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

	// Пытаемся получить параметры тестовой БД
	dbUser := os.Getenv("DB_USER_TEST")
	dbPassword := os.Getenv("DB_PASSWORD_TEST")
	dbName := os.Getenv("DB_TEST_NAME")
	dbHost := os.Getenv("DB_TEST_HOST")
	dbPort := os.Getenv("DB_TEST_PORT")

	// Если нет тестовой БД - используем основную
	if dbUser == "" {
		dbUser = os.Getenv("DB_USER")
		if dbUser == "" {
			dbUser = "golang"
		}
	}
	if dbPassword == "" {
		dbPassword = os.Getenv("DB_PASSWORD")
		if dbPassword == "" {
			dbPassword = "secret"
		}
	}
	if dbName == "" {
		dbName = os.Getenv("DB_NAME")
		if dbName == "" {
			dbName = "app"
		}
	}
	if dbHost == "" {
		dbHost = os.Getenv("DB_HOST")
		if dbHost == "" {
			dbHost = "localhost"
		}
	}
	if dbPort == "" {
		dbPort = os.Getenv("DB_PORT")
		if dbPort == "" {
			dbPort = "5432"
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
	migrationsDir := filepath.Join(filepath.Dir(filename), "../db/migrations")

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
