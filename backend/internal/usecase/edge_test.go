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

const (
	testImageID      = "image-1"
	otherTestImageID = "image-2"
)

func TestEdgeUsecaseCreateAcceptsValidReadingOrderEdge(t *testing.T) {
	fixture := newEdgeFixture(t)

	edge, err := fixture.usecase.Create(fixture.ctx, testImageID, "ann-a", "ann-b", domain.EdgeTypeReadingOrder)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if edge.ID == "" {
		t.Fatal("Create returned edge without ID")
	}
	if edge.ImageID != testImageID ||
		edge.SourceAnnotationID != "ann-a" ||
		edge.TargetAnnotationID != "ann-b" ||
		edge.Type != domain.EdgeTypeReadingOrder {
		t.Fatalf("Create returned edge = %+v, want image/source/target/type preserved", edge)
	}
}

func TestEdgeUsecaseCreateRejectsInvalidAnnotationReferences(t *testing.T) {
	tests := []struct {
		name               string
		sourceAnnotationID string
		targetAnnotationID string
		wantErr            error
	}{
		{
			name:               "missing source annotation",
			sourceAnnotationID: "missing",
			targetAnnotationID: "ann-b",
			wantErr:            usecase.ErrEdgeAnnotationNotFound,
		},
		{
			name:               "self edge",
			sourceAnnotationID: "ann-a",
			targetAnnotationID: "ann-a",
			wantErr:            usecase.ErrSelfEdge,
		},
		{
			name:               "cross-image target annotation",
			sourceAnnotationID: "ann-a",
			targetAnnotationID: "ann-other-image",
			wantErr:            usecase.ErrEdgeImageMismatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newEdgeFixture(t)

			_, err := fixture.usecase.Create(fixture.ctx, testImageID, tt.sourceAnnotationID, tt.targetAnnotationID, domain.EdgeTypeReadingOrder)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Create error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestEdgeUsecaseCreateRejectsDuplicateAndReadingOrderCycle(t *testing.T) {
	fixture := newEdgeFixture(t)
	if _, err := fixture.usecase.Create(fixture.ctx, testImageID, "ann-a", "ann-b", domain.EdgeTypeReadingOrder); err != nil {
		t.Fatalf("Create initial edge returned error: %v", err)
	}

	_, err := fixture.usecase.Create(fixture.ctx, testImageID, "ann-a", "ann-b", domain.EdgeTypeReadingOrder)
	if !errors.Is(err, usecase.ErrDuplicateEdge) {
		t.Fatalf("Create duplicate error = %v, want %v", err, usecase.ErrDuplicateEdge)
	}

	_, err = fixture.usecase.Create(fixture.ctx, testImageID, "ann-b", "ann-a", domain.EdgeTypeReadingOrder)
	if !errors.Is(err, usecase.ErrReadingOrderCycle) {
		t.Fatalf("Create cycle error = %v, want %v", err, usecase.ErrReadingOrderCycle)
	}
}

func TestEdgeUsecaseCreateValidatesKeyValueEdges(t *testing.T) {
	fixture := newEdgeFixture(t)

	if _, err := fixture.usecase.Create(fixture.ctx, testImageID, "ann-key", "ann-value", domain.EdgeTypeKeyValue); err != nil {
		t.Fatalf("Create valid key_value edge returned error: %v", err)
	}

	_, err := fixture.usecase.Create(fixture.ctx, testImageID, "ann-value-2", "ann-key", domain.EdgeTypeKeyValue)
	if !errors.Is(err, usecase.ErrInvalidEdgeCategory) {
		t.Fatalf("Create wrong-category key_value error = %v, want %v", err, usecase.ErrInvalidEdgeCategory)
	}

	_, err = fixture.usecase.Create(fixture.ctx, testImageID, "ann-key", "ann-value-2", domain.EdgeTypeKeyValue)
	if !errors.Is(err, usecase.ErrEdgeCardinalityViolation) {
		t.Fatalf("Create second key_value edge error = %v, want %v", err, usecase.ErrEdgeCardinalityViolation)
	}
}

func TestEdgeUsecaseCreateValidatesTableCellEdges(t *testing.T) {
	fixture := newEdgeFixture(t)

	if _, err := fixture.usecase.Create(fixture.ctx, testImageID, "ann-table", "ann-cell", domain.EdgeTypeTableCell); err != nil {
		t.Fatalf("Create valid table_cell edge returned error: %v", err)
	}
	if _, err := fixture.usecase.Create(fixture.ctx, testImageID, "ann-table", "ann-cell-2", domain.EdgeTypeTableCell); err != nil {
		t.Fatalf("Create second table_cell edge for the same table returned error: %v", err)
	}

	_, err := fixture.usecase.Create(fixture.ctx, testImageID, "ann-cell-2", "ann-table", domain.EdgeTypeTableCell)
	if !errors.Is(err, usecase.ErrInvalidEdgeCategory) {
		t.Fatalf("Create wrong-category table_cell error = %v, want %v", err, usecase.ErrInvalidEdgeCategory)
	}

	_, err = fixture.usecase.Create(fixture.ctx, testImageID, "ann-table-2", "ann-cell", domain.EdgeTypeTableCell)
	if !errors.Is(err, usecase.ErrEdgeCardinalityViolation) {
		t.Fatalf("Create second table_cell source error = %v, want %v", err, usecase.ErrEdgeCardinalityViolation)
	}
}

func TestEdgeUsecaseBulkReplaceRejectsInvalidSetWithoutChangingExistingEdges(t *testing.T) {
	fixture := newEdgeFixture(t)
	if _, err := fixture.usecase.Create(fixture.ctx, testImageID, "ann-a", "ann-b", domain.EdgeTypeReadingOrder); err != nil {
		t.Fatalf("Create initial edge returned error: %v", err)
	}

	_, err := fixture.usecase.BulkReplace(fixture.ctx, testImageID, []domain.Edge{
		{
			SourceAnnotationID: "ann-a",
			TargetAnnotationID: "ann-a",
			Type:               domain.EdgeTypeReadingOrder,
		},
	})
	if !errors.Is(err, usecase.ErrSelfEdge) {
		t.Fatalf("BulkReplace error = %v, want %v", err, usecase.ErrSelfEdge)
	}

	edges, err := fixture.usecase.ListByImage(fixture.ctx, testImageID)
	if err != nil {
		t.Fatalf("ListByImage returned error: %v", err)
	}
	if len(edges) != 1 || edges[0].SourceAnnotationID != "ann-a" || edges[0].TargetAnnotationID != "ann-b" {
		t.Fatalf("ListByImage after rejected replace = %+v, want original edge preserved", edges)
	}
}

type edgeFixture struct {
	ctx     context.Context
	db      *sql.DB
	usecase *usecase.EdgeUsecase
}

func newEdgeFixture(t testing.TB) edgeFixture {
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
	insertEdgeFixtures(t, ctx, db)

	queries := sqlcgen.New(db)
	annotationRepo := sqlite.NewAnnotationRepository(db, queries)
	labelRepo := sqlite.NewLabelRepository(queries)
	edgeRepo := sqlite.NewEdgeRepository(db, queries)

	return edgeFixture{
		ctx:     ctx,
		db:      db,
		usecase: usecase.NewEdgeUsecase(edgeRepo, annotationRepo, labelRepo),
	}
}

func insertEdgeFixtures(t testing.TB, ctx context.Context, db *sql.DB) {
	t.Helper()

	execSQL(t, ctx, db, `INSERT INTO projects (id, name) VALUES (?, ?)`, "project-1", "Project 1")
	execSQL(t, ctx, db, `INSERT INTO images (id, project_id, filename, original_width, original_height, width, height) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		testImageID, "project-1", "image-1.png", 100, 100, 100, 100)
	execSQL(t, ctx, db, `INSERT INTO images (id, project_id, filename, original_width, original_height, width, height) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		otherTestImageID, "project-1", "image-2.png", 100, 100, 100, 100)

	for _, label := range []struct {
		id       string
		name     string
		category domain.LabelCategory
	}{
		{id: "label-object", name: "object", category: domain.LabelCategoryObject},
		{id: "label-key", name: "key", category: domain.LabelCategoryKey},
		{id: "label-value", name: "value", category: domain.LabelCategoryValue},
		{id: "label-table", name: "table", category: domain.LabelCategoryTable},
		{id: "label-cell", name: "cell", category: domain.LabelCategoryCell},
	} {
		execSQL(t, ctx, db, `INSERT INTO label_definitions (id, project_id, name, color, category) VALUES (?, ?, ?, ?, ?)`,
			label.id, "project-1", label.name, "#ffffff", string(label.category))
	}

	for _, annotation := range []struct {
		id      string
		imageID string
		labelID string
	}{
		{id: "ann-a", imageID: testImageID, labelID: "label-object"},
		{id: "ann-b", imageID: testImageID, labelID: "label-object"},
		{id: "ann-other-image", imageID: otherTestImageID, labelID: "label-object"},
		{id: "ann-key", imageID: testImageID, labelID: "label-key"},
		{id: "ann-value", imageID: testImageID, labelID: "label-value"},
		{id: "ann-value-2", imageID: testImageID, labelID: "label-value"},
		{id: "ann-table", imageID: testImageID, labelID: "label-table"},
		{id: "ann-table-2", imageID: testImageID, labelID: "label-table"},
		{id: "ann-cell", imageID: testImageID, labelID: "label-cell"},
		{id: "ann-cell-2", imageID: testImageID, labelID: "label-cell"},
	} {
		execSQL(t, ctx, db, `INSERT INTO annotations (id, image_id, type, coordinates, label_id) VALUES (?, ?, ?, ?, ?)`,
			annotation.id, annotation.imageID, string(domain.AnnotationTypeBBox), `{"x":0,"y":0,"width":0.1,"height":0.1}`, annotation.labelID)
	}
}

func execSQL(t testing.TB, ctx context.Context, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.ExecContext(ctx, query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}
