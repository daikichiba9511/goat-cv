package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/google/uuid"
)

// ErrInvalidGuideline indicates fields that cannot form a Guideline.
var ErrInvalidGuideline = errors.New("invalid guideline")

type guidelineRepository interface {
	Create(ctx context.Context, guideline domain.Guideline) (domain.Guideline, error)
	ListByProject(ctx context.Context, projectID string) ([]domain.Guideline, error)
	Get(ctx context.Context, projectID, guidelineID string) (domain.Guideline, error)
	Update(ctx context.Context, projectID string, guideline domain.Guideline) (domain.Guideline, error)
	Delete(ctx context.Context, projectID, guidelineID string) error
}

// GuidelineUsecase coordinates Project-scoped Guideline operations.
type GuidelineUsecase struct {
	repository guidelineRepository
}

// NewGuidelineUsecase creates a GuidelineUsecase.
func NewGuidelineUsecase(repository guidelineRepository) *GuidelineUsecase {
	return &GuidelineUsecase{repository: repository}
}

// Create adds a Guideline to a Project.
func (usecase *GuidelineUsecase) Create(
	ctx context.Context,
	projectID string,
	title string,
	body string,
	displayOrder int,
) (domain.Guideline, error) {
	if err := validateGuidelineFields(title, displayOrder); err != nil {
		return domain.Guideline{}, err
	}
	guideline := domain.Guideline{
		ID:           uuid.Must(uuid.NewV7()).String(),
		ProjectID:    projectID,
		Title:        strings.TrimSpace(title),
		Body:         body,
		DisplayOrder: displayOrder,
	}
	return usecase.repository.Create(ctx, guideline)
}

// ListByProject returns Guidelines in their persisted display order.
func (usecase *GuidelineUsecase) ListByProject(
	ctx context.Context,
	projectID string,
) ([]domain.Guideline, error) {
	return usecase.repository.ListByProject(ctx, projectID)
}

// Get returns a Guideline only when it belongs to the route Project.
func (usecase *GuidelineUsecase) Get(
	ctx context.Context,
	projectID string,
	guidelineID string,
) (domain.Guideline, error) {
	return usecase.repository.Get(ctx, projectID, guidelineID)
}

// Update changes a Guideline only when it belongs to the route Project.
func (usecase *GuidelineUsecase) Update(
	ctx context.Context,
	projectID string,
	guidelineID string,
	title string,
	body string,
	displayOrder int,
) (domain.Guideline, error) {
	if err := validateGuidelineFields(title, displayOrder); err != nil {
		return domain.Guideline{}, err
	}
	return usecase.repository.Update(ctx, projectID, domain.Guideline{
		ID:           guidelineID,
		ProjectID:    projectID,
		Title:        strings.TrimSpace(title),
		Body:         body,
		DisplayOrder: displayOrder,
	})
}

// Delete removes a Guideline only when it belongs to the route Project.
func (usecase *GuidelineUsecase) Delete(
	ctx context.Context,
	projectID string,
	guidelineID string,
) error {
	return usecase.repository.Delete(ctx, projectID, guidelineID)
}

func validateGuidelineFields(title string, displayOrder int) error {
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("%w: title is required", ErrInvalidGuideline)
	}
	if displayOrder < 0 {
		return fmt.Errorf("%w: display_order must be zero or greater", ErrInvalidGuideline)
	}
	return nil
}
