package handler

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/daikichiba9511/goat-cv/backend/internal/usecase"
	"github.com/go-chi/chi/v5"
)

type labelResponse struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	Name      string `json:"name"`
	Color     string `json:"color"`
	Category  string `json:"category"`
}

func toLabelResponse(l domain.LabelDefinition) labelResponse {
	return labelResponse{
		ID:        l.ID,
		ProjectID: l.ProjectID,
		Name:      l.Name,
		Color:     l.Color,
		Category:  string(l.Category),
	}
}

// LabelHandler serves label definition API routes.
type LabelHandler struct {
	uc *usecase.LabelUsecase
}

// NewLabelHandler creates a LabelHandler.
func NewLabelHandler(uc *usecase.LabelUsecase) *LabelHandler {
	return &LabelHandler{uc: uc}
}

// Routes registers label definition routes on a project-scoped router.
func (h *LabelHandler) Routes(r chi.Router) {
	r.Post("/", h.create)
	r.Get("/", h.list)
	r.Patch("/{labelId}", h.update)
	r.Delete("/{labelId}", h.delete)
}

func (h *LabelHandler) create(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	var req struct {
		Name     string `json:"name"`
		Color    string `json:"color"`
		Category string `json:"category"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.Color == "" || req.Category == "" {
		writeError(w, http.StatusBadRequest, "name, color, and category are required")
		return
	}

	label, err := h.uc.Create(r.Context(), projectID, req.Name, req.Color, domain.LabelCategory(req.Category))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, toLabelResponse(label))
}

func (h *LabelHandler) list(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	labels, err := h.uc.ListByProject(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	items := make([]labelResponse, len(labels))
	for i, label := range labels {
		items[i] = toLabelResponse(label)
	}
	writeJSON(w, http.StatusOK, listResponse{Items: items})
}

func (h *LabelHandler) update(w http.ResponseWriter, r *http.Request) {
	labelID := chi.URLParam(r, "labelId")
	var req struct {
		Name     string `json:"name"`
		Color    string `json:"color"`
		Category string `json:"category"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	label, err := h.uc.Update(r.Context(), labelID, req.Name, req.Color, domain.LabelCategory(req.Category))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "label not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toLabelResponse(label))
}

func (h *LabelHandler) delete(w http.ResponseWriter, r *http.Request) {
	labelID := chi.URLParam(r, "labelId")
	if err := h.uc.Delete(r.Context(), labelID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
