package usecase_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/daikichiba9511/goat-cv/backend/internal/usecase"
)

func TestDatasetExportUsecaseWritesCOCOBBoxPolygonAndClassMapping(t *testing.T) {
	fixture := newDatasetExportFixture(t, []domain.LabelDefinition{
		{ID: "label-table", ProjectID: exportProjectID, Name: "table", Category: domain.LabelCategoryTable},
		{ID: "label-object", ProjectID: exportProjectID, Name: "object", Category: domain.LabelCategoryObject},
	}, []domain.Image{
		{
			ID: "image-a", ProjectID: exportProjectID, Filename: "source.png",
			OriginalWidth: 100, OriginalHeight: 50, Width: 50, Height: 100,
			Rotation: domain.Rotation90, FlipH: true,
		},
	}, map[string][]domain.Annotation{
		"image-a": {
			{
				ID: "annotation-bbox", ImageID: "image-a", Type: domain.AnnotationTypeBBox,
				Coordinates: domain.Coordinates(`{"x":0,"y":0,"width":1,"height":1}`),
				LabelID:     stringPointer("label-object"),
			},
			{
				ID: "annotation-polygon", ImageID: "image-a", Type: domain.AnnotationTypePolygon,
				Coordinates: domain.Coordinates(`{"points":[{"x":0,"y":0},{"x":1,"y":0},{"x":1,"y":1}]}`),
				LabelID:     stringPointer("label-table"),
			},
		},
	})

	var archive bytes.Buffer
	err := fixture.usecase.WriteProjectArchive(
		fixture.ctx,
		exportProjectID,
		usecase.DatasetExportFormatCOCO,
		&archive,
	)
	if err != nil {
		t.Fatalf("WriteProjectArchive returned error: %v", err)
	}

	entries := readZIPEntries(t, archive.Bytes())
	if string(entries["images/default/image-a.png"]) != "image-a" {
		t.Fatalf("exported image = %q, want original bytes", entries["images/default/image-a.png"])
	}

	var dataset struct {
		Images []struct {
			ID       int    `json:"id"`
			FileName string `json:"file_name"`
			Width    int    `json:"width"`
			Height   int    `json:"height"`
		} `json:"images"`
		Categories []struct {
			ID          int    `json:"id"`
			Name        string `json:"name"`
			GOATLabelID string `json:"goat_label_id"`
		} `json:"categories"`
		Annotations []struct {
			ID           int         `json:"id"`
			CategoryID   int         `json:"category_id"`
			BBox         []float64   `json:"bbox"`
			Segmentation [][]float64 `json:"segmentation"`
			Area         float64     `json:"area"`
		} `json:"annotations"`
	}
	decodeJSONEntry(t, entries, "annotations/instances_default.json", &dataset)

	if len(dataset.Images) != 1 || dataset.Images[0].FileName != "images/default/image-a.png" ||
		dataset.Images[0].Width != 100 || dataset.Images[0].Height != 50 {
		t.Fatalf("COCO images = %+v, want original image dimensions and archive path", dataset.Images)
	}
	if len(dataset.Categories) != 2 ||
		dataset.Categories[0].ID != 1 || dataset.Categories[0].GOATLabelID != "label-object" ||
		dataset.Categories[1].ID != 2 || dataset.Categories[1].GOATLabelID != "label-table" {
		t.Fatalf("COCO categories = %+v, want stable name order with Label IDs", dataset.Categories)
	}
	if len(dataset.Annotations) != 2 {
		t.Fatalf("COCO annotations count = %d, want 2", len(dataset.Annotations))
	}
	boundingBox := dataset.Annotations[0]
	if boundingBox.CategoryID != 1 || !equalFloatSlices(boundingBox.BBox, []float64{0, 0, 100, 50}) ||
		boundingBox.Area != 5000 || len(boundingBox.Segmentation) != 0 {
		t.Fatalf("COCO BBox annotation = %+v", boundingBox)
	}
	polygon := dataset.Annotations[1]
	if polygon.CategoryID != 2 || !equalFloatSlices(polygon.BBox, []float64{0, 0, 100, 50}) ||
		polygon.Area != 2500 || len(polygon.Segmentation) != 1 ||
		!equalFloatSlices(polygon.Segmentation[0], []float64{100, 50, 100, 0, 0, 0}) {
		t.Fatalf("COCO Polygon annotation = %+v", polygon)
	}

	manifest := decodeManifest(t, entries)
	if manifest.EdgesIncluded || len(manifest.Warnings) != 0 || len(manifest.ClassMapping) != 2 {
		t.Fatalf("manifest = %+v, want explicit Edge exclusion and complete class mapping", manifest)
	}
	if len(manifest.Images) != 1 || manifest.Images[0].ImageID != "image-a" ||
		manifest.Images[0].OriginalFilename != "source.png" ||
		manifest.Images[0].ExportedPath != "images/default/image-a.png" ||
		manifest.Images[0].Rotation != 90 || !manifest.Images[0].FlipH {
		t.Fatalf("manifest images = %+v, want original Image and transform metadata", manifest.Images)
	}
}

func TestDatasetExportUsecaseWritesYOLOObjectBBoxAndReportsExcludedAnnotations(t *testing.T) {
	fixture := newDatasetExportFixture(t, []domain.LabelDefinition{
		{ID: "label-zebra", ProjectID: exportProjectID, Name: "zebra", Category: domain.LabelCategoryObject},
		{ID: "label-key", ProjectID: exportProjectID, Name: "key", Category: domain.LabelCategoryKey},
		{ID: "label-apple", ProjectID: exportProjectID, Name: "apple", Category: domain.LabelCategoryObject},
	}, []domain.Image{
		{
			ID: "image-a", ProjectID: exportProjectID, Filename: "source.jpg",
			OriginalWidth: 100, OriginalHeight: 50, Width: 50, Height: 100,
			Rotation: domain.Rotation90, FlipH: true,
		},
		{
			ID: "image-empty", ProjectID: exportProjectID, Filename: "empty.png",
			OriginalWidth: 20, OriginalHeight: 10, Width: 20, Height: 10,
		},
	}, map[string][]domain.Annotation{
		"image-a": {
			{
				ID: "annotation-object", ImageID: "image-a", Type: domain.AnnotationTypeBBox,
				Coordinates: domain.Coordinates(`{"x":0.1,"y":0.2,"width":0.3,"height":0.4}`),
				LabelID:     stringPointer("label-apple"),
			},
			{
				ID: "annotation-key", ImageID: "image-a", Type: domain.AnnotationTypeBBox,
				Coordinates: domain.Coordinates(`{"x":0,"y":0,"width":0.1,"height":0.1}`),
				LabelID:     stringPointer("label-key"),
			},
			{
				ID: "annotation-polygon", ImageID: "image-a", Type: domain.AnnotationTypePolygon,
				Coordinates: domain.Coordinates(`{"points":[{"x":0,"y":0},{"x":1,"y":0},{"x":1,"y":1}]}`),
				LabelID:     stringPointer("label-apple"),
			},
		},
	})

	var archive bytes.Buffer
	err := fixture.usecase.WriteProjectArchive(
		fixture.ctx,
		exportProjectID,
		usecase.DatasetExportFormatYOLO,
		&archive,
	)
	if err != nil {
		t.Fatalf("WriteProjectArchive returned error: %v", err)
	}

	entries := readZIPEntries(t, archive.Bytes())
	if got := string(entries["labels/train/image-a.txt"]); got != "0 0.6 0.75 0.4 0.3\n" {
		t.Fatalf("YOLO label = %q, want inverse-transformed normalized BBox", got)
	}
	if got := string(entries["labels/train/image-empty.txt"]); got != "" {
		t.Fatalf("empty image label file = %q, want empty", got)
	}
	if got := string(entries["data.yaml"]); !strings.Contains(got, "0: \"apple\"") ||
		!strings.Contains(got, "1: \"zebra\"") || strings.Contains(got, "key") {
		t.Fatalf("data.yaml = %q, want only object classes in stable order", got)
	}

	var classes []struct {
		ClassID int    `json:"class_id"`
		LabelID string `json:"label_id"`
	}
	decodeJSONEntry(t, entries, "classes.json", &classes)
	if len(classes) != 2 || classes[0].ClassID != 0 || classes[0].LabelID != "label-apple" ||
		classes[1].ClassID != 1 || classes[1].LabelID != "label-zebra" {
		t.Fatalf("classes.json = %+v, want reproducible class-to-Label mapping", classes)
	}

	manifest := decodeManifest(t, entries)
	if manifest.EdgesIncluded || len(manifest.Warnings) != 2 {
		t.Fatalf("manifest = %+v, want Edge exclusion and 2 annotation warnings", manifest)
	}
	warningByAnnotationID := make(map[string]string, len(manifest.Warnings))
	for _, warning := range manifest.Warnings {
		warningByAnnotationID[warning.AnnotationID] = warning.Reason
	}
	if warningByAnnotationID["annotation-key"] != "unsupported_label_category" ||
		warningByAnnotationID["annotation-polygon"] != "unsupported_annotation_type" {
		t.Fatalf("warnings = %+v, want explicit exclusion reasons", manifest.Warnings)
	}
}

func TestDatasetExportUsecaseWritesValidArchiveForEmptyProject(t *testing.T) {
	formats := []usecase.DatasetExportFormat{
		usecase.DatasetExportFormatCOCO,
		usecase.DatasetExportFormatYOLO,
	}
	for _, format := range formats {
		t.Run(string(format), func(t *testing.T) {
			fixture := newDatasetExportFixture(t, nil, nil, nil)

			var archive bytes.Buffer
			if err := fixture.usecase.WriteProjectArchive(fixture.ctx, exportProjectID, format, &archive); err != nil {
				t.Fatalf("WriteProjectArchive returned error: %v", err)
			}
			entries := readZIPEntries(t, archive.Bytes())
			manifest := decodeManifest(t, entries)
			if manifest.ProjectID != exportProjectID || len(manifest.ClassMapping) != 0 {
				t.Fatalf("manifest = %+v, want empty Project metadata", manifest)
			}
			if format == usecase.DatasetExportFormatCOCO {
				var dataset struct {
					Images      []json.RawMessage `json:"images"`
					Annotations []json.RawMessage `json:"annotations"`
				}
				decodeJSONEntry(t, entries, "annotations/instances_default.json", &dataset)
				if len(dataset.Images) != 0 || len(dataset.Annotations) != 0 {
					t.Fatalf("empty COCO dataset = %+v", dataset)
				}
			} else if got := string(entries["data.yaml"]); !strings.Contains(got, "names: {}") {
				t.Fatalf("empty YOLO data.yaml = %q, want empty class map", got)
			}
		})
	}
}

func TestDatasetExportUsecaseRejectsInvalidStoredAnnotationWithoutArchive(t *testing.T) {
	tests := []struct {
		name       string
		annotation domain.Annotation
	}{
		{
			name: "invalid coordinates",
			annotation: domain.Annotation{
				ID: "annotation-invalid", ImageID: "image-a", Type: domain.AnnotationTypeBBox,
				Coordinates: domain.Coordinates(`{"x":0,"y":0,"width":0,"height":1}`),
				LabelID:     stringPointer("label-object"),
			},
		},
		{
			name: "missing label",
			annotation: domain.Annotation{
				ID: "annotation-unlabeled", ImageID: "image-a", Type: domain.AnnotationTypeBBox,
				Coordinates: domain.Coordinates(`{"x":0,"y":0,"width":1,"height":1}`),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newDatasetExportFixture(t, []domain.LabelDefinition{
				{ID: "label-object", ProjectID: exportProjectID, Name: "object", Category: domain.LabelCategoryObject},
			}, []domain.Image{
				{
					ID: "image-a", ProjectID: exportProjectID, Filename: "source.png",
					OriginalWidth: 100, OriginalHeight: 50, Width: 100, Height: 50,
				},
			}, map[string][]domain.Annotation{"image-a": {test.annotation}})

			var archive bytes.Buffer
			err := fixture.usecase.WriteProjectArchive(
				fixture.ctx,
				exportProjectID,
				usecase.DatasetExportFormatCOCO,
				&archive,
			)
			if !errors.Is(err, usecase.ErrInvalidDatasetExport) {
				t.Fatalf("WriteProjectArchive error = %v, want ErrInvalidDatasetExport", err)
			}
			if archive.Len() != 0 {
				t.Fatalf("archive size = %d, want no partial archive", archive.Len())
			}
		})
	}
}

const exportProjectID = "project-export"

type datasetExportFixture struct {
	ctx     context.Context
	usecase *usecase.DatasetExportUsecase
}

func newDatasetExportFixture(
	t *testing.T,
	labels []domain.LabelDefinition,
	images []domain.Image,
	annotationsByImageID map[string][]domain.Annotation,
) datasetExportFixture {
	t.Helper()
	storagePath := t.TempDir()
	for _, image := range images {
		path := filepath.Join(storagePath, image.ID+filepath.Ext(image.Filename))
		if err := os.WriteFile(path, []byte(image.ID), 0o600); err != nil {
			t.Fatalf("write fixture image: %v", err)
		}
	}

	return datasetExportFixture{
		ctx: context.Background(),
		usecase: usecase.NewDatasetExportUsecase(
			datasetExportProjectRepository{project: domain.Project{ID: exportProjectID, Name: "Export Project"}},
			datasetExportImageRepository{images: images},
			datasetExportAnnotationRepository{annotationsByImageID: annotationsByImageID},
			datasetExportLabelRepository{labels: labels},
			storagePath,
		),
	}
}

type datasetExportProjectRepository struct {
	project domain.Project
}

func (r datasetExportProjectRepository) Get(context.Context, string) (domain.Project, error) {
	return r.project, nil
}

type datasetExportImageRepository struct {
	images []domain.Image
}

func (r datasetExportImageRepository) ListByProject(context.Context, string) ([]domain.Image, error) {
	return r.images, nil
}

type datasetExportAnnotationRepository struct {
	annotationsByImageID map[string][]domain.Annotation
}

func (r datasetExportAnnotationRepository) ListByImage(_ context.Context, imageID string) ([]domain.Annotation, error) {
	return r.annotationsByImageID[imageID], nil
}

type datasetExportLabelRepository struct {
	labels []domain.LabelDefinition
}

func (r datasetExportLabelRepository) ListByProject(context.Context, string) ([]domain.LabelDefinition, error) {
	return r.labels, nil
}

type exportManifest struct {
	ProjectID     string `json:"project_id"`
	EdgesIncluded bool   `json:"edges_included"`
	ClassMapping  []struct {
		ClassID int    `json:"class_id"`
		LabelID string `json:"label_id"`
	} `json:"class_mapping"`
	Images []struct {
		ImageID          string `json:"image_id"`
		OriginalFilename string `json:"original_filename"`
		ExportedPath     string `json:"exported_path"`
		Rotation         int    `json:"rotation"`
		FlipH            bool   `json:"flip_h"`
	} `json:"images"`
	Warnings []struct {
		ImageID      string `json:"image_id"`
		AnnotationID string `json:"annotation_id"`
		Reason       string `json:"reason"`
	} `json:"warnings"`
}

func decodeManifest(t *testing.T, entries map[string][]byte) exportManifest {
	t.Helper()
	var manifest exportManifest
	decodeJSONEntry(t, entries, "manifest.json", &manifest)
	return manifest
}

func readZIPEntries(t *testing.T, archive []byte) map[string][]byte {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		t.Fatalf("open ZIP: %v", err)
	}
	entries := make(map[string][]byte, len(reader.File))
	for _, file := range reader.File {
		content, err := func() ([]byte, error) {
			entry, err := file.Open()
			if err != nil {
				return nil, err
			}
			defer entry.Close()
			var buffer bytes.Buffer
			_, err = buffer.ReadFrom(entry)
			return buffer.Bytes(), err
		}()
		if err != nil {
			t.Fatalf("read ZIP entry %q: %v", file.Name, err)
		}
		entries[file.Name] = content
	}
	return entries
}

func decodeJSONEntry(t *testing.T, entries map[string][]byte, name string, target any) {
	t.Helper()
	content, ok := entries[name]
	if !ok {
		t.Fatalf("ZIP entry %q not found", name)
	}
	if err := json.Unmarshal(content, target); err != nil {
		t.Fatalf("decode %s: %v\n%s", name, err, content)
	}
}

func equalFloatSlices(left, right []float64) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func stringPointer(value string) *string {
	return &value
}
