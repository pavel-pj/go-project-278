//go:build integration

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"db200/internal/db/generated"
	"db200/internal/dto"
	"db200/internal/services"

	"github.com/gin-gonic/gin"
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Получаем queries и cleanup
			q, cleanup := SetupTx(t)
			defer cleanup()

			ctx := context.Background()

			// Подготовка данных (если нужно)
			if tt.setupData != nil {
				tt.setupData(ctx, q)
			}

			// Создаем хендлеры
			linkService := services.NewLinkService(q)
			linkHandler := NewLinkHandler(q, linkService)

			// Настраиваем роутер
			r := gin.New()
			r.POST("/api/links", linkHandler.Create)

			// Формируем запрос
			jsonBody, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/links", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Выполняем запрос
			r.ServeHTTP(w, req)

			// Проверяем статус
			assert.Equal(t, tt.expectedStatus, w.Code, "Response body: %s", w.Body.String())

			// Проверяем ошибки
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

			// Проверяем успешный ответ
			if tt.expectedStatus == http.StatusCreated && tt.checkResponse != nil {
				var response dto.LinkResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				tt.checkResponse(t, response, q, ctx)
			}
		})
	}
}
