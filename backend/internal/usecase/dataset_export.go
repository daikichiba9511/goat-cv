package usecase

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
)

// DatasetExportFormat identifies a Project-level standard dataset archive.
type DatasetExportFormat string

const (
	// DatasetExportFormatCOCO exports COCO object detection and segmentation data.
	DatasetExportFormatCOCO DatasetExportFormat = "coco"
	// DatasetExportFormatYOLO exports YOLO object detection data.
	DatasetExportFormatYOLO DatasetExportFormat = "yolo"
)

// ErrInvalidDatasetExport indicates persisted data that cannot be converted safely.
var ErrInvalidDatasetExport = errors.New("invalid dataset export")

type datasetExportProjectRepository interface {
	Get(ctx context.Context, id string) (domain.Project, error)
}

type datasetExportImageRepository interface {
	ListByProject(ctx context.Context, projectID string) ([]domain.Image, error)
}

type datasetExportAnnotationRepository interface {
	ListByImage(ctx context.Context, imageID string) ([]domain.Annotation, error)
}

type datasetExportLabelRepository interface {
	ListByProject(ctx context.Context, projectID string) ([]domain.LabelDefinition, error)
}

// DatasetExportUsecase validates and writes Project-level COCO and YOLO archives.
type DatasetExportUsecase struct {
	projectRepository    datasetExportProjectRepository
	imageRepository      datasetExportImageRepository
	annotationRepository datasetExportAnnotationRepository
	labelRepository      datasetExportLabelRepository
	storagePath          string
}

// NewDatasetExportUsecase creates a DatasetExportUsecase from read-only dataset repositories.
func NewDatasetExportUsecase(
	projectRepository datasetExportProjectRepository,
	imageRepository datasetExportImageRepository,
	annotationRepository datasetExportAnnotationRepository,
	labelRepository datasetExportLabelRepository,
	storagePath string,
) *DatasetExportUsecase {
	return &DatasetExportUsecase{
		projectRepository:    projectRepository,
		imageRepository:      imageRepository,
		annotationRepository: annotationRepository,
		labelRepository:      labelRepository,
		storagePath:          storagePath,
	}
}

// WriteProjectArchive writes a fully validated COCO or YOLO ZIP archive to destination.
func (u *DatasetExportUsecase) WriteProjectArchive(
	ctx context.Context,
	projectID string,
	format DatasetExportFormat,
	destination io.Writer,
) error {
	if format != DatasetExportFormatCOCO && format != DatasetExportFormatYOLO {
		return fmt.Errorf("%w: unsupported format %q", ErrInvalidDatasetExport, format)
	}

	dataset, err := u.loadDataset(ctx, projectID)
	if err != nil {
		return err
	}

	archive := zip.NewWriter(destination)
	switch format {
	case DatasetExportFormatCOCO:
		err = u.writeCOCOArchive(archive, dataset)
	case DatasetExportFormatYOLO:
		err = u.writeYOLOArchive(archive, dataset)
	}
	if err != nil {
		_ = archive.Close()
		return err
	}
	if err := archive.Close(); err != nil {
		return fmt.Errorf("close dataset archive: %w", err)
	}
	return nil
}

type exportDataset struct {
	project   domain.Project
	labels    []domain.LabelDefinition
	labelByID map[string]domain.LabelDefinition
	images    []exportDatasetImage
}

type exportDatasetImage struct {
	image       domain.Image
	annotations []domain.Annotation
	filePath    string
}

func (u *DatasetExportUsecase) loadDataset(ctx context.Context, projectID string) (exportDataset, error) {
	project, err := u.projectRepository.Get(ctx, projectID)
	if err != nil {
		return exportDataset{}, err
	}
	labels, err := u.labelRepository.ListByProject(ctx, projectID)
	if err != nil {
		return exportDataset{}, err
	}
	images, err := u.imageRepository.ListByProject(ctx, projectID)
	if err != nil {
		return exportDataset{}, err
	}

	sort.Slice(labels, func(leftIndex, rightIndex int) bool {
		if labels[leftIndex].Name == labels[rightIndex].Name {
			return labels[leftIndex].ID < labels[rightIndex].ID
		}
		return labels[leftIndex].Name < labels[rightIndex].Name
	})
	labelByID := make(map[string]domain.LabelDefinition, len(labels))
	for _, label := range labels {
		if label.ProjectID != projectID {
			return exportDataset{}, invalidDatasetError("label %s belongs to project %s", label.ID, label.ProjectID)
		}
		labelByID[label.ID] = label
	}

	sort.Slice(images, func(leftIndex, rightIndex int) bool {
		return images[leftIndex].ID < images[rightIndex].ID
	})
	exportImages := make([]exportDatasetImage, len(images))
	for imageIndex, image := range images {
		if err := validateExportImage(image, projectID); err != nil {
			return exportDataset{}, err
		}
		imageFilePath := filepath.Join(u.storagePath, image.ID+filepath.Ext(image.Filename))
		if _, err := os.Stat(imageFilePath); err != nil {
			return exportDataset{}, fmt.Errorf("stat image %s: %w", image.ID, err)
		}

		annotations, err := u.annotationRepository.ListByImage(ctx, image.ID)
		if err != nil {
			return exportDataset{}, err
		}
		sort.Slice(annotations, func(leftIndex, rightIndex int) bool {
			return annotations[leftIndex].ID < annotations[rightIndex].ID
		})
		for _, annotation := range annotations {
			if annotation.ImageID != image.ID {
				return exportDataset{}, invalidDatasetError(
					"annotation %s belongs to image %s, not %s",
					annotation.ID,
					annotation.ImageID,
					image.ID,
				)
			}
			if err := validateAnnotationCoordinates(annotation.Type, annotation.Coordinates); err != nil {
				return exportDataset{}, invalidDatasetError(
					"image %s annotation %s: %v",
					image.ID,
					annotation.ID,
					err,
				)
			}
			if annotation.LabelID == nil {
				return exportDataset{}, invalidDatasetError(
					"image %s annotation %s has no label",
					image.ID,
					annotation.ID,
				)
			}
			if _, exists := labelByID[*annotation.LabelID]; !exists {
				return exportDataset{}, invalidDatasetError(
					"image %s annotation %s references label %s outside the project",
					image.ID,
					annotation.ID,
					*annotation.LabelID,
				)
			}
		}
		exportImages[imageIndex] = exportDatasetImage{
			image:       image,
			annotations: annotations,
			filePath:    imageFilePath,
		}
	}

	return exportDataset{
		project:   project,
		labels:    labels,
		labelByID: labelByID,
		images:    exportImages,
	}, nil
}

func validateExportImage(image domain.Image, projectID string) error {
	if image.ProjectID != projectID {
		return invalidDatasetError("image %s belongs to project %s", image.ID, image.ProjectID)
	}
	if image.OriginalWidth <= 0 || image.OriginalHeight <= 0 {
		return invalidDatasetError("image %s has invalid original dimensions", image.ID)
	}
	if image.Rotation != domain.Rotation0 && image.Rotation != domain.Rotation90 &&
		image.Rotation != domain.Rotation180 && image.Rotation != domain.Rotation270 {
		return invalidDatasetError("image %s has unsupported rotation %d", image.ID, image.Rotation)
	}
	expectedWidth, expectedHeight := domain.EffectiveDimensions(
		image.OriginalWidth,
		image.OriginalHeight,
		image.Rotation,
	)
	if image.Width != expectedWidth || image.Height != expectedHeight {
		return invalidDatasetError(
			"image %s dimensions are %dx%d, want %dx%d for rotation %d",
			image.ID,
			image.Width,
			image.Height,
			expectedWidth,
			expectedHeight,
			image.Rotation,
		)
	}
	return nil
}

func invalidDatasetError(format string, arguments ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidDatasetExport, fmt.Sprintf(format, arguments...))
}

type datasetClassMapping struct {
	ClassID  int    `json:"class_id"`
	LabelID  string `json:"label_id"`
	Name     string `json:"name"`
	Category string `json:"category"`
}

type datasetExportWarning struct {
	ImageID      string `json:"image_id"`
	AnnotationID string `json:"annotation_id"`
	Reason       string `json:"reason"`
}

type datasetImageManifest struct {
	ImageID          string `json:"image_id"`
	OriginalFilename string `json:"original_filename"`
	ExportedPath     string `json:"exported_path"`
	OriginalWidth    int    `json:"original_width"`
	OriginalHeight   int    `json:"original_height"`
	Rotation         int    `json:"rotation"`
	FlipH            bool   `json:"flip_h"`
	FlipV            bool   `json:"flip_v"`
}

type datasetManifest struct {
	Format            string                 `json:"format"`
	Version           string                 `json:"version"`
	ProjectID         string                 `json:"project_id"`
	ProjectName       string                 `json:"project_name"`
	CoordinateSpace   string                 `json:"coordinate_space"`
	TransformHandling string                 `json:"transform_handling"`
	EdgesIncluded     bool                   `json:"edges_included"`
	EdgeNote          string                 `json:"edge_note"`
	ClassMapping      []datasetClassMapping  `json:"class_mapping"`
	Images            []datasetImageManifest `json:"images"`
	Warnings          []datasetExportWarning `json:"warnings"`
}

func newDatasetManifest(
	dataset exportDataset,
	format DatasetExportFormat,
	classMapping []datasetClassMapping,
	warnings []datasetExportWarning,
) datasetManifest {
	images := make([]datasetImageManifest, len(dataset.images))
	imageDirectory := "images/default"
	if format == DatasetExportFormatYOLO {
		imageDirectory = "images/train"
	}
	for imageIndex, datasetImage := range dataset.images {
		image := datasetImage.image
		images[imageIndex] = datasetImageManifest{
			ImageID: image.ID, OriginalFilename: image.Filename,
			ExportedPath:  archiveImagePath(imageDirectory, image),
			OriginalWidth: image.OriginalWidth, OriginalHeight: image.OriginalHeight,
			Rotation: int(image.Rotation), FlipH: image.FlipH, FlipV: image.FlipV,
		}
	}
	return datasetManifest{
		Format:            string(format),
		Version:           "1.0",
		ProjectID:         dataset.project.ID,
		ProjectName:       dataset.project.Name,
		CoordinateSpace:   "original_image",
		TransformHandling: "inverse_to_original_image",
		EdgesIncluded:     false,
		EdgeNote:          "Edges are available only in GOAT JSON export.",
		ClassMapping:      classMapping,
		Images:            images,
		Warnings:          warnings,
	}
}

func buildClassMapping(labels []domain.LabelDefinition, format DatasetExportFormat) []datasetClassMapping {
	mapping := make([]datasetClassMapping, 0, len(labels))
	for _, label := range labels {
		if format == DatasetExportFormatYOLO && label.Category != domain.LabelCategoryObject {
			continue
		}
		classID := len(mapping)
		if format == DatasetExportFormatCOCO {
			classID++
		}
		mapping = append(mapping, datasetClassMapping{
			ClassID:  classID,
			LabelID:  label.ID,
			Name:     label.Name,
			Category: string(label.Category),
		})
	}
	return mapping
}

func buildClassIDByLabelID(mapping []datasetClassMapping) map[string]int {
	classIDs := make(map[string]int, len(mapping))
	for _, class := range mapping {
		classIDs[class.LabelID] = class.ClassID
	}
	return classIDs
}

type cocoDataset struct {
	Info        cocoInfo         `json:"info"`
	Licenses    []any            `json:"licenses"`
	Images      []cocoImage      `json:"images"`
	Annotations []cocoAnnotation `json:"annotations"`
	Categories  []cocoCategory   `json:"categories"`
}

type cocoInfo struct {
	Description string `json:"description"`
	Version     string `json:"version"`
}

type cocoImage struct {
	ID          int    `json:"id"`
	FileName    string `json:"file_name"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	GOATImageID string `json:"goat_image_id"`
}

type cocoAnnotation struct {
	ID               int         `json:"id"`
	ImageID          int         `json:"image_id"`
	CategoryID       int         `json:"category_id"`
	BBox             []float64   `json:"bbox"`
	Segmentation     [][]float64 `json:"segmentation"`
	Area             float64     `json:"area"`
	IsCrowd          int         `json:"iscrowd"`
	GOATAnnotationID string      `json:"goat_annotation_id"`
}

type cocoCategory struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Supercategory string `json:"supercategory"`
	GOATLabelID   string `json:"goat_label_id"`
}

func (u *DatasetExportUsecase) writeCOCOArchive(archive *zip.Writer, dataset exportDataset) error {
	mapping := buildClassMapping(dataset.labels, DatasetExportFormatCOCO)
	classIDs := buildClassIDByLabelID(mapping)
	coco := cocoDataset{
		Info:        cocoInfo{Description: dataset.project.Name, Version: "1.0"},
		Licenses:    make([]any, 0),
		Images:      make([]cocoImage, 0, len(dataset.images)),
		Annotations: make([]cocoAnnotation, 0),
		Categories:  make([]cocoCategory, 0, len(mapping)),
	}
	for _, class := range mapping {
		coco.Categories = append(coco.Categories, cocoCategory{
			ID:            class.ClassID,
			Name:          class.Name,
			Supercategory: class.Category,
			GOATLabelID:   class.LabelID,
		})
	}

	for imageIndex, datasetImage := range dataset.images {
		archivePath := archiveImagePath("images/default", datasetImage.image)
		if err := writeZIPFileFromPath(archive, archivePath, datasetImage.filePath); err != nil {
			return fmt.Errorf("write image %s: %w", datasetImage.image.ID, err)
		}
		imageID := imageIndex + 1
		coco.Images = append(coco.Images, cocoImage{
			ID:          imageID,
			FileName:    archivePath,
			Width:       datasetImage.image.OriginalWidth,
			Height:      datasetImage.image.OriginalHeight,
			GOATImageID: datasetImage.image.ID,
		})
		for _, annotation := range datasetImage.annotations {
			converted, err := toCOCOAnnotation(
				len(coco.Annotations)+1,
				imageID,
				classIDs[*annotation.LabelID],
				datasetImage.image,
				annotation,
			)
			if err != nil {
				return err
			}
			coco.Annotations = append(coco.Annotations, converted)
		}
	}

	if err := writeJSONToZIP(archive, "annotations/instances_default.json", coco); err != nil {
		return err
	}
	manifest := newDatasetManifest(
		dataset,
		DatasetExportFormatCOCO,
		mapping,
		make([]datasetExportWarning, 0),
	)
	return writeJSONToZIP(archive, "manifest.json", manifest)
}

func toCOCOAnnotation(
	annotationID int,
	imageID int,
	categoryID int,
	image domain.Image,
	annotation domain.Annotation,
) (cocoAnnotation, error) {
	converted := cocoAnnotation{
		ID:               annotationID,
		ImageID:          imageID,
		CategoryID:       categoryID,
		Segmentation:     make([][]float64, 0),
		IsCrowd:          0,
		GOATAnnotationID: annotation.ID,
	}
	switch annotation.Type {
	case domain.AnnotationTypeBBox:
		var coordinates domain.BBoxCoordinates
		if err := json.Unmarshal(annotation.Coordinates, &coordinates); err != nil {
			return cocoAnnotation{}, invalidDatasetError("decode annotation %s: %v", annotation.ID, err)
		}
		original := inverseDisplayBBox(coordinates, image)
		converted.BBox = pixelBBox(original, image)
		converted.Area = roundedFloat(converted.BBox[2] * converted.BBox[3])
	case domain.AnnotationTypePolygon:
		var coordinates domain.PolygonCoordinates
		if err := json.Unmarshal(annotation.Coordinates, &coordinates); err != nil {
			return cocoAnnotation{}, invalidDatasetError("decode annotation %s: %v", annotation.ID, err)
		}
		originalPoints := inverseDisplayPolygon(coordinates.Points, image)
		pixelPoints := make([]float64, 0, len(originalPoints)*2)
		for _, point := range originalPoints {
			pixelPoints = append(
				pixelPoints,
				roundedFloat(point.X*float64(image.OriginalWidth)),
				roundedFloat(point.Y*float64(image.OriginalHeight)),
			)
		}
		converted.Segmentation = [][]float64{pixelPoints}
		converted.BBox = polygonPixelBBox(pixelPoints)
		converted.Area = polygonArea(pixelPoints)
	default:
		return cocoAnnotation{}, invalidDatasetError("annotation %s has unsupported type %s", annotation.ID, annotation.Type)
	}
	return converted, nil
}

func pixelBBox(coordinates domain.BBoxCoordinates, image domain.Image) []float64 {
	return []float64{
		roundedFloat(coordinates.X * float64(image.OriginalWidth)),
		roundedFloat(coordinates.Y * float64(image.OriginalHeight)),
		roundedFloat(coordinates.Width * float64(image.OriginalWidth)),
		roundedFloat(coordinates.Height * float64(image.OriginalHeight)),
	}
}

func polygonPixelBBox(points []float64) []float64 {
	minimumX := math.Inf(1)
	maximumX := math.Inf(-1)
	minimumY := math.Inf(1)
	maximumY := math.Inf(-1)
	for pointIndex := 0; pointIndex < len(points); pointIndex += 2 {
		minimumX = math.Min(minimumX, points[pointIndex])
		maximumX = math.Max(maximumX, points[pointIndex])
		minimumY = math.Min(minimumY, points[pointIndex+1])
		maximumY = math.Max(maximumY, points[pointIndex+1])
	}
	return []float64{
		roundedFloat(minimumX),
		roundedFloat(minimumY),
		roundedFloat(maximumX - minimumX),
		roundedFloat(maximumY - minimumY),
	}
}

func polygonArea(points []float64) float64 {
	doubledArea := 0.0
	pointCount := len(points) / 2
	for pointIndex := 0; pointIndex < pointCount; pointIndex++ {
		nextPointIndex := (pointIndex + 1) % pointCount
		x := points[pointIndex*2]
		y := points[pointIndex*2+1]
		nextX := points[nextPointIndex*2]
		nextY := points[nextPointIndex*2+1]
		doubledArea += x*nextY - nextX*y
	}
	return roundedFloat(math.Abs(doubledArea) / 2)
}

func (u *DatasetExportUsecase) writeYOLOArchive(archive *zip.Writer, dataset exportDataset) error {
	mapping := buildClassMapping(dataset.labels, DatasetExportFormatYOLO)
	classIDs := buildClassIDByLabelID(mapping)
	warnings := make([]datasetExportWarning, 0)

	for _, datasetImage := range dataset.images {
		archivePath := archiveImagePath("images/train", datasetImage.image)
		if err := writeZIPFileFromPath(archive, archivePath, datasetImage.filePath); err != nil {
			return fmt.Errorf("write image %s: %w", datasetImage.image.ID, err)
		}

		var labels strings.Builder
		for _, annotation := range datasetImage.annotations {
			label := dataset.labelByID[*annotation.LabelID]
			if label.Category != domain.LabelCategoryObject {
				warnings = append(warnings, datasetExportWarning{
					ImageID:      datasetImage.image.ID,
					AnnotationID: annotation.ID,
					Reason:       "unsupported_label_category",
				})
				continue
			}
			if annotation.Type != domain.AnnotationTypeBBox {
				warnings = append(warnings, datasetExportWarning{
					ImageID:      datasetImage.image.ID,
					AnnotationID: annotation.ID,
					Reason:       "unsupported_annotation_type",
				})
				continue
			}

			var coordinates domain.BBoxCoordinates
			if err := json.Unmarshal(annotation.Coordinates, &coordinates); err != nil {
				return invalidDatasetError("decode annotation %s: %v", annotation.ID, err)
			}
			original := inverseDisplayBBox(coordinates, datasetImage.image)
			centerX := original.X + original.Width/2
			centerY := original.Y + original.Height/2
			line := []float64{centerX, centerY, original.Width, original.Height}
			labels.WriteString(strconv.Itoa(classIDs[*annotation.LabelID]))
			for _, value := range line {
				labels.WriteByte(' ')
				labels.WriteString(formatExportFloat(value))
			}
			labels.WriteByte('\n')
		}

		labelPath := path.Join("labels/train", datasetImage.image.ID+".txt")
		if err := writeZIPFile(archive, labelPath, []byte(labels.String())); err != nil {
			return err
		}
	}

	if err := writeZIPFile(archive, "data.yaml", []byte(yoloDataYAML(mapping))); err != nil {
		return err
	}
	if err := writeJSONToZIP(archive, "classes.json", mapping); err != nil {
		return err
	}
	manifest := newDatasetManifest(dataset, DatasetExportFormatYOLO, mapping, warnings)
	return writeJSONToZIP(archive, "manifest.json", manifest)
}

func yoloDataYAML(mapping []datasetClassMapping) string {
	var data strings.Builder
	data.WriteString("path: .\ntrain: images/train\n")
	if len(mapping) == 0 {
		data.WriteString("names: {}\n")
		return data.String()
	}
	data.WriteString("names:\n")
	for _, class := range mapping {
		encodedName, _ := json.Marshal(class.Name)
		fmt.Fprintf(&data, "  %d: %s\n", class.ClassID, encodedName)
	}
	return data.String()
}

func inverseDisplayBBox(coordinates domain.BBoxCoordinates, image domain.Image) domain.BBoxCoordinates {
	displayCorners := []domain.Point{
		{X: coordinates.X, Y: coordinates.Y},
		{X: coordinates.X + coordinates.Width, Y: coordinates.Y},
		{X: coordinates.X, Y: coordinates.Y + coordinates.Height},
		{X: coordinates.X + coordinates.Width, Y: coordinates.Y + coordinates.Height},
	}
	minimumX := math.Inf(1)
	maximumX := math.Inf(-1)
	minimumY := math.Inf(1)
	maximumY := math.Inf(-1)
	for _, displayPoint := range displayCorners {
		originalPoint := inverseDisplayPoint(displayPoint, image)
		minimumX = math.Min(minimumX, originalPoint.X)
		maximumX = math.Max(maximumX, originalPoint.X)
		minimumY = math.Min(minimumY, originalPoint.Y)
		maximumY = math.Max(maximumY, originalPoint.Y)
	}
	return domain.BBoxCoordinates{
		X:      roundedFloat(minimumX),
		Y:      roundedFloat(minimumY),
		Width:  roundedFloat(maximumX - minimumX),
		Height: roundedFloat(maximumY - minimumY),
	}
}

func inverseDisplayPolygon(points []domain.Point, image domain.Image) []domain.Point {
	originalPoints := make([]domain.Point, len(points))
	for pointIndex, point := range points {
		originalPoints[pointIndex] = inverseDisplayPoint(point, image)
	}
	return originalPoints
}

func inverseDisplayPoint(point domain.Point, image domain.Image) domain.Point {
	var original domain.Point
	switch image.Rotation {
	case domain.Rotation0:
		original = point
	case domain.Rotation90:
		original = domain.Point{X: point.Y, Y: 1 - point.X}
	case domain.Rotation180:
		original = domain.Point{X: 1 - point.X, Y: 1 - point.Y}
	case domain.Rotation270:
		original = domain.Point{X: 1 - point.Y, Y: point.X}
	}
	// Why: Konvaの表示行列はsource軸のflip後にrotationするため、逆変換はrotationから戻す。
	if image.FlipV {
		original.Y = 1 - original.Y
	}
	if image.FlipH {
		original.X = 1 - original.X
	}
	original.X = roundedFloat(original.X)
	original.Y = roundedFloat(original.Y)
	return original
}

func archiveImagePath(directory string, image domain.Image) string {
	return path.Join(directory, image.ID+filepath.Ext(image.Filename))
}

func writeJSONToZIP(archive *zip.Writer, name string, value any) error {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", name, err)
	}
	content = append(content, '\n')
	return writeZIPFile(archive, name, content)
}

func writeZIPFile(archive *zip.Writer, name string, content []byte) error {
	entry, err := archive.Create(name)
	if err != nil {
		return fmt.Errorf("create ZIP entry %s: %w", name, err)
	}
	if _, err := entry.Write(content); err != nil {
		return fmt.Errorf("write ZIP entry %s: %w", name, err)
	}
	return nil
}

func writeZIPFileFromPath(archive *zip.Writer, archivePath, sourcePath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()
	entry, err := archive.Create(archivePath)
	if err != nil {
		return err
	}
	if _, err := io.Copy(entry, source); err != nil {
		return err
	}
	return nil
}

func roundedFloat(value float64) float64 {
	const precision = 1_000_000_000_000
	result := math.Round(value*precision) / precision
	if result == 0 {
		return 0
	}
	return result
}

func formatExportFloat(value float64) string {
	return strconv.FormatFloat(roundedFloat(value), 'f', -1, 64)
}
