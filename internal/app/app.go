package app

import (
	"database/sql"
	linksdb "db200/internal/db/links"
	"db200/services"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// App хранит ВСЕ зависимости приложения
type App struct {
	DB       *sql.DB
	Queries  *Queries
	Services *Services
}

// Queries содержит ВСЕ SQLC queries
type Queries struct {
	Links *linksdb.Queries
}

type Services struct {
	Links  *services.LinkService
	Visits *services.VisitService
}

func NewApp(db *sql.DB) *App {
	queries := &Queries{
		Links: linksdb.New(db),
	}
	// Создаем сервисы (передаем queries!)
	services := &Services{
		Links: services.NewLinkService(queries.Links),
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
