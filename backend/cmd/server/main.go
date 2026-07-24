package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/daikichiba9511/goat-cv/backend/internal/handler"
	"github.com/daikichiba9511/goat-cv/backend/internal/repository/sqlite"
	"github.com/daikichiba9511/goat-cv/backend/internal/sqlcgen"
	"github.com/daikichiba9511/goat-cv/backend/internal/usecase"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	dbPath := envOrDefault("DB_PATH", "goat.db")
	storagePath := envOrDefault("STORAGE_PATH", "storage")
	addr := envOrDefault("ADDR", ":8080")

	if err := os.MkdirAll(storagePath, 0o755); err != nil {
		log.Fatalf("failed to create storage directory: %v", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := runMigrations(db); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		// Why: Phase 1はローカル開発前提なので、Vite dev serverだけを明示的に許可する。
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Repositories
	queries := sqlcgen.New(db)
	projectRepo := sqlite.NewProjectRepository(queries)
	labelRepo := sqlite.NewLabelRepository(queries)
	guidelineRepo := sqlite.NewGuidelineRepository(queries)
	commentRepo := sqlite.NewCommentRepository(queries)
	imageRepo := sqlite.NewImageRepository(queries)
	annotationRepo := sqlite.NewAnnotationRepository(db, queries)
	edgeRepo := sqlite.NewEdgeRepository(db, queries)
	imageGraphRepo := sqlite.NewImageGraphRepository(db, queries)

	// Usecases
	projectUC := usecase.NewProjectUsecase(projectRepo)
	labelUC := usecase.NewLabelUsecase(labelRepo)
	guidelineUC := usecase.NewGuidelineUsecase(guidelineRepo)
	commentUC := usecase.NewCommentUsecase(commentRepo, imageRepo, annotationRepo)
	imageUC := usecase.NewImageUsecase(imageRepo, storagePath)
	annotationUC := usecase.NewAnnotationUsecase(annotationRepo)
	edgeUC := usecase.NewEdgeUsecase(edgeRepo, annotationRepo, labelRepo)
	imageGraphUC := usecase.NewImageGraphUsecase(imageGraphRepo, labelRepo)
	datasetExportUC := usecase.NewDatasetExportUsecase(
		projectRepo,
		imageRepo,
		annotationRepo,
		labelRepo,
		storagePath,
	)

	// Handlers
	projectHandler := handler.NewProjectHandler(projectUC)
	labelHandler := handler.NewLabelHandler(labelUC)
	guidelineHandler := handler.NewGuidelineHandler(guidelineUC)
	commentHandler := handler.NewCommentHandler(commentUC)
	imageHandler := handler.NewImageHandler(imageUC)
	annotationHandler := handler.NewAnnotationHandler(annotationUC)
	edgeHandler := handler.NewEdgeHandler(edgeUC)
	imageGraphHandler := handler.NewImageGraphHandler(imageGraphUC)
	exportHandler := handler.NewExportHandler(
		projectUC,
		imageUC,
		annotationUC,
		labelUC,
		edgeUC,
		datasetExportUC,
	)

	// Routes
	r.Get("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/api/v1/projects", func(r chi.Router) {
		r.Mount("/", projectHandler.Routes())
		r.Route("/{projectId}/labels", func(r chi.Router) {
			labelHandler.Routes(r)
		})
		r.Route("/{projectId}/guidelines", func(r chi.Router) {
			guidelineHandler.Routes(r)
		})
		r.Route("/{projectId}/images", func(r chi.Router) {
			imageHandler.ProjectRoutes(r)
		})
		r.Get("/{projectId}/export", exportHandler.ProjectExport)
	})

	r.Mount("/api/v1/images", imageHandler.ImageRoutes())

	r.Route("/api/v1/images/{imageId}/annotations", func(r chi.Router) {
		annotationHandler.ImageRoutes(r)
	})
	r.Route("/api/v1/images/{imageId}/edges", func(r chi.Router) {
		edgeHandler.ImageRoutes(r)
	})
	r.Route("/api/v1/images/{imageId}/graph", func(r chi.Router) {
		imageGraphHandler.ImageRoutes(r)
	})
	r.Route("/api/v1/images/{imageId}/comments", func(r chi.Router) {
		commentHandler.ImageRoutes(r)
	})

	r.Mount("/api/v1/annotations", annotationHandler.AnnotationRoutes())
	r.Mount("/api/v1/edges", edgeHandler.EdgeRoutes())

	r.Get("/api/v1/images/{imageId}/export", exportHandler.ImageExport)

	log.Printf("starting server on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func runMigrations(db *sql.DB) error {
	return runMigrationsFromDir(db, findMigrationDir())
}

func runMigrationsFromDir(db *sql.DB, migrationDir string) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name TEXT PRIMARY KEY,
			applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		)
	`); err != nil {
		return fmt.Errorf("create migration history: %w", err)
	}

	entries, err := os.ReadDir(migrationDir)
	if err != nil {
		return fmt.Errorf("read migration directory: %w", err)
	}

	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		var alreadyApplied bool
		if err := db.QueryRow(
			`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE name = ?)`,
			entry.Name(),
		).Scan(&alreadyApplied); err != nil {
			return fmt.Errorf("check migration %s: %w", entry.Name(), err)
		}
		if alreadyApplied {
			continue
		}

		data, err := os.ReadFile(filepath.Join(migrationDir, entry.Name()))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}

		transaction, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", entry.Name(), err)
		}
		if _, err := transaction.Exec(string(data)); err != nil {
			transaction.Rollback()
			return fmt.Errorf("apply migration %s: %w", entry.Name(), err)
		}
		if _, err := transaction.Exec(
			`INSERT INTO schema_migrations (name) VALUES (?)`,
			entry.Name(),
		); err != nil {
			transaction.Rollback()
			return fmt.Errorf("record migration %s: %w", entry.Name(), err)
		}
		if err := transaction.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", entry.Name(), err)
		}
		log.Printf("applied migration: %s", entry.Name())
	}
	return nil
}

func findMigrationDir() string {
	// Why: `go run ./cmd/server` と repo root からの起動の両方を許容し、開発時の起動場所に依存させない。
	candidates := []string{
		"db/migrations",
		"backend/db/migrations",
		"../db/migrations",
		"../../db/migrations",
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return "db/migrations"
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
