package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/daikichiba9511/goat-cv/backend/internal/usecase"
	"github.com/go-chi/chi/v5"
)

type imageGraphAnnotationRequest struct {
	ClientID    string          `json:"client_id"`
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	Coordinates json.RawMessage `json:"coordinates"`
	LabelID     *string         `json:"label_id"`
}

type imageGraphEdgeRequest struct {
	ClientID                 string `json:"client_id"`
	ID                       string `json:"id"`
	SourceAnnotationClientID string `json:"source_annotation_client_id"`
	TargetAnnotationClientID string `json:"target_annotation_client_id"`
	Type                     string `json:"type"`
}

type imageGraphRequest struct {
	Annotations *[]imageGraphAnnotationRequest `json:"annotations"`
	Edges       *[]imageGraphEdgeRequest       `json:"edges"`
}

type savedImageGraphAnnotationResponse struct {
	ClientID   string             `json:"client_id"`
	Annotation annotationResponse `json:"annotation"`
}

type savedImageGraphEdgeResponse struct {
	ClientID string       `json:"client_id"`
	Edge     edgeResponse `json:"edge"`
}

type savedImageGraphResponse struct {
	Annotations []savedImageGraphAnnotationResponse `json:"annotations"`
	Edges       []savedImageGraphEdgeResponse       `json:"edges"`
}

// ImageGraphHandler serves the atomic image graph save route.
type ImageGraphHandler struct {
	usecase *usecase.ImageGraphUsecase
}

// NewImageGraphHandler creates an ImageGraphHandler.
func NewImageGraphHandler(imageGraphUsecase *usecase.ImageGraphUsecase) *ImageGraphHandler {
	return &ImageGraphHandler{usecase: imageGraphUsecase}
}

// ImageRoutes registers image-scoped graph routes.
func (h *ImageGraphHandler) ImageRoutes(router chi.Router) {
	router.Put("/", h.save)
}

// save replaces an image graph after validating the complete request.
func (h *ImageGraphHandler) save(responseWriter http.ResponseWriter, request *http.Request) {
	imageID := chi.URLParam(request, "imageId")
	var requestBody imageGraphRequest
	if err := decodeJSON(request, &requestBody); err != nil {
		writeError(responseWriter, http.StatusBadRequest, "invalid request body")
		return
	}
	if requestBody.Annotations == nil || requestBody.Edges == nil {
		writeError(responseWriter, http.StatusBadRequest, "annotations and edges are required")
		return
	}

	graphInput := usecase.ImageGraphInput{
		Annotations: make([]usecase.ImageGraphAnnotationInput, len(*requestBody.Annotations)),
		Edges:       make([]usecase.ImageGraphEdgeInput, len(*requestBody.Edges)),
	}
	for annotationIndex, annotation := range *requestBody.Annotations {
		graphInput.Annotations[annotationIndex] = usecase.ImageGraphAnnotationInput{
			ClientID:    annotation.ClientID,
			ID:          annotation.ID,
			Type:        domain.AnnotationType(annotation.Type),
			Coordinates: domain.Coordinates(annotation.Coordinates),
			LabelID:     annotation.LabelID,
		}
	}
	for edgeIndex, edge := range *requestBody.Edges {
		graphInput.Edges[edgeIndex] = usecase.ImageGraphEdgeInput{
			ClientID:                 edge.ClientID,
			ID:                       edge.ID,
			SourceAnnotationClientID: edge.SourceAnnotationClientID,
			TargetAnnotationClientID: edge.TargetAnnotationClientID,
			Type:                     domain.EdgeType(edge.Type),
		}
	}

	savedGraph, err := h.usecase.Save(request.Context(), imageID, graphInput)
	if err != nil {
		writeImageGraphError(responseWriter, err)
		return
	}
	writeJSON(responseWriter, http.StatusOK, toSavedImageGraphResponse(savedGraph))
}

// toSavedImageGraphResponse converts persisted graph resources while retaining their client IDs.
func toSavedImageGraphResponse(savedGraph usecase.SavedImageGraph) savedImageGraphResponse {
	response := savedImageGraphResponse{
		Annotations: make([]savedImageGraphAnnotationResponse, len(savedGraph.Annotations)),
		Edges:       make([]savedImageGraphEdgeResponse, len(savedGraph.Edges)),
	}
	for annotationIndex, savedAnnotation := range savedGraph.Annotations {
		response.Annotations[annotationIndex] = savedImageGraphAnnotationResponse{
			ClientID:   savedAnnotation.ClientID,
			Annotation: toAnnotationResponse(savedAnnotation.Annotation),
		}
	}
	for edgeIndex, savedEdge := range savedGraph.Edges {
		response.Edges[edgeIndex] = savedImageGraphEdgeResponse{
			ClientID: savedEdge.ClientID,
			Edge:     toEdgeResponse(savedEdge.Edge),
		}
	}
	return response
}

// writeImageGraphError maps graph validation and persistence errors to HTTP responses.
func writeImageGraphError(responseWriter http.ResponseWriter, err error) {
	if writeWorkflowOperationError(responseWriter, err) {
		return
	}
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeError(responseWriter, http.StatusNotFound, "image not found")
	case errors.Is(err, usecase.ErrInvalidImageGraph),
		errors.Is(err, usecase.ErrInvalidAnnotationType),
		errors.Is(err, usecase.ErrInvalidAnnotationCoordinates),
		errors.Is(err, usecase.ErrInvalidEdgeType),
		errors.Is(err, usecase.ErrSelfEdge),
		errors.Is(err, usecase.ErrEdgeAnnotationNotFound),
		errors.Is(err, usecase.ErrEdgeImageMismatch),
		errors.Is(err, usecase.ErrInvalidEdgeCategory):
		writeError(responseWriter, http.StatusBadRequest, err.Error())
	case errors.Is(err, usecase.ErrDuplicateEdge),
		errors.Is(err, usecase.ErrReadingOrderCycle),
		errors.Is(err, usecase.ErrEdgeCardinalityViolation):
		writeError(responseWriter, http.StatusConflict, err.Error())
	default:
		writeError(responseWriter, http.StatusInternalServerError, err.Error())
	}
}
