package handlers

import (
	"db200/internal/dto"
	s "db200/services"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	gonanoid "github.com/matoous/go-nanoid/v2"
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
	shortNameByUser = req.ShortName

	// Если пользователь не указал ShortName - генерируем
	if req.ShortName == "" {
		id, err := gonanoid.Generate("abcdefghijklmnopqrstuvwxyz0123456789", 8)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to generate short URL",
			})
			return
		}
		req.ShortName = id
	}

	baseSite := os.Getenv("BASE_SITE")
	if baseSite == "" {
		baseSite = "https://base-site.com"
	}
	// Убираем trailing slash если есть
	baseSite = strings.TrimSuffix(baseSite, "/")
	req.ShortUrl = fmt.Sprintf("%s/%s", baseSite, req.ShortName)

	params := req.ToCreateLinkParams()

	// Пытаемся создать ссылку (до 10 попыток для уникальности)
	var lastError error
	for attempt := 0; attempt < 10; attempt++ {
		err := h.service.Create(c.Request.Context(), params)
		if err == nil {
			// Успех!
			c.Status(http.StatusCreated)
			return
		}

		// Сохраняем ошибку
		lastError = err

		// Проверяем тип ошибки
		if isDuplicateKeyError(err) {
			// Если пользователь указал ShortName и он неуникальный
			if shortNameByUser != "" {
				c.JSON(http.StatusConflict, gin.H{
					"error":   "Short name already taken",
					"code":    "DUPLICATE_SHORT_NAME",
					"details": "Please choose a different short name",
				})
				return
			}

			// Если автоматически сгенерированный неуникальный - генерируем новый
			newShortName, genErr := gonanoid.Generate("abcdefghijklmnopqrstuvwxyz0123456789", 8)
			if genErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Failed to generate unique short name",
				})
				return
			}

			// Обновляем параметры для следующей попытки
			req.ShortName = newShortName
			req.ShortUrl = fmt.Sprintf("%s/%s", baseSite, newShortName)
			params = req.ToCreateLinkParams()

			// Продолжаем цикл
			continue
		}

		// Если не ошибка дубликата - выходим сразу
		break
	}

	// Если дошли сюда - все попытки неудачны
	c.JSON(http.StatusInternalServerError, gin.H{
		"error":   "Failed to create link after 10 attempts",
		"details": lastError.Error(),
	})

}

func (h *LinkHandler) GetAllLinks(c *gin.Context) {
	response, err := h.service.GetAllLinks(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to create link: " + err.Error(),
		})
		return
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

	response, err := h.service.GetLink(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
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
