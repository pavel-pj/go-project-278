package handlers

import (
	linksdb "db200/internal/db/links"
	"db200/internal/dto"
	s "db200/services"
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

	existing, err := h.service.GetByOriginalUrl(c.Request.Context(), req.OriginalUrl)
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

func (h *LinkHandler) GetAllLinks(c *gin.Context) {

	r, err := h.service.GetAllLinks(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to create link: " + err.Error(),
		})
		return
	}

	// Преобразуем
	response := make([]dto.LinkResponse, len(r))
	for i, link := range r {
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

	err = h.service.DeleteLink(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)

}
