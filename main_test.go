//go:build integration

package main

import (
	"bytes"
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"db200/handlers"
	linksdb "db200/internal/db/links"
	"db200/internal/dto"
	"db200/repositories" // ← изменено: services → repositories

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

//go:embed db/migrations/*.sql
var migrationsFS embed.FS

var (
	testDB      *sql.DB
	testHandler *handlers.LinkHandler
	testRouter  *gin.Engine
)

// withTx - обертка для изоляции каждого теста в транзакции
func withTx(t *testing.T, fn func(ctx context.Context, q *linksdb.Queries, tx *sql.Tx)) {
	t.Helper()

	// Используем новый контекст для транзакции, не зависящий от t.Context()
	ctx := context.Background()

	tx, err := testDB.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	// Откат транзакции после теста - используем контекст без таймаута
	t.Cleanup(func() {
		_ = tx.Rollback() // игнорируем ошибку, т.к. контекст уже может быть отменен
	})

	// Создаем контекст с таймаутом для тестовой логики
	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	t.Cleanup(cancel)

	qtx := linksdb.New(tx)
	fn(testCtx, qtx, tx)
}

func TestMain(m *testing.M) {
	// Устанавливаем Gin в release mode для тестов
	gin.SetMode(gin.ReleaseMode)
	ctx := context.Background()

	fmt.Println("🚀 Starting PostgreSQL container...")

	pgContainer, err := postgres.Run(ctx, "postgres:16",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		fmt.Printf("❌ Failed to start PostgreSQL container: %v\n", err)
		os.Exit(1)
	}

	defer func() {
		fmt.Println("🧹 Cleaning up PostgreSQL container...")
		if err := pgContainer.Terminate(ctx); err != nil {
			fmt.Printf("⚠️ Failed to terminate container: %v\n", err)
		}
		fmt.Println("✅ Container removed")
	}()

	host, err := pgContainer.Host(ctx)
	if err != nil {
		fmt.Printf("❌ Failed to get host: %v\n", err)
		os.Exit(1)
	}

	port, err := pgContainer.MappedPort(ctx, "5432/tcp")
	if err != nil {
		fmt.Printf("❌ Failed to get port: %v\n", err)
		os.Exit(1)
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s user=testuser password=testpass dbname=testdb sslmode=disable",
		host, port.Port(),
	)

	fmt.Printf("📡 Connecting to database at %s:%s\n", host, port.Port())

	testDB, err = sql.Open("pgx", dsn)
	if err != nil {
		fmt.Printf("❌ Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer testDB.Close()

	ctxPing, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := testDB.PingContext(ctxPing); err != nil {
		fmt.Printf("❌ Failed to ping database: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Connected to database")

	fmt.Println("📦 Running migrations...")

	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("postgres"); err != nil {
		fmt.Printf("❌ Failed to set dialect: %v\n", err)
		os.Exit(1)
	}

	if err := goose.Up(testDB, "db/migrations"); err != nil {
		fmt.Printf("❌ Failed to run migrations: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Migrations completed")

	// ← ИСПРАВЛЕНО: используем repositories вместо services
	testQueries := linksdb.New(testDB)
	testRepository := repositories.NewLinkRepository(testQueries) // ← изменено
	testHandler = handlers.NewLinkHandler(testRepository)         // ← изменено

	testRouter = gin.New()

	testRouter.POST("/api/links", testHandler.Create)
	testRouter.GET("/api/links", testHandler.GetAllLinks)
	testRouter.GET("/api/links/:id", testHandler.GetLink)
	testRouter.PUT("/api/links/:id", testHandler.UpdateLink)
	testRouter.DELETE("/api/links/:id", testHandler.DeleteLink)

	os.Setenv("BASE_SITE", "https://test.com")

	fmt.Println("🏃 Running tests...")
	code := m.Run()

	fmt.Printf("📊 Tests completed with code: %d\n", code)
	os.Exit(code)
}

// ==================== ТАБЛИЧНЫЕ ТЕСТЫ ====================

// TestCreateLink - табличный тест для создания ссылок
func TestCreateLink(t *testing.T) {
	// Каждый подтест будет в своей транзакции
	tests := []struct {
		name           string
		requestBody    dto.CreateLinkRequest
		setupData      func(ctx context.Context, q *linksdb.Queries)
		expectedStatus int
		expectedCode   string
		checkResponse  func(t *testing.T, response dto.LinkResponse, q *linksdb.Queries, ctx context.Context)
	}{
		{
			name: "Success - Create link with custom short name",
			requestBody: dto.CreateLinkRequest{
				OriginalUrl: "https://example.com",
				ShortName:   "custom",
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, response dto.LinkResponse, q *linksdb.Queries, ctx context.Context) {
				if response.OriginalUrl != "https://example.com" {
					t.Errorf("Expected original_url 'https://example.com', got '%s'", response.OriginalUrl)
				}
				if response.ShortName != "custom" {
					t.Errorf("Expected short_name 'custom', got '%s'", response.ShortName)
				}
				if response.ID == 0 {
					t.Error("Expected ID to be set, got 0")
				}

				// Проверяем, что запись действительно есть в БД
				link, err := q.GetLink(ctx, response.ID)
				if err != nil {
					t.Errorf("Failed to get link from DB: %v", err)
				}
				if link.OriginalUrl != "https://example.com" {
					t.Errorf("DB check: expected original_url 'https://example.com', got '%s'", link.OriginalUrl)
				}
			},
		},
		{
			name: "Success - Create link with auto-generated short name",
			requestBody: dto.CreateLinkRequest{
				OriginalUrl: "https://example.com/auto",
				ShortName:   "",
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, response dto.LinkResponse, q *linksdb.Queries, ctx context.Context) {
				if response.OriginalUrl != "https://example.com/auto" {
					t.Errorf("Expected original_url 'https://example.com/auto', got '%s'", response.OriginalUrl)
				}
				if response.ShortName == "" {
					t.Error("Expected short_name to be generated, got empty")
				}
				if response.ID == 0 {
					t.Error("Expected ID to be set, got 0")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTx(t, func(ctx context.Context, q *linksdb.Queries, tx *sql.Tx) {
				// Подготовка данных через sqlc внутри транзакции
				if tt.setupData != nil {
					tt.setupData(ctx, q)
				}

				// ← ИСПРАВЛЕНО: создаем временный репозиторий и хендлер с этой транзакцией
				tempRepository := repositories.NewLinkRepository(q)    // ← изменено
				tempHandler := handlers.NewLinkHandler(tempRepository) // ← изменено

				// Создаем временный роутер для этого теста
				tempRouter := gin.New()
				tempRouter.POST("/api/links", tempHandler.Create)

				// Выполняем запрос
				jsonBody, _ := json.Marshal(tt.requestBody)
				req := httptest.NewRequest(http.MethodPost, "/api/links", bytes.NewBuffer(jsonBody))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()
				tempRouter.ServeHTTP(w, req)

				// Проверяем статус
				if w.Code != tt.expectedStatus {
					t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
				}

				// Проверяем код ошибки
				if tt.expectedCode != "" && w.Code == http.StatusConflict {
					var response map[string]interface{}
					if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
						t.Fatalf("Failed to parse response: %v", err)
					}
					if response["code"] != tt.expectedCode {
						t.Errorf("Expected code '%s', got '%v'", tt.expectedCode, response["code"])
					}
				}

				// Проверяем успешный ответ
				if tt.expectedStatus == http.StatusCreated && tt.checkResponse != nil {
					var response dto.LinkResponse
					if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
						t.Fatalf("Failed to parse response: %v", err)
					}
					tt.checkResponse(t, response, q, ctx)
				}
			})
		})
	}
}

// TestUpdateLink - табличный тест для обновления ссылок
func TestUpdateLink(t *testing.T) {
	tests := []struct {
		name           string
		setupData      func(ctx context.Context, q *linksdb.Queries) int32 // создает данные и возвращает ID
		requestBody    map[string]interface{}
		expectedStatus int
		expectedCode   string
		checkResponse  func(t *testing.T, response dto.LinkResponse, q *linksdb.Queries, ctx context.Context)
	}{
		{
			name: "Success - Update only original_url",
			setupData: func(ctx context.Context, q *linksdb.Queries) int32 {
				link, err := q.CreateLink(ctx, linksdb.CreateLinkParams{
					OriginalUrl: "https://old.com",
					ShortName:   "oldname",
					ShortUrl:    "https://test.com/oldname",
				})
				if err != nil {
					t.Fatalf("Failed to create test link: %v", err)
				}
				return link.ID
			},
			requestBody: map[string]interface{}{
				"original_url": "https://new.com",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response dto.LinkResponse, q *linksdb.Queries, ctx context.Context) {
				if response.OriginalUrl != "https://new.com" {
					t.Errorf("Expected original_url 'https://new.com', got '%s'", response.OriginalUrl)
				}
				if response.ShortName != "oldname" {
					t.Errorf("Expected short_name 'oldname', got '%s'", response.ShortName)
				}
				expectedShortUrl := "https://test.com/oldname"
				if response.ShortUrl != expectedShortUrl {
					t.Errorf("Expected short_url '%s', got '%s'", expectedShortUrl, response.ShortUrl)
				}
				// Проверяем в БД
				link, err := q.GetLink(ctx, response.ID)
				if err != nil {
					t.Errorf("Failed to get link from DB: %v", err)
				}
				if link.OriginalUrl != "https://new.com" {
					t.Errorf("DB check: expected original_url 'https://new.com', got '%s'", link.OriginalUrl)
				}
				if link.ShortUrl != expectedShortUrl {
					t.Errorf("DB check: expected short_url '%s', got '%s'", expectedShortUrl, link.ShortUrl)
				}
			},
		},
		// ... остальные тесты аналогично
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTx(t, func(ctx context.Context, q *linksdb.Queries, tx *sql.Tx) {
				// Создаем тестовые данные
				id := tt.setupData(ctx, q)

				// ← ИСПРАВЛЕНО: создаем временный репозиторий и хендлер с этой транзакцией
				tempRepository := repositories.NewLinkRepository(q)    // ← изменено
				tempHandler := handlers.NewLinkHandler(tempRepository) // ← изменено

				// Создаем временный роутер для этого теста
				tempRouter := gin.New()
				tempRouter.PUT("/api/links/:id", tempHandler.UpdateLink)

				// Выполняем запрос
				jsonBody, _ := json.Marshal(tt.requestBody)
				req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/links/%d", id), bytes.NewBuffer(jsonBody))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()
				tempRouter.ServeHTTP(w, req)

				// Проверяем статус
				if w.Code != tt.expectedStatus {
					t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
				}

				// Проверяем код ошибки
				if tt.expectedCode != "" && w.Code == http.StatusConflict {
					var response map[string]interface{}
					if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
						t.Fatalf("Failed to parse response: %v", err)
					}
					if response["code"] != tt.expectedCode {
						t.Errorf("Expected code '%s', got '%v'", tt.expectedCode, response["code"])
					}
				}

				// Проверяем успешный ответ
				if tt.expectedStatus == http.StatusOK && tt.checkResponse != nil {
					var response dto.LinkResponse
					if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
						t.Fatalf("Failed to parse response: %v", err)
					}
					tt.checkResponse(t, response, q, ctx)
				}
			})
		})
	}
}

// TestGetLink - табличный тест для получения ссылки по ID
func TestGetLink(t *testing.T) {
	tests := []struct {
		name           string
		setupData      func(ctx context.Context, q *linksdb.Queries) int32
		expectedStatus int
		checkResponse  func(t *testing.T, response dto.LinkResponse, q *linksdb.Queries, ctx context.Context)
	}{
		{
			name: "Success - Get existing link",
			setupData: func(ctx context.Context, q *linksdb.Queries) int32 {
				link, err := q.CreateLink(ctx, linksdb.CreateLinkParams{
					OriginalUrl: "https://example.com",
					ShortName:   "testlink",
					ShortUrl:    "https://test.com/testlink",
				})
				if err != nil {
					t.Fatalf("Failed to create test link: %v", err)
				}
				return link.ID
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response dto.LinkResponse, q *linksdb.Queries, ctx context.Context) {
				if response.OriginalUrl != "https://example.com" {
					t.Errorf("Expected original_url 'https://example.com', got '%s'", response.OriginalUrl)
				}
				if response.ShortName != "testlink" {
					t.Errorf("Expected short_name 'testlink', got '%s'", response.ShortName)
				}
			},
		},
		{
			name: "Error - Link not found",
			setupData: func(ctx context.Context, q *linksdb.Queries) int32 {
				return 99999
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTx(t, func(ctx context.Context, q *linksdb.Queries, tx *sql.Tx) {
				id := tt.setupData(ctx, q)

				// ← ИСПРАВЛЕНО
				tempRepository := repositories.NewLinkRepository(q)
				tempHandler := handlers.NewLinkHandler(tempRepository)

				tempRouter := gin.New()
				tempRouter.GET("/api/links/:id", tempHandler.GetLink)

				req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/links/%d", id), nil)
				w := httptest.NewRecorder()
				tempRouter.ServeHTTP(w, req)

				if w.Code != tt.expectedStatus {
					t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
				}

				if tt.expectedStatus == http.StatusOK && tt.checkResponse != nil {
					var response dto.LinkResponse
					if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
						t.Fatalf("Failed to parse response: %v", err)
					}
					tt.checkResponse(t, response, q, ctx)
				}
			})
		})
	}
}

// TestDeleteLink - табличный тест для удаления ссылки
func TestDeleteLink(t *testing.T) {
	tests := []struct {
		name           string
		setupData      func(ctx context.Context, q *linksdb.Queries) int32
		expectedStatus int
		verifyDeleted  func(t *testing.T, ctx context.Context, q *linksdb.Queries, id int32)
	}{
		{
			name: "Success - Delete existing link",
			setupData: func(ctx context.Context, q *linksdb.Queries) int32 {
				link, err := q.CreateLink(ctx, linksdb.CreateLinkParams{
					OriginalUrl: "https://delete.com",
					ShortName:   "delete",
					ShortUrl:    "https://test.com/delete",
				})
				if err != nil {
					t.Fatalf("Failed to create test link: %v", err)
				}
				return link.ID
			},
			expectedStatus: http.StatusNoContent,
			verifyDeleted: func(t *testing.T, ctx context.Context, q *linksdb.Queries, id int32) {
				_, err := q.GetLink(ctx, id)
				if err == nil {
					t.Error("Expected link to be deleted, but it still exists")
				}
			},
		},
		{
			name: "Error - Link not found",
			setupData: func(ctx context.Context, q *linksdb.Queries) int32 {
				return 99999
			},
			expectedStatus: http.StatusNotFound,
			verifyDeleted:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTx(t, func(ctx context.Context, q *linksdb.Queries, tx *sql.Tx) {
				id := tt.setupData(ctx, q)

				// ← ИСПРАВЛЕНО
				tempRepository := repositories.NewLinkRepository(q)
				tempHandler := handlers.NewLinkHandler(tempRepository)

				tempRouter := gin.New()
				tempRouter.DELETE("/api/links/:id", tempHandler.DeleteLink)

				req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/links/%d", id), nil)
				w := httptest.NewRecorder()
				tempRouter.ServeHTTP(w, req)

				if w.Code != tt.expectedStatus {
					t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
				}

				if tt.expectedStatus == http.StatusNoContent && tt.verifyDeleted != nil {
					tt.verifyDeleted(t, ctx, q, id)
				}
			})
		})
	}
}

// TestGetAllLinks - табличный тест для получения всех ссылок с пагинацией
func TestGetAllLinks(t *testing.T) {
	tests := []struct {
		name           string
		setupData      func(ctx context.Context, q *linksdb.Queries) int
		rangeParam     string
		expectedCount  int
		expectedStart  int64
		expectedEnd    int64
		expectedTotal  int64
		expectedStatus int
	}{
		{
			name: "Success - Get all links without range (default first 10)",
			setupData: func(ctx context.Context, q *linksdb.Queries) int {
				links := []linksdb.CreateLinkParams{
					{OriginalUrl: "https://test1.com", ShortName: "link1", ShortUrl: "https://test.com/link1"},
					{OriginalUrl: "https://test2.com", ShortName: "link2", ShortUrl: "https://test.com/link2"},
					{OriginalUrl: "https://test3.com", ShortName: "link3", ShortUrl: "https://test.com/link3"},
				}
				for _, link := range links {
					_, err := q.CreateLink(ctx, link)
					if err != nil {
						t.Fatalf("Failed to create test link: %v", err)
					}
				}
				return len(links)
			},
			rangeParam:     "",
			expectedCount:  3,
			expectedStart:  1,
			expectedEnd:    3,
			expectedTotal:  3,
			expectedStatus: http.StatusOK,
		},
		// ... остальные тесты
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTx(t, func(ctx context.Context, q *linksdb.Queries, tx *sql.Tx) {
				tt.setupData(ctx, q)

				// ← ИСПРАВЛЕНО
				tempRepository := repositories.NewLinkRepository(q)
				tempHandler := handlers.NewLinkHandler(tempRepository)

				tempRouter := gin.New()
				tempRouter.GET("/api/links", tempHandler.GetAllLinks)

				url := "/api/links"
				if tt.rangeParam != "" {
					url = url + "?range=" + tt.rangeParam
				}

				req := httptest.NewRequest(http.MethodGet, url, nil)
				w := httptest.NewRecorder()
				tempRouter.ServeHTTP(w, req)

				if w.Code != tt.expectedStatus {
					t.Fatalf("Expected status %d, got %d", tt.expectedStatus, w.Code)
				}

				if tt.expectedStatus != http.StatusOK {
					return
				}

				contentRange := w.Header().Get("Content-Range")
				expectedContentRange := fmt.Sprintf("links %d-%d/%d", tt.expectedStart, tt.expectedEnd, tt.expectedTotal)
				if contentRange != expectedContentRange {
					t.Errorf("Expected Content-Range: %s, got: %s", expectedContentRange, contentRange)
				}

				var response []dto.LinkResponse
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}

				if len(response) != tt.expectedCount {
					t.Errorf("Expected %d links, got %d", tt.expectedCount, len(response))
				}
			})
		})
	}
}

// TestGetAllLinksWithLargeDataset - тест с большим количеством данных
func TestGetAllLinksWithLargeDataset(t *testing.T) {
	withTx(t, func(ctx context.Context, q *linksdb.Queries, tx *sql.Tx) {
		// Создаем 25 ссылок
		for i := 1; i <= 25; i++ {
			_, err := q.CreateLink(ctx, linksdb.CreateLinkParams{
				OriginalUrl: fmt.Sprintf("https://test%d.com", i),
				ShortName:   fmt.Sprintf("link%d", i),
				ShortUrl:    fmt.Sprintf("https://test.com/link%d", i),
			})
			if err != nil {
				t.Fatalf("Failed to create test link: %v", err)
			}
		}

		// ← ИСПРАВЛЕНО
		tempRepository := repositories.NewLinkRepository(q)
		tempHandler := handlers.NewLinkHandler(tempRepository)

		tests := []struct {
			rangeParam    string
			expectedCount int
			expectedStart int64
			expectedEnd   int64
			expectedTotal int64
		}{
			{"[0,9]", 10, 1, 10, 25},
			{"[4,13]", 10, 5, 14, 25},
			{"[19,28]", 6, 20, 25, 25},
			{"[0,24]", 25, 1, 25, 25},
			{"[10,14]", 5, 11, 15, 25},
		}

		for _, tt := range tests {
			t.Run(tt.rangeParam, func(t *testing.T) {
				url := "/api/links?range=" + tt.rangeParam
				req := httptest.NewRequest(http.MethodGet, url, nil)
				w := httptest.NewRecorder()

				tempRouter := gin.New()
				tempRouter.GET("/api/links", tempHandler.GetAllLinks)
				tempRouter.ServeHTTP(w, req)

				if w.Code != http.StatusOK {
					t.Fatalf("Expected status OK, got %d", w.Code)
				}

				contentRange := w.Header().Get("Content-Range")
				expected := fmt.Sprintf("links %d-%d/%d", tt.expectedStart, tt.expectedEnd, tt.expectedTotal)
				if contentRange != expected {
					t.Errorf("Expected Content-Range: %s, got: %s", expected, contentRange)
				}

				var response []dto.LinkResponse
				json.Unmarshal(w.Body.Bytes(), &response)
				if len(response) != tt.expectedCount {
					t.Errorf("Expected %d links, got %d", tt.expectedCount, len(response))
				}
			})
		}
	})
}
