package sqlite

import (
	"database/sql"
	"time"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/daikichiba9511/goat-cv/backend/internal/sqlcgen"
)

const timeFormat = "2006-01-02T15:04:05Z"

func parseTime(s string) time.Time {
	// Why: 現在の時刻文字列はDBのDEFAULTまたはアプリ内Formatからだけ来る前提で、変換エラーを呼び出し側の分岐にしない。
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

func toGuideline(row sqlcgen.Guideline) domain.Guideline {
	return domain.Guideline{
		ID:           row.ID,
		ProjectID:    row.ProjectID,
		Title:        row.Title,
		Body:         row.Body,
		DisplayOrder: int(row.DisplayOrder),
		UpdatedAt:    parseTime(row.UpdatedAt),
	}
}

func toComment(row sqlcgen.Comment) domain.Comment {
	return toCommentWithTargetState(row, false)
}

func toCommentWithTargetState(row sqlcgen.Comment, targetDeleted bool) domain.Comment {
	return domain.Comment{
		ID:            row.ID,
		ImageID:       row.ImageID,
		AnnotationID:  fromNullString(row.AnnotationID),
		Type:          domain.CommentType(row.Type),
		Body:          row.Body,
		Author:        row.Author,
		Resolved:      row.Resolved,
		TargetDeleted: targetDeleted,
		CreatedAt:     parseTime(row.CreatedAt),
		UpdatedAt:     parseTime(row.UpdatedAt),
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
		Escalated:      row.Escalated,
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
