package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/daikichiba9511/goat-cv/backend/internal/usecase"
	"github.com/go-chi/chi/v5"
)

func TestProjectExportWritesRequestedDatasetArchive(t *testing.T) {
	exporter := &stubDatasetExporter{content: []byte("dataset archive")}
	handler := &ExportHandler{datasetExporter: exporter}
	router := chi.NewRouter()
	router.Get("/projects/{projectId}/export", handler.ProjectExport)

	request := httptest.NewRequest(http.MethodGet, "/projects/project-1/export?format=coco", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", response.Code, response.Body.String())
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/zip" {
		t.Fatalf("Content-Type = %q, want application/zip", contentType)
	}
	if disposition := response.Header().Get("Content-Disposition"); !strings.Contains(disposition, "goat-project-1-coco.zip") {
		t.Fatalf("Content-Disposition = %q, want archive filename", disposition)
	}
	if response.Body.String() != "dataset archive" {
		t.Fatalf("body = %q, want archive bytes", response.Body.String())
	}
	if exporter.projectID != "project-1" || exporter.format != usecase.DatasetExportFormatCOCO {
		t.Fatalf("export request = (%q, %q), want project-1 COCO", exporter.projectID, exporter.format)
	}
}

func TestExportHandlersRejectUnsupportedFormatAtHTTPBoundary(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		handle      func(*ExportHandler, http.ResponseWriter, *http.Request)
		exportError error
		wantStatus  int
	}{
		{
			name: "unknown project format",
			path: "/projects/project-1/export?format=pascal-voc",
			handle: func(handler *ExportHandler, writer http.ResponseWriter, request *http.Request) {
				handler.ProjectExport(writer, request)
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "image-level YOLO",
			path: "/images/image-1/export?format=yolo",
			handle: func(handler *ExportHandler, writer http.ResponseWriter, request *http.Request) {
				handler.ImageExport(writer, request)
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid persisted dataset",
			path: "/projects/project-1/export?format=yolo",
			handle: func(handler *ExportHandler, writer http.ResponseWriter, request *http.Request) {
				handler.ProjectExport(writer, request)
			},
			exportError: usecase.ErrInvalidDatasetExport,
			wantStatus:  http.StatusUnprocessableEntity,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			exporter := &stubDatasetExporter{err: test.exportError}
			handler := &ExportHandler{datasetExporter: exporter}
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			routeContext := chi.NewRouteContext()
			if strings.HasPrefix(test.path, "/projects/") {
				routeContext.URLParams.Add("projectId", "project-1")
			} else {
				routeContext.URLParams.Add("imageId", "image-1")
			}
			request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
			response := httptest.NewRecorder()

			test.handle(handler, response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d: %s", response.Code, test.wantStatus, response.Body.String())
			}
			if test.wantStatus == http.StatusBadRequest && exporter.calls != 0 {
				t.Fatalf("dataset exporter calls = %d, want 0", exporter.calls)
			}
		})
	}
}

func TestBuildExportImageIncludesGraphEdges(t *testing.T) {
	handler := &ExportHandler{
		annotationLister: stubAnnotationLister{annotations: []domain.Annotation{
			{
				ID: "annotation-a", ImageID: "image-1", Type: domain.AnnotationTypeBBox,
				Coordinates: domain.Coordinates(`{"x":0,"y":0,"width":0.5,"height":1}`),
			},
		}},
		edgeLister: stubEdgeLister{edges: []domain.Edge{
			{
				ID: "edge-1", ImageID: "image-1", SourceAnnotationID: "annotation-a",
				TargetAnnotationID: "annotation-b", Type: domain.EdgeTypeReadingOrder,
			},
		}},
	}

	exportedImage, err := handler.buildExportImage(
		context.Background(),
		domain.Image{ID: "image-1"},
		map[string]string{},
	)
	if err != nil {
		t.Fatalf("buildExportImage returned error: %v", err)
	}
	if len(exportedImage.Edges) != 1 {
		t.Fatalf("edges count = %d, want 1", len(exportedImage.Edges))
	}
	edge := exportedImage.Edges[0]
	if edge.ID != "edge-1" || edge.Source != "annotation-a" ||
		edge.Target != "annotation-b" || edge.Type != "reading_order" {
		t.Fatalf("edge = %+v, want complete GOAT JSON edge", edge)
	}
}

type stubDatasetExporter struct {
	content   []byte
	err       error
	calls     int
	projectID string
	format    usecase.DatasetExportFormat
}

func (s *stubDatasetExporter) WriteProjectArchive(
	_ context.Context,
	projectID string,
	format usecase.DatasetExportFormat,
	destination io.Writer,
) error {
	s.calls++
	s.projectID = projectID
	s.format = format
	if s.err != nil {
		return s.err
	}
	_, err := destination.Write(s.content)
	return err
}

type stubAnnotationLister struct {
	annotations []domain.Annotation
}

func (s stubAnnotationLister) ListByImage(context.Context, string) ([]domain.Annotation, error) {
	return s.annotations, nil
}

type stubEdgeLister struct {
	edges []domain.Edge
}

func (s stubEdgeLister) ListByImage(context.Context, string) ([]domain.Edge, error) {
	return s.edges, nil
}
