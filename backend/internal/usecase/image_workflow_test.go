package usecase_test

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"testing"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/daikichiba9511/goat-cv/backend/internal/usecase"
)

func TestImageWorkflowUsecaseAppliesAllowedTransitions(t *testing.T) {
	tests := []struct {
		name             string
		currentStatus    domain.ImageStatus
		currentEscalated bool
		event            domain.ImageWorkflowEvent
		wantStatus       domain.ImageStatus
		wantEscalated    bool
	}{
		{name: "complete initial annotation", currentStatus: domain.ImageStatusPending, event: domain.ImageWorkflowEventAnnotationCompleted, wantStatus: domain.ImageStatusAnnotated},
		{name: "reopen annotation", currentStatus: domain.ImageStatusAnnotated, event: domain.ImageWorkflowEventAnnotationReopened, wantStatus: domain.ImageStatusPending},
		{name: "start review", currentStatus: domain.ImageStatusAnnotated, event: domain.ImageWorkflowEventReviewStarted, wantStatus: domain.ImageStatusInReview},
		{name: "cancel review", currentStatus: domain.ImageStatusInReview, event: domain.ImageWorkflowEventReviewCancelled, wantStatus: domain.ImageStatusAnnotated},
		{name: "approve review", currentStatus: domain.ImageStatusInReview, event: domain.ImageWorkflowEventReviewApproved, wantStatus: domain.ImageStatusApproved},
		{name: "reject review", currentStatus: domain.ImageStatusInReview, event: domain.ImageWorkflowEventReviewRejected, wantStatus: domain.ImageStatusRejected},
		{name: "complete rejected revision", currentStatus: domain.ImageStatusRejected, event: domain.ImageWorkflowEventAnnotationCompleted, wantStatus: domain.ImageStatusAnnotated},
		{name: "reopen approval", currentStatus: domain.ImageStatusApproved, event: domain.ImageWorkflowEventApprovalReopened, wantStatus: domain.ImageStatusInReview},
		{name: "start escalation", currentStatus: domain.ImageStatusAnnotated, event: domain.ImageWorkflowEventEscalationStarted, wantStatus: domain.ImageStatusAnnotated, wantEscalated: true},
		{name: "resolve escalation", currentStatus: domain.ImageStatusRejected, currentEscalated: true, event: domain.ImageWorkflowEventEscalationResolved, wantStatus: domain.ImageStatusRejected},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &workflowImageRepository{image: domain.Image{
				ID:        workflowImageID,
				Status:    test.currentStatus,
				Escalated: test.currentEscalated,
			}}
			imageUsecase := usecase.NewImageUsecase(repository, "")

			updatedImage, err := imageUsecase.ApplyWorkflowEvent(context.Background(), workflowImageID, test.event)
			if err != nil {
				t.Fatalf("ApplyWorkflowEvent returned error: %v", err)
			}
			if updatedImage.Status != test.wantStatus || updatedImage.Escalated != test.wantEscalated {
				t.Fatalf(
					"updated workflow = (%q, %t), want (%q, %t)",
					updatedImage.Status,
					updatedImage.Escalated,
					test.wantStatus,
					test.wantEscalated,
				)
			}
			if repository.image.Status != test.wantStatus || repository.image.Escalated != test.wantEscalated {
				t.Fatalf("persisted workflow = %+v, want returned workflow state", repository.image)
			}
		})
	}
}

func TestImageWorkflowUsecaseRejectsDisallowedEventWithoutChangingState(t *testing.T) {
	repository := &workflowImageRepository{image: domain.Image{
		ID:     workflowImageID,
		Status: domain.ImageStatusApproved,
	}}
	imageUsecase := usecase.NewImageUsecase(repository, "")

	_, err := imageUsecase.ApplyWorkflowEvent(
		context.Background(),
		workflowImageID,
		domain.ImageWorkflowEventEscalationStarted,
	)
	var conflictError *usecase.ImageWorkflowConflictError
	if !errors.As(err, &conflictError) {
		t.Fatalf("ApplyWorkflowEvent error = %v, want ImageWorkflowConflictError", err)
	}
	if conflictError.Current.Status != domain.ImageStatusApproved || conflictError.Current.Escalated {
		t.Fatalf("conflict current state = %+v, want approved without escalation", conflictError.Current)
	}
	wantAllowedEvents := []domain.ImageWorkflowEvent{domain.ImageWorkflowEventApprovalReopened}
	if !slices.Equal(conflictError.AllowedEvents, wantAllowedEvents) {
		t.Fatalf("allowed events = %v, want %v", conflictError.AllowedEvents, wantAllowedEvents)
	}
	if repository.updateCount != 0 || repository.image.Status != domain.ImageStatusApproved || repository.image.Escalated {
		t.Fatalf("repository after rejected event = %+v, updates = %d; want unchanged", repository.image, repository.updateCount)
	}
}

func TestImageWorkflowUsecaseRejectsUnknownEventWithoutChangingState(t *testing.T) {
	repository := &workflowImageRepository{image: domain.Image{
		ID:     workflowImageID,
		Status: domain.ImageStatusPending,
	}}
	imageUsecase := usecase.NewImageUsecase(repository, "")

	_, err := imageUsecase.ApplyWorkflowEvent(
		context.Background(),
		workflowImageID,
		domain.ImageWorkflowEvent("workflow_skipped"),
	)
	if !errors.Is(err, usecase.ErrUnknownImageWorkflowEvent) {
		t.Fatalf("ApplyWorkflowEvent error = %v, want ErrUnknownImageWorkflowEvent", err)
	}
	if repository.updateCount != 0 || repository.image.Status != domain.ImageStatusPending || repository.image.Escalated {
		t.Fatalf("repository after unknown event = %+v, updates = %d; want unchanged", repository.image, repository.updateCount)
	}
}

func TestAllowedImageWorkflowEventsMatchPublishedStateMachine(t *testing.T) {
	tests := []struct {
		name       string
		image      domain.Image
		wantEvents []domain.ImageWorkflowEvent
	}{
		{
			name:  "pending",
			image: domain.Image{Status: domain.ImageStatusPending},
			wantEvents: []domain.ImageWorkflowEvent{
				domain.ImageWorkflowEventAnnotationCompleted,
				domain.ImageWorkflowEventEscalationStarted,
			},
		},
		{
			name:  "annotated",
			image: domain.Image{Status: domain.ImageStatusAnnotated},
			wantEvents: []domain.ImageWorkflowEvent{
				domain.ImageWorkflowEventAnnotationReopened,
				domain.ImageWorkflowEventReviewStarted,
				domain.ImageWorkflowEventEscalationStarted,
			},
		},
		{
			name:  "in review",
			image: domain.Image{Status: domain.ImageStatusInReview},
			wantEvents: []domain.ImageWorkflowEvent{
				domain.ImageWorkflowEventReviewCancelled,
				domain.ImageWorkflowEventReviewApproved,
				domain.ImageWorkflowEventReviewRejected,
				domain.ImageWorkflowEventEscalationStarted,
			},
		},
		{
			name:  "rejected",
			image: domain.Image{Status: domain.ImageStatusRejected},
			wantEvents: []domain.ImageWorkflowEvent{
				domain.ImageWorkflowEventAnnotationCompleted,
				domain.ImageWorkflowEventEscalationStarted,
			},
		},
		{
			name:  "approved",
			image: domain.Image{Status: domain.ImageStatusApproved},
			wantEvents: []domain.ImageWorkflowEvent{
				domain.ImageWorkflowEventApprovalReopened,
			},
		},
		{
			name:  "escalated",
			image: domain.Image{Status: domain.ImageStatusRejected, Escalated: true},
			wantEvents: []domain.ImageWorkflowEvent{
				domain.ImageWorkflowEventEscalationResolved,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			allowedEvents := usecase.AllowedImageWorkflowEvents(test.image)
			if !slices.Equal(allowedEvents, test.wantEvents) {
				t.Fatalf("allowed events = %v, want %v", allowedEvents, test.wantEvents)
			}
		})
	}
}

func TestImageTransformRequiresPendingWorkflowWithoutEscalation(t *testing.T) {
	tests := []struct {
		name         string
		image        domain.Image
		wantConflict bool
	}{
		{
			name:  "pending",
			image: domain.Image{ID: workflowImageID, OriginalWidth: 100, OriginalHeight: 200, Status: domain.ImageStatusPending},
		},
		{
			name:         "rejected",
			image:        domain.Image{ID: workflowImageID, OriginalWidth: 100, OriginalHeight: 200, Status: domain.ImageStatusRejected},
			wantConflict: true,
		},
		{
			name:         "escalated pending",
			image:        domain.Image{ID: workflowImageID, OriginalWidth: 100, OriginalHeight: 200, Status: domain.ImageStatusPending, Escalated: true},
			wantConflict: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &workflowImageRepository{image: test.image}
			imageUsecase := usecase.NewImageUsecase(repository, "")

			_, err := imageUsecase.UpdateTransform(
				context.Background(),
				workflowImageID,
				domain.Rotation90,
				false,
				false,
			)
			if test.wantConflict {
				var conflictError *usecase.ImageWorkflowOperationConflictError
				if !errors.As(err, &conflictError) {
					t.Fatalf("UpdateTransform error = %v, want ImageWorkflowOperationConflictError", err)
				}
				if conflictError.Operation != usecase.ImageWorkflowOperationTransformEdit {
					t.Fatalf("conflict operation = %q, want transform edit", conflictError.Operation)
				}
				if repository.transformUpdateCount != 0 {
					t.Fatalf("transform updates = %d, want 0", repository.transformUpdateCount)
				}
				return
			}
			if err != nil {
				t.Fatalf("UpdateTransform returned error: %v", err)
			}
			if repository.transformUpdateCount != 1 {
				t.Fatalf("transform updates = %d, want 1", repository.transformUpdateCount)
			}
		})
	}
}

const workflowImageID = "workflow-image"

type workflowImageRepository struct {
	image                domain.Image
	updateCount          int
	transformUpdateCount int
}

func (repository *workflowImageRepository) Create(_ context.Context, image domain.Image) (domain.Image, error) {
	repository.image = image
	return image, nil
}

func (repository *workflowImageRepository) Get(_ context.Context, imageID string) (domain.Image, error) {
	if repository.image.ID != imageID {
		return domain.Image{}, sql.ErrNoRows
	}
	return repository.image, nil
}

func (repository *workflowImageRepository) ListByProject(_ context.Context, _ string) ([]domain.Image, error) {
	return []domain.Image{repository.image}, nil
}

func (repository *workflowImageRepository) ListByProjectFiltered(
	_ context.Context,
	_ string,
	_ *domain.ImageStatus,
	_ *bool,
) ([]domain.Image, error) {
	return []domain.Image{repository.image}, nil
}

func (repository *workflowImageRepository) UpdateTransform(
	_ context.Context,
	_ string,
	rotation domain.Rotation,
	flipH bool,
	flipV bool,
	width int,
	height int,
) (domain.Image, error) {
	repository.transformUpdateCount++
	repository.image.Rotation = rotation
	repository.image.FlipH = flipH
	repository.image.FlipV = flipV
	repository.image.Width = width
	repository.image.Height = height
	return repository.image, nil
}

func (repository *workflowImageRepository) UpdateWorkflow(
	_ context.Context,
	_ string,
	status domain.ImageStatus,
	escalated bool,
) (domain.Image, error) {
	repository.updateCount++
	repository.image.Status = status
	repository.image.Escalated = escalated
	return repository.image, nil
}

func (repository *workflowImageRepository) Delete(_ context.Context, _ string) error {
	return nil
}
