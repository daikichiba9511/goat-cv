package sqlite

import (
	"context"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/daikichiba9511/goat-cv/backend/internal/sqlcgen"
)

// ProjectRepository persists projects in SQLite.
type ProjectRepository struct {
	queries *sqlcgen.Queries
}

// NewProjectRepository creates a ProjectRepository.
func NewProjectRepository(queries *sqlcgen.Queries) *ProjectRepository {
	return &ProjectRepository{queries: queries}
}

// Create inserts a project.
func (r *ProjectRepository) Create(ctx context.Context, id, name string) (domain.Project, error) {
	row, err := r.queries.CreateProject(ctx, sqlcgen.CreateProjectParams{ID: id, Name: name})
	if err != nil {
		return domain.Project{}, err
	}
	return toProject(row), nil
}

// Get returns a project by ID.
func (r *ProjectRepository) Get(ctx context.Context, id string) (domain.Project, error) {
	row, err := r.queries.GetProject(ctx, id)
	if err != nil {
		return domain.Project{}, err
	}
	return toProject(row), nil
}

// List returns all projects.
func (r *ProjectRepository) List(ctx context.Context) ([]domain.Project, error) {
	rows, err := r.queries.ListProjects(ctx)
	if err != nil {
		return nil, err
	}
	projects := make([]domain.Project, len(rows))
	for i, row := range rows {
		projects[i] = toProject(row)
	}
	return projects, nil
}

// Update changes a project's name.
func (r *ProjectRepository) Update(ctx context.Context, id, name string) (domain.Project, error) {
	row, err := r.queries.UpdateProject(ctx, sqlcgen.UpdateProjectParams{Name: name, ID: id})
	if err != nil {
		return domain.Project{}, err
	}
	return toProject(row), nil
}

// Delete removes a project by ID.
func (r *ProjectRepository) Delete(ctx context.Context, id string) error {
	return r.queries.DeleteProject(ctx, id)
}
