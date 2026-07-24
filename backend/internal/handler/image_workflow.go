package handler

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/daikichiba9511/goat-cv/backend/internal/usecase"
	"github.com/go-chi/chi/v5"
)

type imageWorkflowStateResponse struct {
	Status    string `json:"status"`
	Escalated bool   `json:"escalated"`
}

type imageWorkflowConflictResponse struct {
	Error         string                      `json:"error"`
	Current       imageWorkflowStateResponse  `json:"current"`
	AllowedEvents []domain.ImageWorkflowEvent `json:"allowed_events"`
}

func (handler *ImageHandler) applyWorkflowEvent(responseWriter http.ResponseWriter, request *http.Request) {
	var requestBody struct {
		Event string `json:"event"`
	}
	if err := decodeJSON(request, &requestBody); err != nil {
		writeError(responseWriter, http.StatusBadRequest, "invalid request body")
		return
	}

	imageID := chi.URLParam(request, "imageId")
	image, err := handler.uc.ApplyWorkflowEvent(
		request.Context(),
		imageID,
		domain.ImageWorkflowEvent(requestBody.Event),
	)
	if err != nil {
		writeWorkflowTransitionError(responseWriter, err)
		return
	}
	writeJSON(responseWriter, http.StatusOK, toImageResponse(image))
}

func writeWorkflowTransitionError(responseWriter http.ResponseWriter, err error) {
	var conflictError *usecase.ImageWorkflowConflictError
	switch {
	case errors.Is(err, usecase.ErrUnknownImageWorkflowEvent):
		writeError(responseWriter, http.StatusBadRequest, err.Error())
	case errors.Is(err, sql.ErrNoRows):
		writeError(responseWriter, http.StatusNotFound, "image not found")
	case errors.As(err, &conflictError):
		writeJSON(responseWriter, http.StatusConflict, imageWorkflowConflictResponse{
			Error: "workflow transition not allowed",
			Current: imageWorkflowStateResponse{
				Status:    string(conflictError.Current.Status),
				Escalated: conflictError.Current.Escalated,
			},
			AllowedEvents: conflictError.AllowedEvents,
		})
	default:
		writeError(responseWriter, http.StatusInternalServerError, err.Error())
	}
}

func writeWorkflowOperationError(responseWriter http.ResponseWriter, err error) bool {
	var conflictError *usecase.ImageWorkflowOperationConflictError
	if !errors.As(err, &conflictError) {
		return false
	}
	writeJSON(responseWriter, http.StatusConflict, imageWorkflowConflictResponse{
		Error: "workflow operation not allowed",
		Current: imageWorkflowStateResponse{
			Status:    string(conflictError.Current.Status),
			Escalated: conflictError.Current.Escalated,
		},
		AllowedEvents: conflictError.AllowedEvents,
	})
	return true
}
