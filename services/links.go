package services

import (
	"context"
	"database/sql"
	linksdb "db200/internal/db/links"
	"errors"
	"math/rand"
	"time"
)

var (
	ErrCannotGenerateUniqueName = errors.New("cannot generate unique short name")
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

func (s *LinkService) Create(ctx context.Context, params linksdb.CreateLinkParams) (linksdb.Link, error) {

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

func (s *LinkService) GetByOriginalUrl(ctx context.Context, originalUrl string) (linksdb.Link, error) {
	return s.queries.GetLinkByOriginUrl(ctx, originalUrl)
}
func (s *LinkService) GetLinkByOriginUrlExludedId(
	ctx context.Context,
	params linksdb.GetLinkByOriginUrlExludedIdParams) (linksdb.Link, error) {
	return s.queries.GetLinkByOriginUrlExludedId(ctx, params)
}

func (s *LinkService) GetLinkByShortName(ctx context.Context, shortName string) (linksdb.Link, error) {
	return s.queries.GetLinkByShortName(ctx, shortName)
}

func (s *LinkService) GetLinkByShortNameExludedId(
	ctx context.Context,
	params linksdb.GetLinkByShortNameExcluedeIdParams) (linksdb.Link, error) {
	return s.queries.GetLinkByShortNameExcluedeId(ctx, params)
}

func (s *LinkService) GenerateShortName(ctx context.Context) (string, error) {

	//Generate a string
	length := 10
	base62Chars := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

	for attempt := 1; attempt <= 20; attempt++ {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		result := make([]byte, length)
		for i := range result {
			result[i] = base62Chars[r.Intn(len(base62Chars))]
		}

		_, err := s.queries.GetLinkByShortName(ctx, string(result))
		if err != nil && errors.Is(err, sql.ErrNoRows) {
			return string(result), nil
		}

	}

	return "", ErrCannotGenerateUniqueName

}

func (s *LinkService) UpdateLink(
	ctx context.Context,
	params linksdb.UpdateLinkParams) (linksdb.Link, error) {
	return s.queries.UpdateLink(ctx, params)
}
