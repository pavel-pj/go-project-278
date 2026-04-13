package app

import (
	"context"
	"db200/internal/db/generated"
	"db200/internal/services"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// App хранит ВСЕ зависимости приложения
type App struct {
	DB       *pgxpool.Pool
	Queries  *generated.Queries
	Services *Service
}

type Service struct {
	Links *services.LinkService
}

func NewApp(dbURL string) (*App, error) { // ← принимаем URL, а не *sql.DB
	// Создаем пул соединений pgx
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Проверяем подключение
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("SUPER ERROR failed to ping database: %w", err)
	}

	// Создаем queries (теперь совместимо!)
	queries := generated.New(pool)

	services := &Service{
		Links: services.NewLinkService(queries),
	}

	return &App{
		DB:       pool,
		Queries:  queries,
		Services: services,
	}, nil
}

// Close закрывает соединение с БД
func (a *App) Close() error {
	if a.DB != nil {
		a.DB.Close()
	}
	return nil
}
