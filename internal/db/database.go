package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Config содержит настройки подключения к БД
type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	SSLMode    string
}

// Connect устанавливает соединение с базой данных
func Connect() (*sql.DB, error) {
	// ПРОВЕРЯЕМ RENDER DATABASE URL В ПЕРВУЮ ОЧЕРЕДЬ!
	renderDBURL := os.Getenv("DATABASE_URL")
	if renderDBURL != "" {
		log.Println("✅ Using Render DATABASE_URL")
		return connectFromURL(renderDBURL)
	}

	// Если нет Render URL, используем обычный конфиг
	cfg := Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "linksdb"),
		SSLMode:    getEnv("SSL_MODE", "disable"),
	}

	log.Println("🔄 Using local database configuration")
	return connectFromConfig(cfg)
}

// connectFromURL подключается по DATABASE_URL (для Render)
func connectFromURL(dbURL string) (*sql.DB, error) {
	// Render использует формат: postgresql://...
	// Преобразуем в postgres:// если нужно
	dbURL = strings.Replace(dbURL, "postgresql://", "postgres://", 1)

	// Добавляем sslmode=require если нет
	if !strings.Contains(dbURL, "sslmode=") {
		if strings.Contains(dbURL, "?") {
			dbURL += "&sslmode=require"
		} else {
			dbURL += "?sslmode=require"
		}
	}

	log.Printf("🔗 Connecting with URL: %s", dbURL)

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database from URL: %w", err)
	}

	return configureDB(db)
}

// connectFromConfig подключается по конфигу (для локальной разработки)
func connectFromConfig(cfg Config) (*sql.DB, error) {
	// Для локальной разработки используем disable
	if cfg.SSLMode == "" {
		cfg.SSLMode = "disable"
	}

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.SSLMode)

	log.Printf("🔗 Connecting to: %s:%s/%s", cfg.DBHost, cfg.DBPort, cfg.DBName)

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return configureDB(db)
}

// configureDB настраивает и проверяет соединение
func configureDB(db *sql.DB) (*sql.DB, error) {
	// Настройка пула соединений
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Проверяем соединение
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := db.PingContext(ctx)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("database not reachable: %w", err)
	}

	log.Println("✅ Database connection established")
	return db, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

/*
package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Load configuration (you might want to use environment variables)

// Config содержит настройки подключения к БД
type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	SSLMode    string
}

// Connect устанавливает соединение с базой данных
func Connect() (*sql.DB, error) {

	// ПРОВЕРЯЕМ RENDER DATABASE URL В ПЕРВУЮ ОЧЕРЕДЬ!
	renderDBURL := os.Getenv("DATABASE_URL")
	if renderDBURL != "" {
		log.Println("Using Render DATABASE_URL")
		return connectFromURL(renderDBURL)
	}


	cfg := Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "linksdb"),
	}

	// Build connection string
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	// Connect to database
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	//defer db.Close()

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatal("Database not reachable:", err)
	}

	fmt.Println("✅ Database connection established")

	// Именно здесь устанавливается реальное соединение с базой.
	if err := db.PingContext(ctx); err != nil {
		log.Fatal("database unreachable:", err)
	}

	return db, nil
}


func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
*/
