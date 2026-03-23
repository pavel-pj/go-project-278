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
	"db200/services"

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

	testQueries := linksdb.New(testDB)
	testService := services.NewLinkService(testQueries)
	testHandler = handlers.NewLinkHandler(testService)

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
// TestCreateLink - табличный тест для создания ссылок
// TestCreateLink - табличный тест для создания ссылок
func TestCreateLink(t *testing.T) {
	// Каждый подтест будет в своей транзакции
	tests := []struct {
		name           string
		requestBody    dto.CreateLinkRequest
		setupData      func(ctx context.Context, q *linksdb.Queries) // теперь принимает q
		expectedStatus int
		expectedCode   string
		checkResponse  func(t *testing.T, response dto.LinkResponse, q *linksdb.Queries, ctx context.Context) // ← добавили ctx
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
				link, err := q.GetLink(ctx, response.ID) // ← теперь ctx есть
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
		// ... остальные тесты
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTx(t, func(ctx context.Context, q *linksdb.Queries, tx *sql.Tx) {
				// Подготовка данных через sqlc внутри транзакции
				if tt.setupData != nil {
					tt.setupData(ctx, q)
				}

				// Создаем временный хендлер с этой транзакцией
				tempService := services.NewLinkService(q)
				tempHandler := handlers.NewLinkHandler(tempService)

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
					tt.checkResponse(t, response, q, ctx) // ← передаем ctx
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
				//log.Println(fmt.Sprintf("Значение req.ShortName :%+v", link))
				if err != nil {
					t.Fatalf("Failed to update test link: %v", err)
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
				// Проверяем что short_url НЕ изменился
				expectedShortUrl := "https://test.com/oldname"
				//log.Println(fmt.Sprintf("RESPONSE :%+v", response))
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
		{
			name: "Success - Update only short_name",
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
				"short_name": "newname",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response dto.LinkResponse, q *linksdb.Queries, ctx context.Context) {
				if response.OriginalUrl != "https://old.com" {
					t.Errorf("Expected original_url 'https://old.com', got '%s'", response.OriginalUrl)
				}
				if response.ShortName != "newname" {
					t.Errorf("Expected short_name 'newname', got '%s'", response.ShortName)
				}
				// Проверяем что short_url обновился
				expectedShortUrl := "https://test.com/newname"
				if response.ShortUrl != expectedShortUrl {
					t.Errorf("Expected short_url '%s', got '%s'", expectedShortUrl, response.ShortUrl)
				}
				// Проверяем в БД
				link, err := q.GetLink(ctx, response.ID)
				if err != nil {
					t.Errorf("Failed to get link from DB: %v", err)
				}
				if link.ShortName != "newname" {
					t.Errorf("DB check: expected short_name 'newname', got '%s'", link.ShortName)
				}
				if link.ShortUrl != expectedShortUrl {
					t.Errorf("DB check: expected short_url '%s', got '%s'", expectedShortUrl, link.ShortUrl)
				}
			},
		},
		{
			name: "Success - Update both fields",
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
				"short_name":   "newname",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response dto.LinkResponse, q *linksdb.Queries, ctx context.Context) {
				if response.OriginalUrl != "https://new.com" {
					t.Errorf("Expected original_url 'https://new.com', got '%s'", response.OriginalUrl)
				}
				if response.ShortName != "newname" {
					t.Errorf("Expected short_name 'newname', got '%s'", response.ShortName)
				}
				// Проверяем что short_url обновился
				expectedShortUrl := "https://test.com/newname"
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
				if link.ShortName != "newname" {
					t.Errorf("DB check: expected short_name 'newname', got '%s'", link.ShortName)
				}
				if link.ShortUrl != expectedShortUrl {
					t.Errorf("DB check: expected short_url '%s', got '%s'", expectedShortUrl, link.ShortUrl)
				}
			},
		},
		{
			name: "Error - Duplicate original_url",
			setupData: func(ctx context.Context, q *linksdb.Queries) int32 {
				// Создаем первую ссылку (которая будет дубликатом)
				_, err := q.CreateLink(ctx, linksdb.CreateLinkParams{
					OriginalUrl: "https://existing.com",
					ShortName:   "existing",
					ShortUrl:    "https://test.com/existing",
				})

				if err != nil {
					t.Fatalf("Failed to create existing link: %v", err)
				}
				// Создаем вторую ссылку для обновления
				link, err := q.CreateLink(ctx, linksdb.CreateLinkParams{
					OriginalUrl: "https://to-update.com",
					ShortName:   "update",
					ShortUrl:    "https://test.com/update",
				})
				if err != nil {
					t.Fatalf("Failed to create test link: %v", err)
				}
				return link.ID
			},
			requestBody: map[string]interface{}{
				"original_url": "https://existing.com",
			},
			expectedStatus: http.StatusConflict,
			expectedCode:   "DUPLICATE_ORIGINAL_URL",
		},
		{
			name: "Error - Duplicate short_name",
			setupData: func(ctx context.Context, q *linksdb.Queries) int32 {
				// Создаем первую ссылку с занятым short_name
				_, err := q.CreateLink(ctx, linksdb.CreateLinkParams{
					OriginalUrl: "https://first.com",
					ShortName:   "taken",
					ShortUrl:    "https://test.com/taken",
				})
				if err != nil {
					t.Fatalf("Failed to create existing link: %v", err)
				}
				// Создаем вторую ссылку для обновления
				link, err := q.CreateLink(ctx, linksdb.CreateLinkParams{
					OriginalUrl: "https://second.com",
					ShortName:   "update",
					ShortUrl:    "https://test.com/update",
				})
				if err != nil {
					t.Fatalf("Failed to create test link: %v", err)
				}
				return link.ID
			},
			requestBody: map[string]interface{}{
				"short_name": "taken",
			},
			expectedStatus: http.StatusConflict,
			expectedCode:   "DUPLICATE_SHORT_NAME",
		},
		{
			name: "Error - Invalid ID (not found)",
			setupData: func(ctx context.Context, q *linksdb.Queries) int32 {
				return 99999 // несуществующий ID
			},
			requestBody: map[string]interface{}{
				"original_url": "https://new.com",
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTx(t, func(ctx context.Context, q *linksdb.Queries, tx *sql.Tx) {
				// Создаем тестовые данные
				id := tt.setupData(ctx, q)

				// Создаем временный хендлер с этой транзакцией
				tempService := services.NewLinkService(q)
				tempHandler := handlers.NewLinkHandler(tempService)

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
					//log.Println(fmt.Sprintf("Значение RESPONSE в ЦИКЛе idx %d:%+v", idx, response))
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
				// Проверяем в БД
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
			name: "Error - Link not found",
			setupData: func(ctx context.Context, q *linksdb.Queries) int32 {
				return 99999 // несуществующий ID
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTx(t, func(ctx context.Context, q *linksdb.Queries, tx *sql.Tx) {
				// Создаем тестовые данные
				id := tt.setupData(ctx, q)

				// Создаем временный хендлер с этой транзакцией
				tempService := services.NewLinkService(q)
				tempHandler := handlers.NewLinkHandler(tempService)

				// Создаем временный роутер для этого теста
				tempRouter := gin.New()
				tempRouter.GET("/api/links/:id", tempHandler.GetLink)

				// Выполняем запрос
				req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/links/%d", id), nil)
				w := httptest.NewRecorder()
				tempRouter.ServeHTTP(w, req)

				// Проверяем статус
				if w.Code != tt.expectedStatus {
					t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
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
				// Проверяем, что ссылка действительно удалена
				_, err := q.GetLink(ctx, id)
				if err == nil {
					t.Error("Expected link to be deleted, but it still exists")
				}
			},
		},
		{
			name: "Error - Link not found",
			setupData: func(ctx context.Context, q *linksdb.Queries) int32 {
				return 99999 // несуществующий ID
			},
			expectedStatus: http.StatusNotFound,
			verifyDeleted:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTx(t, func(ctx context.Context, q *linksdb.Queries, tx *sql.Tx) {
				// Создаем тестовые данные
				id := tt.setupData(ctx, q)

				// Создаем временный хендлер с этой транзакцией
				tempService := services.NewLinkService(q)
				tempHandler := handlers.NewLinkHandler(tempService)

				// Создаем временный роутер для этого теста
				tempRouter := gin.New()
				tempRouter.DELETE("/api/links/:id", tempHandler.DeleteLink)

				// Выполняем запрос на удаление
				req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/links/%d", id), nil)
				w := httptest.NewRecorder()
				tempRouter.ServeHTTP(w, req)

				// Проверяем статус
				if w.Code != tt.expectedStatus {
					t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
				}

				// Проверяем, что ссылка удалена
				if tt.expectedStatus == http.StatusNoContent && tt.verifyDeleted != nil {
					tt.verifyDeleted(t, ctx, q, id)
				}
			})
		})
	}
}

// TestGetAllLinks - табличный тест для получения всех ссылок
func TestGetAllLinks(t *testing.T) {
	tests := []struct {
		name          string
		setupData     func(ctx context.Context, q *linksdb.Queries) int // количество созданных ссылок
		expectedCount int
	}{
		{
			name: "Success - Get all links when multiple exist",
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
			expectedCount: 3,
		},
		{
			name: "Success - Get all links when empty",
			setupData: func(ctx context.Context, q *linksdb.Queries) int {
				return 0
			},
			expectedCount: 0,
		},
		{
			name: "Success - Get all links with one item",
			setupData: func(ctx context.Context, q *linksdb.Queries) int {
				_, err := q.CreateLink(ctx, linksdb.CreateLinkParams{
					OriginalUrl: "https://single.com",
					ShortName:   "single",
					ShortUrl:    "https://test.com/single",
				})
				if err != nil {
					t.Fatalf("Failed to create test link: %v", err)
				}
				return 1
			},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTx(t, func(ctx context.Context, q *linksdb.Queries, tx *sql.Tx) {
				// Создаем тестовые данные
				createdCount := tt.setupData(ctx, q)

				// Создаем временный хендлер с этой транзакцией
				tempService := services.NewLinkService(q)
				tempHandler := handlers.NewLinkHandler(tempService)

				// Создаем временный роутер для этого теста
				tempRouter := gin.New()
				tempRouter.GET("/api/links", tempHandler.GetAllLinks)

				// Выполняем запрос
				req := httptest.NewRequest(http.MethodGet, "/api/links", nil)
				w := httptest.NewRecorder()
				tempRouter.ServeHTTP(w, req)

				// Проверяем статус
				if w.Code != http.StatusOK {
					t.Fatalf("Expected status %d, got %d", http.StatusOK, w.Code)
				}

				// Проверяем количество
				var response []dto.LinkResponse
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}

				if len(response) != createdCount {
					t.Errorf("Expected %d links, got %d", createdCount, len(response))
				}

				// Проверяем, что все созданные ссылки есть в ответе
				if createdCount > 0 {
					// Получаем все ID из БД
					allLinks, err := q.GetAllLinks(ctx)
					if err != nil {
						t.Fatalf("Failed to get links from DB: %v", err)
					}

					if len(allLinks) != len(response) {
						t.Errorf("DB and response count mismatch: DB=%d, Response=%d", len(allLinks), len(response))
					}
				}
			})
		})
	}
}
