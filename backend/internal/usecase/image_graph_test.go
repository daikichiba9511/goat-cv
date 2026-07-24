package usecase_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/daikichiba9511/goat-cv/backend/internal/repository/sqlite"
	"github.com/daikichiba9511/goat-cv/backend/internal/sqlcgen"
	"github.com/daikichiba9511/goat-cv/backend/internal/usecase"
)

func TestImageGraphUsecaseSaveResolvesClientReferencesWithoutResponseOrder(t *testing.T) {
	fixture := newImageGraphFixture(t)

	savedGraph, err := fixture.usecase.Save(fixture.ctx, testImageID, usecase.ImageGraphInput{
		Annotations: []usecase.ImageGraphAnnotationInput{
			{
				ClientID:    "client-second",
				Type:        domain.AnnotationTypeBBox,
				Coordinates: domain.Coordinates(`{"x":0.5,"y":0,"width":0.5,"height":1}`),
			},
			{
				ClientID:    "client-first",
				Type:        domain.AnnotationTypeBBox,
				Coordinates: domain.Coordinates(`{"x":0,"y":0,"width":0.5,"height":1}`),
			},
		},
		Edges: []usecase.ImageGraphEdgeInput{
			{
				ClientID:                 "client-edge",
				SourceAnnotationClientID: "client-first",
				TargetAnnotationClientID: "client-second",
				Type:                     domain.EdgeTypeReadingOrder,
			},
		},
	})
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	annotationIDByClientID := make(map[string]string, len(savedGraph.Annotations))
	for _, savedAnnotation := range savedGraph.Annotations {
		annotationIDByClientID[savedAnnotation.ClientID] = savedAnnotation.Annotation.ID
	}
	if annotationIDByClientID["client-first"] == "" || annotationIDByClientID["client-second"] == "" {
		t.Fatalf("saved annotations = %+v, want IDs for both client IDs", savedGraph.Annotations)
	}
	if len(savedGraph.Edges) != 1 {
		t.Fatalf("saved edges count = %d, want 1", len(savedGraph.Edges))
	}
	savedEdge := savedGraph.Edges[0]
	if savedEdge.ClientID != "client-edge" ||
		savedEdge.Edge.SourceAnnotationID != annotationIDByClientID["client-first"] ||
		savedEdge.Edge.TargetAnnotationID != annotationIDByClientID["client-second"] {
		t.Fatalf("saved edge = %+v, want endpoints resolved by client ID", savedEdge)
	}
}

func TestImageGraphUsecaseSaveRejectsInvalidGraphWithoutChangingExistingGraph(t *testing.T) {
	tests := []struct {
		name      string
		input     usecase.ImageGraphInput
		wantError error
	}{
		{
			name: "invalid annotation coordinates",
			input: usecase.ImageGraphInput{
				Annotations: []usecase.ImageGraphAnnotationInput{
					{
						ClientID:    "client-a",
						Type:        domain.AnnotationTypeBBox,
						Coordinates: domain.Coordinates(`{"x":0,"y":0,"width":0,"height":1}`),
					},
				},
			},
			wantError: usecase.ErrInvalidAnnotationCoordinates,
		},
		{
			name: "unknown annotation client reference",
			input: usecase.ImageGraphInput{
				Annotations: []usecase.ImageGraphAnnotationInput{
					validGraphAnnotationInput("client-a", 0),
				},
				Edges: []usecase.ImageGraphEdgeInput{
					{
						ClientID:                 "client-edge",
						SourceAnnotationClientID: "client-a",
						TargetAnnotationClientID: "missing",
						Type:                     domain.EdgeTypeReadingOrder,
					},
				},
			},
			wantError: usecase.ErrInvalidImageGraph,
		},
		{
			name: "invalid edge relation",
			input: usecase.ImageGraphInput{
				Annotations: []usecase.ImageGraphAnnotationInput{
					validGraphAnnotationInput("client-a", 0),
				},
				Edges: []usecase.ImageGraphEdgeInput{
					{
						ClientID:                 "client-edge",
						SourceAnnotationClientID: "client-a",
						TargetAnnotationClientID: "client-a",
						Type:                     domain.EdgeTypeReadingOrder,
					},
				},
			},
			wantError: usecase.ErrSelfEdge,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newImageGraphFixture(t)
			originalEdge, err := fixture.edgeUsecase.Create(
				fixture.ctx,
				testImageID,
				"ann-a",
				"ann-b",
				domain.EdgeTypeReadingOrder,
			)
			if err != nil {
				t.Fatalf("Create original edge: %v", err)
			}

			_, err = fixture.usecase.Save(fixture.ctx, testImageID, test.input)
			if !errors.Is(err, test.wantError) {
				t.Fatalf("Save error = %v, want %v", err, test.wantError)
			}

			annotations, err := fixture.annotationRepository.ListByImage(fixture.ctx, testImageID)
			if err != nil {
				t.Fatalf("List annotations after rejected save: %v", err)
			}
			edges, err := fixture.edgeRepository.ListByImage(fixture.ctx, testImageID)
			if err != nil {
				t.Fatalf("List edges after rejected save: %v", err)
			}
			if len(annotations) != 9 || len(edges) != 1 || edges[0].ID != originalEdge.ID {
				t.Fatalf("graph after rejected save has %d annotations and edges %+v, want original graph", len(annotations), edges)
			}
		})
	}
}

func TestImageGraphUsecaseSaveRollsBackAnnotationsAndEdgesWhenEdgeInsertFails(t *testing.T) {
	fixture := newImageGraphFixture(t)
	originalEdge, err := fixture.edgeUsecase.Create(
		fixture.ctx,
		testImageID,
		"ann-a",
		"ann-b",
		domain.EdgeTypeReadingOrder,
	)
	if err != nil {
		t.Fatalf("Create original edge: %v", err)
	}

	execSQL(
		t,
		fixture.ctx,
		fixture.db,
		`INSERT INTO annotations (id, image_id, type, coordinates, label_id) VALUES (?, ?, ?, ?, ?)`,
		"ann-other-image-2",
		otherTestImageID,
		string(domain.AnnotationTypeBBox),
		`{"x":0.2,"y":0,"width":0.1,"height":0.1}`,
		"label-object",
	)
	execSQL(
		t,
		fixture.ctx,
		fixture.db,
		`INSERT INTO edges (id, image_id, source_annotation_id, target_annotation_id, type) VALUES (?, ?, ?, ?, ?)`,
		"edge-other-image",
		otherTestImageID,
		"ann-other-image",
		"ann-other-image-2",
		string(domain.EdgeTypeReadingOrder),
	)

	_, err = fixture.usecase.Save(fixture.ctx, testImageID, usecase.ImageGraphInput{
		Annotations: []usecase.ImageGraphAnnotationInput{
			validGraphAnnotationInput("client-a", 0),
			validGraphAnnotationInput("client-b", 0.5),
		},
		Edges: []usecase.ImageGraphEdgeInput{
			{
				ClientID:                 "client-edge",
				ID:                       "edge-other-image",
				SourceAnnotationClientID: "client-a",
				TargetAnnotationClientID: "client-b",
				Type:                     domain.EdgeTypeReadingOrder,
			},
		},
	})
	if err == nil {
		t.Fatal("Save returned nil error, want edge ID constraint failure")
	}

	annotations, err := fixture.annotationRepository.ListByImage(fixture.ctx, testImageID)
	if err != nil {
		t.Fatalf("List annotations after failed transaction: %v", err)
	}
	edges, err := fixture.edgeRepository.ListByImage(fixture.ctx, testImageID)
	if err != nil {
		t.Fatalf("List edges after failed transaction: %v", err)
	}
	if len(annotations) != 9 || len(edges) != 1 || edges[0].ID != originalEdge.ID {
		t.Fatalf("graph after failed transaction has %d annotations and edges %+v, want original graph", len(annotations), edges)
	}
}

func TestImageGraphUsecaseSaveRejectsLockedWorkflowWithoutChangingGraph(t *testing.T) {
	fixture := newImageGraphFixture(t)
	execSQL(
		t,
		fixture.ctx,
		fixture.db,
		`UPDATE images SET status = ?, escalated = 1 WHERE id = ?`,
		domain.ImageStatusPending,
		testImageID,
	)

	_, err := fixture.usecase.Save(fixture.ctx, testImageID, usecase.ImageGraphInput{
		Annotations: []usecase.ImageGraphAnnotationInput{validGraphAnnotationInput("replacement", 0)},
	})
	var conflictError *usecase.ImageWorkflowOperationConflictError
	if !errors.As(err, &conflictError) {
		t.Fatalf("Save error = %v, want ImageWorkflowOperationConflictError", err)
	}
	if conflictError.Operation != usecase.ImageWorkflowOperationGraphEdit {
		t.Fatalf("conflict operation = %q, want graph edit", conflictError.Operation)
	}

	annotations, listError := fixture.annotationRepository.ListByImage(fixture.ctx, testImageID)
	if listError != nil {
		t.Fatalf("List annotations after rejected Save: %v", listError)
	}
	if len(annotations) != 9 {
		t.Fatalf("annotations after rejected Save = %d, want original 9", len(annotations))
	}
}

func TestImageGraphUsecaseSaveAllowsRejectedRevision(t *testing.T) {
	fixture := newImageGraphFixture(t)
	execSQL(
		t,
		fixture.ctx,
		fixture.db,
		`UPDATE images SET status = ? WHERE id = ?`,
		domain.ImageStatusRejected,
		testImageID,
	)

	_, err := fixture.usecase.Save(fixture.ctx, testImageID, usecase.ImageGraphInput{
		Annotations: []usecase.ImageGraphAnnotationInput{validGraphAnnotationInput("revision", 0)},
	})
	if err != nil {
		t.Fatalf("Save rejected revision returned error: %v", err)
	}
}

type imageGraphFixture struct {
	ctx                  context.Context
	db                   *sql.DB
	usecase              *usecase.ImageGraphUsecase
	edgeUsecase          *usecase.EdgeUsecase
	annotationRepository *sqlite.AnnotationRepository
	edgeRepository       *sqlite.EdgeRepository
}

func newImageGraphFixture(t testing.TB) imageGraphFixture {
	t.Helper()

	edgeFixture := newEdgeFixture(t)
	queries := sqlcgen.New(edgeFixture.db)
	annotationRepository := sqlite.NewAnnotationRepository(edgeFixture.db, queries)
	edgeRepository := sqlite.NewEdgeRepository(edgeFixture.db, queries)
	labelRepository := sqlite.NewLabelRepository(queries)
	graphRepository := sqlite.NewImageGraphRepository(edgeFixture.db, queries)
	imageRepository := sqlite.NewImageRepository(queries)

	return imageGraphFixture{
		ctx: edgeFixture.ctx,
		db:  edgeFixture.db,
		usecase: usecase.NewImageGraphUsecase(
			reorderingImageGraphRepository{repository: graphRepository},
			labelRepository,
			imageRepository,
		),
		edgeUsecase:          edgeFixture.usecase,
		annotationRepository: annotationRepository,
		edgeRepository:       edgeRepository,
	}
}

type reorderingImageGraphRepository struct {
	repository *sqlite.ImageGraphRepository
}

func (r reorderingImageGraphRepository) Replace(
	ctx context.Context,
	imageID string,
	annotations []domain.Annotation,
	edges []domain.Edge,
) ([]domain.Annotation, []domain.Edge, error) {
	persistedAnnotations, persistedEdges, err := r.repository.Replace(ctx, imageID, annotations, edges)
	if err != nil {
		return nil, nil, err
	}
	slices.Reverse(persistedAnnotations)
	slices.Reverse(persistedEdges)
	return persistedAnnotations, persistedEdges, nil
}

func validGraphAnnotationInput(clientID string, x float64) usecase.ImageGraphAnnotationInput {
	return usecase.ImageGraphAnnotationInput{
		ClientID:    clientID,
		Type:        domain.AnnotationTypeBBox,
		Coordinates: domain.Coordinates(fmt.Sprintf(`{"x":%g,"y":0,"width":0.5,"height":1}`, x)),
	}
}
