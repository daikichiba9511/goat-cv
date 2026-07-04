package usecase

import (
	"context"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/daikichiba9511/goat-cv/backend/internal/repository/sqlite"
	"github.com/google/uuid"
)

type LabelUsecase struct {
	repo *sqlite.LabelRepository
}

func NewLabelUsecase(repo *sqlite.LabelRepository) *LabelUsecase {
	return &LabelUsecase{repo: repo}
}

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

func (u *LabelUsecase) ListByProject(ctx context.Context, projectID string) ([]domain.LabelDefinition, error) {
	return u.repo.ListByProject(ctx, projectID)
}

func (u *LabelUsecase) Update(ctx context.Context, id, name, color string, category domain.LabelCategory) (domain.LabelDefinition, error) {
	label := domain.LabelDefinition{
		ID:       id,
		Name:     name,
		Color:    color,
		Category: category,
	}
	return u.repo.Update(ctx, label)
}

func (u *LabelUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, id)
}
