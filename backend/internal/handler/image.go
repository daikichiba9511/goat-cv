package handler

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/chibadaimare/goat/backend/internal/domain"
	"github.com/chibadaimare/goat/backend/internal/usecase"
	"github.com/go-chi/chi/v5"
)

type imageResponse struct {
	ID             string `json:"id"`
	ProjectID      string `json:"project_id"`
	Filename       string `json:"filename"`
	OriginalWidth  int    `json:"original_width"`
	OriginalHeight int    `json:"original_height"`
	Width          int    `json:"width"`
	Height         int    `json:"height"`
	Rotation       int    `json:"rotation"`
	FlipH          bool   `json:"flip_h"`
	FlipV          bool   `json:"flip_v"`
	Status         string `json:"status"`
	UploadedAt     string `json:"uploaded_at"`
}

func toImageResponse(img domain.Image) imageResponse {
	return imageResponse{
		ID:             img.ID,
		ProjectID:      img.ProjectID,
		Filename:       img.Filename,
		OriginalWidth:  img.OriginalWidth,
		OriginalHeight: img.OriginalHeight,
		Width:          img.Width,
		Height:         img.Height,
		Rotation:       int(img.Rotation),
		FlipH:          img.FlipH,
		FlipV:          img.FlipV,
		Status:         string(img.Status),
		UploadedAt:     img.UploadedAt.Format("2006-01-02T15:04:05Z"),
	}
}

type ImageHandler struct {
	uc *usecase.ImageUsecase
}

func NewImageHandler(uc *usecase.ImageUsecase) *ImageHandler {
	return &ImageHandler{uc: uc}
}

func (h *ImageHandler) ProjectRoutes(r chi.Router) {
	r.Post("/", h.upload)
	r.Get("/", h.list)
}

func (h *ImageHandler) ImageRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/{imageId}", h.get)
	r.Get("/{imageId}/file", h.file)
	r.Patch("/{imageId}", h.updateTransform)
	r.Delete("/{imageId}", h.delete)
	return r
}

func (h *ImageHandler) upload(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	img, err := h.uc.Upload(r.Context(), projectID, header.Filename, file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, toImageResponse(img))
}

func (h *ImageHandler) list(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	images, err := h.uc.ListByProject(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	items := make([]imageResponse, len(images))
	for i, img := range images {
		items[i] = toImageResponse(img)
	}
	writeJSON(w, http.StatusOK, listResponse{Items: items})
}

func (h *ImageHandler) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "imageId")
	img, err := h.uc.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "image not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toImageResponse(img))
}

func (h *ImageHandler) file(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "imageId")
	img, err := h.uc.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "image not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	http.ServeFile(w, r, h.uc.FilePath(img))
}

func (h *ImageHandler) updateTransform(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "imageId")
	var req struct {
		Rotation int  `json:"rotation"`
		FlipH    bool `json:"flip_h"`
		FlipV    bool `json:"flip_v"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	img, err := h.uc.UpdateTransform(r.Context(), id, domain.Rotation(req.Rotation), req.FlipH, req.FlipV)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "image not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toImageResponse(img))
}

func (h *ImageHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "imageId")
	if err := h.uc.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
