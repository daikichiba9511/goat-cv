package domain

import "time"

// Guideline stores one ordered Markdown page for a project's annotation rules.
type Guideline struct {
	ID           string
	ProjectID    string
	Title        string
	Body         string
	DisplayOrder int
	UpdatedAt    time.Time
}
