package app

import (
	"database/sql"
	linksdb "db200/internal/db/links"
	visitsdb "db200/internal/db/visits"
	"db200/repositories"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// App хранит ВСЕ зависимости приложения
type App struct {
	DB           *sql.DB
	Queries      *Queries
	Repositories *Repository
}

// Queries содержит ВСЕ SQLC queries
type Queries struct {
	Links  *linksdb.Queries
	Visits *visitsdb.Queries
}

type Repository struct {
	Links  *repositories.LinkRepository
	Visits *repositories.VisitRepository
}

func NewApp(db *sql.DB) *App {
	queries := &Queries{
		Links:  linksdb.New(db),
		Visits: visitsdb.New(db),
	}
	// Создаем репозитории (передаем queries!)
	repositories := &Repository{
		Links:  repositories.NewLinkRepository(queries.Links),
		Visits: repositories.NewVisitRepository(queries.Visits),
	}

	return &App{
		DB:           db,
		Queries:      queries,
		Repositories: repositories,
	}
}

// Close закрывает соединение с БД
func (a *App) Close() error {

	if closeErr := a.DB.Close(); closeErr != nil {
		return fmt.Errorf("failed to close database: %w", closeErr)
	}
	return nil
}
