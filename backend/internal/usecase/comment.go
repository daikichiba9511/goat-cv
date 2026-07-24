package usecase

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/google/uuid"
)

// ErrInvalidComment indicates fields that cannot form a Comment.
var ErrInvalidComment = errors.New("invalid comment")

type commentRepository interface {
	Create(ctx context.Context, comment domain.Comment) (domain.Comment, error)
	ListByImage(ctx context.Context, imageID string) ([]domain.Comment, error)
	SetResolved(ctx context.Context, imageID, commentID string, resolved bool) (domain.Comment, error)
	Delete(ctx context.Context, imageID, commentID string) error
}

type commentImageRepository interface {
	Get(ctx context.Context, imageID string) (domain.Image, error)
}

type commentAnnotationRepository interface {
	Get(ctx context.Context, annotationID string) (domain.Annotation, error)
}

// CreateCommentInput contains the fields required to create an Image-scoped Comment.
type CreateCommentInput struct {
	ImageID      string
	AnnotationID *string
	Author       string
	Body         string
	Type         domain.CommentType
}

// SetCommentResolvedInput identifies a Comment and its next resolved state.
type SetCommentResolvedInput struct {
	ImageID   string
	CommentID string
	Resolved  bool
}

// CommentUsecase coordinates Image-scoped QA Comment operations.
type CommentUsecase struct {
	repository           commentRepository
	imageRepository      commentImageRepository
	annotationRepository commentAnnotationRepository
}

// NewCommentUsecase creates a CommentUsecase.
func NewCommentUsecase(
	repository commentRepository,
	imageRepository commentImageRepository,
	annotationRepository commentAnnotationRepository,
) *CommentUsecase {
	return &CommentUsecase{
		repository:           repository,
		imageRepository:      imageRepository,
		annotationRepository: annotationRepository,
	}
}

// Create adds an Image-level or Annotation-level QA Comment.
func (usecase *CommentUsecase) Create(
	ctx context.Context,
	input CreateCommentInput,
) (domain.Comment, error) {
	trimmedAuthor := strings.TrimSpace(input.Author)
	if err := validateCommentFields(trimmedAuthor, input.Body, input.Type); err != nil {
		return domain.Comment{}, err
	}
	if _, err := usecase.imageRepository.Get(ctx, input.ImageID); err != nil {
		return domain.Comment{}, err
	}
	if input.AnnotationID != nil {
		annotation, err := usecase.annotationRepository.Get(ctx, *input.AnnotationID)
		if err != nil {
			return domain.Comment{}, err
		}
		if annotation.ImageID != input.ImageID {
			// Why: 別ImageのAnnotationの存在をAPIから推測できないよう、不一致も未存在と同じ扱いにする。
			return domain.Comment{}, sql.ErrNoRows
		}
	}

	return usecase.repository.Create(ctx, domain.Comment{
		ID:           uuid.Must(uuid.NewV7()).String(),
		ImageID:      input.ImageID,
		AnnotationID: input.AnnotationID,
		Author:       trimmedAuthor,
		Body:         input.Body,
		Type:         input.Type,
	})
}

// ListByImage returns Comments after confirming the route Image exists.
func (usecase *CommentUsecase) ListByImage(
	ctx context.Context,
	imageID string,
) ([]domain.Comment, error) {
	if _, err := usecase.imageRepository.Get(ctx, imageID); err != nil {
		return nil, err
	}
	return usecase.repository.ListByImage(ctx, imageID)
}

// SetResolved changes a Comment between resolved and unresolved.
func (usecase *CommentUsecase) SetResolved(
	ctx context.Context,
	input SetCommentResolvedInput,
) (domain.Comment, error) {
	return usecase.repository.SetResolved(ctx, input.ImageID, input.CommentID, input.Resolved)
}

// Delete removes a Comment only when it belongs to the route Image.
func (usecase *CommentUsecase) Delete(
	ctx context.Context,
	imageID string,
	commentID string,
) error {
	return usecase.repository.Delete(ctx, imageID, commentID)
}

func validateCommentFields(author string, body string, commentType domain.CommentType) error {
	if author == "" {
		return fmt.Errorf("%w: author is required", ErrInvalidComment)
	}
	if strings.TrimSpace(body) == "" {
		return fmt.Errorf("%w: body is required", ErrInvalidComment)
	}
	switch commentType {
	case domain.CommentTypeQuestion, domain.CommentTypeIssue, domain.CommentTypeNote:
		return nil
	default:
		return fmt.Errorf("%w: unsupported type %q", ErrInvalidComment, commentType)
	}
}
