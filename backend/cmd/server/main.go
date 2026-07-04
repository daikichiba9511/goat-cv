package main

import (
	"database/sql"
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
	imageRepo := sqlite.NewImageRepository(queries)
	annotationRepo := sqlite.NewAnnotationRepository(db, queries)

	// Usecases
	projectUC := usecase.NewProjectUsecase(projectRepo)
	labelUC := usecase.NewLabelUsecase(labelRepo)
	imageUC := usecase.NewImageUsecase(imageRepo, storagePath)
	annotationUC := usecase.NewAnnotationUsecase(annotationRepo)

	// Handlers
	projectHandler := handler.NewProjectHandler(projectUC)
	labelHandler := handler.NewLabelHandler(labelUC)
	imageHandler := handler.NewImageHandler(imageUC)
	annotationHandler := handler.NewAnnotationHandler(annotationUC)
	exportHandler := handler.NewExportHandler(projectUC, imageUC, annotationUC, labelUC)

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
		r.Route("/{projectId}/images", func(r chi.Router) {
			imageHandler.ProjectRoutes(r)
		})
		r.Get("/{projectId}/export", exportHandler.ProjectExport)
	})

	r.Mount("/api/v1/images", imageHandler.ImageRoutes())

	r.Route("/api/v1/images/{imageId}/annotations", func(r chi.Router) {
		annotationHandler.ImageRoutes(r)
	})

	r.Mount("/api/v1/annotations", annotationHandler.AnnotationRoutes())

	r.Get("/api/v1/images/{imageId}/export", exportHandler.ImageExport)

	log.Printf("starting server on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func runMigrations(db *sql.DB) error {
	migrationDir := findMigrationDir()
	entries, err := os.ReadDir(migrationDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(migrationDir, entry.Name()))
		if err != nil {
			return err
		}
		if _, err := db.Exec(string(data)); err != nil {
			return err
		}
		log.Printf("applied migration: %s", entry.Name())
	}
	return nil
}

func findMigrationDir() string {
	candidates := []string{
		"db/migrations",
		"backend/db/migrations",
		"../db/migrations",
		"../../db/migrations",
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return c
		}
	}
	return "db/migrations"
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
