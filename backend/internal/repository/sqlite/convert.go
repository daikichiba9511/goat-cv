package sqlite

import (
	"database/sql"
	"time"

	"github.com/chibadaimare/goat/backend/internal/domain"
	"github.com/chibadaimare/goat/backend/internal/sqlcgen"
)

const timeFormat = "2006-01-02T15:04:05Z"

func parseTime(s string) time.Time {
	t, _ := time.Parse(timeFormat, s)
	return t
}

func toNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func fromNullString(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

func toProject(row sqlcgen.Project) domain.Project {
	return domain.Project{
		ID:        row.ID,
		Name:      row.Name,
		CreatedAt: parseTime(row.CreatedAt),
	}
}

func toLabelDefinition(row sqlcgen.LabelDefinition) domain.LabelDefinition {
	return domain.LabelDefinition{
		ID:        row.ID,
		ProjectID: row.ProjectID,
		Name:      row.Name,
		Color:     row.Color,
		Category:  domain.LabelCategory(row.Category),
	}
}

func toImage(row sqlcgen.Image) domain.Image {
	return domain.Image{
		ID:             row.ID,
		ProjectID:      row.ProjectID,
		Filename:       row.Filename,
		OriginalWidth:  int(row.OriginalWidth),
		OriginalHeight: int(row.OriginalHeight),
		Width:          int(row.Width),
		Height:         int(row.Height),
		Rotation:       domain.Rotation(row.Rotation),
		FlipH:          row.FlipH,
		FlipV:          row.FlipV,
		Status:         domain.ImageStatus(row.Status),
		UploadedAt:     parseTime(row.UploadedAt),
	}
}

func toAnnotation(row sqlcgen.Annotation) domain.Annotation {
	return domain.Annotation{
		ID:          row.ID,
		ImageID:     row.ImageID,
		Type:        domain.AnnotationType(row.Type),
		Coordinates: domain.Coordinates(row.Coordinates),
		LabelID:     fromNullString(row.LabelID),
		CreatedAt:   parseTime(row.CreatedAt),
	}
}

func toEdge(row sqlcgen.Edge) domain.Edge {
	return domain.Edge{
		ID:                 row.ID,
		ImageID:            row.ImageID,
		SourceAnnotationID: row.SourceAnnotationID,
		TargetAnnotationID: row.TargetAnnotationID,
		Type:               domain.EdgeType(row.Type),
	}
}
