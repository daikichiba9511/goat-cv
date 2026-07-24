package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/google/uuid"
)

var (
	// ErrInvalidAnnotationType indicates an unsupported annotation type.
	ErrInvalidAnnotationType = errors.New("invalid annotation type")
	// ErrInvalidAnnotationCoordinates indicates coordinates that do not satisfy the annotation type schema.
	ErrInvalidAnnotationCoordinates = errors.New("invalid annotation coordinates")
	// ErrDuplicateAnnotationID indicates repeated persistent IDs in one replacement set.
	ErrDuplicateAnnotationID = errors.New("duplicate annotation id")
)

type annotationRepository interface {
	Create(ctx context.Context, annotation domain.Annotation) (domain.Annotation, error)
	Get(ctx context.Context, id string) (domain.Annotation, error)
	ListByImage(ctx context.Context, imageID string) ([]domain.Annotation, error)
	Update(ctx context.Context, annotation domain.Annotation) (domain.Annotation, error)
	Delete(ctx context.Context, id string) error
	BulkReplace(ctx context.Context, imageID string, annotations []domain.Annotation) ([]domain.Annotation, error)
}

// AnnotationUsecase coordinates annotation operations.
type AnnotationUsecase struct {
	repo           annotationRepository
	workflowImages imageWorkflowReader
}

// NewAnnotationUsecase creates an AnnotationUsecase.
func NewAnnotationUsecase(repo annotationRepository, workflowImages imageWorkflowReader) *AnnotationUsecase {
	return &AnnotationUsecase{repo: repo, workflowImages: workflowImages}
}

// Create creates an annotation for an image.
func (u *AnnotationUsecase) Create(ctx context.Context, imageID string, annType domain.AnnotationType, coordinates domain.Coordinates, labelID *string) (domain.Annotation, error) {
	if err := requireImageWorkflowOperationForImage(
		ctx,
		u.workflowImages,
		imageID,
		ImageWorkflowOperationGraphEdit,
	); err != nil {
		return domain.Annotation{}, err
	}
	if err := validateAnnotationCoordinates(annType, coordinates); err != nil {
		return domain.Annotation{}, err
	}

	ann := domain.Annotation{
		ID:          uuid.Must(uuid.NewV7()).String(),
		ImageID:     imageID,
		Type:        annType,
		Coordinates: coordinates,
		LabelID:     labelID,
	}
	return u.repo.Create(ctx, ann)
}

// ListByImage returns annotations for an image.
func (u *AnnotationUsecase) ListByImage(ctx context.Context, imageID string) ([]domain.Annotation, error) {
	return u.repo.ListByImage(ctx, imageID)
}

// Update changes an annotation.
func (u *AnnotationUsecase) Update(ctx context.Context, id string, annType domain.AnnotationType, coordinates domain.Coordinates, labelID *string) (domain.Annotation, error) {
	if err := u.requireGraphEditForAnnotation(ctx, id); err != nil {
		return domain.Annotation{}, err
	}
	if err := validateAnnotationCoordinates(annType, coordinates); err != nil {
		return domain.Annotation{}, err
	}

	ann := domain.Annotation{
		ID:          id,
		Type:        annType,
		Coordinates: coordinates,
		LabelID:     labelID,
	}
	return u.repo.Update(ctx, ann)
}

// Delete removes an annotation by ID.
func (u *AnnotationUsecase) Delete(ctx context.Context, id string) error {
	if err := u.requireGraphEditForAnnotation(ctx, id); err != nil {
		return err
	}
	return u.repo.Delete(ctx, id)
}

func (u *AnnotationUsecase) requireGraphEditForAnnotation(ctx context.Context, annotationID string) error {
	// Why: item routeにはImage IDがないため、永続化済みAnnotationの所有Imageを認可判定の正とする。
	existingAnnotation, err := u.repo.Get(ctx, annotationID)
	if err != nil {
		return err
	}
	return requireImageWorkflowOperationForImage(
		ctx,
		u.workflowImages,
		existingAnnotation.ImageID,
		ImageWorkflowOperationGraphEdit,
	)
}

// BulkReplace replaces all annotations for an image and returns the persisted rows.
// Annotations with an empty ID are treated as new records and receive UUID v7 IDs.
func (u *AnnotationUsecase) BulkReplace(ctx context.Context, imageID string, annotations []domain.Annotation) ([]domain.Annotation, error) {
	if err := requireImageWorkflowOperationForImage(
		ctx,
		u.workflowImages,
		imageID,
		ImageWorkflowOperationGraphEdit,
	); err != nil {
		return nil, err
	}
	// Why: フロントエンドは未保存Annotationを一時IDで扱うため、永続化境界でだけUUID v7へ置き換える。
	// Why not: Phase 1では操作ログ同期をしないので、個別差分ではなく画像単位の現在状態を正とする。
	candidateAnnotations := make([]domain.Annotation, len(annotations))
	usedAnnotationIDs := make(map[string]struct{}, len(annotations))
	for i, annotation := range annotations {
		if err := validateAnnotationCoordinates(annotation.Type, annotation.Coordinates); err != nil {
			return nil, fmt.Errorf("annotations[%d]: %w", i, err)
		}
		if annotation.ID == "" {
			annotation.ID = uuid.Must(uuid.NewV7()).String()
		}
		if _, exists := usedAnnotationIDs[annotation.ID]; exists {
			return nil, fmt.Errorf("annotations[%d]: %w %q", i, ErrDuplicateAnnotationID, annotation.ID)
		}
		usedAnnotationIDs[annotation.ID] = struct{}{}
		annotation.ImageID = imageID
		candidateAnnotations[i] = annotation
	}
	return u.repo.BulkReplace(ctx, imageID, candidateAnnotations)
}
