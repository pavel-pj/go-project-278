package configs

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	// Только для основной БД
	DBUser     string
	DBPassword string
	DBName     string
	DBHost     string
	DBPort     string

	// Для Render.com
	DatabaseURL string
	UseURL      bool // флаг, что используем URL вместо отдельных параметров
}

func Load() (*Config, error) {
	// Загружаем .env файл
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	cfg := &Config{}

	// ПРОВЕРЯЕМ RENDER DATABASE URL В ПЕРВУЮ ОЧЕРЕДЬ!
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		log.Println("✅ Using Render DATABASE_URL")
		cfg.DatabaseURL = dbURL
		cfg.UseURL = true
		return cfg, nil
	}

	// Если нет Render URL, используем обычный конфиг
	cfg.UseURL = false
	cfg.DBUser = getEnv("DB_USER", "golang")
	cfg.DBPassword = getEnv("DB_PASSWORD", "secret")
	cfg.DBName = getEnv("DB_NAME", "app")
	cfg.DBHost = getEnv("DB_HOST", "postgres")
	cfg.DBPort = getEnv("DB_PORT", "5432")

	log.Println("Using local database configuration")
	return cfg, nil
}

func (c *Config) GetDBURL() string {
	if c.UseURL {
		// Преобразуем postgresql:// в postgres:// если нужно
		url := strings.Replace(c.DatabaseURL, "postgresql://", "postgres://", 1)

		// Добавляем sslmode=require если нет (нужно для Render)
		if !strings.Contains(url, "sslmode=") {
			if strings.Contains(url, "?") {
				url += "&sslmode=require"
			} else {
				url += "?sslmode=require"
			}
		}
		return url
	}

	// Локальная разработка - sslmode=disable
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
