package handlers

import (
	"database/sql"
	"db200/internal/db/generated"
	"db200/internal/dto"
	"errors"
	"os"
	"strconv"
	"strings"

	s "db200/internal/services"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/lib/pq"
)

type LinkHandler struct {
	queries   *generated.Queries
	validator *validator.Validate
	service   *s.LinkService
}

func NewLinkHandler(queries *generated.Queries, service *s.LinkService) *LinkHandler {
	return &LinkHandler{
		queries:   queries,
		validator: validator.New(),
		service:   s.NewLinkService(queries),
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
	_, err := h.queries.GetLinkByOriginURL(c.Request.Context(), req.OriginalUrl)
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
		_, err = h.queries.GetLinkByShortName(c.Request.Context(), req.ShortName)
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
		req.ShortName, err = h.service.GenerateShortName(c.Request.Context())
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

	params := generated.CreateLinkParams{
		OriginalUrl: req.OriginalUrl,
		ShortName:   req.ShortName,
		ShortUrl:    req.ShortUrl,
	}

	r, err := h.queries.CreateLink(c.Request.Context(), params)
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

	_, err = h.queries.GetLink(c.Request.Context(), id)
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
		_, err := h.queries.GetLinkByOriginURLExludedID(c.Request.Context(),
			generated.GetLinkByOriginURLExludedIDParams{
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
		_, err := h.queries.GetLinkByShortNameExcludeID(
			c.Request.Context(),
			generated.GetLinkByShortNameExcludeIDParams{
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

	// Подготовка параметров для обновления - ИСПРАВЛЕНО!
	updateParams := generated.UpdateLinkParams{
		ID:          id,
		OriginalUrl: pgtype.Text{Valid: false}, // ← Используем pgtype.Text
		ShortName:   pgtype.Text{Valid: false}, // ← Используем pgtype.Text
		ShortUrl:    pgtype.Text{Valid: false}, // ← Добавляем ShortUrl
	}

	// Устанавливаем значения только если они были переданы
	if req.OriginalUrl != nil {
		updateParams.OriginalUrl = pgtype.Text{
			String: *req.OriginalUrl,
			Valid:  true,
		}
	}

	if req.ShortName != nil {
		updateParams.ShortName = pgtype.Text{
			String: *req.ShortName,
			Valid:  true,
		}

		// Генерируем новый short_url
		baseSite := os.Getenv("BASE_SITE")
		if baseSite == "" {
			baseSite = "https://base-site.com"
		}
		baseSite = strings.TrimSuffix(baseSite, "/")

		updateParams.ShortUrl = pgtype.Text{
			String: baseSite + "/r/" + *req.ShortName,
			Valid:  true,
		}
	}

	updateLink, err := h.queries.UpdateLink(c.Request.Context(), updateParams)

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
	startRange, endRange, hasRange := h.service.ParseRangeParam(c)

	// Получаем общее количество
	totalCount, err := h.queries.GetLinksCount(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get total count: " + err.Error(),
		})
		return
	}

	// Корректируем значения
	offset, limit, newStart, newEnd := h.service.CalculatePagination(startRange, endRange, hasRange, totalCount)

	// Устанавливаем заголовок
	contentRange := fmt.Sprintf("links %d-%d/%d", newStart, newEnd, totalCount)

	c.Writer.Header().Set("Content-Range", contentRange)

	// Если limit == 0, возвращаем пустой массив
	if limit == 0 {
		c.JSON(http.StatusOK, []dto.LinkResponse{})
		return
	}

	// Получаем данные
	params := generated.GetAllLinksParams{
		Limit:  limit,
		Offset: offset,
	}

	res, err := h.queries.GetAllLinks(c.Request.Context(), params)
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

	r, err := h.queries.GetLink(c.Request.Context(), id)
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

	_, err = h.queries.GetLink(c.Request.Context(), id)
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

	err = h.queries.DeleteLink(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)

}
