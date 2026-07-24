package sqlite

import (
	"context"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/daikichiba9511/goat-cv/backend/internal/sqlcgen"
)

// ImageRepository persists image metadata in SQLite.
type ImageRepository struct {
	queries *sqlcgen.Queries
}

// NewImageRepository creates an ImageRepository.
func NewImageRepository(queries *sqlcgen.Queries) *ImageRepository {
	return &ImageRepository{queries: queries}
}

// Create inserts image metadata.
func (r *ImageRepository) Create(ctx context.Context, img domain.Image) (domain.Image, error) {
	row, err := r.queries.CreateImage(ctx, sqlcgen.CreateImageParams{
		ID:             img.ID,
		ProjectID:      img.ProjectID,
		Filename:       img.Filename,
		OriginalWidth:  int64(img.OriginalWidth),
		OriginalHeight: int64(img.OriginalHeight),
		Width:          int64(img.Width),
		Height:         int64(img.Height),
		Rotation:       int64(img.Rotation),
		FlipH:          img.FlipH,
		FlipV:          img.FlipV,
	})
	if err != nil {
		return domain.Image{}, err
	}
	return toImage(row), nil
}

// Get returns image metadata by ID.
func (r *ImageRepository) Get(ctx context.Context, id string) (domain.Image, error) {
	row, err := r.queries.GetImage(ctx, id)
	if err != nil {
		return domain.Image{}, err
	}
	return toImage(row), nil
}

// ListByProject returns images for a project.
func (r *ImageRepository) ListByProject(ctx context.Context, projectID string) ([]domain.Image, error) {
	rows, err := r.queries.ListImagesByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	images := make([]domain.Image, len(rows))
	for i, row := range rows {
		images[i] = toImage(row)
	}
	return images, nil
}

// ListByProjectAndStatus returns images for a project filtered by status.
func (r *ImageRepository) ListByProjectAndStatus(ctx context.Context, projectID string, status domain.ImageStatus) ([]domain.Image, error) {
	rows, err := r.queries.ListImagesByProjectAndStatus(ctx, sqlcgen.ListImagesByProjectAndStatusParams{
		ProjectID: projectID,
		Status:    string(status),
	})
	if err != nil {
		return nil, err
	}
	images := make([]domain.Image, len(rows))
	for i, row := range rows {
		images[i] = toImage(row)
	}
	return images, nil
}

// UpdateTransform changes image transform metadata.
func (r *ImageRepository) UpdateTransform(ctx context.Context, id string, rotation domain.Rotation, flipH, flipV bool, width, height int) (domain.Image, error) {
	row, err := r.queries.UpdateImageTransform(ctx, sqlcgen.UpdateImageTransformParams{
		Rotation: int64(rotation),
		FlipH:    flipH,
		FlipV:    flipV,
		Width:    int64(width),
		Height:   int64(height),
		ID:       id,
	})
	if err != nil {
		return domain.Image{}, err
	}
	return toImage(row), nil
}

// UpdateWorkflow changes both Image workflow dimensions in one statement.
func (r *ImageRepository) UpdateWorkflow(ctx context.Context, id string, status domain.ImageStatus, escalated bool) (domain.Image, error) {
	row, err := r.queries.UpdateImageWorkflow(ctx, sqlcgen.UpdateImageWorkflowParams{
		Status:    string(status),
		Escalated: escalated,
		ID:        id,
	})
	if err != nil {
		return domain.Image{}, err
	}
	return toImage(row), nil
}

// Delete removes image metadata by ID.
func (r *ImageRepository) Delete(ctx context.Context, id string) error {
	return r.queries.DeleteImage(ctx, id)
}
