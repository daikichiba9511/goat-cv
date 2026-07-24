package sqlite

import (
	"context"
	"database/sql"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/daikichiba9511/goat-cv/backend/internal/sqlcgen"
)

// GuidelineRepository persists Project-scoped Guideline pages in SQLite.
type GuidelineRepository struct {
	queries *sqlcgen.Queries
}

// NewGuidelineRepository creates a GuidelineRepository.
func NewGuidelineRepository(queries *sqlcgen.Queries) *GuidelineRepository {
	return &GuidelineRepository{queries: queries}
}

// Create inserts a Guideline page.
func (repository *GuidelineRepository) Create(
	ctx context.Context,
	guideline domain.Guideline,
) (domain.Guideline, error) {
	row, err := repository.queries.CreateGuideline(ctx, sqlcgen.CreateGuidelineParams{
		ID:           guideline.ID,
		ProjectID:    guideline.ProjectID,
		Title:        guideline.Title,
		Body:         guideline.Body,
		DisplayOrder: int64(guideline.DisplayOrder),
	})
	if err != nil {
		return domain.Guideline{}, err
	}
	return toGuideline(row), nil
}

// ListByProject returns Guideline pages in stable display order.
func (repository *GuidelineRepository) ListByProject(
	ctx context.Context,
	projectID string,
) ([]domain.Guideline, error) {
	rows, err := repository.queries.ListGuidelinesByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	guidelines := make([]domain.Guideline, len(rows))
	for guidelineIndex, row := range rows {
		guidelines[guidelineIndex] = toGuideline(row)
	}
	return guidelines, nil
}

// Get returns a Guideline only when it belongs to projectID.
func (repository *GuidelineRepository) Get(
	ctx context.Context,
	projectID string,
	guidelineID string,
) (domain.Guideline, error) {
	row, err := repository.queries.GetGuideline(ctx, sqlcgen.GetGuidelineParams{
		ID:        guidelineID,
		ProjectID: projectID,
	})
	if err != nil {
		return domain.Guideline{}, err
	}
	return toGuideline(row), nil
}

// Update changes a Guideline only when it belongs to projectID.
func (repository *GuidelineRepository) Update(
	ctx context.Context,
	projectID string,
	guideline domain.Guideline,
) (domain.Guideline, error) {
	row, err := repository.queries.UpdateGuideline(ctx, sqlcgen.UpdateGuidelineParams{
		Title:        guideline.Title,
		Body:         guideline.Body,
		DisplayOrder: int64(guideline.DisplayOrder),
		ID:           guideline.ID,
		ProjectID:    projectID,
	})
	if err != nil {
		return domain.Guideline{}, err
	}
	return toGuideline(row), nil
}

// Delete removes a Guideline only when it belongs to projectID.
func (repository *GuidelineRepository) Delete(
	ctx context.Context,
	projectID string,
	guidelineID string,
) error {
	deletedRows, err := repository.queries.DeleteGuideline(ctx, sqlcgen.DeleteGuidelineParams{
		ID:        guidelineID,
		ProjectID: projectID,
	})
	if err != nil {
		return err
	}
	if deletedRows == 0 {
		return sql.ErrNoRows
	}
	return nil
}
