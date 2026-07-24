package domain

import "time"

// ImageStatus represents the annotation workflow state of an image.
type ImageStatus string

const (
	// ImageStatusPending indicates an image that has not been annotated.
	ImageStatusPending ImageStatus = "pending"
	// ImageStatusAnnotated indicates an image marked complete by an annotator.
	ImageStatusAnnotated ImageStatus = "annotated"
	// ImageStatusInReview indicates an image currently being reviewed.
	ImageStatusInReview ImageStatus = "in_review"
	// ImageStatusApproved indicates an image accepted by a reviewer.
	ImageStatusApproved ImageStatus = "approved"
	// ImageStatusRejected indicates an image returned for correction.
	ImageStatusRejected ImageStatus = "rejected"
)

// Rotation represents a right-angle image rotation in degrees.
type Rotation int

const (
	// Rotation0 leaves the image orientation unchanged.
	Rotation0 Rotation = 0
	// Rotation90 rotates the image 90 degrees.
	Rotation90 Rotation = 90
	// Rotation180 rotates the image 180 degrees.
	Rotation180 Rotation = 180
	// Rotation270 rotates the image 270 degrees.
	Rotation270 Rotation = 270
)

// Image stores file metadata and the display transform used for annotation.
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
	Escalated      bool
	UploadedAt     time.Time
}

// EffectiveDimensions returns image dimensions after applying a right-angle rotation.
// Only 90 and 270 degree rotations swap width and height.
func EffectiveDimensions(originalWidth, originalHeight int, rotation Rotation) (width, height int) {
	// Why: 画像ファイル自体は変換せず、表示後の座標空間だけをメタデータで表す。
	// Why not: 90°/270°以外では幅と高さを入れ替えない。任意角度回転はPhase 1の対象外。
	switch rotation {
	case Rotation90, Rotation270:
		return originalHeight, originalWidth
	default:
		return originalWidth, originalHeight
	}
}
