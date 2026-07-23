package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/daikichiba9511/goat-cv/backend/internal/usecase"
	"github.com/go-chi/chi/v5"
)

// ExportHandler serves dataset export routes.
type ExportHandler struct {
	projectUC    *usecase.ProjectUsecase
	imageUC      *usecase.ImageUsecase
	annotationUC *usecase.AnnotationUsecase
	labelUC      *usecase.LabelUsecase
}

// NewExportHandler creates an ExportHandler.
func NewExportHandler(
	projectUC *usecase.ProjectUsecase,
	imageUC *usecase.ImageUsecase,
	annotationUC *usecase.AnnotationUsecase,
	labelUC *usecase.LabelUsecase,
) *ExportHandler {
	return &ExportHandler{
		projectUC:    projectUC,
		imageUC:      imageUC,
		annotationUC: annotationUC,
		labelUC:      labelUC,
	}
}

// ProjectExport writes a GOAT JSON export for a project.
func (h *ExportHandler) ProjectExport(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	ctx := r.Context()

	project, err := h.projectUC.Get(ctx, projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	labels, err := h.labelUC.ListByProject(ctx, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	images, err := h.imageUC.ListByProject(ctx, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	labelMap := make(map[string]string, len(labels))
	for _, label := range labels {
		labelMap[label.ID] = label.Name
	}

	exportImages := make([]exportImage, 0, len(images))
	for _, img := range images {
		ei, err := h.buildExportImage(ctx, img, labelMap)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		exportImages = append(exportImages, ei)
	}

	exportLabels := make([]exportLabel, len(labels))
	for i, label := range labels {
		exportLabels[i] = exportLabel{
			ID:       label.ID,
			Name:     label.Name,
			Color:    label.Color,
			Category: string(label.Category),
		}
	}

	result := goatExport{
		Format:  "goat_json",
		Version: "1.0",
		Project: exportProject{ID: project.ID, Name: project.Name},
		Labels:  exportLabels,
		Images:  exportImages,
	}

	writeJSON(w, http.StatusOK, result)
}

// ImageExport writes a GOAT JSON export for a single image.
func (h *ExportHandler) ImageExport(w http.ResponseWriter, r *http.Request) {
	imageID := chi.URLParam(r, "imageId")
	ctx := r.Context()

	img, err := h.imageUC.Get(ctx, imageID)
	if err != nil {
		writeError(w, http.StatusNotFound, "image not found")
		return
	}

	labels, err := h.labelUC.ListByProject(ctx, img.ProjectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	labelMap := make(map[string]string, len(labels))
	for _, label := range labels {
		labelMap[label.ID] = label.Name
	}

	ei, err := h.buildExportImage(ctx, img, labelMap)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, ei)
}

func (h *ExportHandler) buildExportImage(ctx context.Context, img domain.Image, labelMap map[string]string) (exportImage, error) {
	annotations, err := h.annotationUC.ListByImage(ctx, img.ID)
	if err != nil {
		return exportImage{}, err
	}

	exportAnns := make([]exportAnnotation, len(annotations))
	for i, annotation := range annotations {
		var labelName string
		if annotation.LabelID != nil {
			// Why: label_idはON DELETE SET NULLなので、欠落したラベル名は空文字のまま出して古いAnnotationを壊さない。
			labelName = labelMap[*annotation.LabelID]
		}
		exportAnns[i] = exportAnnotation{
			ID:          annotation.ID,
			Type:        string(annotation.Type),
			Coordinates: json.RawMessage(annotation.Coordinates),
			LabelID:     annotation.LabelID,
			Label:       labelName,
		}
	}

	return exportImage{
		ID:             img.ID,
		Filename:       img.Filename,
		OriginalWidth:  img.OriginalWidth,
		OriginalHeight: img.OriginalHeight,
		Width:          img.Width,
		Height:         img.Height,
		Rotation:       int(img.Rotation),
		FlipH:          img.FlipH,
		FlipV:          img.FlipV,
		Annotations:    exportAnns,
		// Why not: Edgeの完全ExportはPhase 2以降で扱う。Phase 1のJSON形状だけ先に固定する。
		Edges: []exportEdge{},
	}, nil
}

type goatExport struct {
	Format  string        `json:"format"`
	Version string        `json:"version"`
	Project exportProject `json:"project"`
	Labels  []exportLabel `json:"labels"`
	Images  []exportImage `json:"images"`
}

type exportProject struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type exportLabel struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Color    string `json:"color"`
	Category string `json:"category"`
}

type exportImage struct {
	ID             string             `json:"id"`
	Filename       string             `json:"filename"`
	OriginalWidth  int                `json:"original_width"`
	OriginalHeight int                `json:"original_height"`
	Width          int                `json:"width"`
	Height         int                `json:"height"`
	Rotation       int                `json:"rotation"`
	FlipH          bool               `json:"flip_h"`
	FlipV          bool               `json:"flip_v"`
	Annotations    []exportAnnotation `json:"annotations"`
	Edges          []exportEdge       `json:"edges"`
}

type exportAnnotation struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	Coordinates json.RawMessage `json:"coordinates"`
	LabelID     *string         `json:"label_id"`
	Label       string          `json:"label"`
}

type exportEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
}
