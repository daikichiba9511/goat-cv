package sqlite

import (
	"context"
	"database/sql"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/daikichiba9511/goat-cv/backend/internal/sqlcgen"
)

// AnnotationRepository persists annotations in SQLite.
type AnnotationRepository struct {
	db      *sql.DB
	queries *sqlcgen.Queries
}

// NewAnnotationRepository creates an AnnotationRepository.
func NewAnnotationRepository(db *sql.DB, queries *sqlcgen.Queries) *AnnotationRepository {
	return &AnnotationRepository{db: db, queries: queries}
}

// Create inserts an annotation.
func (r *AnnotationRepository) Create(ctx context.Context, ann domain.Annotation) (domain.Annotation, error) {
	row, err := r.queries.CreateAnnotation(ctx, sqlcgen.CreateAnnotationParams{
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

// Get returns an annotation by ID.
func (r *AnnotationRepository) Get(ctx context.Context, id string) (domain.Annotation, error) {
	row, err := r.queries.GetAnnotation(ctx, id)
	if err != nil {
		return domain.Annotation{}, err
	}
	return toAnnotation(row), nil
}

// ListByImage returns annotations for an image.
func (r *AnnotationRepository) ListByImage(ctx context.Context, imageID string) ([]domain.Annotation, error) {
	rows, err := r.queries.ListAnnotationsByImage(ctx, imageID)
	if err != nil {
		return nil, err
	}
	annotations := make([]domain.Annotation, len(rows))
	for i, row := range rows {
		annotations[i] = toAnnotation(row)
	}
	return annotations, nil
}

// Update changes an annotation.
func (r *AnnotationRepository) Update(ctx context.Context, ann domain.Annotation) (domain.Annotation, error) {
	row, err := r.queries.UpdateAnnotation(ctx, sqlcgen.UpdateAnnotationParams{
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

// Delete removes an annotation by ID.
func (r *AnnotationRepository) Delete(ctx context.Context, id string) error {
	return r.queries.DeleteAnnotation(ctx, id)
}

// BulkReplace replaces all annotations for an image and returns the persisted rows.
// The operation is atomic: if any insert fails, the previous annotation set remains in place.
func (r *AnnotationRepository) BulkReplace(ctx context.Context, imageID string, annotations []domain.Annotation) ([]domain.Annotation, error) {
	// Why: Phase 1のUIは画像単位の同期保存なので、差分操作ではなく全置換で保存境界を単純に保つ。
	// Why not: Delete後のInsert失敗で空状態を残さないよう、必ず1トランザクションで実行する。
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	txQueries := r.queries.WithTx(tx)

	if err := txQueries.DeleteAnnotationsByImage(ctx, imageID); err != nil {
		return nil, err
	}

	result := make([]domain.Annotation, len(annotations))
	for i, ann := range annotations {
		row, err := txQueries.CreateAnnotation(ctx, sqlcgen.CreateAnnotationParams{
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
