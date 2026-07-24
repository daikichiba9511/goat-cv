package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
)

type imageWorkflowReader interface {
	Get(ctx context.Context, imageID string) (domain.Image, error)
}

// ImageWorkflowConflictError reports a known event that is not allowed from the current Image state.
type ImageWorkflowConflictError struct {
	Event         domain.ImageWorkflowEvent
	Current       domain.Image
	AllowedEvents []domain.ImageWorkflowEvent
}

// ImageWorkflowOperation identifies a state-sensitive Image mutation.
type ImageWorkflowOperation string

const (
	// ImageWorkflowOperationGraphEdit changes an Image Annotation Graph.
	ImageWorkflowOperationGraphEdit ImageWorkflowOperation = "graph_edit"
	// ImageWorkflowOperationTransformEdit changes Image transform metadata.
	ImageWorkflowOperationTransformEdit ImageWorkflowOperation = "transform_edit"
)

// ImageWorkflowOperationConflictError reports an operation blocked by the current Image state.
type ImageWorkflowOperationConflictError struct {
	Operation     ImageWorkflowOperation
	Current       domain.Image
	AllowedEvents []domain.ImageWorkflowEvent
}

// ErrUnknownImageWorkflowEvent indicates an event outside the published workflow contract.
var ErrUnknownImageWorkflowEvent = errors.New("unknown image workflow event")

// Error describes the attempted event and current workflow state.
func (workflowError *ImageWorkflowConflictError) Error() string {
	return fmt.Sprintf(
		"image workflow event %q is not allowed from status %q with escalated=%t",
		workflowError.Event,
		workflowError.Current.Status,
		workflowError.Current.Escalated,
	)
}

// Error describes the blocked operation and current workflow state.
func (workflowError *ImageWorkflowOperationConflictError) Error() string {
	return fmt.Sprintf(
		"image workflow operation %q is not allowed from status %q with escalated=%t",
		workflowError.Operation,
		workflowError.Current.Status,
		workflowError.Current.Escalated,
	)
}

// Why: conflict responseとUIのaction順を状態ごとに安定させるため、eventの列挙順を一か所に固定する。
var imageWorkflowEvents = [...]domain.ImageWorkflowEvent{
	domain.ImageWorkflowEventAnnotationCompleted,
	domain.ImageWorkflowEventAnnotationReopened,
	domain.ImageWorkflowEventReviewStarted,
	domain.ImageWorkflowEventReviewCancelled,
	domain.ImageWorkflowEventReviewApproved,
	domain.ImageWorkflowEventReviewRejected,
	domain.ImageWorkflowEventApprovalReopened,
	domain.ImageWorkflowEventEscalationStarted,
	domain.ImageWorkflowEventEscalationResolved,
}

// ApplyWorkflowEvent applies an allowed event to an Image and persists both workflow dimensions together.
func (usecase *ImageUsecase) ApplyWorkflowEvent(
	ctx context.Context,
	imageID string,
	event domain.ImageWorkflowEvent,
) (domain.Image, error) {
	if !isKnownImageWorkflowEvent(event) {
		return domain.Image{}, fmt.Errorf("%w: %q", ErrUnknownImageWorkflowEvent, event)
	}

	currentImage, err := usecase.repo.Get(ctx, imageID)
	if err != nil {
		return domain.Image{}, err
	}

	updatedImage, allowed := applyImageWorkflowEvent(currentImage, event)
	if !allowed {
		return domain.Image{}, &ImageWorkflowConflictError{
			Event:         event,
			Current:       currentImage,
			AllowedEvents: AllowedImageWorkflowEvents(currentImage),
		}
	}
	return usecase.repo.UpdateWorkflow(ctx, imageID, updatedImage.Status, updatedImage.Escalated)
}

// AllowedImageWorkflowEvents returns events accepted from the current lifecycle and escalation state.
// The order is stable so API clients can present actions consistently.
func AllowedImageWorkflowEvents(currentImage domain.Image) []domain.ImageWorkflowEvent {
	allowedEvents := make([]domain.ImageWorkflowEvent, 0, len(imageWorkflowEvents))
	for _, event := range imageWorkflowEvents {
		if _, allowed := applyImageWorkflowEvent(currentImage, event); allowed {
			allowedEvents = append(allowedEvents, event)
		}
	}
	return allowedEvents
}

func isKnownImageWorkflowEvent(event domain.ImageWorkflowEvent) bool {
	for _, knownEvent := range imageWorkflowEvents {
		if event == knownEvent {
			return true
		}
	}
	return false
}

func isKnownImageStatus(status domain.ImageStatus) bool {
	switch status {
	case domain.ImageStatusPending,
		domain.ImageStatusAnnotated,
		domain.ImageStatusInReview,
		domain.ImageStatusRejected,
		domain.ImageStatusApproved:
		return true
	default:
		return false
	}
}

func requireImageWorkflowOperation(currentImage domain.Image, operation ImageWorkflowOperation) error {
	allowed := false
	if !currentImage.Escalated {
		// Why: rejectedでは指摘修正のGraph編集だけを許可し、座標全体を無効にし得るtransformは再開しない。
		switch operation {
		case ImageWorkflowOperationGraphEdit:
			allowed = currentImage.Status == domain.ImageStatusPending || currentImage.Status == domain.ImageStatusRejected
		case ImageWorkflowOperationTransformEdit:
			allowed = currentImage.Status == domain.ImageStatusPending
		}
	}
	if allowed {
		return nil
	}
	return &ImageWorkflowOperationConflictError{
		Operation:     operation,
		Current:       currentImage,
		AllowedEvents: AllowedImageWorkflowEvents(currentImage),
	}
}

func requireImageWorkflowOperationForImage(
	ctx context.Context,
	images imageWorkflowReader,
	imageID string,
	operation ImageWorkflowOperation,
) error {
	currentImage, err := images.Get(ctx, imageID)
	if err != nil {
		return err
	}
	return requireImageWorkflowOperation(currentImage, operation)
}

func applyImageWorkflowEvent(
	currentImage domain.Image,
	event domain.ImageWorkflowEvent,
) (domain.Image, bool) {
	// Why: escalationはlifecycleと直交するが、判断待ち中に消費できるeventは解除だけに制限する。
	if currentImage.Escalated {
		if event != domain.ImageWorkflowEventEscalationResolved {
			return domain.Image{}, false
		}
		currentImage.Escalated = false
		return currentImage, true
	}

	switch event {
	case domain.ImageWorkflowEventAnnotationCompleted:
		if currentImage.Status != domain.ImageStatusPending && currentImage.Status != domain.ImageStatusRejected {
			return domain.Image{}, false
		}
		currentImage.Status = domain.ImageStatusAnnotated
	case domain.ImageWorkflowEventAnnotationReopened:
		if currentImage.Status != domain.ImageStatusAnnotated {
			return domain.Image{}, false
		}
		currentImage.Status = domain.ImageStatusPending
	case domain.ImageWorkflowEventReviewStarted:
		if currentImage.Status != domain.ImageStatusAnnotated {
			return domain.Image{}, false
		}
		currentImage.Status = domain.ImageStatusInReview
	case domain.ImageWorkflowEventReviewCancelled:
		if currentImage.Status != domain.ImageStatusInReview {
			return domain.Image{}, false
		}
		currentImage.Status = domain.ImageStatusAnnotated
	case domain.ImageWorkflowEventReviewApproved:
		if currentImage.Status != domain.ImageStatusInReview {
			return domain.Image{}, false
		}
		currentImage.Status = domain.ImageStatusApproved
	case domain.ImageWorkflowEventReviewRejected:
		if currentImage.Status != domain.ImageStatusInReview {
			return domain.Image{}, false
		}
		currentImage.Status = domain.ImageStatusRejected
	case domain.ImageWorkflowEventApprovalReopened:
		if currentImage.Status != domain.ImageStatusApproved {
			return domain.Image{}, false
		}
		currentImage.Status = domain.ImageStatusInReview
	case domain.ImageWorkflowEventEscalationStarted:
		if currentImage.Status == domain.ImageStatusApproved {
			return domain.Image{}, false
		}
		currentImage.Escalated = true
	case domain.ImageWorkflowEventEscalationResolved:
		return domain.Image{}, false
	default:
		return domain.Image{}, false
	}

	return currentImage, true
}
