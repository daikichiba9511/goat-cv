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

func TestGuidelineUsecaseManagesGuidelinesInStableDisplayOrder(t *testing.T) {
	fixture := newGuidelineFixture(t)

	guidelines, err := fixture.usecase.ListByProject(fixture.ctx, guidelineProjectID)
	if err != nil {
		t.Fatalf("ListByProject returned error: %v", err)
	}
	if len(guidelines) != 0 {
		t.Fatalf("initial guidelines = %+v, want empty list", guidelines)
	}

	zeta := fixture.create(t, guidelineProjectID, "Zeta", "zeta body", 1)
	later := fixture.create(t, guidelineProjectID, "Later", "later body", 2)
	alpha := fixture.create(t, guidelineProjectID, "Alpha", "alpha body", 1)
	assertGuidelineTitles(t, fixture.list(t, guidelineProjectID), "Alpha", "Zeta", "Later")

	updated, err := fixture.usecase.Update(
		fixture.ctx,
		guidelineProjectID,
		later.ID,
		"First",
		"updated body",
		0,
	)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if updated.Title != "First" || updated.Body != "updated body" || updated.DisplayOrder != 0 {
		t.Fatalf("updated guideline = %+v, want updated fields", updated)
	}
	assertGuidelineTitles(t, fixture.list(t, guidelineProjectID), "First", "Alpha", "Zeta")

	if err := fixture.usecase.Delete(fixture.ctx, guidelineProjectID, alpha.ID); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if _, err := fixture.usecase.Get(fixture.ctx, guidelineProjectID, alpha.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("Get deleted guideline error = %v, want sql.ErrNoRows", err)
	}
	assertGuidelineTitles(t, fixture.list(t, guidelineProjectID), "First", "Zeta")

	if zeta.ProjectID != guidelineProjectID {
		t.Fatalf("created guideline project = %q, want %q", zeta.ProjectID, guidelineProjectID)
	}
}

func TestGuidelineUsecaseDoesNotAccessGuidelineThroughAnotherProject(t *testing.T) {
	fixture := newGuidelineFixture(t)
	guideline := fixture.create(t, guidelineProjectID, "Private rules", "project one", 0)

	if guidelines := fixture.list(t, otherGuidelineProjectID); len(guidelines) != 0 {
		t.Fatalf("other Project guidelines = %+v, want empty list", guidelines)
	}

	operations := []struct {
		name string
		run  func() error
	}{
		{
			name: "get",
			run: func() error {
				_, err := fixture.usecase.Get(fixture.ctx, otherGuidelineProjectID, guideline.ID)
				return err
			},
		},
		{
			name: "update",
			run: func() error {
				_, err := fixture.usecase.Update(
					fixture.ctx,
					otherGuidelineProjectID,
					guideline.ID,
					"Changed",
					"changed",
					1,
				)
				return err
			},
		},
		{
			name: "delete",
			run: func() error {
				return fixture.usecase.Delete(fixture.ctx, otherGuidelineProjectID, guideline.ID)
			},
		},
	}

	for _, operation := range operations {
		t.Run(operation.name, func(t *testing.T) {
			if err := operation.run(); !errors.Is(err, sql.ErrNoRows) {
				t.Fatalf("error = %v, want sql.ErrNoRows", err)
			}
		})
	}

	unchanged, err := fixture.usecase.Get(fixture.ctx, guidelineProjectID, guideline.ID)
	if err != nil {
		t.Fatalf("Get through owning Project returned error: %v", err)
	}
	if unchanged.Title != "Private rules" || unchanged.Body != "project one" || unchanged.DisplayOrder != 0 {
		t.Fatalf("guideline after cross-Project operations = %+v, want unchanged", unchanged)
	}
}

func TestGuidelineUsecaseRejectsInvalidFieldsWithoutCreatingGuideline(t *testing.T) {
	fixture := newGuidelineFixture(t)
	tests := []struct {
		name         string
		title        string
		displayOrder int
	}{
		{name: "blank title", title: "  ", displayOrder: 0},
		{name: "negative display order", title: "Rules", displayOrder: -1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := fixture.usecase.Create(
				fixture.ctx,
				guidelineProjectID,
				test.title,
				"body",
				test.displayOrder,
			)
			if !errors.Is(err, usecase.ErrInvalidGuideline) {
				t.Fatalf("Create error = %v, want ErrInvalidGuideline", err)
			}
		})
	}

	if guidelines := fixture.list(t, guidelineProjectID); len(guidelines) != 0 {
		t.Fatalf("guidelines after invalid creates = %+v, want empty list", guidelines)
	}
}

const (
	guidelineProjectID      = "guideline-project-1"
	otherGuidelineProjectID = "guideline-project-2"
)

type guidelineFixture struct {
	ctx     context.Context
	usecase *usecase.GuidelineUsecase
}

func newGuidelineFixture(t testing.TB) guidelineFixture {
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

	for _, migrationPath := range []string{
		"../../db/migrations/001_init.sql",
		"../../db/migrations/002_guidelines.sql",
	} {
		migration, err := os.ReadFile(migrationPath)
		if err != nil {
			t.Fatalf("read migration %s: %v", migrationPath, err)
		}
		if _, err := db.Exec(string(migration)); err != nil {
			t.Fatalf("apply migration %s: %v", migrationPath, err)
		}
	}

	ctx := context.Background()
	for _, project := range []struct {
		id   string
		name string
	}{
		{id: guidelineProjectID, name: "Guideline Project"},
		{id: otherGuidelineProjectID, name: "Other Project"},
	} {
		if _, err := db.ExecContext(
			ctx,
			`INSERT INTO projects (id, name) VALUES (?, ?)`,
			project.id,
			project.name,
		); err != nil {
			t.Fatalf("insert Project %s: %v", project.id, err)
		}
	}

	repository := sqlite.NewGuidelineRepository(sqlcgen.New(db))
	return guidelineFixture{
		ctx:     ctx,
		usecase: usecase.NewGuidelineUsecase(repository),
	}
}

func (f guidelineFixture) create(
	t testing.TB,
	projectID string,
	title string,
	body string,
	displayOrder int,
) domain.Guideline {
	t.Helper()
	guideline, err := f.usecase.Create(f.ctx, projectID, title, body, displayOrder)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	return guideline
}

func (f guidelineFixture) list(t testing.TB, projectID string) []domain.Guideline {
	t.Helper()
	guidelines, err := f.usecase.ListByProject(f.ctx, projectID)
	if err != nil {
		t.Fatalf("ListByProject returned error: %v", err)
	}
	return guidelines
}

func assertGuidelineTitles(t testing.TB, guidelines []domain.Guideline, want ...string) {
	t.Helper()
	if len(guidelines) != len(want) {
		t.Fatalf("guideline count = %d, want %d: %+v", len(guidelines), len(want), guidelines)
	}
	for index, title := range want {
		if guidelines[index].Title != title {
			t.Fatalf("guideline[%d].Title = %q, want %q", index, guidelines[index].Title, title)
		}
	}
}
