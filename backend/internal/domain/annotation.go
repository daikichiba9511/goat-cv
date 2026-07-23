package domain

import "time"

// AnnotationType identifies the coordinate schema used by an annotation.
type AnnotationType string

const (
	// AnnotationTypeBBox represents a rectangular bounding box annotation.
	AnnotationTypeBBox AnnotationType = "bbox"
	// AnnotationTypePolygon represents a polygon annotation.
	AnnotationTypePolygon AnnotationType = "polygon"
)

// BBoxCoordinates stores normalized rectangle coordinates.
type BBoxCoordinates struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// Point stores a normalized 2D point.
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// PolygonCoordinates stores normalized polygon vertices.
type PolygonCoordinates struct {
	Points []Point `json:"points"`
}

// Coordinates stores raw annotation coordinate JSON.
// The payload is decoded as BBoxCoordinates or PolygonCoordinates according to Annotation.Type.
type Coordinates []byte

// Annotation represents a labeled region on an image.
type Annotation struct {
	ID          string
	ImageID     string
	Type        AnnotationType
	Coordinates Coordinates
	LabelID     *string
	CreatedAt   time.Time
}
