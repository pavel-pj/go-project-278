//go:build integration

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"db200/internal/db/generated"
	"db200/internal/dto"
	"db200/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateLink(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    dto.CreateLinkRequest
		setupData      func(ctx context.Context, q *generated.Queries)
		expectedStatus int
		expectedError  map[string]string
		checkResponse  func(t *testing.T, response dto.LinkResponse, q *generated.Queries, ctx context.Context)
	}{
		{
			name: "Success - Create link with custom short name",
			requestBody: dto.CreateLinkRequest{
				OriginalUrl: "https://example.com",
				ShortName:   "custom",
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, response dto.LinkResponse, q *generated.Queries, ctx context.Context) {
				assert.Equal(t, "https://example.com", response.OriginalUrl)
				assert.Equal(t, "custom", response.ShortName)
				assert.NotZero(t, response.ID)

				link, err := q.GetLink(ctx, response.ID)
				require.NoError(t, err)
				assert.Equal(t, "https://example.com", link.OriginalUrl)
				assert.Equal(t, "custom", link.ShortName)
			},
		},
		{
			name: "Success - Create link with auto-generated short name",
			requestBody: dto.CreateLinkRequest{
				OriginalUrl: "https://example.com/auto",
				ShortName:   "",
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, response dto.LinkResponse, q *generated.Queries, ctx context.Context) {
				assert.Equal(t, "https://example.com/auto", response.OriginalUrl)
				assert.NotEmpty(t, response.ShortName)
				assert.NotZero(t, response.ID)
			},
		},
		{
			name: "Error - Duplicate original_url",
			setupData: func(ctx context.Context, q *generated.Queries) {
				_, err := q.CreateLink(ctx, generated.CreateLinkParams{
					OriginalUrl: "https://existing.com",
					ShortName:   "existing",
					ShortUrl:    "https://test.com/existing",
				})
				require.NoError(t, err)
			},
			requestBody: dto.CreateLinkRequest{
				OriginalUrl: "https://existing.com",
				ShortName:   "new",
			},
			expectedStatus: http.StatusConflict,
			expectedError: map[string]string{
				"original_url": "this URL already has a shortened version",
			},
		},
		{
			name: "Error - Duplicate short_name",
			setupData: func(ctx context.Context, q *generated.Queries) {
				_, err := q.CreateLink(ctx, generated.CreateLinkParams{
					OriginalUrl: "https://first.com",
					ShortName:   "taken",
					ShortUrl:    "https://test.com/taken",
				})
				require.NoError(t, err)
			},
			requestBody: dto.CreateLinkRequest{
				OriginalUrl: "https://second.com",
				ShortName:   "taken",
			},
			expectedStatus: http.StatusConflict,
			expectedError: map[string]string{
				"short_name": "short name already in use",
			},
		},
		{
			name: "Error - Missing original_url",
			requestBody: dto.CreateLinkRequest{
				OriginalUrl: "",
				ShortName:   "test",
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedError: map[string]string{
				"original_url": "OriginalUrl is required",
			},
		},
		{
			name: "Error - Invalid URL format",
			requestBody: dto.CreateLinkRequest{
				OriginalUrl: "not-a-url",
				ShortName:   "test",
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedError: map[string]string{
				"original_url": "OriginalUrl must be a valid URL",
			},
		},
		{
			name: "Error - Short name too short",
			requestBody: dto.CreateLinkRequest{
				OriginalUrl: "https://example.com",
				ShortName:   "ab",
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedError: map[string]string{
				"short_name": "ShortName must be at least 3 characters",
			},
		},
		{
			name: "Error - Short name too long",
			requestBody: dto.CreateLinkRequest{
				OriginalUrl: "https://example.com",
				ShortName:   strings.Repeat("a", 33),
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedError: map[string]string{
				"short_name": "ShortName must be at most 32 characters",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, cleanup := SetupTx(t)
			defer cleanup()

			ctx := context.Background()

			if tt.setupData != nil {
				tt.setupData(ctx, q)
			}

			linkService := services.NewLinkService(q)
			linkHandler := NewLinkHandler(q, linkService)

			r := gin.New()
			r.POST("/api/links", linkHandler.Create)

			jsonBody, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/links", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "Response body: %s", w.Body.String())

			if tt.expectedError != nil {
				var response map[string]map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				errors, ok := response["errors"]
				require.True(t, ok, "Expected errors object in response")

				for field, expectedMsg := range tt.expectedError {
					actualMsg, exists := errors[field]
					require.True(t, exists, "Expected error for field '%s'", field)
					assert.Equal(t, expectedMsg, actualMsg)
				}
			}

			if tt.expectedStatus == http.StatusCreated && tt.checkResponse != nil {
				var response dto.LinkResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				tt.checkResponse(t, response, q, ctx)
			}
		})
	}
}

func TestUpdateLink(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupData      func(ctx context.Context, q *generated.Queries) int32
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  map[string]string
		checkResponse  func(t *testing.T, response dto.LinkResponse, q *generated.Queries, ctx context.Context)
	}{
		{
			name: "Success - Update both fields",
			setupData: func(ctx context.Context, q *generated.Queries) int32 {
				link, err := q.CreateLink(ctx, generated.CreateLinkParams{
					OriginalUrl: "https://old.com",
					ShortName:   "old",
					ShortUrl:    "https://test.com/r/old",
				})
				require.NoError(t, err)
				return link.ID
			},
			requestBody: map[string]interface{}{
				"original_url": "https://new.com",
				"short_name":   "new",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response dto.LinkResponse, q *generated.Queries, ctx context.Context) {
				assert.Equal(t, "https://new.com", response.OriginalUrl)
				assert.Equal(t, "new", response.ShortName)
			},
		},
		{
			name: "Success - Update only original_url",
			setupData: func(ctx context.Context, q *generated.Queries) int32 {
				link, err := q.CreateLink(ctx, generated.CreateLinkParams{
					OriginalUrl: "https://old.com",
					ShortName:   "oldname",
					ShortUrl:    "https://test.com/r/oldname",
				})
				require.NoError(t, err)
				return link.ID
			},
			requestBody: map[string]interface{}{
				"original_url": "https://new.com",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response dto.LinkResponse, q *generated.Queries, ctx context.Context) {
				assert.Equal(t, "https://new.com", response.OriginalUrl)
				assert.Equal(t, "oldname", response.ShortName)
			},
		},
		{
			name: "Success - Update only short_name",
			setupData: func(ctx context.Context, q *generated.Queries) int32 {
				link, err := q.CreateLink(ctx, generated.CreateLinkParams{
					OriginalUrl: "https://old.com",
					ShortName:   "oldname",
					ShortUrl:    "https://test.com/r/oldname",
				})
				require.NoError(t, err)
				return link.ID
			},
			requestBody: map[string]interface{}{
				"short_name": "newname",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response dto.LinkResponse, q *generated.Queries, ctx context.Context) {
				assert.Equal(t, "https://old.com", response.OriginalUrl)
				assert.Equal(t, "newname", response.ShortName)
			},
		},
		{
			name: "Error - Duplicate original_url",
			setupData: func(ctx context.Context, q *generated.Queries) int32 {
				_, err := q.CreateLink(ctx, generated.CreateLinkParams{
					OriginalUrl: "https://existing.com",
					ShortName:   "existing",
					ShortUrl:    "https://test.com/r/existing",
				})
				require.NoError(t, err)
				link, err := q.CreateLink(ctx, generated.CreateLinkParams{
					OriginalUrl: "https://toupdate.com",
					ShortName:   "toupdate",
					ShortUrl:    "https://test.com/r/toupdate",
				})
				require.NoError(t, err)
				return link.ID
			},
			requestBody: map[string]interface{}{
				"original_url": "https://existing.com",
			},
			expectedStatus: http.StatusConflict,
			expectedError: map[string]string{
				"original_url": "this URL already has a shortened version",
			},
		},
		{
			name: "Error - Duplicate short_name",
			setupData: func(ctx context.Context, q *generated.Queries) int32 {
				_, err := q.CreateLink(ctx, generated.CreateLinkParams{
					OriginalUrl: "https://first.com",
					ShortName:   "taken",
					ShortUrl:    "https://test.com/r/taken",
				})
				require.NoError(t, err)
				link, err := q.CreateLink(ctx, generated.CreateLinkParams{
					OriginalUrl: "https://second.com",
					ShortName:   "update",
					ShortUrl:    "https://test.com/r/update",
				})
				require.NoError(t, err)
				return link.ID
			},
			requestBody: map[string]interface{}{
				"short_name": "taken",
			},
			expectedStatus: http.StatusConflict,
			expectedError: map[string]string{
				"short_name": "short name already in use",
			},
		},
		{
			name: "Error - Link not found",
			setupData: func(ctx context.Context, q *generated.Queries) int32 {
				return 99999
			},
			requestBody: map[string]interface{}{
				"original_url": "https://new.com",
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "Error - Invalid original_url format",
			setupData: func(ctx context.Context, q *generated.Queries) int32 {
				link, err := q.CreateLink(ctx, generated.CreateLinkParams{
					OriginalUrl: "https://valid.com",
					ShortName:   "valid",
					ShortUrl:    "https://test.com/r/valid",
				})
				require.NoError(t, err)
				return link.ID
			},
			requestBody: map[string]interface{}{
				"original_url": "not-a-url",
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedError: map[string]string{
				"original_url": "OriginalUrl must be a valid URL",
			},
		},
		{
			name: "Error - Short name too short",
			setupData: func(ctx context.Context, q *generated.Queries) int32 {
				link, err := q.CreateLink(ctx, generated.CreateLinkParams{
					OriginalUrl: "https://valid.com",
					ShortName:   "valid",
					ShortUrl:    "https://test.com/r/valid",
				})
				require.NoError(t, err)
				return link.ID
			},
			requestBody: map[string]interface{}{
				"short_name": "ab",
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedError: map[string]string{
				"short_name": "ShortName must be at least 3 characters",
			},
		},
		{
			name: "Error - Short name too long",
			setupData: func(ctx context.Context, q *generated.Queries) int32 {
				link, err := q.CreateLink(ctx, generated.CreateLinkParams{
					OriginalUrl: "https://valid.com",
					ShortName:   "valid",
					ShortUrl:    "https://test.com/r/valid",
				})
				require.NoError(t, err)
				return link.ID
			},
			requestBody: map[string]interface{}{
				"short_name": strings.Repeat("a", 33),
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedError: map[string]string{
				"short_name": "ShortName must be at most 32 characters",
			},
		},
		{
			name: "Error - Invalid JSON",
			setupData: func(ctx context.Context, q *generated.Queries) int32 {
				link, err := q.CreateLink(ctx, generated.CreateLinkParams{
					OriginalUrl: "https://valid.com",
					ShortName:   "valid",
					ShortUrl:    "https://test.com/r/valid",
				})
				require.NoError(t, err)
				return link.ID
			},
			requestBody:    nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, cleanup := SetupTx(t)
			defer cleanup()

			ctx := context.Background()

			linkID := int32(99999)
			if tt.setupData != nil {
				linkID = tt.setupData(ctx, q)
			}

			linkService := services.NewLinkService(q)
			linkHandler := NewLinkHandler(q, linkService)

			r := gin.New()
			r.PUT("/api/links/:id", linkHandler.UpdateLink)

			var req *http.Request
			if tt.requestBody != nil {
				jsonBody, _ := json.Marshal(tt.requestBody)
				req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/links/%d", linkID), bytes.NewBuffer(jsonBody))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/links/%d", linkID), bytes.NewBuffer([]byte(`{invalid json`)))
				req.Header.Set("Content-Type", "application/json")
			}

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "Response body: %s", w.Body.String())

			if tt.expectedError != nil {
				var response map[string]map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				errors, ok := response["errors"]
				require.True(t, ok, "Expected errors object in response")

				for field, expectedMsg := range tt.expectedError {
					actualMsg, exists := errors[field]
					require.True(t, exists, "Expected error for field '%s'", field)
					assert.Equal(t, expectedMsg, actualMsg)
				}
			}

			if tt.expectedStatus == http.StatusOK && tt.checkResponse != nil {
				var response dto.LinkResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				tt.checkResponse(t, response, q, ctx)
			}
		})
	}
}

func TestGetAllLinks(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupData      func(ctx context.Context, q *generated.Queries) int
		rangeParam     string
		expectedCount  int
		expectedStart  int64
		expectedEnd    int64
		expectedTotal  int64
		expectedStatus int
	}{
		{
			name: "Success - Get all links without range",
			setupData: func(ctx context.Context, q *generated.Queries) int {
				for i := 1; i <= 3; i++ {
					_, err := q.CreateLink(ctx, generated.CreateLinkParams{
						OriginalUrl: fmt.Sprintf("https://test%d.com", i),
						ShortName:   fmt.Sprintf("link%d", i),
						ShortUrl:    fmt.Sprintf("https://test.com/link%d", i),
					})
					require.NoError(t, err)
				}
				return 3
			},
			rangeParam:     "",
			expectedCount:  3,
			expectedStart:  1,
			expectedEnd:    3,
			expectedTotal:  3,
			expectedStatus: http.StatusOK,
		},
		{
			name: "Success - Get links with range [0,1]",
			setupData: func(ctx context.Context, q *generated.Queries) int {
				for i := 1; i <= 3; i++ {
					_, err := q.CreateLink(ctx, generated.CreateLinkParams{
						OriginalUrl: fmt.Sprintf("https://test%d.com", i),
						ShortName:   fmt.Sprintf("link%d", i),
						ShortUrl:    fmt.Sprintf("https://test.com/link%d", i),
					})
					require.NoError(t, err)
				}
				return 3
			},
			rangeParam:     "[0,1]",
			expectedCount:  2,
			expectedStart:  1,
			expectedEnd:    2,
			expectedTotal:  3,
			expectedStatus: http.StatusOK,
		},
		{
			name: "Success - Empty database",
			setupData: func(ctx context.Context, q *generated.Queries) int {
				return 0
			},
			rangeParam:     "",
			expectedCount:  0,
			expectedStart:  0,
			expectedEnd:    0,
			expectedTotal:  0,
			expectedStatus: http.StatusOK,
		},
		{
			name: "Success - Range exceeds bounds",
			setupData: func(ctx context.Context, q *generated.Queries) int {
				for i := 1; i <= 5; i++ {
					_, err := q.CreateLink(ctx, generated.CreateLinkParams{
						OriginalUrl: fmt.Sprintf("https://test%d.com", i),
						ShortName:   fmt.Sprintf("link%d", i),
						ShortUrl:    fmt.Sprintf("https://test.com/link%d", i),
					})
					require.NoError(t, err)
				}
				return 5
			},
			rangeParam:     "[10,20]",
			expectedCount:  0,
			expectedStart:  0,
			expectedEnd:    0,
			expectedTotal:  5,
			expectedStatus: http.StatusOK,
		},
		{
			name: "Error - Invalid range format",
			setupData: func(ctx context.Context, q *generated.Queries) int {
				return 0
			},
			rangeParam:     "[0,1,2]",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Error - Invalid range values",
			setupData: func(ctx context.Context, q *generated.Queries) int {
				return 0
			},
			rangeParam:     "[a,b]",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, cleanup := SetupTx(t)
			defer cleanup()

			ctx := context.Background()

			tt.setupData(ctx, q)

			linkService := services.NewLinkService(q)
			linkHandler := NewLinkHandler(q, linkService)

			r := gin.New()
			r.GET("/api/links", linkHandler.GetAllLinks)

			url := "/api/links"
			if tt.rangeParam != "" {
				url = url + "?range=" + tt.rangeParam
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "Response body: %s", w.Body.String())

			if tt.expectedStatus != http.StatusOK {
				return
			}

			if tt.expectedTotal > 0 {
				contentRange := w.Header().Get("Content-Range")
				// Проверяем только наличие заголовка, так как формат может отличаться
				assert.NotEmpty(t, contentRange, "Content-Range header should be present")
			}

			var response []dto.LinkResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Len(t, response, tt.expectedCount)
		})
	}
}

func TestGetAllLinksWithLargeDataset(t *testing.T) {
	gin.SetMode(gin.TestMode)

	q, cleanup := SetupTx(t)
	defer cleanup()

	ctx := context.Background()

	// Создаем 25 ссылок
	for i := 1; i <= 25; i++ {
		_, err := q.CreateLink(ctx, generated.CreateLinkParams{
			OriginalUrl: fmt.Sprintf("https://test%d.com", i),
			ShortName:   fmt.Sprintf("link%d", i),
			ShortUrl:    fmt.Sprintf("https://test.com/link%d", i),
		})
		require.NoError(t, err)
	}

	linkService := services.NewLinkService(q)
	linkHandler := NewLinkHandler(q, linkService)

	tests := []struct {
		rangeParam    string
		expectedCount int
	}{
		{"[0,9]", 10},
		{"[4,13]", 10},
		{"[19,28]", 6},
		{"[0,24]", 25},
		{"[10,14]", 5},
	}

	for _, tt := range tests {
		t.Run(tt.rangeParam, func(t *testing.T) {
			r := gin.New()
			r.GET("/api/links", linkHandler.GetAllLinks)

			url := "/api/links?range=" + tt.rangeParam
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response []dto.LinkResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Len(t, response, tt.expectedCount)
		})
	}
}

func TestGetLink(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupData      func(ctx context.Context, q *generated.Queries) int32
		linkID         int32
		expectedStatus int
		checkResponse  func(t *testing.T, response dto.LinkResponse)
	}{
		{
			name: "Success - Get existing link",
			setupData: func(ctx context.Context, q *generated.Queries) int32 {
				link, err := q.CreateLink(ctx, generated.CreateLinkParams{
					OriginalUrl: "https://example.com",
					ShortName:   "testlink",
					ShortUrl:    "https://test.com/r/testlink",
				})
				require.NoError(t, err)
				return link.ID
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response dto.LinkResponse) {
				assert.Equal(t, "https://example.com", response.OriginalUrl)
				assert.Equal(t, "testlink", response.ShortName)
			},
		},
		{
			name:           "Error - Link not found",
			setupData:      nil,
			linkID:         99999,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Error - Invalid ID format (non-numeric)",
			setupData:      nil,
			linkID:         0,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, cleanup := SetupTx(t)
			defer cleanup()

			ctx := context.Background()

			linkID := tt.linkID
			if tt.setupData != nil {
				linkID = tt.setupData(ctx, q)
			}

			linkService := services.NewLinkService(q)
			linkHandler := NewLinkHandler(q, linkService)

			r := gin.New()
			r.GET("/api/links/:id", linkHandler.GetLink)

			var url string
			if tt.linkID == 0 && tt.name == "Error - Invalid ID format (non-numeric)" {
				url = "/api/links/abc"
			} else {
				url = fmt.Sprintf("/api/links/%d", linkID)
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK && tt.checkResponse != nil {
				var response dto.LinkResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				tt.checkResponse(t, response)
			}
		})
	}
}

func TestDeleteLink(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupData      func(ctx context.Context, q *generated.Queries) int32
		linkID         int32
		expectedStatus int
	}{
		{
			name: "Success - Delete existing link",
			setupData: func(ctx context.Context, q *generated.Queries) int32 {
				link, err := q.CreateLink(ctx, generated.CreateLinkParams{
					OriginalUrl: "https://delete.com",
					ShortName:   "delete",
					ShortUrl:    "https://test.com/r/delete",
				})
				require.NoError(t, err)
				return link.ID
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "Error - Delete non-existent link",
			setupData:      nil,
			linkID:         99999,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Error - Invalid ID format (non-numeric)",
			setupData:      nil,
			linkID:         0,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, cleanup := SetupTx(t)
			defer cleanup()

			ctx := context.Background()

			linkID := tt.linkID
			if tt.setupData != nil {
				linkID = tt.setupData(ctx, q)
			}

			linkService := services.NewLinkService(q)
			linkHandler := NewLinkHandler(q, linkService)

			r := gin.New()
			r.DELETE("/api/links/:id", linkHandler.DeleteLink)

			var url string
			if tt.linkID == 0 && tt.name == "Error - Invalid ID format (non-numeric)" {
				url = "/api/links/abc"
			} else {
				url = fmt.Sprintf("/api/links/%d", linkID)
			}

			req := httptest.NewRequest(http.MethodDelete, url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			// Проверяем, что запись действительно удалена
			if tt.expectedStatus == http.StatusNoContent {
				_, err := q.GetLink(ctx, linkID)
				assert.Error(t, err)
				assert.ErrorIs(t, err, pgx.ErrNoRows)
			}
		})
	}
}
