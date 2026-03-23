package domain

import "time"

type AnnotationType string

const (
	AnnotationTypeBBox    AnnotationType = "bbox"
	AnnotationTypePolygon AnnotationType = "polygon"
)

type BBoxCoordinates struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type PolygonCoordinates struct {
	Points []Point `json:"points"`
}

// Coordinates holds the raw JSON of annotation coordinates.
// Deserialized to BBoxCoordinates or PolygonCoordinates based on AnnotationType.
type Coordinates []byte

type Annotation struct {
	ID          string
	ImageID     string
	Type        AnnotationType
	Coordinates Coordinates
	LabelID     *string
	CreatedAt   time.Time
}
