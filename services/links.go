package services

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// LinkService обрабатывает бизнес-логику для ссылок
type LinkService struct {
}

// NewLinkService создает сервис
func NewLinkService() *LinkService {
	return &LinkService{}
}

// parseRangeParam парсит параметр range и возвращает start, end и флаг наличия
func (s *LinkService) ParseRangeParam(c *gin.Context) (start, end int64, hasRange bool) {
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
func (h *LinkService) CalculatePagination(startRange, endRange int64, hasRange bool, totalCount int64) (offset, limit int32, newStart, newEnd int64) {
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
