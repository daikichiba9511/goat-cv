package domain

// ImageWorkflowEvent represents a command that may change an Image workflow state.
type ImageWorkflowEvent string

const (
	// ImageWorkflowEventAnnotationCompleted marks initial annotation or a rejected revision complete.
	ImageWorkflowEventAnnotationCompleted ImageWorkflowEvent = "annotation_completed"
	// ImageWorkflowEventAnnotationReopened returns a completed annotation to editing.
	ImageWorkflowEventAnnotationReopened ImageWorkflowEvent = "annotation_reopened"
	// ImageWorkflowEventReviewStarted starts review of a completed annotation.
	ImageWorkflowEventReviewStarted ImageWorkflowEvent = "review_started"
	// ImageWorkflowEventReviewCancelled returns an unfinished review to annotated.
	ImageWorkflowEventReviewCancelled ImageWorkflowEvent = "review_cancelled"
	// ImageWorkflowEventReviewApproved accepts an Image under review.
	ImageWorkflowEventReviewApproved ImageWorkflowEvent = "review_approved"
	// ImageWorkflowEventReviewRejected returns an Image under review for correction.
	ImageWorkflowEventReviewRejected ImageWorkflowEvent = "review_rejected"
	// ImageWorkflowEventApprovalReopened returns an approved Image to review.
	ImageWorkflowEventApprovalReopened ImageWorkflowEvent = "approval_reopened"
	// ImageWorkflowEventEscalationStarted pauses work while external judgment is pending.
	ImageWorkflowEventEscalationStarted ImageWorkflowEvent = "escalation_started"
	// ImageWorkflowEventEscalationResolved resumes work without changing its lifecycle status.
	ImageWorkflowEventEscalationResolved ImageWorkflowEvent = "escalation_resolved"
)
