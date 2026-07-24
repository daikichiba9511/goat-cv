package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/daikichiba9511/goat-cv/backend/internal/usecase"
	"github.com/go-chi/chi/v5"
)

func TestImageGraphSaveReturnsExplicitClientMappings(t *testing.T) {
	router := newImageGraphTestRouter()
	body := `{"annotations":[` +
		`{"client_id":"client-b","type":"bbox","coordinates":{"x":0.5,"y":0,"width":0.5,"height":1}},` +
		`{"client_id":"client-a","type":"bbox","coordinates":{"x":0,"y":0,"width":0.5,"height":1}}` +
		`],"edges":[` +
		`{"client_id":"client-edge","source_annotation_client_id":"client-a","target_annotation_client_id":"client-b","type":"reading_order"}` +
		`]}`
	request := httptest.NewRequest(http.MethodPut, "/images/image-1/graph/", strings.NewReader(body))
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
	var responseBody savedImageGraphResponse
	if err := json.Unmarshal(response.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	annotationIDByClientID := make(map[string]string, len(responseBody.Annotations))
	for _, savedAnnotation := range responseBody.Annotations {
		annotationIDByClientID[savedAnnotation.ClientID] = savedAnnotation.Annotation.ID
	}
	if len(responseBody.Edges) != 1 ||
		responseBody.Edges[0].ClientID != "client-edge" ||
		responseBody.Edges[0].Edge.SourceAnnotationID != annotationIDByClientID["client-a"] ||
		responseBody.Edges[0].Edge.TargetAnnotationID != annotationIDByClientID["client-b"] {
		t.Fatalf("response = %+v, want edge endpoints resolved through client IDs", responseBody)
	}
}

func TestImageGraphSaveReturnsBadRequestForUnknownClientReference(t *testing.T) {
	router := newImageGraphTestRouter()
	body := `{"annotations":[` +
		`{"client_id":"client-a","type":"bbox","coordinates":{"x":0,"y":0,"width":1,"height":1}}` +
		`],"edges":[` +
		`{"client_id":"client-edge","source_annotation_client_id":"client-a","target_annotation_client_id":"missing","type":"reading_order"}` +
		`]}`
	request := httptest.NewRequest(http.MethodPut, "/images/image-1/graph/", strings.NewReader(body))
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusBadRequest, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "target_annotation_client_id") {
		t.Fatalf("body = %s, want invalid reference field", response.Body.String())
	}
}

func TestImageGraphSaveReturnsConflictWhenWorkflowIsLocked(t *testing.T) {
	imageGraphUsecase := usecase.NewImageGraphUsecase(
		imageGraphHandlerTestRepository{},
		imageGraphHandlerTestLabelRepository{},
		imageGraphHandlerTestImageRepository{status: domain.ImageStatusPending, escalated: true},
	)
	imageGraphHandler := NewImageGraphHandler(imageGraphUsecase)
	router := chi.NewRouter()
	router.Route("/images/{imageId}/graph", imageGraphHandler.ImageRoutes)
	request := httptest.NewRequest(
		http.MethodPut,
		"/images/image-1/graph/",
		strings.NewReader(`{"annotations":[],"edges":[]}`),
	)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusConflict, response.Body.String())
	}
}

func newImageGraphTestRouter() chi.Router {
	imageGraphUsecase := usecase.NewImageGraphUsecase(
		imageGraphHandlerTestRepository{},
		imageGraphHandlerTestLabelRepository{},
		imageGraphHandlerTestImageRepository{status: domain.ImageStatusPending},
	)
	imageGraphHandler := NewImageGraphHandler(imageGraphUsecase)
	router := chi.NewRouter()
	router.Route("/images/{imageId}/graph", imageGraphHandler.ImageRoutes)
	return router
}

type imageGraphHandlerTestRepository struct{}

func (imageGraphHandlerTestRepository) Replace(
	_ context.Context,
	_ string,
	annotations []domain.Annotation,
	edges []domain.Edge,
) ([]domain.Annotation, []domain.Edge, error) {
	persistedAnnotations := slices.Clone(annotations)
	persistedEdges := slices.Clone(edges)
	slices.Reverse(persistedAnnotations)
	slices.Reverse(persistedEdges)
	return persistedAnnotations, persistedEdges, nil
}

type imageGraphHandlerTestLabelRepository struct{}

func (imageGraphHandlerTestLabelRepository) Get(_ context.Context, id string) (domain.LabelDefinition, error) {
	return domain.LabelDefinition{ID: id, Category: domain.LabelCategoryObject}, nil
}

type imageGraphHandlerTestImageRepository struct {
	status    domain.ImageStatus
	escalated bool
}

func (repository imageGraphHandlerTestImageRepository) Get(_ context.Context, imageID string) (domain.Image, error) {
	return domain.Image{ID: imageID, Status: repository.status, Escalated: repository.escalated}, nil
}
