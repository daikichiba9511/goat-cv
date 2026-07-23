package usecase

import (
	"context"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/daikichiba9511/goat-cv/backend/internal/repository/sqlite"
	"github.com/google/uuid"
)

// LabelUsecase coordinates label definition operations.
type LabelUsecase struct {
	repo *sqlite.LabelRepository
}

// NewLabelUsecase creates a LabelUsecase.
func NewLabelUsecase(repo *sqlite.LabelRepository) *LabelUsecase {
	return &LabelUsecase{repo: repo}
}

// Create creates a label definition for a project.
func (u *LabelUsecase) Create(ctx context.Context, projectID, name, color string, category domain.LabelCategory) (domain.LabelDefinition, error) {
	label := domain.LabelDefinition{
		ID:        uuid.Must(uuid.NewV7()).String(),
		ProjectID: projectID,
		Name:      name,
		Color:     color,
		Category:  category,
	}
	return u.repo.Create(ctx, label)
}

// ListByProject returns label definitions for a project.
func (u *LabelUsecase) ListByProject(ctx context.Context, projectID string) ([]domain.LabelDefinition, error) {
	return u.repo.ListByProject(ctx, projectID)
}

// Update changes a label definition.
func (u *LabelUsecase) Update(ctx context.Context, id, name, color string, category domain.LabelCategory) (domain.LabelDefinition, error) {
	label := domain.LabelDefinition{
		ID:       id,
		Name:     name,
		Color:    color,
		Category: category,
	}
	return u.repo.Update(ctx, label)
}

// Delete removes a label definition by ID.
func (u *LabelUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, id)
}
