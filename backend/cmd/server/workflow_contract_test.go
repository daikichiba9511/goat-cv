package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

type workflowContractImage struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Escalated bool   `json:"escalated"`
}

type workflowContractConflict struct {
	Current       workflowContractImage `json:"current"`
	AllowedEvents []string              `json:"allowed_events"`
}

type workflowContractFixture struct {
	handler http.Handler
}

func TestImageWorkflowHTTPContract(t *testing.T) {
	fixture := newWorkflowContractFixture(t)

	t.Run("advances a new Image through approval", func(t *testing.T) {
		workflowImage := fixture.createImage(t, "approval")
		for _, transition := range []struct {
			event      string
			wantStatus string
		}{
			{event: "annotation_completed", wantStatus: "annotated"},
			{event: "review_started", wantStatus: "in_review"},
			{event: "review_approved", wantStatus: "approved"},
		} {
			response := fixture.requestJSON(t, http.MethodPost, workflowPath(workflowImage.ID), map[string]string{
				"event": transition.event,
			})
			assertStatusCode(t, response, http.StatusOK)
			assertWorkflowState(t, decodeWorkflowImage(t, response), transition.wantStatus, false, transition.event)
		}

		persistedImage := fixture.getImage(t, workflowImage.ID)
		assertWorkflowState(t, persistedImage, "approved", false, "retrieved approved Image")
	})

	t.Run("allows rejected Graph revision before resubmission", func(t *testing.T) {
		workflowImage := fixture.createImage(t, "revision")
		for _, event := range []string{
			"annotation_completed",
			"review_started",
			"review_rejected",
		} {
			response := fixture.requestJSON(t, http.MethodPost, workflowPath(workflowImage.ID), map[string]string{
				"event": event,
			})
			assertStatusCode(t, response, http.StatusOK)
		}
		assertWorkflowState(t, fixture.getImage(t, workflowImage.ID), "rejected", false, "review rejection")

		graphResponse := fixture.requestJSON(
			t,
			http.MethodPut,
			"/api/v1/images/"+workflowImage.ID+"/graph/",
			map[string]any{"annotations": []any{}, "edges": []any{}},
		)
		assertStatusCode(t, graphResponse, http.StatusOK)

		resubmission := fixture.requestJSON(t, http.MethodPost, workflowPath(workflowImage.ID), map[string]string{
			"event": "annotation_completed",
		})
		assertStatusCode(t, resubmission, http.StatusOK)
		assertWorkflowState(t, decodeWorkflowImage(t, resubmission), "annotated", false, "revision resubmission")
	})

	t.Run("keeps lifecycle while escalation blocks edits but permits Comments", func(t *testing.T) {
		workflowImage := fixture.createImage(t, "escalation")
		escalationResponse := fixture.requestJSON(t, http.MethodPost, workflowPath(workflowImage.ID), map[string]string{
			"event": "escalation_started",
		})
		assertStatusCode(t, escalationResponse, http.StatusOK)
		assertWorkflowState(t, decodeWorkflowImage(t, escalationResponse), "pending", true, "escalation start")

		graphResponse := fixture.requestJSON(
			t,
			http.MethodPut,
			"/api/v1/images/"+workflowImage.ID+"/graph/",
			map[string]any{"annotations": []any{}, "edges": []any{}},
		)
		assertStatusCode(t, graphResponse, http.StatusConflict)
		var graphConflict workflowContractConflict
		decodeJSONResponse(t, graphResponse, &graphConflict)
		assertWorkflowState(t, graphConflict.Current, "pending", true, "escalated Graph conflict")

		transformResponse := fixture.requestJSON(
			t,
			http.MethodPatch,
			"/api/v1/images/"+workflowImage.ID,
			map[string]any{"rotation": 90, "flip_h": false, "flip_v": false},
		)
		assertStatusCode(t, transformResponse, http.StatusConflict)

		commentResponse := fixture.requestJSON(
			t,
			http.MethodPost,
			"/api/v1/images/"+workflowImage.ID+"/comments/",
			map[string]any{
				"annotation_id": nil,
				"author":        "reviewer",
				"body":          "Clarify this Image",
				"type":          "question",
			},
		)
		assertStatusCode(t, commentResponse, http.StatusCreated)

		resolutionResponse := fixture.requestJSON(t, http.MethodPost, workflowPath(workflowImage.ID), map[string]string{
			"event": "escalation_resolved",
		})
		assertStatusCode(t, resolutionResponse, http.StatusOK)
		assertWorkflowState(t, fixture.getImage(t, workflowImage.ID), "pending", false, "escalation resolution")
	})

	t.Run("distinguishes malformed events and state conflicts without mutation", func(t *testing.T) {
		workflowImage := fixture.createImage(t, "errors")

		unknownResponse := fixture.requestJSON(t, http.MethodPost, workflowPath(workflowImage.ID), map[string]string{
			"event": "workflow_skipped",
		})
		assertStatusCode(t, unknownResponse, http.StatusBadRequest)

		conflictResponse := fixture.requestJSON(t, http.MethodPost, workflowPath(workflowImage.ID), map[string]string{
			"event": "review_approved",
		})
		assertStatusCode(t, conflictResponse, http.StatusConflict)
		var conflict workflowContractConflict
		decodeJSONResponse(t, conflictResponse, &conflict)
		assertWorkflowState(t, conflict.Current, "pending", false, "conflict current state")
		if len(conflict.AllowedEvents) != 2 ||
			conflict.AllowedEvents[0] != "annotation_completed" ||
			conflict.AllowedEvents[1] != "escalation_started" {
			t.Fatalf("allowed events = %v, want annotation_completed and escalation_started", conflict.AllowedEvents)
		}
		assertWorkflowState(t, fixture.getImage(t, workflowImage.ID), "pending", false, "state after rejected events")

		missingResponse := fixture.requestJSON(t, http.MethodPost, workflowPath("missing-image"), map[string]string{
			"event": "annotation_completed",
		})
		assertStatusCode(t, missingResponse, http.StatusNotFound)
	})

	t.Run("combines lifecycle and escalation Image filters", func(t *testing.T) {
		projectID := fixture.createProject(t, "filters")
		pendingClear := fixture.uploadImage(t, projectID, "pending-clear")
		pendingEscalated := fixture.uploadImage(t, projectID, "pending-escalated")
		annotatedClear := fixture.uploadImage(t, projectID, "annotated-clear")

		assertStatusCode(t, fixture.requestJSON(
			t,
			http.MethodPost,
			workflowPath(pendingEscalated.ID),
			map[string]string{"event": "escalation_started"},
		), http.StatusOK)
		assertStatusCode(t, fixture.requestJSON(
			t,
			http.MethodPost,
			workflowPath(annotatedClear.ID),
			map[string]string{"event": "annotation_completed"},
		), http.StatusOK)

		filterResponse := fixture.requestJSON(
			t,
			http.MethodGet,
			"/api/v1/projects/"+projectID+"/images/?status=pending&escalated=false",
			nil,
		)
		assertStatusCode(t, filterResponse, http.StatusOK)
		var imageList struct {
			Items []workflowContractImage `json:"items"`
		}
		decodeJSONResponse(t, filterResponse, &imageList)
		if len(imageList.Items) != 1 || imageList.Items[0].ID != pendingClear.ID {
			t.Fatalf("filtered Images = %+v, want only %s", imageList.Items, pendingClear.ID)
		}
	})
}

func newWorkflowContractFixture(t testing.TB) workflowContractFixture {
	t.Helper()
	databasePath := filepath.Join(t.TempDir(), "goat.db")
	database, err := sql.Open("sqlite3", databasePath+"?_foreign_keys=on")
	if err != nil {
		t.Fatalf("open workflow contract database: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close workflow contract database: %v", err)
		}
	})
	if err := runMigrationsFromDir(database, "../../db/migrations"); err != nil {
		t.Fatalf("run workflow contract migrations: %v", err)
	}
	return workflowContractFixture{
		handler: buildRouter(database, t.TempDir()),
	}
}

func (fixture workflowContractFixture) createImage(t testing.TB, name string) workflowContractImage {
	t.Helper()
	return fixture.uploadImage(t, fixture.createProject(t, name), name)
}

func (fixture workflowContractFixture) createProject(t testing.TB, name string) string {
	t.Helper()
	projectResponse := fixture.requestJSON(t, http.MethodPost, "/api/v1/projects/", map[string]string{
		"name": name,
	})
	assertStatusCode(t, projectResponse, http.StatusCreated)
	var project struct {
		ID string `json:"id"`
	}
	decodeJSONResponse(t, projectResponse, &project)
	return project.ID
}

func (fixture workflowContractFixture) uploadImage(
	t testing.TB,
	projectID string,
	name string,
) workflowContractImage {
	t.Helper()
	var requestBody bytes.Buffer
	multipartWriter := multipart.NewWriter(&requestBody)
	filePart, err := multipartWriter.CreateFormFile("file", name+".png")
	if err != nil {
		t.Fatalf("create upload form: %v", err)
	}
	if err := png.Encode(filePart, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatalf("encode upload Image: %v", err)
	}
	if err := multipartWriter.Close(); err != nil {
		t.Fatalf("close upload form: %v", err)
	}
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/projects/"+projectID+"/images/",
		&requestBody,
	)
	request.Header.Set("Content-Type", multipartWriter.FormDataContentType())
	response := httptest.NewRecorder()
	fixture.handler.ServeHTTP(response, request)
	assertStatusCode(t, response, http.StatusCreated)

	var createdImage workflowContractImage
	decodeJSONResponse(t, response, &createdImage)
	return createdImage
}

func (fixture workflowContractFixture) getImage(t testing.TB, imageID string) workflowContractImage {
	t.Helper()
	response := fixture.requestJSON(t, http.MethodGet, "/api/v1/images/"+imageID, nil)
	assertStatusCode(t, response, http.StatusOK)
	return decodeWorkflowImage(t, response)
}

func (fixture workflowContractFixture) requestJSON(
	t testing.TB,
	method string,
	path string,
	body any,
) *httptest.ResponseRecorder {
	t.Helper()
	var requestBody bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&requestBody).Encode(body); err != nil {
			t.Fatalf("encode %s %s request: %v", method, path, err)
		}
	}
	request := httptest.NewRequest(method, path, &requestBody)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	fixture.handler.ServeHTTP(response, request)
	return response
}

func workflowPath(imageID string) string {
	return "/api/v1/images/" + imageID + "/workflow-transitions"
}

func assertStatusCode(t testing.TB, response *httptest.ResponseRecorder, wantStatusCode int) {
	t.Helper()
	if response.Code != wantStatusCode {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, wantStatusCode, response.Body.String())
	}
}

func assertWorkflowState(
	t testing.TB,
	workflowImage workflowContractImage,
	wantStatus string,
	wantEscalated bool,
	operation string,
) {
	t.Helper()
	if workflowImage.Status != wantStatus || workflowImage.Escalated != wantEscalated {
		t.Fatalf(
			"%s workflow = (%q, %t), want (%q, %t)",
			operation,
			workflowImage.Status,
			workflowImage.Escalated,
			wantStatus,
			wantEscalated,
		)
	}
}

func decodeWorkflowImage(t testing.TB, response *httptest.ResponseRecorder) workflowContractImage {
	t.Helper()
	var workflowImage workflowContractImage
	decodeJSONResponse(t, response, &workflowImage)
	return workflowImage
}

func decodeJSONResponse(t testing.TB, response *httptest.ResponseRecorder, destination any) {
	t.Helper()
	if err := json.Unmarshal(response.Body.Bytes(), destination); err != nil {
		t.Fatalf("decode response body %q: %v", response.Body.String(), err)
	}
}
