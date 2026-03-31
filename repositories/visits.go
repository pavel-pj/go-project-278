package repositories

import (
	visitsdb "db200/internal/db/visits"
)

// LinkService обрабатывает бизнес-логику для ссылок
type VisitRepository struct {
	queries *visitsdb.Queries
}

// NewLinkService создает сервис
func NewLVisitRepository(queries *visitsdb.Queries) *VisitRepository {
	return &VisitRepository{
		queries: queries,
	}
}
