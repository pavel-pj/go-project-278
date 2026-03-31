package services

import (
	visitsdb "db200/internal/db/visits"
)

// LinkService обрабатывает бизнес-логику для ссылок
type VisitService struct {
	queries *visitsdb.Queries
}

// NewLinkService создает сервис
func NewLVisitService(queries *visitsdb.Queries) *VisitService {
	return &VisitService{
		queries: queries,
	}
}
