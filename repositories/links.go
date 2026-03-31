// Package services provides business logic layer for handling link operations.
// It encapsulates the database queries and implements validation, generation,
// and other business rules for managing shortened links.
package repositories

import (
	"context"
	"crypto/rand"
	"database/sql"
	linksdb "db200/internal/db/links"
	"errors"
	"fmt"
	"math/big"
)

// ErrCannotGenerateUniqueName is returned when the system fails to generate
// a unique short name after multiple attempts.
var (
	ErrCannotGenerateUniqueName = errors.New("cannot generate unique short name")
)

// LinkService обрабатывает бизнес-логику для ссылок
type LinkRepository struct {
	queries *linksdb.Queries
}

// NewLinkService создает сервис
func NewLinkRepository(queries *linksdb.Queries) *LinkRepository {
	return &LinkRepository{
		queries: queries,
	}
}

// Create creates a new link in the database with the provided parameters.
// It returns the created link and any error encountered during the operation.
func (s *LinkRepository) Create(ctx context.Context, params linksdb.CreateLinkParams) (linksdb.Link, error) {

	link, err := s.queries.CreateLink(ctx, params)
	if err != nil {
		return linksdb.Link{}, fmt.Errorf("failed to create link: %w", err)
	}
	return link, nil

}

// GetAllLinks retrieves all links from the database.
// It returns a slice of links and any error encountered during the operation.
func (s *LinkRepository) GetAllLinks(ctx context.Context, params linksdb.GetAllLinksParams) ([]linksdb.Link, error) {
	links, err := s.queries.GetAllLinks(ctx, params)
	if err != nil {
		return []linksdb.Link{}, fmt.Errorf("failed to get all links:  %w", err)
	}
	return links, nil
}

// GetLinksCount retrieves count of links from the database.
func (s *LinkRepository) GetLinksCount(ctx context.Context) (int64, error) {
	count, err := s.queries.GetLinksCount(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get all links:  %w", err)
	}
	return count, nil
}

// GetLink retrieves a link from the database by its unique ID.
// It returns the link and any error encountered during the operation.
func (s *LinkRepository) GetLink(ctx context.Context, id int32) (linksdb.Link, error) {

	link, err := s.queries.GetLink(ctx, id)
	if err != nil {
		return linksdb.Link{}, fmt.Errorf("failed to get link by id: %d: %w", id, err)
	}
	return link, nil
}

// DeleteLink removes a link from the database by its ID.
// It returns an error if the deletion operation fails.
func (s *LinkRepository) DeleteLink(ctx context.Context, id int32) error {
	err := s.queries.DeleteLink(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete link by id: %d: %w", id, err)
	}
	return nil
}

// GetByOriginalURL retrieves a link by its original URL.
func (s *LinkRepository) GetByOriginalURL(ctx context.Context, originalURL string) (linksdb.Link, error) {
	link, err := s.queries.GetLinkByOriginURL(ctx, originalURL)
	if err != nil {
		return linksdb.Link{}, fmt.Errorf("failed to get link by orinal url: %s: %w", originalURL, err)
	}
	return link, nil

}

// GetLinkByOriginURLExludedID retrieves a link by its original URL while excluding
// a specific link ID from the search results. This is useful for checking if a URL
// already exists when updating a link, excluding the current link being edited.
func (s *LinkRepository) GetLinkByOriginURLExludedID(
	ctx context.Context,
	params linksdb.GetLinkByOriginURLExludedIDParams) (linksdb.Link, error) {

	link, err := s.queries.GetLinkByOriginURLExludedID(ctx, params)
	if err != nil {
		return linksdb.Link{}, fmt.Errorf("failed get link by origin url:%s exluded id: %d: %w", params.OriginalUrl, params.ID, err)
	}
	return link, nil

}

// GetLinkByShortName retrieves a link by its short name from the database.
// It returns the link and any error encountered during the operation.
func (s *LinkRepository) GetLinkByShortName(ctx context.Context, shortName string) (linksdb.Link, error) {
	link, err := s.queries.GetLinkByShortName(ctx, shortName)
	if err != nil {
		return linksdb.Link{}, fmt.Errorf("failed get link by short name %s: %w", shortName, err)
	}
	return link, nil

}

// GetLinkByShortNameExludedID retrieves a link by short name excluding a specific ID.
// Used for checking duplicate short names during updates.
func (s *LinkRepository) GetLinkByShortNameExludedID(
	ctx context.Context,
	params linksdb.GetLinkByShortNameExcluedeIDParams) (linksdb.Link, error) {
	link, err := s.queries.GetLinkByShortNameExcluedeID(ctx, params)
	if err != nil {
		return linksdb.Link{}, fmt.Errorf("failed to get link by short name: %s, with id %d: %w", params.ShortName, params.ID, err)
	}
	return link, nil

}

// GenerateShortName create a unique URL for the link if a user doesnt provide a short_name param
func (s *LinkRepository) GenerateShortName(ctx context.Context) (string, error) {
	length := 10
	base62Chars := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

	for attempt := 1; attempt <= 20; attempt++ {
		result := make([]byte, length)
		for i := range result {
			// Генерируем случайное число от 0 до len(base62Chars)-1
			num, err := rand.Int(rand.Reader, big.NewInt(int64(len(base62Chars))))
			if err != nil {
				return "", fmt.Errorf("failed to generate random number: %w", err)
			}
			result[i] = base62Chars[num.Int64()]
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
func (s *LinkRepository) UpdateLink(
	ctx context.Context,
	params linksdb.UpdateLinkParams) (linksdb.Link, error) {
	link, err := s.queries.UpdateLink(ctx, params)
	if err != nil {
		return linksdb.Link{}, fmt.Errorf("failed to update link with id %d: %w", params.ID, err)
	}
	return link, nil
}
