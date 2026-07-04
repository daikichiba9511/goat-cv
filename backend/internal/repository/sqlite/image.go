package sqlite

import (
	"context"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/daikichiba9511/goat-cv/backend/internal/sqlcgen"
)

type ImageRepository struct {
	q *sqlcgen.Queries
}

func NewImageRepository(q *sqlcgen.Queries) *ImageRepository {
	return &ImageRepository{q: q}
}

func (r *ImageRepository) Create(ctx context.Context, img domain.Image) (domain.Image, error) {
	row, err := r.q.CreateImage(ctx, sqlcgen.CreateImageParams{
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

func (r *ImageRepository) Get(ctx context.Context, id string) (domain.Image, error) {
	row, err := r.q.GetImage(ctx, id)
	if err != nil {
		return domain.Image{}, err
	}
	return toImage(row), nil
}

func (r *ImageRepository) ListByProject(ctx context.Context, projectID string) ([]domain.Image, error) {
	rows, err := r.q.ListImagesByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	images := make([]domain.Image, len(rows))
	for i, row := range rows {
		images[i] = toImage(row)
	}
	return images, nil
}

func (r *ImageRepository) ListByProjectAndStatus(ctx context.Context, projectID string, status domain.ImageStatus) ([]domain.Image, error) {
	rows, err := r.q.ListImagesByProjectAndStatus(ctx, sqlcgen.ListImagesByProjectAndStatusParams{
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

func (r *ImageRepository) UpdateTransform(ctx context.Context, id string, rotation domain.Rotation, flipH, flipV bool, width, height int) (domain.Image, error) {
	row, err := r.q.UpdateImageTransform(ctx, sqlcgen.UpdateImageTransformParams{
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

func (r *ImageRepository) UpdateStatus(ctx context.Context, id string, status domain.ImageStatus) (domain.Image, error) {
	row, err := r.q.UpdateImageStatus(ctx, sqlcgen.UpdateImageStatusParams{
		Status: string(status),
		ID:     id,
	})
	if err != nil {
		return domain.Image{}, err
	}
	return toImage(row), nil
}

func (r *ImageRepository) Delete(ctx context.Context, id string) error {
	return r.q.DeleteImage(ctx, id)
}
