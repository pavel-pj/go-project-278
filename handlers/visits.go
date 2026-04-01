package handlers

import (
	visitsdb "db200/internal/db/visits"
	r "db200/repositories"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type VisitHandler struct {
	VisitRepository *r.VisitRepository
	LinkRepository  *r.LinkRepository
}

func NewVisitHandler(visitRepository *r.VisitRepository, linkRepository *r.LinkRepository) *VisitHandler {
	return &VisitHandler{
		LinkRepository:  linkRepository,
		VisitRepository: visitRepository,
	}
}

func (h *VisitHandler) Redirect(c *gin.Context) {

	shortName := c.Param("code")
	if shortName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "short name is required"})
		return
	}

	// Get link by short name
	link, err := h.LinkRepository.GetLinkByShortName(c.Request.Context(), shortName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
		return
	}

	// Get client information
	ip := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	referer := c.GetHeader("Referer")

	// Determine redirect status (302 Found for temporary redirect)
	status := http.StatusFound // 302

	// Record visit
	_, err = h.VisitRepository.Create(c.Request.Context(), visitsdb.CreateVisitParams{
		LinkID:    link.ID,
		Ip:        ip,
		UserAgent: userAgent,
		Status:    int32(status),
		Referer:   referer,
	})
	if err != nil {
		// Log error but still redirect
		_ = c.Error(err) // игнорируем ошибку, так как редирект важнее
	}

	// Perform redirect
	c.Redirect(status, link.OriginalUrl)
}

func (h *VisitHandler) GetVisits(c *gin.Context) {
	// Парсим параметры
	startRange, endRange, hasRange := h.parseRangeParam(c)

	// Получаем общее количество
	totalCount, err := h.VisitRepository.GetVisitsCount(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get total count: " + err.Error(),
		})
		return
	}

	// Корректируем значения
	offset, limit, newStart, newEnd := h.calculatePagination(startRange, endRange, hasRange, totalCount)

	// Устанавливаем заголовок
	contentRange := fmt.Sprintf("visits %d-%d/%d", newStart, newEnd, totalCount)
	c.Writer.Header().Set("Content-Range", contentRange)

	// Если limit == 0, возвращаем пустой массив
	if limit == 0 {
		c.JSON(http.StatusOK, []visitsdb.LinkVisit{})
		return
	}

	// Получаем данные
	params := visitsdb.GetVisitsParams{
		Limit:  limit,
		Offset: offset,
	}

	res, err := h.VisitRepository.GetVisits(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed get visits: " + err.Error(),
		})
		return
	}

	// Преобразуем в response DTO
	response := make([]visitsdb.LinkVisit, len(res))
	for i, visit := range res {
		response[i] = visitsdb.LinkVisit{
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

// parseRangeParam парсит параметр range и возвращает start, end и флаг наличия
func (h *VisitHandler) parseRangeParam(c *gin.Context) (start, end int64, hasRange bool) {
	rangeParam := c.Query("range")
	if rangeParam == "" {
		return 0, 9, false // default: первые 10 записей (0-9)
	}

	trimmed := strings.Trim(rangeParam, "[]")
	parts := strings.Split(trimmed, ",")

	if len(parts) != 2 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid range format, expected [start,end]",
		})
		return 0, 0, true
	}

	var err1, err2 error
	start, err1 = strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	end, err2 = strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)

	if err1 != nil || err2 != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid range values, expected numbers",
		})
		return 0, 0, true
	}

	if start < 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "start must be >= 0",
		})
		return 0, 0, true
	}

	if end <= start {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "end must be greater than start",
		})
		return 0, 0, true
	}

	return start, end, true
}

// calculatePagination вычисляет offset, limit и скорректированные start/end
func (h *VisitHandler) calculatePagination(startRange, endRange int64, hasRange bool, totalCount int64) (offset, limit int32, newStart, newEnd int64) {
	if hasRange {
		if totalCount == 0 {
			return 0, 0, 0, 0
		}

		if endRange >= totalCount {
			endRange = totalCount - 1
		}

		if startRange <= endRange {
			//nolint:gosec // G115: startRange не может превышать int32, так как это индекс в БД
			offset = int32(startRange)

			limitVal := endRange - startRange + 1
			//nolint:gosec // G115: limitVal не может превышать int32, так как это количество записей в диапазоне
			limit = int32(limitVal)

			newStart = startRange + 1
			newEnd = endRange + 1

			return offset, limit, newStart, newEnd
		}

		return 0, 0, startRange + 1, 0
	}

	if totalCount == 0 {
		return 0, 0, 0, 0
	}

	limit = 10
	if totalCount < 10 {
		//nolint:gosec // G115: totalCount не может превышать int32, так как это количество записей в БД
		limit = int32(totalCount)
	}

	offset = 0
	newStart = 1
	newEnd = int64(limit)

	return offset, limit, newStart, newEnd
}
