package handlers

import (
	"db200/internal/dto"
	s "db200/services"
	"net/http"

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

	// Генерируем короткий URL (если не указан)
	if req.ShortUrl == "" {
		req.ShortUrl = generateShortURL()
	}

	params := req.ToCreateLinkParams()

	// ВЫЗЫВАЕМ СЕРВИС
	link, err := h.service.Create(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create link: " + err.Error(),
		})
		return
	}

	response := dto.FromLink(link)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Link created successfully",
		"data":    response,
	})

}

// Генерация короткого URL (заглушка)
func generateShortURL() string {
	return "https://short.ly/abc123" // TODO: реальная логика
}
