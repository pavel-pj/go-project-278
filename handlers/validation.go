package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// HandleValidationErrors обрабатывает ошибки валидации и возвращает единый формат
func HandleValidationErrors(c *gin.Context, err error) {
	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		errorsMap := make(map[string]string)

		for _, ve := range validationErrors {
			field := ve.Field()
			// Преобразуем camelCase в snake_case для JSON
			jsonField := toSnakeCase(field)

			switch ve.Tag() {
			case "required":
				errorsMap[jsonField] = field + " is required"
			case "url":
				errorsMap[jsonField] = field + " must be a valid URL"
			case "min":
				errorsMap[jsonField] = field + " must be at least " + ve.Param() + " characters"
			case "max":
				errorsMap[jsonField] = field + " must be at most " + ve.Param() + " characters"
			case "alphanum":
				errorsMap[jsonField] = field + " must contain only alphanumeric characters"
			default:
				errorsMap[jsonField] = field + " is invalid"
			}
		}
		/*
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"errors": errorsMap,
			})
		*/
		c.JSON(http.StatusEarlyHints, gin.H{
			"errors": errorsMap,
		})

		c.Abort()
		return
	}

	// Если это не ошибка валидации
	c.JSON(http.StatusBadRequest, gin.H{
		"error": "invalid request",
	})
	c.Abort()
}

// toSnakeCase конвертирует CamelCase в snake_case
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}
