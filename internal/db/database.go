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
