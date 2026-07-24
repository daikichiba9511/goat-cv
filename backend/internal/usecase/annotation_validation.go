package usecase

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
)

type bboxCoordinateInput struct {
	X      *float64 `json:"x"`
	Y      *float64 `json:"y"`
	Width  *float64 `json:"width"`
	Height *float64 `json:"height"`
}

type polygonCoordinateInput struct {
	Points *[]pointCoordinateInput `json:"points"`
}

type pointCoordinateInput struct {
	X *float64 `json:"x"`
	Y *float64 `json:"y"`
}

// validateAnnotationCoordinates verifies that raw coordinates match the annotation type and normalized coordinate rules.
func validateAnnotationCoordinates(annotationType domain.AnnotationType, rawCoordinates domain.Coordinates) error {
	switch annotationType {
	case domain.AnnotationTypeBBox:
		return validateBBoxCoordinates(rawCoordinates)
	case domain.AnnotationTypePolygon:
		return validatePolygonCoordinates(rawCoordinates)
	default:
		return fmt.Errorf("%w: %q", ErrInvalidAnnotationType, annotationType)
	}
}

// validateBBoxCoordinates verifies the required rectangle fields, finite values, and image bounds.
func validateBBoxCoordinates(rawCoordinates domain.Coordinates) error {
	var coordinates bboxCoordinateInput
	if err := decodeCoordinateObject(rawCoordinates, &coordinates); err != nil {
		return fmt.Errorf("%w: bbox schema: %v", ErrInvalidAnnotationCoordinates, err)
	}

	// Why: 0は有効な正規化座標なので、ポインタで必須項目の欠落とゼロ値を区別する。
	if coordinates.X == nil || coordinates.Y == nil || coordinates.Width == nil || coordinates.Height == nil {
		return fmt.Errorf("%w: bbox requires x, y, width, and height", ErrInvalidAnnotationCoordinates)
	}
	if !isNormalizedCoordinate(*coordinates.X) || !isNormalizedCoordinate(*coordinates.Y) ||
		!isNormalizedCoordinate(*coordinates.Width) || !isNormalizedCoordinate(*coordinates.Height) {
		return fmt.Errorf("%w: bbox values must be finite and between 0 and 1", ErrInvalidAnnotationCoordinates)
	}
	if *coordinates.Width <= 0 || *coordinates.Height <= 0 {
		return fmt.Errorf("%w: bbox width and height must be greater than zero", ErrInvalidAnnotationCoordinates)
	}
	if *coordinates.X+*coordinates.Width > 1 || *coordinates.Y+*coordinates.Height > 1 {
		return fmt.Errorf("%w: bbox must remain within normalized image bounds", ErrInvalidAnnotationCoordinates)
	}
	return nil
}

// validatePolygonCoordinates verifies required point fields, finite values, image bounds, and minimum distinct vertices.
func validatePolygonCoordinates(rawCoordinates domain.Coordinates) error {
	var coordinates polygonCoordinateInput
	if err := decodeCoordinateObject(rawCoordinates, &coordinates); err != nil {
		return fmt.Errorf("%w: polygon schema: %v", ErrInvalidAnnotationCoordinates, err)
	}
	if coordinates.Points == nil {
		return fmt.Errorf("%w: polygon requires points", ErrInvalidAnnotationCoordinates)
	}

	distinctPoints := make(map[domain.Point]struct{}, len(*coordinates.Points))
	for pointIndex, point := range *coordinates.Points {
		if point.X == nil || point.Y == nil {
			return fmt.Errorf("%w: polygon points[%d] requires x and y", ErrInvalidAnnotationCoordinates, pointIndex)
		}
		if !isNormalizedCoordinate(*point.X) || !isNormalizedCoordinate(*point.Y) {
			return fmt.Errorf("%w: polygon points[%d] must be finite and between 0 and 1", ErrInvalidAnnotationCoordinates, pointIndex)
		}
		distinctPoints[domain.Point{X: *point.X, Y: *point.Y}] = struct{}{}
	}
	if len(distinctPoints) < 3 {
		return fmt.Errorf("%w: polygon requires at least three distinct points", ErrInvalidAnnotationCoordinates)
	}

	// Why not: 自己交差の許否はAnnotationの用途に依存するため、Issue #11では座標Schemaと範囲だけを検証する。
	return nil
}

// decodeCoordinateObject decodes exactly one JSON object and rejects fields outside the selected coordinate schema.
func decodeCoordinateObject(rawCoordinates domain.Coordinates, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(rawCoordinates))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("coordinates must contain one JSON object")
		}
		return err
	}
	return nil
}

// isNormalizedCoordinate reports whether a coordinate is finite and within the closed normalized range.
func isNormalizedCoordinate(coordinate float64) bool {
	return !math.IsNaN(coordinate) && !math.IsInf(coordinate, 0) && coordinate >= 0 && coordinate <= 1
}
