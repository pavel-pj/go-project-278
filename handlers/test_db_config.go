//go:build integration

package handlers

import (
	"context"
	"database/sql"
	"db200/internal/db/generated"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

var testPool *pgxpool.Pool

// InitTestDB - инициализация тестовой БД (вызывается вручную)
func InitTestDB() {
	if testPool != nil {
		return
	}

	gin.SetMode(gin.TestMode)

	// Загружаем .env файл (если есть)
	if err := godotenv.Load("../.env"); err != nil {
		log.Println("No .env file found, using defaults or CI env vars")
	}

	// Определяем, запущены ли тесты в Docker
	inDocker := os.Getenv("DOCKER_ENV") == "true"

	// Получаем параметры тестовой БД
	dbUser := os.Getenv("DB_USER_TEST")
	if dbUser == "" {
		dbUser = "golang_test"
	}
	dbPassword := os.Getenv("DB_PASSWORD_TEST")
	if dbPassword == "" {
		dbPassword = "secret_test"
	}
	dbName := os.Getenv("DB_TEST_NAME")
	if dbName == "" {
		dbName = "app_test"
	}

	// Хост: в Docker используем имя сервиса, локально - localhost
	dbHost := os.Getenv("DB_TEST_HOST")
	if dbHost == "" {
		if inDocker {
			dbHost = "postgres_test" // Имя сервиса в docker-compose
		} else {
			dbHost = "localhost"
		}
	}

	// Порт: в Docker стандартный 5432, локально может быть 5501
	dbPort := os.Getenv("DB_TEST_PORT")
	if dbPort == "" {
		if inDocker {
			dbPort = "5432"
		} else {
			dbPort = "5501"
		}
	}

	// Формируем URL (приоритет у переменной TEST_DATABASE_URL)
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	log.Printf("Connecting to test DB at: postgres://%s:***@%s:%s/%s", dbUser, dbHost, dbPort, dbName)

	// Применяем миграции
	sqlDB, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer sqlDB.Close()

	// Находим путь к миграциям
	_, filename, _, _ := runtime.Caller(0)
	migrationsDir := filepath.Join(filepath.Dir(filename), "../internal/db/migrations")

	// Делаем путь абсолютным для Docker
	if inDocker {
		// В Docker проект монтируется в /app
		migrationsDir = "/app/internal/db/migrations"
	}

	log.Printf("Migrations directory: %s", migrationsDir)

	if err := goose.Up(sqlDB, migrationsDir); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	// Создаем пул соединений pgx
	testPool, err = pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatal("Failed to create connection pool:", err)
	}

	log.Println("Test database initialized successfully")
}

// SetupTx возвращает queries и cleanup функцию
func SetupTx(t *testing.T) (*generated.Queries, func()) {
	if testPool == nil {
		InitTestDB()
	}

	tx, err := testPool.Begin(context.Background())
	require.NoError(t, err)

	q := generated.New(tx)

	cleanup := func() {
		err := tx.Rollback(context.Background())
		require.NoError(t, err)
	}

	return q, cleanup
}
