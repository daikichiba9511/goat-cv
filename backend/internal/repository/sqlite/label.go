package sqlite

import (
	"context"

	"github.com/chibadaimare/goat/backend/internal/domain"
	"github.com/chibadaimare/goat/backend/internal/sqlcgen"
)

type LabelRepository struct {
	q *sqlcgen.Queries
}

func NewLabelRepository(q *sqlcgen.Queries) *LabelRepository {
	return &LabelRepository{q: q}
}

func (r *LabelRepository) Create(ctx context.Context, label domain.LabelDefinition) (domain.LabelDefinition, error) {
	row, err := r.q.CreateLabelDefinition(ctx, sqlcgen.CreateLabelDefinitionParams{
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

func (r *LabelRepository) Get(ctx context.Context, id string) (domain.LabelDefinition, error) {
	row, err := r.q.GetLabelDefinition(ctx, id)
	if err != nil {
		return domain.LabelDefinition{}, err
	}
	return toLabelDefinition(row), nil
}

func (r *LabelRepository) ListByProject(ctx context.Context, projectID string) ([]domain.LabelDefinition, error) {
	rows, err := r.q.ListLabelDefinitions(ctx, projectID)
	if err != nil {
		return nil, err
	}
	labels := make([]domain.LabelDefinition, len(rows))
	for i, row := range rows {
		labels[i] = toLabelDefinition(row)
	}
	return labels, nil
}

func (r *LabelRepository) Update(ctx context.Context, label domain.LabelDefinition) (domain.LabelDefinition, error) {
	row, err := r.q.UpdateLabelDefinition(ctx, sqlcgen.UpdateLabelDefinitionParams{
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

func (r *LabelRepository) Delete(ctx context.Context, id string) error {
	return r.q.DeleteLabelDefinition(ctx, id)
}
