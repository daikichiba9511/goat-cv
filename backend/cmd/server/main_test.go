package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestRunMigrationsCreatesWorkflowSchemaAndCanRunAgain(t *testing.T) {
	database := openMigrationTestDatabase(t)

	for run := 1; run <= 2; run++ {
		if err := runMigrationsFromDir(database, "../../db/migrations"); err != nil {
			t.Fatalf("run migrations %d: %v", run, err)
		}
	}

	if _, err := database.Exec(`INSERT INTO projects (id, name) VALUES ('project-1', 'Workflow')`); err != nil {
		t.Fatalf("insert Project: %v", err)
	}
	if _, err := database.Exec(`
		INSERT INTO images (
			id, project_id, filename, original_width, original_height, width, height
		) VALUES ('image-1', 'project-1', 'image.png', 100, 100, 100, 100)
	`); err != nil {
		t.Fatalf("insert Image: %v", err)
	}

	var status string
	var escalated bool
	if err := database.QueryRow(
		`SELECT status, escalated FROM images WHERE id = 'image-1'`,
	).Scan(&status, &escalated); err != nil {
		t.Fatalf("read workflow defaults: %v", err)
	}
	if status != "pending" || escalated {
		t.Fatalf("new Image workflow = (%q, %t), want (pending, false)", status, escalated)
	}

	if _, err := database.Exec(`UPDATE images SET status = 'escalated' WHERE id = 'image-1'`); err == nil {
		t.Fatal("legacy escalated status update succeeded, want constraint error")
	}
	if _, err := database.Exec(`UPDATE images SET status = 'approved', escalated = 1 WHERE id = 'image-1'`); err == nil {
		t.Fatal("approved escalation update succeeded, want constraint error")
	}
}

func TestRunMigrationsRejectsLegacyEscalatedStatusWithoutConversion(t *testing.T) {
	database := openMigrationTestDatabase(t)
	initialSchema, err := os.ReadFile("../../db/migrations/001_init.sql")
	if err != nil {
		t.Fatalf("read initial migration: %v", err)
	}
	if _, err := database.Exec(string(initialSchema)); err != nil {
		t.Fatalf("apply initial migration: %v", err)
	}
	if _, err := database.Exec(`INSERT INTO projects (id, name) VALUES ('project-1', 'Legacy')`); err != nil {
		t.Fatalf("insert legacy Project: %v", err)
	}
	if _, err := database.Exec(`
		INSERT INTO images (
			id, project_id, filename, original_width, original_height, width, height, status
		) VALUES ('image-legacy', 'project-1', 'legacy.png', 100, 100, 100, 100, 'escalated')
	`); err != nil {
		t.Fatalf("insert legacy escalated Image: %v", err)
	}

	err = runMigrationsFromDir(database, "../../db/migrations")
	if err == nil || !strings.Contains(err.Error(), "004_image_workflow.sql") || !strings.Contains(err.Error(), "image-legacy") {
		t.Fatalf("run migrations error = %v, want workflow migration failure identifying image-legacy", err)
	}

	var status string
	if err := database.QueryRow(
		`SELECT status FROM images WHERE id = 'image-legacy'`,
	).Scan(&status); err != nil {
		t.Fatalf("read legacy Image after rejected migration: %v", err)
	}
	if status != "escalated" {
		t.Fatalf("legacy Image status = %q, want unchanged escalated", status)
	}
}

func openMigrationTestDatabase(t testing.TB) *sql.DB {
	t.Helper()
	databasePath := filepath.Join(t.TempDir(), "goat.db")
	database, err := sql.Open("sqlite3", databasePath+"?_foreign_keys=on")
	if err != nil {
		t.Fatalf("open migration test database: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close migration test database: %v", err)
		}
	})
	return database
}
