package sqlite

import (
	"context"
	"database/sql"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/daikichiba9511/goat-cv/backend/internal/sqlcgen"
)

// CommentRepository persists Image-scoped QA Comments in SQLite.
type CommentRepository struct {
	queries *sqlcgen.Queries
}

// NewCommentRepository creates a CommentRepository.
func NewCommentRepository(queries *sqlcgen.Queries) *CommentRepository {
	return &CommentRepository{queries: queries}
}

// Create inserts a Comment.
func (repository *CommentRepository) Create(
	ctx context.Context,
	comment domain.Comment,
) (domain.Comment, error) {
	row, err := repository.queries.CreateComment(ctx, sqlcgen.CreateCommentParams{
		ID:           comment.ID,
		ImageID:      comment.ImageID,
		AnnotationID: toNullString(comment.AnnotationID),
		Author:       comment.Author,
		Body:         comment.Body,
		Type:         string(comment.Type),
	})
	if err != nil {
		return domain.Comment{}, err
	}
	return toComment(row), nil
}

// ListByImage returns Comments in stable creation order.
func (repository *CommentRepository) ListByImage(
	ctx context.Context,
	imageID string,
) ([]domain.Comment, error) {
	rows, err := repository.queries.ListCommentsByImage(ctx, imageID)
	if err != nil {
		return nil, err
	}
	comments := make([]domain.Comment, len(rows))
	for commentIndex, row := range rows {
		comments[commentIndex] = toCommentWithTargetState(row.Comment, row.TargetDeleted != 0)
	}
	return comments, nil
}

// SetResolved changes a Comment's resolved state only within imageID.
func (repository *CommentRepository) SetResolved(
	ctx context.Context,
	imageID string,
	commentID string,
	resolved bool,
) (domain.Comment, error) {
	updatedRows, err := repository.queries.SetCommentResolved(ctx, sqlcgen.SetCommentResolvedParams{
		Resolved: resolved,
		ID:       commentID,
		ImageID:  imageID,
	})
	if err != nil {
		return domain.Comment{}, err
	}
	if updatedRows == 0 {
		return domain.Comment{}, sql.ErrNoRows
	}
	return repository.get(ctx, imageID, commentID)
}

// Delete removes a Comment only within imageID.
func (repository *CommentRepository) Delete(
	ctx context.Context,
	imageID string,
	commentID string,
) error {
	deletedRows, err := repository.queries.DeleteComment(ctx, sqlcgen.DeleteCommentParams{
		ID:      commentID,
		ImageID: imageID,
	})
	if err != nil {
		return err
	}
	if deletedRows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (repository *CommentRepository) get(
	ctx context.Context,
	imageID string,
	commentID string,
) (domain.Comment, error) {
	row, err := repository.queries.GetComment(ctx, sqlcgen.GetCommentParams{
		ID:      commentID,
		ImageID: imageID,
	})
	if err != nil {
		return domain.Comment{}, err
	}
	return toCommentWithTargetState(row.Comment, row.TargetDeleted != 0), nil
}
