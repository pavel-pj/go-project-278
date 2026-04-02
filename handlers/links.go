package handlers

import (
	"database/sql"
	linksdb "db200/internal/db/links"
	"db200/internal/dto"
	r "db200/repositories"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/lib/pq"
)

type LinkHandler struct {
	repository *r.LinkRepository
	validator  *validator.Validate
}

func NewLinkHandler(repository *r.LinkRepository) *LinkHandler {
	return &LinkHandler{
		repository: repository,
		validator:  validator.New(),
	}
}

func (h *LinkHandler) Create(c *gin.Context) {
	var req dto.CreateLinkRequest

	// Валидируем JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			HandleValidationErrors(c, validationErrors)
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request",
		})
		return
	}

	// Валидация через validator
	if err := h.validator.Struct(req); err != nil {
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			HandleValidationErrors(c, validationErrors)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "validation failed",
		})
		return
	}

	// Проверяем уникальность original_url
	_, err := h.repository.GetByOriginalURL(c.Request.Context(), req.OriginalUrl)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"errors": map[string]string{
				"original_url": "this URL already has a shortened version",
			},
		})
		return
	}

	baseSite := os.Getenv("BASE_SITE")
	if baseSite == "" {
		baseSite = "https://base-site.com"
	}

	// Если short_name передан - проверяем уникальность
	if req.ShortName != "" {
		_, err = h.repository.GetLinkByShortName(c.Request.Context(), req.ShortName)
		if err == nil {
			c.JSON(http.StatusConflict, gin.H{
				"errors": map[string]string{
					"short_name": "short name already in use",
				},
			})
			return
		}
	} else {
		// Generate a random short name
		req.ShortName, err = h.repository.GenerateShortName(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to generate unique short name",
			})
			return
		}
	}

	// Убираем trailing slash если есть
	baseSite = strings.TrimSuffix(baseSite, "/")
	req.ShortUrl = fmt.Sprintf("%s/r/%s", baseSite, req.ShortName)

	params := linksdb.CreateLinkParams{
		OriginalUrl: req.OriginalUrl,
		ShortName:   req.ShortName,
		ShortUrl:    req.ShortUrl,
	}

	r, err := h.repository.Create(c.Request.Context(), params)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			if strings.Contains(pqErr.Message, "original_url") {
				c.JSON(http.StatusConflict, gin.H{
					"errors": map[string]string{
						"original_url": "this URL already has a shortened version",
					},
				})
			} else if strings.Contains(pqErr.Message, "short_name") {
				c.JSON(http.StatusConflict, gin.H{
					"errors": map[string]string{
						"short_name": "short name already in use",
					},
				})
			}
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create a new link",
		})
		return
	}

	response := dto.LinkResponse{
		ID:          r.ID,
		OriginalUrl: r.OriginalUrl,
		ShortName:   r.ShortName,
		ShortUrl:    r.ShortUrl,
	}

	c.JSON(http.StatusCreated, response)
}

func (h *LinkHandler) UpdateLink(c *gin.Context) {
	idParam := c.Param("id")
	id64, err := strconv.ParseInt(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid ID format",
			"hint":  "ID must be a number",
		})
		return
	}

	id := int32(id64)

	var req dto.UpdateLinkRequest

	// Валидируем JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			HandleValidationErrors(c, validationErrors)
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request",
		})
		return
	}

	// Дополнительная валидация
	if err := h.validator.Struct(req); err != nil {
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			HandleValidationErrors(c, validationErrors)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "validation failed",
		})
		return
	}

	if req.OriginalUrl != nil {
		// проверка на уникальность адреса
		_, err := h.repository.GetLinkByOriginURLExludedID(c.Request.Context(),
			linksdb.GetLinkByOriginURLExludedIDParams{
				ID:          id,
				OriginalUrl: *req.OriginalUrl,
			})

		if err == nil {
			c.JSON(http.StatusConflict, gin.H{
				"errors": map[string]string{
					"original_url": "this URL already has a shortened version",
				},
			})
			return
		}
	}

	if req.ShortName != nil {
		// проверка на уникальность shortName
		_, err := h.repository.GetLinkByShortNameExludedID(
			c.Request.Context(),
			linksdb.GetLinkByShortNameExcluedeIDParams{
				ID:        id,
				ShortName: *req.ShortName,
			})

		if err == nil {
			c.JSON(http.StatusConflict, gin.H{
				"errors": map[string]string{
					"short_name": "short name already in use",
				},
			})
			return
		}
	}

	// Подготовка параметров для обновления
	updateParams := linksdb.UpdateLinkParams{
		ID: id,
	}

	// Устанавливаем значения только если они были переданы
	if req.OriginalUrl != nil {
		updateParams.OriginalUrl = sql.NullString{
			String: *req.OriginalUrl,
			Valid:  true,
		}
	}

	if req.ShortName != nil {
		updateParams.ShortName = sql.NullString{
			String: *req.ShortName,
			Valid:  true,
		}

		// Генерируем новый short_url
		baseSite := os.Getenv("BASE_SITE")
		if baseSite == "" {
			baseSite = "https://base-site.com"
		}
		baseSite = strings.TrimSuffix(baseSite, "/")

		updateParams.ShortUrl = sql.NullString{
			String: baseSite + "/r/" + *req.ShortName,
			Valid:  true,
		}

	}

	updateLink, err := h.repository.UpdateLink(c.Request.Context(), updateParams)

	if err != nil {
		// Обработка ошибки уникальности
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			if strings.Contains(pqErr.Message, "original_url") {
				c.JSON(http.StatusConflict, gin.H{
					"errors": map[string]string{
						"original_url": "this URL already has a shortened version",
					},
				})
			} else if strings.Contains(pqErr.Message, "short_name") {
				c.JSON(http.StatusConflict, gin.H{
					"errors": map[string]string{
						"short_name": "short name already in use",
					},
				})
			}
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"details": fmt.Sprintf("%s", err),
		})
		return
	}

	response := dto.LinkResponse{
		ID:          updateLink.ID,
		OriginalUrl: updateLink.OriginalUrl,
		ShortName:   updateLink.ShortName,
		ShortUrl:    updateLink.ShortUrl,
	}

	c.JSON(http.StatusOK, response)
}

func (h *LinkHandler) GetAllLinks(c *gin.Context) {

	// Парсим параметры
	startRange, endRange, hasRange := h.parseRangeParam(c)

	// Получаем общее количество
	totalCount, err := h.repository.GetLinksCount(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get total count: " + err.Error(),
		})
		return
	}

	// Корректируем значения
	offset, limit, newStart, newEnd := h.calculatePagination(startRange, endRange, hasRange, totalCount)

	// Устанавливаем заголовок
	contentRange := fmt.Sprintf("links %d-%d/%d", newStart, newEnd, totalCount)

	c.Writer.Header().Set("Content-Range", contentRange)

	// Если limit == 0, возвращаем пустой массив
	if limit == 0 {
		c.JSON(http.StatusOK, []dto.LinkResponse{})
		return
	}

	// Получаем данные
	params := linksdb.GetAllLinksParams{
		Limit:  limit,
		Offset: offset,
	}

	res, err := h.repository.GetAllLinks(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed get links: " + err.Error(),
		})
		return
	}

	// Преобразуем в response DTO
	response := make([]dto.LinkResponse, len(res))
	for i, link := range res {
		response[i] = dto.LinkResponse{
			ID:          link.ID,
			OriginalUrl: link.OriginalUrl,
			ShortName:   link.ShortName,
			ShortUrl:    link.ShortUrl,
		}
	}

	c.JSON(http.StatusOK, response)
}

// parseRangeParam парсит параметр range и возвращает start, end и флаг наличия
func (h *LinkHandler) parseRangeParam(c *gin.Context) (start, end int64, hasRange bool) {
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
func (h *LinkHandler) calculatePagination(startRange, endRange int64, hasRange bool, totalCount int64) (offset, limit int32, newStart, newEnd int64) {
	if hasRange {
		// Если totalCount == 0, возвращаем пустой результат
		if totalCount == 0 {
			return 0, 0, 0, 0
		}

		// Корректируем endRange (не выходим за пределы)
		if endRange >= totalCount {
			endRange = totalCount - 1
		}

		// Проверяем, есть ли данные в диапазоне
		if startRange <= endRange {
			//nolint:gosec
			offset = int32(startRange)
			//nolint:gosec
			limit = int32(endRange - startRange + 1)

			// Для Content-Range используем единичную индексацию (как требует HTTP)
			newStart = startRange + 1
			newEnd = endRange + 1

			return offset, limit, newStart, newEnd
		}

		// Нет данных в диапазоне
		return 0, 0, startRange + 1, 0
	}

	// Без range - дефолтные значения
	if totalCount == 0 {
		return 0, 0, 0, 0
	}

	// По умолчанию берем первые 10 записей
	limit = 10
	if totalCount < 10 {
		//nolint:gosec
		limit = int32(totalCount)
	}

	offset = 0
	newStart = 1
	newEnd = int64(limit)

	return offset, limit, newStart, newEnd
}

func (h *LinkHandler) GetLink(c *gin.Context) {

	idParam := c.Param("id")
	id64, err := strconv.ParseInt(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid ID format",
			"hint":  "ID must be a number",
		})
		return
	}

	id := int32(id64)

	r, err := h.repository.GetLink(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "link not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	response := dto.LinkResponse{
		ID:          r.ID,
		OriginalUrl: r.OriginalUrl,
		ShortName:   r.ShortName,
		ShortUrl:    r.ShortUrl,
	}

	c.JSON(http.StatusOK,
		response,
	)

}

func (h *LinkHandler) DeleteLink(c *gin.Context) {

	idParam := c.Param("id")
	id64, err := strconv.ParseInt(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid ID format",
			"hint":  "ID must be a number",
		})
		return
	}

	id := int32(id64)

	_, err = h.repository.GetLink(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Link not found",
				"code":    "NOT_FOUND",
				"details": fmt.Sprintf("Link with ID %d does not exist", id),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to check link existence",
		})
		return
	}

	err = h.repository.DeleteLink(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)

}
