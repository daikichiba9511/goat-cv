package usecase

import (
	"context"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/google/uuid"
)

type projectRepository interface {
	Create(ctx context.Context, id, name string) (domain.Project, error)
	Get(ctx context.Context, id string) (domain.Project, error)
	List(ctx context.Context) ([]domain.Project, error)
	Update(ctx context.Context, id, name string) (domain.Project, error)
	Delete(ctx context.Context, id string) error
}

// ProjectUsecase coordinates project operations.
type ProjectUsecase struct {
	repo projectRepository
}

// NewProjectUsecase creates a ProjectUsecase.
func NewProjectUsecase(repo projectRepository) *ProjectUsecase {
	return &ProjectUsecase{repo: repo}
}

// Create creates a project with a generated UUID v7.
func (u *ProjectUsecase) Create(ctx context.Context, name string) (domain.Project, error) {
	id := uuid.Must(uuid.NewV7()).String()
	return u.repo.Create(ctx, id, name)
}

// Get returns a project by ID.
func (u *ProjectUsecase) Get(ctx context.Context, id string) (domain.Project, error) {
	return u.repo.Get(ctx, id)
}

// List returns all projects.
func (u *ProjectUsecase) List(ctx context.Context) ([]domain.Project, error) {
	return u.repo.List(ctx)
}

// Update changes a project's name.
func (u *ProjectUsecase) Update(ctx context.Context, id, name string) (domain.Project, error) {
	return u.repo.Update(ctx, id, name)
}

// Delete removes a project by ID.
func (u *ProjectUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, id)
}
