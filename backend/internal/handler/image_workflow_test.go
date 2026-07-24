package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/daikichiba9511/goat-cv/backend/internal/repository/sqlite"
	"github.com/daikichiba9511/goat-cv/backend/internal/sqlcgen"
	"github.com/daikichiba9511/goat-cv/backend/internal/usecase"
	"github.com/go-chi/chi/v5"
	_ "github.com/mattn/go-sqlite3"
)

func TestImageWorkflowTransitionHTTPContract(t *testing.T) {
	tests := []struct {
		name           string
		image          domain.Image
		event          string
		wantStatusCode int
		wantStatus     string
		wantEscalated  bool
		wantAllowed    []string
	}{
		{
			name:           "allowed event",
			image:          domain.Image{ID: "image-1", Status: domain.ImageStatusPending},
			event:          string(domain.ImageWorkflowEventAnnotationCompleted),
			wantStatusCode: http.StatusOK,
			wantStatus:     string(domain.ImageStatusAnnotated),
		},
		{
			name:           "unknown event",
			image:          domain.Image{ID: "image-1", Status: domain.ImageStatusPending},
			event:          "workflow_skipped",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "missing image",
			image:          domain.Image{},
			event:          string(domain.ImageWorkflowEventAnnotationCompleted),
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:           "disallowed event",
			image:          domain.Image{ID: "image-1", Status: domain.ImageStatusApproved},
			event:          string(domain.ImageWorkflowEventEscalationStarted),
			wantStatusCode: http.StatusConflict,
			wantStatus:     string(domain.ImageStatusApproved),
			wantAllowed:    []string{string(domain.ImageWorkflowEventApprovalReopened)},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newImageWorkflowHandlerFixture(t)
			if test.image.ID != "" {
				fixture.insertImage(t, test.image)
			}
			router := chi.NewRouter()
			router.Mount("/images", fixture.handler.ImageRoutes())
			request := httptest.NewRequest(
				http.MethodPost,
				"/images/image-1/workflow-transitions",
				strings.NewReader(`{"event":"`+test.event+`"}`),
			)
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			if response.Code != test.wantStatusCode {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatusCode, response.Body.String())
			}
			if test.wantStatus == "" {
				return
			}
			var responseBody struct {
				Status    string `json:"status"`
				Escalated bool   `json:"escalated"`
				Current   struct {
					Status    string `json:"status"`
					Escalated bool   `json:"escalated"`
				} `json:"current"`
				AllowedEvents []string `json:"allowed_events"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &responseBody); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if test.wantStatusCode == http.StatusConflict {
				if responseBody.Current.Status != test.wantStatus || responseBody.Current.Escalated != test.wantEscalated {
					t.Fatalf("current workflow = %+v, want (%q, %t)", responseBody.Current, test.wantStatus, test.wantEscalated)
				}
				if len(responseBody.AllowedEvents) != len(test.wantAllowed) || responseBody.AllowedEvents[0] != test.wantAllowed[0] {
					t.Fatalf("allowed events = %v, want %v", responseBody.AllowedEvents, test.wantAllowed)
				}
				return
			}
			if responseBody.Status != test.wantStatus || responseBody.Escalated != test.wantEscalated {
				t.Fatalf("workflow = (%q, %t), want (%q, %t)", responseBody.Status, responseBody.Escalated, test.wantStatus, test.wantEscalated)
			}
		})
	}
}

func TestImageListFiltersLifecycleAndEscalationTogether(t *testing.T) {
	fixture := newImageWorkflowHandlerFixture(t)
	for _, image := range []struct {
		id        string
		status    domain.ImageStatus
		escalated bool
	}{
		{id: "pending-clear", status: domain.ImageStatusPending},
		{id: "pending-escalated", status: domain.ImageStatusPending, escalated: true},
		{id: "rejected-clear", status: domain.ImageStatusRejected},
	} {
		fixture.insertImage(t, domain.Image{ID: image.id, Status: image.status, Escalated: image.escalated})
	}

	router := chi.NewRouter()
	router.Route("/projects/{projectId}/images", fixture.handler.ProjectRoutes)
	request := httptest.NewRequest(
		http.MethodGet,
		"/projects/project-1/images/?status=pending&escalated=false",
		nil,
	)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
	var responseBody struct {
		Items []imageResponse `json:"items"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(responseBody.Items) != 1 || responseBody.Items[0].ID != "pending-clear" {
		t.Fatalf("filtered Images = %+v, want pending-clear only", responseBody.Items)
	}
}

func TestImageTransformReturnsConflictWhenWorkflowIsLocked(t *testing.T) {
	fixture := newImageWorkflowHandlerFixture(t)
	fixture.insertImage(t, domain.Image{
		ID:     "image-1",
		Status: domain.ImageStatusRejected,
	})
	router := chi.NewRouter()
	router.Mount("/images", fixture.handler.ImageRoutes())
	request := httptest.NewRequest(
		http.MethodPatch,
		"/images/image-1",
		strings.NewReader(`{"rotation":90,"flip_h":false,"flip_v":false}`),
	)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusConflict, response.Body.String())
	}
}

type imageWorkflowHandlerFixture struct {
	database *sql.DB
	handler  *ImageHandler
}

func newImageWorkflowHandlerFixture(t testing.TB) imageWorkflowHandlerFixture {
	t.Helper()
	database, err := sql.Open("sqlite3", "file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared&_foreign_keys=on")
	if err != nil {
		t.Fatalf("open workflow handler database: %v", err)
	}
	database.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close workflow handler database: %v", err)
		}
	})
	for _, migrationPath := range []string{
		"../../db/migrations/001_init.sql",
		"../../db/migrations/004_image_workflow.sql",
	} {
		migration, err := os.ReadFile(migrationPath)
		if err != nil {
			t.Fatalf("read migration %s: %v", migrationPath, err)
		}
		if _, err := database.Exec(string(migration)); err != nil {
			t.Fatalf("apply migration %s: %v", migrationPath, err)
		}
	}
	if _, err := database.Exec(`INSERT INTO projects (id, name) VALUES ('project-1', 'Workflow')`); err != nil {
		t.Fatalf("insert Project: %v", err)
	}
	repository := sqlite.NewImageRepository(sqlcgen.New(database))
	return imageWorkflowHandlerFixture{
		database: database,
		handler:  NewImageHandler(usecase.NewImageUsecase(repository, t.TempDir())),
	}
}

func (fixture imageWorkflowHandlerFixture) insertImage(t testing.TB, image domain.Image) {
	t.Helper()
	if _, err := fixture.database.Exec(`
		INSERT INTO images (
			id, project_id, filename, original_width, original_height, width, height, status, escalated
		) VALUES (?, 'project-1', ?, 100, 200, 100, 200, ?, ?)
	`, image.ID, image.ID+".png", image.Status, image.Escalated); err != nil {
		t.Fatalf("insert Image %s: %v", image.ID, err)
	}
}
