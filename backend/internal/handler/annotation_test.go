package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/daikichiba9511/goat-cv/backend/internal/usecase"
	"github.com/go-chi/chi/v5"
)

func TestAnnotationBulkReplaceReturnsBadRequestWithInvalidArrayPosition(t *testing.T) {
	annotationUsecase := usecase.NewAnnotationUsecase(annotationHandlerTestRepository{})
	annotationHandler := NewAnnotationHandler(annotationUsecase)
	router := chi.NewRouter()
	router.Route("/images/{imageId}/annotations", annotationHandler.ImageRoutes)

	body := `{"annotations":[` +
		`{"type":"bbox","coordinates":{"x":0,"y":0,"width":1,"height":1}},` +
		`{"type":"polygon","coordinates":{"points":[{"x":0,"y":0},{"x":1,"y":0}]}}` +
		`]}`
	request := httptest.NewRequest(http.MethodPut, "/images/image-1/annotations/", strings.NewReader(body))
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusBadRequest, response.Body.String())
	}
	var responseBody map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	errorMessage := responseBody["error"]
	if !strings.Contains(errorMessage, "annotations[1]") || !strings.Contains(errorMessage, "three distinct points") {
		t.Fatalf("error = %q, want array position and validation reason", errorMessage)
	}
}

type annotationHandlerTestRepository struct{}

func (annotationHandlerTestRepository) Create(_ context.Context, annotation domain.Annotation) (domain.Annotation, error) {
	return annotation, nil
}

func (annotationHandlerTestRepository) ListByImage(_ context.Context, _ string) ([]domain.Annotation, error) {
	return nil, nil
}

func (annotationHandlerTestRepository) Update(_ context.Context, annotation domain.Annotation) (domain.Annotation, error) {
	return annotation, nil
}

func (annotationHandlerTestRepository) Delete(_ context.Context, _ string) error {
	return nil
}

func (annotationHandlerTestRepository) BulkReplace(_ context.Context, _ string, annotations []domain.Annotation) ([]domain.Annotation, error) {
	return annotations, nil
}
