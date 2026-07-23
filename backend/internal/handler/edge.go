package handler

import (
	"errors"
	"net/http"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/daikichiba9511/goat-cv/backend/internal/usecase"
	"github.com/go-chi/chi/v5"
)

type edgeResponse struct {
	ID                 string `json:"id"`
	ImageID            string `json:"image_id"`
	SourceAnnotationID string `json:"source_annotation_id"`
	TargetAnnotationID string `json:"target_annotation_id"`
	Type               string `json:"type"`
}

func toEdgeResponse(edge domain.Edge) edgeResponse {
	return edgeResponse{
		ID:                 edge.ID,
		ImageID:            edge.ImageID,
		SourceAnnotationID: edge.SourceAnnotationID,
		TargetAnnotationID: edge.TargetAnnotationID,
		Type:               string(edge.Type),
	}
}

// EdgeHandler serves edge API routes.
type EdgeHandler struct {
	uc *usecase.EdgeUsecase
}

// NewEdgeHandler creates an EdgeHandler.
func NewEdgeHandler(uc *usecase.EdgeUsecase) *EdgeHandler {
	return &EdgeHandler{uc: uc}
}

// ImageRoutes registers image-scoped edge collection routes.
func (h *EdgeHandler) ImageRoutes(r chi.Router) {
	r.Post("/", h.create)
	r.Get("/", h.list)
	r.Put("/", h.bulkReplace)
}

// EdgeRoutes returns edge item routes.
func (h *EdgeHandler) EdgeRoutes() chi.Router {
	r := chi.NewRouter()
	r.Delete("/{edgeId}", h.delete)
	return r
}

func (h *EdgeHandler) create(w http.ResponseWriter, r *http.Request) {
	imageID := chi.URLParam(r, "imageId")
	var req struct {
		SourceAnnotationID string `json:"source_annotation_id"`
		TargetAnnotationID string `json:"target_annotation_id"`
		Type               string `json:"type"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.SourceAnnotationID == "" || req.TargetAnnotationID == "" || req.Type == "" {
		writeError(w, http.StatusBadRequest, "source_annotation_id, target_annotation_id, and type are required")
		return
	}

	edge, err := h.uc.Create(r.Context(), imageID, req.SourceAnnotationID, req.TargetAnnotationID, domain.EdgeType(req.Type))
	if err != nil {
		writeEdgeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toEdgeResponse(edge))
}

func (h *EdgeHandler) list(w http.ResponseWriter, r *http.Request) {
	imageID := chi.URLParam(r, "imageId")
	edges, err := h.uc.ListByImage(r.Context(), imageID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	items := make([]edgeResponse, len(edges))
	for i, edge := range edges {
		items[i] = toEdgeResponse(edge)
	}
	writeJSON(w, http.StatusOK, listResponse{Items: items})
}

func (h *EdgeHandler) bulkReplace(w http.ResponseWriter, r *http.Request) {
	imageID := chi.URLParam(r, "imageId")
	var req struct {
		Edges []struct {
			ID                 string `json:"id"`
			SourceAnnotationID string `json:"source_annotation_id"`
			TargetAnnotationID string `json:"target_annotation_id"`
			Type               string `json:"type"`
		} `json:"edges"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	edges := make([]domain.Edge, len(req.Edges))
	for i, requestEdge := range req.Edges {
		edges[i] = domain.Edge{
			ID:                 requestEdge.ID,
			SourceAnnotationID: requestEdge.SourceAnnotationID,
			TargetAnnotationID: requestEdge.TargetAnnotationID,
			Type:               domain.EdgeType(requestEdge.Type),
		}
	}

	result, err := h.uc.BulkReplace(r.Context(), imageID, edges)
	if err != nil {
		writeEdgeError(w, err)
		return
	}
	items := make([]edgeResponse, len(result))
	for i, edge := range result {
		items[i] = toEdgeResponse(edge)
	}
	writeJSON(w, http.StatusOK, listResponse{Items: items})
}

func (h *EdgeHandler) delete(w http.ResponseWriter, r *http.Request) {
	edgeID := chi.URLParam(r, "edgeId")
	if err := h.uc.Delete(r.Context(), edgeID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeEdgeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrEdgeAnnotationNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, usecase.ErrDuplicateEdge),
		errors.Is(err, usecase.ErrReadingOrderCycle),
		errors.Is(err, usecase.ErrEdgeCardinalityViolation):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, usecase.ErrInvalidEdgeType),
		errors.Is(err, usecase.ErrSelfEdge),
		errors.Is(err, usecase.ErrEdgeImageMismatch),
		errors.Is(err, usecase.ErrInvalidEdgeCategory):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}
