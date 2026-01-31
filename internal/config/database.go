package config

import (
	"fmt"
	"os"
)

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func LoadDBConfig() DBConfig {
	// Для Render/Heroku используем DATABASE_URL
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		return parseDatabaseURL(dbURL)
	}

	return DBConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "golang"),
		Password: getEnv("DB_PASSWORD", "secret"),
		DBName:   getEnv("DB_NAME", "app"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}
}

func parseDatabaseURL(url string) DBConfig {
	// Парсим DATABASE_URL формата: postgres://user:pass@host:port/dbname
	// Упрощённый парсинг - в продакшене лучше использовать net/url
	return DBConfig{
		Host:     "localhost", // замените на реальный парсинг
		Port:     "5432",
		User:     "user",
		Password: "pass",
		DBName:   "dbname",
		SSLMode:  "require",
	}
}

func (c DBConfig) ConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
