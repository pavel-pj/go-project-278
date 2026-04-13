package main

import (
	"db200/internal/app"
	"db200/internal/configs"
	r "db200/router"
	"log"
)

func main() {
	// Загружаем конфиг
	cfg, err := configs.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err) // Изменили с Warning на Fatal
	}

	// Получаем URL из конфига (работает и для Render, и для локальной разработки)
	dbURL := cfg.GetDBURL()

	// Для отладки - посмотрим, какой URL используется (но не логируйте пароль в production!)
	log.Printf("Connecting with URL scheme: %s", dbURL[:20]) // только первые 20 символов

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
