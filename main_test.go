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

// Теперь путь правильный - миграции в db/migrations
//
//go:embed db/migrations/*.sql
var migrationsFS embed.FS

var (
	testDB      *sql.DB
	testHandler *handlers.LinkHandler
	testRouter  *gin.Engine
)

func TestMain(m *testing.M) {
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

	// Применяем миграции - путь "db/migrations" внутри embed.FS
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

func TestCreateLink_DuplicateURL(t *testing.T) {
	firstReq := dto.CreateLinkRequest{
		OriginalUrl: "https://duplicate.com",
		ShortName:   "first",
	}

	jsonBody, _ := json.Marshal(firstReq)
	req := httptest.NewRequest(http.MethodPost, "/api/links", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create first link: %s", w.Body.String())
	}

	secondReq := dto.CreateLinkRequest{
		OriginalUrl: "https://duplicate.com",
		ShortName:   "second",
	}

	jsonBody, _ = json.Marshal(secondReq)
	req = httptest.NewRequest(http.MethodPost, "/api/links", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusConflict, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["code"] != "DUPLICATE_ORIGINAL_URL" {
		t.Errorf("Expected code 'DUPLICATE_ORIGINAL_URL', got '%v'", response["code"])
	}

	t.Log("✅ Test passed!")
}
