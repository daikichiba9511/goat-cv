package handler

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/daikichiba9511/goat-cv/backend/internal/usecase"
	"github.com/go-chi/chi/v5"
)

type projectResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

func toProjectResponse(p domain.Project) projectResponse {
	return projectResponse{
		ID:        p.ID,
		Name:      p.Name,
		CreatedAt: p.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// ProjectHandler serves project API routes.
type ProjectHandler struct {
	uc *usecase.ProjectUsecase
}

// NewProjectHandler creates a ProjectHandler.
func NewProjectHandler(uc *usecase.ProjectUsecase) *ProjectHandler {
	return &ProjectHandler{uc: uc}
}

// Routes returns the project API router.
func (h *ProjectHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.create)
	r.Get("/", h.list)
	r.Get("/{projectId}", h.get)
	r.Patch("/{projectId}", h.update)
	r.Delete("/{projectId}", h.delete)
	return r
}

func (h *ProjectHandler) create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	p, err := h.uc.Create(r.Context(), req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, toProjectResponse(p))
}

func (h *ProjectHandler) list(w http.ResponseWriter, r *http.Request) {
	projects, err := h.uc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	items := make([]projectResponse, len(projects))
	for i, project := range projects {
		items[i] = toProjectResponse(project)
	}
	writeJSON(w, http.StatusOK, listResponse{Items: items})
}

func (h *ProjectHandler) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "projectId")
	p, err := h.uc.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toProjectResponse(p))
}

func (h *ProjectHandler) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "projectId")
	var req struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	p, err := h.uc.Update(r.Context(), id, req.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toProjectResponse(p))
}

func (h *ProjectHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "projectId")
	if err := h.uc.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
