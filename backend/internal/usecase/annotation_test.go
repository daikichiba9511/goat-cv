package usecase_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/daikichiba9511/goat-cv/backend/internal/repository/sqlite"
	"github.com/daikichiba9511/goat-cv/backend/internal/sqlcgen"
	"github.com/daikichiba9511/goat-cv/backend/internal/usecase"
	_ "github.com/mattn/go-sqlite3"
)

const annotationTestImageID = "annotation-image-1"

func TestAnnotationUsecaseCreateAcceptsValidCoordinateSchemas(t *testing.T) {
	fixture := newAnnotationFixture(t)
	tests := []struct {
		name        string
		annType     domain.AnnotationType
		coordinates domain.Coordinates
	}{
		{
			name:        "bounding box on normalized boundaries",
			annType:     domain.AnnotationTypeBBox,
			coordinates: domain.Coordinates(`{"x":0,"y":0,"width":1,"height":1}`),
		},
		{
			name:        "polygon with three boundary vertices",
			annType:     domain.AnnotationTypePolygon,
			coordinates: domain.Coordinates(`{"points":[{"x":0,"y":0},{"x":1,"y":0},{"x":0,"y":1}]}`),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			annotation, err := fixture.usecase.Create(fixture.ctx, annotationTestImageID, test.annType, test.coordinates, nil)
			if err != nil {
				t.Fatalf("Create returned error: %v", err)
			}
			if annotation.Type != test.annType {
				t.Fatalf("Create type = %q, want %q", annotation.Type, test.annType)
			}
		})
	}
}

func TestAnnotationUsecaseCreateRejectsInvalidCoordinateSchemas(t *testing.T) {
	tests := []struct {
		name        string
		annType     domain.AnnotationType
		coordinates domain.Coordinates
		wantErr     error
	}{
		{
			name:        "unsupported annotation type",
			annType:     domain.AnnotationType("circle"),
			coordinates: domain.Coordinates(`{"x":0,"y":0,"width":1,"height":1}`),
			wantErr:     usecase.ErrInvalidAnnotationType,
		},
		{
			name:        "bounding box with polygon schema",
			annType:     domain.AnnotationTypeBBox,
			coordinates: domain.Coordinates(`{"points":[{"x":0,"y":0},{"x":1,"y":0},{"x":0,"y":1}]}`),
			wantErr:     usecase.ErrInvalidAnnotationCoordinates,
		},
		{
			name:        "bounding box with zero width",
			annType:     domain.AnnotationTypeBBox,
			coordinates: domain.Coordinates(`{"x":0,"y":0,"width":0,"height":1}`),
			wantErr:     usecase.ErrInvalidAnnotationCoordinates,
		},
		{
			name:        "bounding box extending outside normalized space",
			annType:     domain.AnnotationTypeBBox,
			coordinates: domain.Coordinates(`{"x":0.8,"y":0,"width":0.3,"height":1}`),
			wantErr:     usecase.ErrInvalidAnnotationCoordinates,
		},
		{
			name:        "bounding box with non-finite coordinate",
			annType:     domain.AnnotationTypeBBox,
			coordinates: domain.Coordinates(`{"x":1e400,"y":0,"width":1,"height":1}`),
			wantErr:     usecase.ErrInvalidAnnotationCoordinates,
		},
		{
			name:        "polygon with fewer than three distinct points",
			annType:     domain.AnnotationTypePolygon,
			coordinates: domain.Coordinates(`{"points":[{"x":0,"y":0},{"x":1,"y":0},{"x":0,"y":0}]}`),
			wantErr:     usecase.ErrInvalidAnnotationCoordinates,
		},
		{
			name:        "polygon point missing y coordinate",
			annType:     domain.AnnotationTypePolygon,
			coordinates: domain.Coordinates(`{"points":[{"x":0,"y":0},{"x":1},{"x":0,"y":1}]}`),
			wantErr:     usecase.ErrInvalidAnnotationCoordinates,
		},
		{
			name:        "polygon point outside normalized space",
			annType:     domain.AnnotationTypePolygon,
			coordinates: domain.Coordinates(`{"points":[{"x":0,"y":0},{"x":1.1,"y":0},{"x":0,"y":1}]}`),
			wantErr:     usecase.ErrInvalidAnnotationCoordinates,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newAnnotationFixture(t)

			_, err := fixture.usecase.Create(fixture.ctx, annotationTestImageID, test.annType, test.coordinates, nil)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Create error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestAnnotationUsecaseUpdateRejectsInvalidCoordinatesWithoutChangingAnnotation(t *testing.T) {
	fixture := newAnnotationFixture(t)
	originalCoordinates := domain.Coordinates(`{"x":0.1,"y":0.2,"width":0.3,"height":0.4}`)
	original, err := fixture.usecase.Create(fixture.ctx, annotationTestImageID, domain.AnnotationTypeBBox, originalCoordinates, nil)
	if err != nil {
		t.Fatalf("Create original annotation: %v", err)
	}

	_, err = fixture.usecase.Update(
		fixture.ctx,
		original.ID,
		domain.AnnotationTypeBBox,
		domain.Coordinates(`{"x":0.1,"y":0.2,"width":0,"height":0.4}`),
		nil,
	)
	if !errors.Is(err, usecase.ErrInvalidAnnotationCoordinates) {
		t.Fatalf("Update error = %v, want %v", err, usecase.ErrInvalidAnnotationCoordinates)
	}

	persisted, err := fixture.repository.Get(fixture.ctx, original.ID)
	if err != nil {
		t.Fatalf("Get annotation after rejected update: %v", err)
	}
	if string(persisted.Coordinates) != string(originalCoordinates) {
		t.Fatalf("coordinates after rejected update = %s, want %s", persisted.Coordinates, originalCoordinates)
	}
}

func TestAnnotationUsecaseBulkReplaceRejectsInvalidSetWithoutChangingAnnotations(t *testing.T) {
	fixture := newAnnotationFixture(t)
	original, err := fixture.usecase.Create(
		fixture.ctx,
		annotationTestImageID,
		domain.AnnotationTypeBBox,
		domain.Coordinates(`{"x":0.1,"y":0.2,"width":0.3,"height":0.4}`),
		nil,
	)
	if err != nil {
		t.Fatalf("Create original annotation: %v", err)
	}

	_, err = fixture.usecase.BulkReplace(fixture.ctx, annotationTestImageID, []domain.Annotation{
		{
			Type:        domain.AnnotationTypeBBox,
			Coordinates: domain.Coordinates(`{"x":0,"y":0,"width":0.5,"height":0.5}`),
		},
		{
			Type:        domain.AnnotationTypePolygon,
			Coordinates: domain.Coordinates(`{"points":[{"x":0,"y":0},{"x":1,"y":0}]}`),
		},
	})
	if !errors.Is(err, usecase.ErrInvalidAnnotationCoordinates) {
		t.Fatalf("BulkReplace error = %v, want %v", err, usecase.ErrInvalidAnnotationCoordinates)
	}
	if !strings.Contains(err.Error(), "annotations[1]") {
		t.Fatalf("BulkReplace error = %q, want invalid array position", err)
	}

	annotations, err := fixture.usecase.ListByImage(fixture.ctx, annotationTestImageID)
	if err != nil {
		t.Fatalf("ListByImage after rejected replace: %v", err)
	}
	if len(annotations) != 1 || annotations[0].ID != original.ID {
		t.Fatalf("annotations after rejected replace = %+v, want original annotation", annotations)
	}
}

func TestAnnotationUsecaseBulkReplaceRejectsDuplicatePersistentIDs(t *testing.T) {
	fixture := newAnnotationFixture(t)
	coordinates := domain.Coordinates(`{"x":0,"y":0,"width":1,"height":1}`)

	_, err := fixture.usecase.BulkReplace(fixture.ctx, annotationTestImageID, []domain.Annotation{
		{ID: "duplicate-id", Type: domain.AnnotationTypeBBox, Coordinates: coordinates},
		{ID: "duplicate-id", Type: domain.AnnotationTypeBBox, Coordinates: coordinates},
	})
	if !errors.Is(err, usecase.ErrDuplicateAnnotationID) {
		t.Fatalf("BulkReplace error = %v, want %v", err, usecase.ErrDuplicateAnnotationID)
	}
	if annotations, listErr := fixture.usecase.ListByImage(fixture.ctx, annotationTestImageID); listErr != nil {
		t.Fatalf("ListByImage after rejected replace: %v", listErr)
	} else if len(annotations) != 0 {
		t.Fatalf("annotations after rejected replace = %+v, want empty list", annotations)
	}
}

type annotationFixture struct {
	ctx        context.Context
	repository *sqlite.AnnotationRepository
	usecase    *usecase.AnnotationUsecase
}

func newAnnotationFixture(t testing.TB) annotationFixture {
	t.Helper()

	databaseName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := sql.Open("sqlite3", "file:"+databaseName+"?mode=memory&cache=shared&_foreign_keys=on")
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("close test database: %v", err)
		}
	})

	migration, err := os.ReadFile("../../db/migrations/001_init.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if _, err := db.Exec(string(migration)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}

	ctx := context.Background()
	execAnnotationSQL(t, ctx, db, `INSERT INTO projects (id, name) VALUES (?, ?)`, "annotation-project-1", "Annotation Project")
	execAnnotationSQL(
		t,
		ctx,
		db,
		`INSERT INTO images (id, project_id, filename, original_width, original_height, width, height) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		annotationTestImageID,
		"annotation-project-1",
		"annotation-image.png",
		100,
		100,
		100,
		100,
	)

	queries := sqlcgen.New(db)
	repository := sqlite.NewAnnotationRepository(db, queries)
	return annotationFixture{
		ctx:        ctx,
		repository: repository,
		usecase:    usecase.NewAnnotationUsecase(repository),
	}
}

func execAnnotationSQL(t testing.TB, ctx context.Context, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.ExecContext(ctx, query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}
