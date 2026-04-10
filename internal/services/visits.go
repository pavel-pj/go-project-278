package services

import (
	"db200/internal/db/generated"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// VisitService обрабатывает бизнес-логику
type VisitService struct {
	queries *generated.Queries
}

// NewLinkService создает сервис
func NewVisitService(queries *generated.Queries) *VisitService {
	return &VisitService{
		queries: queries,
	}
}

// parseRangeParam парсит параметр range и возвращает start, end и флаг наличия
func (s *VisitService) ParseRangeParam(c *gin.Context) (start, end int64, hasRange bool) {
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
func (h *VisitService) CalculatePagination(startRange, endRange int64, hasRange bool, totalCount int64) (offset, limit int32, newStart, newEnd int64) {
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
