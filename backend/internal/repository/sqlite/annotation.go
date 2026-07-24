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

	result, err := replaceAnnotations(ctx, txQueries, imageID, annotations)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

func replaceAnnotations(
	ctx context.Context,
	queries *sqlcgen.Queries,
	imageID string,
	annotations []domain.Annotation,
) ([]domain.Annotation, error) {
	existingRows, err := queries.ListAnnotationsByImage(ctx, imageID)
	if err != nil {
		return nil, err
	}
	existingByID := make(map[string]sqlcgen.Annotation, len(existingRows))
	for _, row := range existingRows {
		existingByID[row.ID] = row
	}
	incomingIDs := make(map[string]struct{}, len(annotations))
	for _, annotation := range annotations {
		incomingIDs[annotation.ID] = struct{}{}
	}

	for annotationID := range existingByID {
		if _, remains := incomingIDs[annotationID]; remains {
			continue
		}
		if err := queries.DeleteAnnotation(ctx, annotationID); err != nil {
			return nil, err
		}
	}

	persisted := make([]domain.Annotation, len(annotations))
	for annotationIndex, annotation := range annotations {
		var row sqlcgen.Annotation
		if _, exists := existingByID[annotation.ID]; exists {
			row, err = queries.UpdateAnnotation(ctx, sqlcgen.UpdateAnnotationParams{
				Type:        string(annotation.Type),
				Coordinates: string(annotation.Coordinates),
				LabelID:     toNullString(annotation.LabelID),
				ID:          annotation.ID,
			})
		} else {
			row, err = queries.CreateAnnotation(ctx, sqlcgen.CreateAnnotationParams{
				ID:          annotation.ID,
				ImageID:     imageID,
				Type:        string(annotation.Type),
				Coordinates: string(annotation.Coordinates),
				LabelID:     toNullString(annotation.LabelID),
			})
		}
		if err != nil {
			return nil, err
		}
		persisted[annotationIndex] = toAnnotation(row)
	}
	return persisted, nil
}
