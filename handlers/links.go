package handlers

import (
	"database/sql"
	linksdb "db200/internal/db/links"
	"db200/internal/dto"
	s "db200/services"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type LinkHandler struct {
	service *s.LinkService
}

func NewLinkHandler(service *s.LinkService) *LinkHandler {
	return &LinkHandler{
		service: service,
	}
}

func (h *LinkHandler) Create(c *gin.Context) {
	var req dto.CreateLinkRequest

	// Валидируем JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	existing, err := h.service.GetByOriginalURL(c.Request.Context(), req.OriginalUrl)
	if err == nil {
		// No error means URL was found (exists in DB)
		c.JSON(http.StatusConflict, gin.H{
			"error":   "This URL already has a shortened version",
			"code":    "DUPLICATE_ORIGINAL_URL",
			"details": fmt.Sprintf("Short URL: %s", existing.ShortUrl),
		})
		return
	}

	baseSite := os.Getenv("BASE_SITE")
	if baseSite == "" {
		baseSite = "https://base-site.com"
	}

	if req.ShortName != "" {
		_, err = h.service.GetLinkByShortName(c.Request.Context(), req.ShortName)
		if err == nil {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "20-This ShortName already has used for other URL",
				"code":    "DUPLICATE_SHORT_NAME_URL",
				"details": fmt.Sprintf("Short name: %s", req.ShortName),
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
	req.ShortUrl = fmt.Sprintf("%s/%s", baseSite, req.ShortName)

	params := linksdb.CreateLinkParams{
		OriginalUrl: req.OriginalUrl,
		ShortName:   req.ShortName,
		ShortUrl:    req.ShortUrl,
	}

	r, err := h.service.Create(c.Request.Context(), params)
	if err != nil {
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
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	if req.OriginalUrl != nil {
		// проверка на уникальность адреса
		existing, err := h.service.GetLinkByOriginURLExludedID(c.Request.Context(),
			linksdb.GetLinkByOriginURLExludedIDParams{
				ID:          id,
				OriginalUrl: *req.OriginalUrl,
			})

		if err == nil {
			// No error means URL was found (exists in DB)
			c.JSON(http.StatusConflict, gin.H{
				"error":   "This URL already has a shortened version",
				"code":    "DUPLICATE_ORIGINAL_URL",
				"details": fmt.Sprintf("Short URL: %s", existing.ShortUrl),
			})
			return
		}
	}
	if req.ShortName != nil {
		// проверка на уникальность shortName
		existing, err := h.service.GetLinkByShortNameExludedID(
			c.Request.Context(),
			linksdb.GetLinkByShortNameExcluedeIDParams{
				ID:        id,
				ShortName: *req.ShortName,
			})

		if err == nil {
			// No error means URL was found (exists in DB)
			c.JSON(http.StatusConflict, gin.H{
				"error":   "This SHORT NAME already has a shortened version",
				"code":    "DUPLICATE_SHORT_NAME",
				"details": fmt.Sprintf("Short URL: %s", existing.ShortUrl),
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
			String: baseSite + "/" + *req.ShortName,
			Valid:  true,
		}
	}

	updateLink, err := h.service.UpdateLink(c.Request.Context(), updateParams)

	if err != nil {
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

	c.JSON(http.StatusOK,
		response,
	)

}

func (h *LinkHandler) GetAllLinks(c *gin.Context) {
	var req dto.AllLinkRequest

	var offset int32
	var limit int32
	offset = 0
	limit = 10
	//Если есть параметр range
	req.Range = c.Query("range")
	if req.Range != "" {
		// ПАРАМЕТР ЕСТЬ - парсим его
		trimmed := strings.Trim(req.Range, "[]")
		parts := strings.Split(trimmed, ",")

		if len(parts) != 2 {
			// Ошибка: неправильный формат
			c.JSON(http.StatusInternalServerError, gin.H{
				"invalid range format": req.Range,
			})
		}

		start, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		end, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))

		if err1 != nil || err2 != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid range values",
			})
			return
		}

		if start < 0 || end < 0 || end <= start {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid range values",
			})
			return
		}

		offset = int32(start - 1)
		limit = int32(end - start + 1)

	}

	params := linksdb.GetAllLinksParams{
		Limit:  limit,
		Offset: offset,
	}

	res, err := h.service.GetAllLinks(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed get link: " + err.Error(),
		})
		return
	}

	// Преобразуем
	response := make([]dto.LinkResponse, len(res))
	for i, link := range res {
		response[i] = dto.LinkResponse{
			ID:          link.ID,
			OriginalUrl: link.OriginalUrl,
			ShortName:   link.ShortName,
			ShortUrl:    link.ShortUrl,
		}
	}

	c.JSON(http.StatusOK,
		response,
	)

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

	r, err := h.service.GetLink(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
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

	_, err = h.service.GetLink(c.Request.Context(), id)
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

	err = h.service.DeleteLink(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)

}
