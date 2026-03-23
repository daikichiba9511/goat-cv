package usecase

import (
	"context"

	"github.com/chibadaimare/goat/backend/internal/domain"
	"github.com/chibadaimare/goat/backend/internal/repository/sqlite"
	"github.com/google/uuid"
)

type AnnotationUsecase struct {
	repo *sqlite.AnnotationRepository
}

func NewAnnotationUsecase(repo *sqlite.AnnotationRepository) *AnnotationUsecase {
	return &AnnotationUsecase{repo: repo}
}

func (u *AnnotationUsecase) Create(ctx context.Context, imageID string, annType domain.AnnotationType, coordinates domain.Coordinates, labelID *string) (domain.Annotation, error) {
	ann := domain.Annotation{
		ID:          uuid.Must(uuid.NewV7()).String(),
		ImageID:     imageID,
		Type:        annType,
		Coordinates: coordinates,
		LabelID:     labelID,
	}
	return u.repo.Create(ctx, ann)
}

func (u *AnnotationUsecase) ListByImage(ctx context.Context, imageID string) ([]domain.Annotation, error) {
	return u.repo.ListByImage(ctx, imageID)
}

func (u *AnnotationUsecase) Update(ctx context.Context, id string, annType domain.AnnotationType, coordinates domain.Coordinates, labelID *string) (domain.Annotation, error) {
	ann := domain.Annotation{
		ID:          id,
		Type:        annType,
		Coordinates: coordinates,
		LabelID:     labelID,
	}
	return u.repo.Update(ctx, ann)
}

func (u *AnnotationUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, id)
}

// BulkReplace replaces all annotations for an image.
// Annotations without an ID get a new UUID assigned.
func (u *AnnotationUsecase) BulkReplace(ctx context.Context, imageID string, annotations []domain.Annotation) ([]domain.Annotation, error) {
	for i := range annotations {
		if annotations[i].ID == "" {
			annotations[i].ID = uuid.Must(uuid.NewV7()).String()
		}
		annotations[i].ImageID = imageID
	}
	return u.repo.BulkReplace(ctx, imageID, annotations)
}
