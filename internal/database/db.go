package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"db200/internal/config"

	_ "github.com/lib/pq"
	// или для pgx:
	// _ "github.com/jackc/pgx/v5/stdlib"
)

var (
	DB      *sql.DB
	Queries *db.Queries // ← SQLC queries
)

// Init инициализирует БД и SQLC queries
func Init() error {
	cfg := config.LoadDBConfig()

	connStr := cfg.ConnectionString()

	// Открываем соединение
	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Настройка пула соединений
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(5)
	DB.SetConnMaxLifetime(5 * time.Minute)

	// Проверка подключения
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := DB.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Инициализируем SQLC queries
	Queries = db.New(DB) // ← Ключевая строка!

	log.Println("✅ Database connected successfully")
	log.Println("✅ SQLC queries initialized")

	return nil
}

// GetQueries возвращает SQLC queries
func GetQueries() *db.Queries {
	return Queries
}

// GetDB возвращает raw DB connection
func GetDB() *sql.DB {
	return DB
}

// Close закрывает соединение
func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}
