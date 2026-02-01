package services

import (
	"context"
	linksdb "db200/internal/db/links"
)

// LinkService обрабатывает бизнес-логику для ссылок
type LinkService struct {
	queries *linksdb.Queries
}

// NewLinkService создает сервис
func NewLinkService(queries *linksdb.Queries) *LinkService {
	return &LinkService{
		queries: queries,
	}
}

func (s *LinkService) Create(ctx context.Context, params linksdb.CreateLinkParams) error {

	return s.queries.CreateLink(ctx, params)
}

func (s *LinkService) GetAllLinks(ctx context.Context) ([]linksdb.Link, error) {
	return s.queries.GetAllLinks(ctx)
}

func (s *LinkService) GetLink(ctx context.Context, id int32) (linksdb.Link, error) {
	return s.queries.GetLink(ctx, id)
}

func (s *LinkService) DeleteLink(ctx context.Context, id int32) error {
	return s.queries.DeleteLink(ctx, id)
}
