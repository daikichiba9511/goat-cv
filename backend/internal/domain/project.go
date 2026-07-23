package domain

import "time"

// Project groups images, labels, annotations, and export settings.
type Project struct {
	ID        string
	Name      string
	CreatedAt time.Time
}
