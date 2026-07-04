package sqlite

import (
	"context"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/daikichiba9511/goat-cv/backend/internal/sqlcgen"
)

type ProjectRepository struct {
	q *sqlcgen.Queries
}

func NewProjectRepository(q *sqlcgen.Queries) *ProjectRepository {
	return &ProjectRepository{q: q}
}

func (r *ProjectRepository) Create(ctx context.Context, id, name string) (domain.Project, error) {
	row, err := r.q.CreateProject(ctx, sqlcgen.CreateProjectParams{ID: id, Name: name})
	if err != nil {
		return domain.Project{}, err
	}
	return toProject(row), nil
}

func (r *ProjectRepository) Get(ctx context.Context, id string) (domain.Project, error) {
	row, err := r.q.GetProject(ctx, id)
	if err != nil {
		return domain.Project{}, err
	}
	return toProject(row), nil
}

func (r *ProjectRepository) List(ctx context.Context) ([]domain.Project, error) {
	rows, err := r.q.ListProjects(ctx)
	if err != nil {
		return nil, err
	}
	projects := make([]domain.Project, len(rows))
	for i, row := range rows {
		projects[i] = toProject(row)
	}
	return projects, nil
}

func (r *ProjectRepository) Update(ctx context.Context, id, name string) (domain.Project, error) {
	row, err := r.q.UpdateProject(ctx, sqlcgen.UpdateProjectParams{Name: name, ID: id})
	if err != nil {
		return domain.Project{}, err
	}
	return toProject(row), nil
}

func (r *ProjectRepository) Delete(ctx context.Context, id string) error {
	return r.q.DeleteProject(ctx, id)
}
