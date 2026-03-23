package services

import (
	"context"
	"database/sql"
	linksdb "db200/internal/db/links"
	"errors"
	"fmt"
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

	link, err := s.queries.CreateLink(ctx, params)
	if err != nil {
		return linksdb.Link{}, fmt.Errorf("failed to create link: %w", err)
	}
	return link, nil

}

func (s *LinkService) GetAllLinks(ctx context.Context) ([]linksdb.Link, error) {
	links, err := s.queries.GetAllLinks(ctx)
	if err != nil {
		return []linksdb.Link{}, fmt.Errorf("failed to get all links:  %w", err)
	}
	return links, nil

}

func (s *LinkService) GetLink(ctx context.Context, id int32) (linksdb.Link, error) {

	link, err := s.queries.GetLink(ctx, id)
	if err != nil {
		return linksdb.Link{}, fmt.Errorf("failed to get link by id: %d: %w", id, err)
	}
	return link, nil
}

func (s *LinkService) DeleteLink(ctx context.Context, id int32) error {
	err := s.queries.DeleteLink(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete link by id: %d: %w", id, err)
	}
	return nil
}

// GetByOriginalURL retrieves a link by its original URL.
func (s *LinkService) GetByOriginalUrl(ctx context.Context, originalUrl string) (linksdb.Link, error) {
	link, err := s.queries.GetLinkByOriginUrl(ctx, originalUrl)
	if err != nil {
		return linksdb.Link{}, fmt.Errorf("failed to get link by orinal url: %s: %w", originalUrl, err)
	}
	return link, nil

}
func (s *LinkService) GetLinkByOriginUrlExludedID(
	ctx context.Context,
	params linksdb.GetLinkByOriginUrlExludedIDParams) (linksdb.Link, error) {

	link, err := s.queries.GetLinkByOriginUrlExludedID(ctx, params)
	if err != nil {
		return linksdb.Link{}, fmt.Errorf("failed get link by origin url:%s exluded id: %d: %w", params.OriginalUrl, params.ID, err)
	}
	return link, nil

}

func (s *LinkService) GetLinkByShortName(ctx context.Context, shortName string) (linksdb.Link, error) {
	link, err := s.queries.GetLinkByShortName(ctx, shortName)
	if err != nil {
		return linksdb.Link{}, fmt.Errorf("failed get link by short name %s: %w", shortName, err)
	}
	return link, nil

}

// GetLinkByShortNameExludedID retrieves a link by short name excluding a specific ID.
// Used for checking duplicate short names during updates.
func (s *LinkService) GetLinkByShortNameExludedID(
	ctx context.Context,
	params linksdb.GetLinkByShortNameExcluedeIDParams) (linksdb.Link, error) {
	link, err := s.queries.GetLinkByShortNameExcluedeID(ctx, params)
	if err != nil {
		return linksdb.Link{}, fmt.Errorf("failed to get link by short name: %s, with id %d: %w", params.ShortName, params.ID, err)
	}
	return link, nil

}

// GenerateShortName create a unique URL for the link if a user doesnt provide a short_name param
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

// UpdateLink updates an existing link in the database.
// It takes a context and update parameters, and returns the updated link.
// If the link doesn't exist or there's a database error, an error is returned.
func (s *LinkService) UpdateLink(
	ctx context.Context,
	params linksdb.UpdateLinkParams) (linksdb.Link, error) {
	link, err := s.queries.UpdateLink(ctx, params)
	if err != nil {
		return linksdb.Link{}, fmt.Errorf("failed to update link with id %d: %w", params.ID, err)
	}
	return link, nil
}
