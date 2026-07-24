package domain

import "time"

// CommentType identifies the review purpose of a Comment.
type CommentType string

const (
	// CommentTypeQuestion represents a question that needs clarification.
	CommentTypeQuestion CommentType = "question"
	// CommentTypeIssue represents a problem that requires a change.
	CommentTypeIssue CommentType = "issue"
	// CommentTypeNote represents supplemental review information.
	CommentTypeNote CommentType = "note"
)

// Comment stores one Image-level or Annotation-level QA record.
type Comment struct {
	ID            string
	ImageID       string
	AnnotationID  *string
	Author        string
	Body          string
	Type          CommentType
	Resolved      bool
	TargetDeleted bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
