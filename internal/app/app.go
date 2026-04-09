package app

import (
	"database/sql"
	"db200/internal/db/generated"
	"db200/internal/services"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// App хранит ВСЕ зависимости приложения
type App struct {
	DB       *sql.DB
	Queries  *generated.Queries
	Services *Service
}

type Service struct {
	Links *services.LinkService
}

func NewApp(db *sql.DB) *App {
	queries := generated.New(db)

	services := &Service{
		Links: services.NewLinkService(),
	}

	return &App{
		DB:       db,
		Queries:  queries,
		Services: services,
	}
}

// Close закрывает соединение с БД
func (a *App) Close() error {

	if closeErr := a.DB.Close(); closeErr != nil {
		return fmt.Errorf("failed to close database: %w", closeErr)
	}
	return nil
}
