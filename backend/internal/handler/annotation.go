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

type annotationResponse struct {
	ID          string          `json:"id"`
	ImageID     string          `json:"image_id"`
	Type        string          `json:"type"`
	Coordinates json.RawMessage `json:"coordinates"`
	LabelID     *string         `json:"label_id"`
	CreatedAt   string          `json:"created_at"`
}

func toAnnotationResponse(a domain.Annotation) annotationResponse {
	return annotationResponse{
		ID:          a.ID,
		ImageID:     a.ImageID,
		Type:        string(a.Type),
		Coordinates: json.RawMessage(a.Coordinates),
		LabelID:     a.LabelID,
		CreatedAt:   a.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// AnnotationHandler serves annotation API routes.
type AnnotationHandler struct {
	uc *usecase.AnnotationUsecase
}

// NewAnnotationHandler creates an AnnotationHandler.
func NewAnnotationHandler(uc *usecase.AnnotationUsecase) *AnnotationHandler {
	return &AnnotationHandler{uc: uc}
}

// ImageRoutes registers image-scoped annotation collection routes.
func (h *AnnotationHandler) ImageRoutes(r chi.Router) {
	r.Post("/", h.create)
	r.Get("/", h.list)
	r.Put("/", h.bulkReplace)
}

// AnnotationRoutes returns annotation item routes.
func (h *AnnotationHandler) AnnotationRoutes() chi.Router {
	r := chi.NewRouter()
	r.Patch("/{annotationId}", h.update)
	r.Delete("/{annotationId}", h.delete)
	return r
}

func (h *AnnotationHandler) create(w http.ResponseWriter, r *http.Request) {
	imageID := chi.URLParam(r, "imageId")
	var req struct {
		Type        string          `json:"type"`
		Coordinates json.RawMessage `json:"coordinates"`
		LabelID     *string         `json:"label_id"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Type == "" || len(req.Coordinates) == 0 {
		writeError(w, http.StatusBadRequest, "type and coordinates are required")
		return
	}

	ann, err := h.uc.Create(r.Context(), imageID, domain.AnnotationType(req.Type), domain.Coordinates(req.Coordinates), req.LabelID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, toAnnotationResponse(ann))
}

func (h *AnnotationHandler) list(w http.ResponseWriter, r *http.Request) {
	imageID := chi.URLParam(r, "imageId")
	annotations, err := h.uc.ListByImage(r.Context(), imageID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	items := make([]annotationResponse, len(annotations))
	for i, annotation := range annotations {
		items[i] = toAnnotationResponse(annotation)
	}
	writeJSON(w, http.StatusOK, listResponse{Items: items})
}

func (h *AnnotationHandler) bulkReplace(w http.ResponseWriter, r *http.Request) {
	imageID := chi.URLParam(r, "imageId")
	var req struct {
		Annotations []struct {
			ID          string          `json:"id"`
			Type        string          `json:"type"`
			Coordinates json.RawMessage `json:"coordinates"`
			LabelID     *string         `json:"label_id"`
		} `json:"annotations"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	annotations := make([]domain.Annotation, len(req.Annotations))
	for i, requestAnnotation := range req.Annotations {
		annotations[i] = domain.Annotation{
			ID:          requestAnnotation.ID,
			Type:        domain.AnnotationType(requestAnnotation.Type),
			Coordinates: domain.Coordinates(requestAnnotation.Coordinates),
			LabelID:     requestAnnotation.LabelID,
		}
	}

	result, err := h.uc.BulkReplace(r.Context(), imageID, annotations)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	items := make([]annotationResponse, len(result))
	for i, annotation := range result {
		items[i] = toAnnotationResponse(annotation)
	}
	writeJSON(w, http.StatusOK, listResponse{Items: items})
}

func (h *AnnotationHandler) update(w http.ResponseWriter, r *http.Request) {
	annID := chi.URLParam(r, "annotationId")
	var req struct {
		Type        string          `json:"type"`
		Coordinates json.RawMessage `json:"coordinates"`
		LabelID     *string         `json:"label_id"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ann, err := h.uc.Update(r.Context(), annID, domain.AnnotationType(req.Type), domain.Coordinates(req.Coordinates), req.LabelID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "annotation not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toAnnotationResponse(ann))
}

func (h *AnnotationHandler) delete(w http.ResponseWriter, r *http.Request) {
	annID := chi.URLParam(r, "annotationId")
	if err := h.uc.Delete(r.Context(), annID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
