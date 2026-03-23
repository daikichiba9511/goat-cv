package domain

import "time"

type ImageStatus string

const (
	ImageStatusPending   ImageStatus = "pending"
	ImageStatusAnnotated ImageStatus = "annotated"
	ImageStatusInReview  ImageStatus = "in_review"
	ImageStatusApproved  ImageStatus = "approved"
	ImageStatusRejected  ImageStatus = "rejected"
	ImageStatusEscalated ImageStatus = "escalated"
)

type Rotation int

const (
	Rotation0   Rotation = 0
	Rotation90  Rotation = 90
	Rotation180 Rotation = 180
	Rotation270 Rotation = 270
)

type Image struct {
	ID             string
	ProjectID      string
	Filename       string
	OriginalWidth  int
	OriginalHeight int
	Width          int
	Height         int
	Rotation       Rotation
	FlipH          bool
	FlipV          bool
	Status         ImageStatus
	UploadedAt     time.Time
}

// EffectiveDimensions returns width and height after applying rotation.
// 90° and 270° rotations swap width and height.
func EffectiveDimensions(originalWidth, originalHeight int, rotation Rotation) (width, height int) {
	switch rotation {
	case Rotation90, Rotation270:
		return originalHeight, originalWidth
	default:
		return originalWidth, originalHeight
	}
}
