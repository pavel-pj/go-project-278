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
	if req.ShortName == "" {
		req.ShortName = "exmpl"
	}

	params := req.ToCreateLinkParams()

	// ВЫЗЫВАЕМ СЕРВИС
	err := h.service.Create(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create link: " + err.Error(),
		})
		return
	}

	c.Status(http.StatusCreated)

}

func (h *LinkHandler) GetAllLinks(c *gin.Context) {
	response, err := h.service.GetAllLinks(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create link: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK,
		response,
	)

}
