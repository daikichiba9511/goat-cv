package usecase

import (
	"context"

	"github.com/chibadaimare/goat/backend/internal/domain"
	"github.com/chibadaimare/goat/backend/internal/repository/sqlite"
	"github.com/google/uuid"
)

type ProjectUsecase struct {
	repo *sqlite.ProjectRepository
}

func NewProjectUsecase(repo *sqlite.ProjectRepository) *ProjectUsecase {
	return &ProjectUsecase{repo: repo}
}

func (u *ProjectUsecase) Create(ctx context.Context, name string) (domain.Project, error) {
	id := uuid.Must(uuid.NewV7()).String()
	return u.repo.Create(ctx, id, name)
}

func (u *ProjectUsecase) Get(ctx context.Context, id string) (domain.Project, error) {
	return u.repo.Get(ctx, id)
}

func (u *ProjectUsecase) List(ctx context.Context) ([]domain.Project, error) {
	return u.repo.List(ctx)
}

func (u *ProjectUsecase) Update(ctx context.Context, id, name string) (domain.Project, error) {
	return u.repo.Update(ctx, id, name)
}

func (u *ProjectUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, id)
}
