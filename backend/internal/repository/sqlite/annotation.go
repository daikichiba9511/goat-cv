package sqlite

import (
	"context"
	"database/sql"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/daikichiba9511/goat-cv/backend/internal/sqlcgen"
)

type AnnotationRepository struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewAnnotationRepository(db *sql.DB, q *sqlcgen.Queries) *AnnotationRepository {
	return &AnnotationRepository{db: db, q: q}
}

func (r *AnnotationRepository) Create(ctx context.Context, ann domain.Annotation) (domain.Annotation, error) {
	row, err := r.q.CreateAnnotation(ctx, sqlcgen.CreateAnnotationParams{
		ID:          ann.ID,
		ImageID:     ann.ImageID,
		Type:        string(ann.Type),
		Coordinates: string(ann.Coordinates),
		LabelID:     toNullString(ann.LabelID),
	})
	if err != nil {
		return domain.Annotation{}, err
	}
	return toAnnotation(row), nil
}

func (r *AnnotationRepository) Get(ctx context.Context, id string) (domain.Annotation, error) {
	row, err := r.q.GetAnnotation(ctx, id)
	if err != nil {
		return domain.Annotation{}, err
	}
	return toAnnotation(row), nil
}

func (r *AnnotationRepository) ListByImage(ctx context.Context, imageID string) ([]domain.Annotation, error) {
	rows, err := r.q.ListAnnotationsByImage(ctx, imageID)
	if err != nil {
		return nil, err
	}
	annotations := make([]domain.Annotation, len(rows))
	for i, row := range rows {
		annotations[i] = toAnnotation(row)
	}
	return annotations, nil
}

func (r *AnnotationRepository) Update(ctx context.Context, ann domain.Annotation) (domain.Annotation, error) {
	row, err := r.q.UpdateAnnotation(ctx, sqlcgen.UpdateAnnotationParams{
		Type:        string(ann.Type),
		Coordinates: string(ann.Coordinates),
		LabelID:     toNullString(ann.LabelID),
		ID:          ann.ID,
	})
	if err != nil {
		return domain.Annotation{}, err
	}
	return toAnnotation(row), nil
}

func (r *AnnotationRepository) Delete(ctx context.Context, id string) error {
	return r.q.DeleteAnnotation(ctx, id)
}

// BulkReplace deletes all annotations for an image and inserts new ones in a transaction.
func (r *AnnotationRepository) BulkReplace(ctx context.Context, imageID string, annotations []domain.Annotation) ([]domain.Annotation, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	qtx := r.q.WithTx(tx)

	if err := qtx.DeleteAnnotationsByImage(ctx, imageID); err != nil {
		return nil, err
	}

	result := make([]domain.Annotation, len(annotations))
	for i, ann := range annotations {
		row, err := qtx.CreateAnnotation(ctx, sqlcgen.CreateAnnotationParams{
			ID:          ann.ID,
			ImageID:     imageID,
			Type:        string(ann.Type),
			Coordinates: string(ann.Coordinates),
			LabelID:     toNullString(ann.LabelID),
		})
		if err != nil {
			return nil, err
		}
		result[i] = toAnnotation(row)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}
