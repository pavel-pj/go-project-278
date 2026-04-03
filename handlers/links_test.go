//go:build integration

package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	linksdb "db200/internal/db/links"
	"db200/internal/dto"
	"db200/internal/testhelpers"
	"db200/repositories"
	"db200/services"

	"github.com/gin-gonic/gin"
)

func TestCreateLink_Success_CustomShortName(t *testing.T) {
	// НАСТРОЙКА - поднимаем БД через pgtest
	fixture := testhelpers.SetupTestDB(t)

	// ВЫПОЛНЯЕМ ТЕСТ В ТРАНЗАКЦИИ
	fixture.WithTxLinks(t, func(tx *sql.Tx, q *linksdb.Queries) {
		// Создаём хендлер
		repo := repositories.NewLinkRepository(q)
		service := services.NewLinkService()
		handler := NewLinkHandler(repo, service)

		// Роутер
		router := gin.New()
		router.POST("/api/links", handler.Create)

		// Запрос
		reqBody := dto.CreateLinkRequest{
			OriginalUrl: "https://example.com",
			ShortName:   "custom",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/links", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// ПРОВЕРКИ
		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d. Body: %s", w.Code, w.Body.String())
		}

		var response dto.LinkResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response.OriginalUrl != "https://example.com" {
			t.Errorf("Expected original_url 'https://example.com', got '%s'", response.OriginalUrl)
		}
		if response.ShortName != "custom" {
			t.Errorf("Expected short_name 'custom', got '%s'", response.ShortName)
		}
		if response.ID == 0 {
			t.Error("Expected ID to be set, got 0")
		}

		// Проверяем, что реально в БД
		link, err := q.GetLink(fixture.Ctx, response.ID)
		if err != nil {
			t.Errorf("Failed to get link from DB: %v", err)
		}
		if link.OriginalUrl != "https://example.com" {
			t.Errorf("DB check: expected 'https://example.com', got '%s'", link.OriginalUrl)
		}
	})
}
