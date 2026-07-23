package sqlite

import (
	"context"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/daikichiba9511/goat-cv/backend/internal/sqlcgen"
)

// LabelRepository persists label definitions in SQLite.
type LabelRepository struct {
	queries *sqlcgen.Queries
}

// NewLabelRepository creates a LabelRepository.
func NewLabelRepository(queries *sqlcgen.Queries) *LabelRepository {
	return &LabelRepository{queries: queries}
}

// Create inserts a label definition.
func (r *LabelRepository) Create(ctx context.Context, label domain.LabelDefinition) (domain.LabelDefinition, error) {
	row, err := r.queries.CreateLabelDefinition(ctx, sqlcgen.CreateLabelDefinitionParams{
		ID:        label.ID,
		ProjectID: label.ProjectID,
		Name:      label.Name,
		Color:     label.Color,
		Category:  string(label.Category),
	})
	if err != nil {
		return domain.LabelDefinition{}, err
	}
	return toLabelDefinition(row), nil
}

// Get returns a label definition by ID.
func (r *LabelRepository) Get(ctx context.Context, id string) (domain.LabelDefinition, error) {
	row, err := r.queries.GetLabelDefinition(ctx, id)
	if err != nil {
		return domain.LabelDefinition{}, err
	}
	return toLabelDefinition(row), nil
}

// ListByProject returns label definitions for a project.
func (r *LabelRepository) ListByProject(ctx context.Context, projectID string) ([]domain.LabelDefinition, error) {
	rows, err := r.queries.ListLabelDefinitions(ctx, projectID)
	if err != nil {
		return nil, err
	}
	labels := make([]domain.LabelDefinition, len(rows))
	for i, row := range rows {
		labels[i] = toLabelDefinition(row)
	}
	return labels, nil
}

// Update changes a label definition.
func (r *LabelRepository) Update(ctx context.Context, label domain.LabelDefinition) (domain.LabelDefinition, error) {
	row, err := r.queries.UpdateLabelDefinition(ctx, sqlcgen.UpdateLabelDefinitionParams{
		Name:     label.Name,
		Color:    label.Color,
		Category: string(label.Category),
		ID:       label.ID,
	})
	if err != nil {
		return domain.LabelDefinition{}, err
	}
	return toLabelDefinition(row), nil
}

// Delete removes a label definition by ID.
func (r *LabelRepository) Delete(ctx context.Context, id string) error {
	return r.queries.DeleteLabelDefinition(ctx, id)
}
