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

type guidelineResponse struct {
	ID           string `json:"id"`
	ProjectID    string `json:"project_id"`
	Title        string `json:"title"`
	Body         string `json:"body"`
	DisplayOrder int    `json:"display_order"`
	UpdatedAt    string `json:"updated_at"`
}

type guidelineRequest struct {
	Title        string  `json:"title"`
	Body         *string `json:"body"`
	DisplayOrder *int    `json:"display_order"`
}

// GuidelineHandler serves Project-scoped Guideline API routes.
type GuidelineHandler struct {
	usecase *usecase.GuidelineUsecase
}

// NewGuidelineHandler creates a GuidelineHandler.
func NewGuidelineHandler(guidelineUsecase *usecase.GuidelineUsecase) *GuidelineHandler {
	return &GuidelineHandler{usecase: guidelineUsecase}
}

// Routes registers Guideline routes on a Project-scoped router.
func (handler *GuidelineHandler) Routes(router chi.Router) {
	router.Post("/", handler.create)
	router.Get("/", handler.list)
	router.Get("/{guidelineId}", handler.get)
	router.Patch("/{guidelineId}", handler.update)
	router.Delete("/{guidelineId}", handler.delete)
}

func (handler *GuidelineHandler) create(writer http.ResponseWriter, request *http.Request) {
	input, ok := decodeGuidelineRequest(writer, request)
	if !ok {
		return
	}
	guideline, err := handler.usecase.Create(
		request.Context(),
		chi.URLParam(request, "projectId"),
		input.Title,
		*input.Body,
		*input.DisplayOrder,
	)
	if err != nil {
		writeGuidelineError(writer, err)
		return
	}
	writeJSON(writer, http.StatusCreated, toGuidelineResponse(guideline))
}

func (handler *GuidelineHandler) list(writer http.ResponseWriter, request *http.Request) {
	guidelines, err := handler.usecase.ListByProject(
		request.Context(),
		chi.URLParam(request, "projectId"),
	)
	if err != nil {
		writeGuidelineError(writer, err)
		return
	}
	items := make([]guidelineResponse, len(guidelines))
	for guidelineIndex, guideline := range guidelines {
		items[guidelineIndex] = toGuidelineResponse(guideline)
	}
	writeJSON(writer, http.StatusOK, listResponse{Items: items})
}

func (handler *GuidelineHandler) get(writer http.ResponseWriter, request *http.Request) {
	guideline, err := handler.usecase.Get(
		request.Context(),
		chi.URLParam(request, "projectId"),
		chi.URLParam(request, "guidelineId"),
	)
	if err != nil {
		writeGuidelineError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, toGuidelineResponse(guideline))
}

func (handler *GuidelineHandler) update(writer http.ResponseWriter, request *http.Request) {
	input, ok := decodeGuidelineRequest(writer, request)
	if !ok {
		return
	}
	guideline, err := handler.usecase.Update(
		request.Context(),
		chi.URLParam(request, "projectId"),
		chi.URLParam(request, "guidelineId"),
		input.Title,
		*input.Body,
		*input.DisplayOrder,
	)
	if err != nil {
		writeGuidelineError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, toGuidelineResponse(guideline))
}

func (handler *GuidelineHandler) delete(writer http.ResponseWriter, request *http.Request) {
	err := handler.usecase.Delete(
		request.Context(),
		chi.URLParam(request, "projectId"),
		chi.URLParam(request, "guidelineId"),
	)
	if err != nil {
		writeGuidelineError(writer, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func decodeGuidelineRequest(
	writer http.ResponseWriter,
	request *http.Request,
) (guidelineRequest, bool) {
	var input guidelineRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid request body")
		return guidelineRequest{}, false
	}
	if input.Body == nil || input.DisplayOrder == nil {
		writeError(writer, http.StatusBadRequest, "title, body, and display_order are required")
		return guidelineRequest{}, false
	}
	return input, true
}

func writeGuidelineError(writer http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrInvalidGuideline):
		writeError(writer, http.StatusBadRequest, err.Error())
	case errors.Is(err, sql.ErrNoRows):
		writeError(writer, http.StatusNotFound, "guideline not found")
	default:
		writeError(writer, http.StatusInternalServerError, err.Error())
	}
}

func toGuidelineResponse(guideline domain.Guideline) guidelineResponse {
	return guidelineResponse{
		ID:           guideline.ID,
		ProjectID:    guideline.ProjectID,
		Title:        guideline.Title,
		Body:         guideline.Body,
		DisplayOrder: guideline.DisplayOrder,
		UpdatedAt:    guideline.UpdatedAt.Format(time.RFC3339),
	}
}
