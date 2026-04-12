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
		log.Printf("Warning: failed to load config: %v", err)
	}

	// Получаем URL из конфига
	dbURL := cfg.GetDBURL()

	// Создаём приложение (NewApp должен принимать URL)
	application, err := app.NewApp(dbURL)
	if err != nil {
		log.Fatal("Failed to create app: ", err)
	}
	defer application.Close()

	router := r.NewRouter(application)

	// Запускаем сервер
	if err := router.Run(":8080"); err != nil {
		log.Printf("Server failed to start: %v", err)
	}
}
