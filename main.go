package main

import (
	"db200/internal/app"
	"db200/internal/configs"
	r "db200/internal/router"
	"log"
)

func main() {
	// Загружаем конфиг
	cfg, err := configs.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Получаем URL из конфига (работает и для Render, и для локальной разработки)
	dbURL := cfg.GetDBURL()

	// Создаём приложение
	application, err := app.NewApp(dbURL)
	if err != nil {
		log.Fatal("Failed to create app: ", err)
	}
	defer func() {
		if err := application.Close(); err != nil {
			log.Printf("Warning: failed to close app: %v", err)
		}
	}()

	router := r.NewRouter(application)

	// Запускаем сервер
	if err := router.Run(":8080"); err != nil {
		log.Printf("Server failed to start: %v", err)
	}
}
