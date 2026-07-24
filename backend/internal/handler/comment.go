package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/daikichiba9511/goat-cv/backend/internal/usecase"
	"github.com/go-chi/chi/v5"
)

type commentResponse struct {
	ID            string  `json:"id"`
	ImageID       string  `json:"image_id"`
	AnnotationID  *string `json:"annotation_id"`
	Author        string  `json:"author"`
	Body          string  `json:"body"`
	Type          string  `json:"type"`
	Resolved      bool    `json:"resolved"`
	TargetDeleted bool    `json:"target_deleted"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type createCommentRequest struct {
	AnnotationID *string `json:"annotation_id"`
	Author       string  `json:"author"`
	Body         string  `json:"body"`
	Type         string  `json:"type"`
}

type setCommentResolvedRequest struct {
	Resolved *bool `json:"resolved"`
}

// CommentHandler serves Image-scoped QA Comment API routes.
type CommentHandler struct {
	usecase *usecase.CommentUsecase
}

// NewCommentHandler creates a CommentHandler.
func NewCommentHandler(commentUsecase *usecase.CommentUsecase) *CommentHandler {
	return &CommentHandler{usecase: commentUsecase}
}

// ImageRoutes registers Comment routes on an Image-scoped router.
func (handler *CommentHandler) ImageRoutes(router chi.Router) {
	router.Post("/", handler.create)
	router.Get("/", handler.list)
	router.Patch("/{commentId}", handler.setResolved)
	router.Delete("/{commentId}", handler.delete)
}

func (handler *CommentHandler) create(writer http.ResponseWriter, request *http.Request) {
	var input createCommentRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid request body")
		return
	}

	comment, err := handler.usecase.Create(request.Context(), usecase.CreateCommentInput{
		ImageID:      chi.URLParam(request, "imageId"),
		AnnotationID: input.AnnotationID,
		Author:       input.Author,
		Body:         input.Body,
		Type:         domain.CommentType(input.Type),
	})
	if err != nil {
		writeCommentError(writer, err)
		return
	}
	writeJSON(writer, http.StatusCreated, toCommentResponse(comment))
}

func (handler *CommentHandler) list(writer http.ResponseWriter, request *http.Request) {
	comments, err := handler.usecase.ListByImage(
		request.Context(),
		chi.URLParam(request, "imageId"),
	)
	if err != nil {
		writeCommentError(writer, err)
		return
	}
	items := make([]commentResponse, len(comments))
	for commentIndex, comment := range comments {
		items[commentIndex] = toCommentResponse(comment)
	}
	writeJSON(writer, http.StatusOK, listResponse{Items: items})
}

func (handler *CommentHandler) setResolved(writer http.ResponseWriter, request *http.Request) {
	var input setCommentResolvedRequest
	if err := decodeJSON(request, &input); err != nil || input.Resolved == nil {
		writeError(writer, http.StatusBadRequest, "resolved is required")
		return
	}

	comment, err := handler.usecase.SetResolved(request.Context(), usecase.SetCommentResolvedInput{
		ImageID:   chi.URLParam(request, "imageId"),
		CommentID: chi.URLParam(request, "commentId"),
		Resolved:  *input.Resolved,
	})
	if err != nil {
		writeCommentError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, toCommentResponse(comment))
}

func (handler *CommentHandler) delete(writer http.ResponseWriter, request *http.Request) {
	err := handler.usecase.Delete(
		request.Context(),
		chi.URLParam(request, "imageId"),
		chi.URLParam(request, "commentId"),
	)
	if err != nil {
		writeCommentError(writer, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func writeCommentError(writer http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrInvalidComment):
		writeError(writer, http.StatusBadRequest, err.Error())
	case errors.Is(err, sql.ErrNoRows):
		writeError(writer, http.StatusNotFound, "comment target not found")
	default:
		writeError(writer, http.StatusInternalServerError, err.Error())
	}
}

func toCommentResponse(comment domain.Comment) commentResponse {
	return commentResponse{
		ID:            comment.ID,
		ImageID:       comment.ImageID,
		AnnotationID:  comment.AnnotationID,
		Author:        comment.Author,
		Body:          comment.Body,
		Type:          string(comment.Type),
		Resolved:      comment.Resolved,
		TargetDeleted: comment.TargetDeleted,
		CreatedAt:     comment.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     comment.UpdatedAt.Format(time.RFC3339),
	}
}
