package repositories

import (
	"context"
	visitsdb "db200/internal/db/visits"
	"fmt"
)

// LinkService обрабатывает бизнес-логику для ссылок
type VisitRepository struct {
	queries *visitsdb.Queries
}

// NewLinkService создает сервис
func NewVisitRepository(queries *visitsdb.Queries) *VisitRepository {
	return &VisitRepository{
		queries: queries,
	}
}

// Create a new visit_link in the database with the provided parameters.
func (r *VisitRepository) Create(ctx context.Context, params visitsdb.CreateVisitParams) (visitsdb.LinkVisit, error) {

	visit, err := r.queries.CreateVisit(ctx, params)
	if err != nil {
		return visitsdb.LinkVisit{}, fmt.Errorf("failed to create visit: %w", err)
	}
	return visit, nil

}

// Get all visit_links
func (r *VisitRepository) GetVisits(ctx context.Context, params visitsdb.GetVisitsParams) ([]visitsdb.LinkVisit, error) {

	visits, err := r.queries.GetVisits(ctx, params)
	if err != nil {
		return []visitsdb.LinkVisit{}, fmt.Errorf("failed to get visits: %w", err)
	}
	return visits, nil

}

// Get visit_links counts
func (r *VisitRepository) GetVisitsCount(ctx context.Context) (int64, error) {

	count, err := r.queries.GetVisitsCount(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get visits: %w", err)
	}
	return count, nil

}
