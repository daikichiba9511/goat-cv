package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strconv"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/daikichiba9511/goat-cv/backend/internal/usecase"
	"github.com/go-chi/chi/v5"
)

type datasetExporter interface {
	WriteProjectArchive(
		ctx context.Context,
		projectID string,
		format usecase.DatasetExportFormat,
		destination io.Writer,
	) error
}

type exportAnnotationLister interface {
	ListByImage(ctx context.Context, imageID string) ([]domain.Annotation, error)
}

type exportEdgeLister interface {
	ListByImage(ctx context.Context, imageID string) ([]domain.Edge, error)
}

// ExportHandler serves dataset export routes.
type ExportHandler struct {
	projectUC        *usecase.ProjectUsecase
	imageUC          *usecase.ImageUsecase
	labelUC          *usecase.LabelUsecase
	annotationLister exportAnnotationLister
	edgeLister       exportEdgeLister
	datasetExporter  datasetExporter
}

// NewExportHandler creates an ExportHandler.
func NewExportHandler(
	projectUC *usecase.ProjectUsecase,
	imageUC *usecase.ImageUsecase,
	annotationUC *usecase.AnnotationUsecase,
	labelUC *usecase.LabelUsecase,
	edgeUC *usecase.EdgeUsecase,
	datasetExportUC *usecase.DatasetExportUsecase,
) *ExportHandler {
	return &ExportHandler{
		projectUC:        projectUC,
		imageUC:          imageUC,
		labelUC:          labelUC,
		annotationLister: annotationUC,
		edgeLister:       edgeUC,
		datasetExporter:  datasetExportUC,
	}
}

// ProjectExport writes the requested Project-level GOAT JSON, COCO, or YOLO export.
func (h *ExportHandler) ProjectExport(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" || format == "json" {
		h.writeProjectGOATJSON(w, r)
		return
	}
	if format != string(usecase.DatasetExportFormatCOCO) &&
		format != string(usecase.DatasetExportFormatYOLO) {
		writeError(w, http.StatusBadRequest, "unsupported export format")
		return
	}
	h.writeProjectDatasetArchive(w, r, usecase.DatasetExportFormat(format))
}

func (h *ExportHandler) writeProjectGOATJSON(w http.ResponseWriter, r *http.Request) {
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
	for _, image := range images {
		exportedImage, err := h.buildExportImage(ctx, image, labelMap)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		exportImages = append(exportImages, exportedImage)
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
	format := r.URL.Query().Get("format")
	if format != "" && format != "json" {
		writeError(w, http.StatusBadRequest, "image export only supports json")
		return
	}

	imageID := chi.URLParam(r, "imageId")
	ctx := r.Context()

	image, err := h.imageUC.Get(ctx, imageID)
	if err != nil {
		writeError(w, http.StatusNotFound, "image not found")
		return
	}

	labels, err := h.labelUC.ListByProject(ctx, image.ProjectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	labelMap := make(map[string]string, len(labels))
	for _, label := range labels {
		labelMap[label.ID] = label.Name
	}

	exportedImage, err := h.buildExportImage(ctx, image, labelMap)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, exportedImage)
}

func (h *ExportHandler) buildExportImage(ctx context.Context, image domain.Image, labelMap map[string]string) (exportImage, error) {
	annotations, err := h.annotationLister.ListByImage(ctx, image.ID)
	if err != nil {
		return exportImage{}, err
	}
	edges, err := h.edgeLister.ListByImage(ctx, image.ID)
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
	exportEdges := make([]exportEdge, len(edges))
	for edgeIndex, edge := range edges {
		exportEdges[edgeIndex] = exportEdge{
			ID:     edge.ID,
			Source: edge.SourceAnnotationID,
			Target: edge.TargetAnnotationID,
			Type:   string(edge.Type),
		}
	}

	return exportImage{
		ID:             image.ID,
		Filename:       image.Filename,
		OriginalWidth:  image.OriginalWidth,
		OriginalHeight: image.OriginalHeight,
		Width:          image.Width,
		Height:         image.Height,
		Rotation:       int(image.Rotation),
		FlipH:          image.FlipH,
		FlipV:          image.FlipV,
		Annotations:    exportAnns,
		Edges:          exportEdges,
	}, nil
}

func (h *ExportHandler) writeProjectDatasetArchive(
	w http.ResponseWriter,
	r *http.Request,
	format usecase.DatasetExportFormat,
) {
	projectID := chi.URLParam(r, "projectId")
	temporaryArchive, err := os.CreateTemp("", "goat-dataset-export-*.zip")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	temporaryPath := temporaryArchive.Name()
	defer os.Remove(temporaryPath)
	defer temporaryArchive.Close()

	// Why: 変換と検証を一時Fileで完了させ、失敗したZIPの一部をHTTP responseへ流さない。
	if err := h.datasetExporter.WriteProjectArchive(
		r.Context(),
		projectID,
		format,
		temporaryArchive,
	); err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidDatasetExport):
			writeError(w, http.StatusUnprocessableEntity, err.Error())
		case errors.Is(err, sql.ErrNoRows):
			writeError(w, http.StatusNotFound, "project not found")
		default:
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	archiveSize, err := temporaryArchive.Seek(0, io.SeekEnd)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := temporaryArchive.Seek(0, io.SeekStart); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	filename := fmt.Sprintf("goat-%s-%s.zip", projectID, format)
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": filename})
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", disposition)
	w.Header().Set("Content-Length", strconv.FormatInt(archiveSize, 10))
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, temporaryArchive)
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
