package handlers

import (
	"db200/internal/db/generated"
	s "db200/internal/services"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type VisitHandler struct {
	queries *generated.Queries
	service *s.VisitService
}

func NewVisitHandler(queries *generated.Queries) *VisitHandler {
	return &VisitHandler{
		queries: queries,
		service: s.NewVisitService(queries),
	}
}

func (h *VisitHandler) Redirect(c *gin.Context) {

	shortName := c.Param("code")
	if shortName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "short name is required"})
		return
	}

	// Получили ссылку по short name
	link, err := h.queries.GetLinkByShortName(c.Request.Context(), shortName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
		return
	}

	// Клиентская информация
	ip := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	referer := c.GetHeader("Referer")

	//  302 статус
	status := http.StatusFound

	// создаем визит
	_, err = h.queries.CreateVisit(c.Request.Context(), generated.CreateVisitParams{
		LinkID:    link.ID,
		Ip:        ip,
		UserAgent: userAgent,
		Status:    int32(status),
		Referer:   referer,
	})
	if err != nil {
		// игнорируем ошибку, так как редирект важнее
		_ = c.Error(err)
	}

	c.Redirect(status, link.OriginalUrl)
}

func (h *VisitHandler) GetVisits(c *gin.Context) {
	// Парсим параметры
	startRange, endRange, hasRange := h.service.ParseRangeParam(c)

	// Получаем общее количество
	totalCount, err := h.queries.GetVisitsCount(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get total count: " + err.Error(),
		})
		return
	}

	// Корректируем значения
	offset, limit, newStart, newEnd := h.service.CalculatePagination(startRange, endRange, hasRange, totalCount)

	// Устанавливаем заголовок
	contentRange := fmt.Sprintf("visits %d-%d/%d", newStart, newEnd, totalCount)
	c.Writer.Header().Set("Content-Range", contentRange)

	// Если limit == 0, возвращаем пустой массив
	if limit == 0 {
		c.JSON(http.StatusOK, []generated.LinkVisit{})
		return
	}

	// Получаем данные
	params := generated.GetVisitsParams{
		Limit:  limit,
		Offset: offset,
	}

	res, err := h.queries.GetVisits(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed get visits: " + err.Error(),
		})
		return
	}

	// Преобразуем в response DTO
	response := make([]generated.LinkVisit, len(res))
	for i, visit := range res {
		response[i] = generated.LinkVisit{
			ID:        visit.ID,
			LinkID:    visit.LinkID,
			CreatedAt: visit.CreatedAt,
			Ip:        visit.Ip,
			UserAgent: visit.UserAgent,
			Status:    visit.Status,
			Referer:   visit.Referer,
		}
	}

	c.JSON(http.StatusOK, response)
}
