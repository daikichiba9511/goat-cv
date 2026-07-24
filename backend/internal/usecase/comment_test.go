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

func TestCommentUsecaseManagesImageAndAnnotationComments(t *testing.T) {
	fixture := newCommentFixture(t)

	if comments := fixture.list(t, commentImageID); len(comments) != 0 {
		t.Fatalf("initial comments = %+v, want empty list", comments)
	}

	imageComment := fixture.create(t, commentImageID, nil, " reviewer ", "Check the whole image", domain.CommentTypeQuestion)
	annotationID := commentAnnotationID
	annotationComment := fixture.create(t, commentImageID, &annotationID, "annotator", "BBox margin", domain.CommentTypeIssue)

	comments := fixture.list(t, commentImageID)
	if len(comments) != 2 || comments[0].ID != imageComment.ID || comments[1].ID != annotationComment.ID {
		t.Fatalf("comments = %+v, want creation order", comments)
	}
	if imageComment.Author != "reviewer" || imageComment.Body != "Check the whole image" || imageComment.Resolved {
		t.Fatalf("image Comment = %+v, want trimmed unresolved Comment", imageComment)
	}

	resolved, err := fixture.usecase.SetResolved(fixture.ctx, usecase.SetCommentResolvedInput{
		ImageID: commentImageID, CommentID: annotationComment.ID, Resolved: true,
	})
	if err != nil {
		t.Fatalf("SetResolved(true) returned error: %v", err)
	}
	if !resolved.Resolved {
		t.Fatalf("resolved = %+v, want resolved Comment", resolved)
	}
	reopened, err := fixture.usecase.SetResolved(fixture.ctx, usecase.SetCommentResolvedInput{
		ImageID: commentImageID, CommentID: annotationComment.ID, Resolved: false,
	})
	if err != nil {
		t.Fatalf("SetResolved(false) returned error: %v", err)
	}
	if reopened.Resolved {
		t.Fatalf("reopened = %+v, want unresolved Comment", reopened)
	}

	if err := fixture.usecase.Delete(fixture.ctx, commentImageID, imageComment.ID); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	comments = fixture.list(t, commentImageID)
	if len(comments) != 1 || comments[0].ID != annotationComment.ID {
		t.Fatalf("comments after Delete = %+v, want Annotation Comment only", comments)
	}
}

func TestCommentUsecaseRejectsInvalidOrOutOfScopeTargets(t *testing.T) {
	fixture := newCommentFixture(t)
	otherAnnotationID := otherCommentAnnotationID
	otherProjectAnnotationID := otherProjectCommentAnnotationID
	missingAnnotationID := "missing-annotation"

	tests := []struct {
		name         string
		imageID      string
		annotationID *string
		author       string
		body         string
		commentType  domain.CommentType
		wantError    error
	}{
		{name: "blank author", imageID: commentImageID, author: "  ", body: "body", commentType: domain.CommentTypeNote, wantError: usecase.ErrInvalidComment},
		{name: "blank body", imageID: commentImageID, author: "author", body: "\n\t", commentType: domain.CommentTypeNote, wantError: usecase.ErrInvalidComment},
		{name: "unsupported type", imageID: commentImageID, author: "author", body: "body", commentType: "answer", wantError: usecase.ErrInvalidComment},
		{name: "missing image", imageID: "missing-image", author: "author", body: "body", commentType: domain.CommentTypeNote, wantError: sql.ErrNoRows},
		{name: "missing annotation", imageID: commentImageID, annotationID: &missingAnnotationID, author: "author", body: "body", commentType: domain.CommentTypeIssue, wantError: sql.ErrNoRows},
		{name: "annotation from another image", imageID: commentImageID, annotationID: &otherAnnotationID, author: "author", body: "body", commentType: domain.CommentTypeIssue, wantError: sql.ErrNoRows},
		{name: "annotation from another Project", imageID: commentImageID, annotationID: &otherProjectAnnotationID, author: "author", body: "body", commentType: domain.CommentTypeIssue, wantError: sql.ErrNoRows},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := fixture.usecase.Create(fixture.ctx, usecase.CreateCommentInput{
				ImageID:      test.imageID,
				AnnotationID: test.annotationID,
				Author:       test.author,
				Body:         test.body,
				Type:         test.commentType,
			})
			if !errors.Is(err, test.wantError) {
				t.Fatalf("Create error = %v, want %v", err, test.wantError)
			}
		})
	}

	comment := fixture.create(t, commentImageID, nil, "reviewer", "Scoped", domain.CommentTypeNote)
	if _, err := fixture.usecase.SetResolved(fixture.ctx, usecase.SetCommentResolvedInput{
		ImageID: otherCommentImageID, CommentID: comment.ID, Resolved: true,
	}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("cross-Image SetResolved error = %v, want sql.ErrNoRows", err)
	}
	if err := fixture.usecase.Delete(fixture.ctx, otherCommentImageID, comment.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("cross-Image Delete error = %v, want sql.ErrNoRows", err)
	}
	if len(fixture.list(t, commentImageID)) != 1 {
		t.Fatal("cross-Image operations changed the owning Image's Comment")
	}
}

func TestAnnotationCommentFollowsPersistentAnnotationIdentityAfterDeletion(t *testing.T) {
	fixture := newCommentFixture(t)
	annotationID := commentAnnotationID
	created := fixture.create(t, commentImageID, &annotationID, "reviewer", "Keep until deletion", domain.CommentTypeIssue)

	updatedAnnotation := domain.Annotation{
		ID:          commentAnnotationID,
		ImageID:     commentImageID,
		Type:        domain.AnnotationTypeBBox,
		Coordinates: domain.Coordinates(`{"x":0.2,"y":0.2,"width":0.4,"height":0.4}`),
	}
	if _, _, err := fixture.graphRepository.Replace(
		fixture.ctx,
		commentImageID,
		[]domain.Annotation{updatedAnnotation},
		nil,
	); err != nil {
		t.Fatalf("Replace preserving Annotation returned error: %v", err)
	}
	comments := fixture.list(t, commentImageID)
	if len(comments) != 1 || comments[0].ID != created.ID {
		t.Fatalf("comments after Annotation update = %+v, want preserved Comment", comments)
	}

	if _, _, err := fixture.graphRepository.Replace(fixture.ctx, commentImageID, nil, nil); err != nil {
		t.Fatalf("Replace deleting Annotation returned error: %v", err)
	}
	if _, err := fixture.db.ExecContext(fixture.ctx, `
		INSERT INTO annotations (id, image_id, type, coordinates)
		VALUES (?, ?, 'bbox', '{"x":0,"y":0,"width":1,"height":1}')
	`, annotationID, otherCommentImageID); err != nil {
		t.Fatalf("reuse deleted Annotation ID in another Image: %v", err)
	}
	comments = fixture.list(t, commentImageID)
	if len(comments) != 1 || comments[0].ID != created.ID || !comments[0].TargetDeleted {
		t.Fatalf("comments after Annotation deletion = %+v, want retained Comment with deleted target", comments)
	}
	if comments[0].AnnotationID == nil || *comments[0].AnnotationID != annotationID {
		t.Fatalf("annotation ID = %v, want retained logical reference %q", comments[0].AnnotationID, annotationID)
	}
}

const (
	commentProjectID                = "comment-project-1"
	otherCommentProjectID           = "comment-project-2"
	commentImageID                  = "comment-image-1"
	otherCommentImageID             = "comment-image-2"
	otherProjectCommentImageID      = "comment-image-3"
	commentAnnotationID             = "comment-annotation-1"
	otherCommentAnnotationID        = "comment-annotation-2"
	otherProjectCommentAnnotationID = "comment-annotation-3"
)

type commentFixture struct {
	ctx             context.Context
	db              *sql.DB
	usecase         *usecase.CommentUsecase
	graphRepository *sqlite.ImageGraphRepository
}

func newCommentFixture(t testing.TB) commentFixture {
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
		"../../db/migrations/003_comments.sql",
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
	for _, project := range []struct{ id, name string }{
		{id: commentProjectID, name: "Comment Project"},
		{id: otherCommentProjectID, name: "Other Comment Project"},
	} {
		if _, err := db.ExecContext(ctx, `INSERT INTO projects (id, name) VALUES (?, ?)`, project.id, project.name); err != nil {
			t.Fatalf("insert Project %s: %v", project.id, err)
		}
	}
	for _, image := range []struct{ id, projectID string }{
		{id: commentImageID, projectID: commentProjectID},
		{id: otherCommentImageID, projectID: commentProjectID},
		{id: otherProjectCommentImageID, projectID: otherCommentProjectID},
	} {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO images (
				id, project_id, filename, original_width, original_height, width, height
			) VALUES (?, ?, ?, 100, 100, 100, 100)
		`, image.id, image.projectID, image.id+".png"); err != nil {
			t.Fatalf("insert Image %s: %v", image.id, err)
		}
	}
	for _, annotation := range []struct{ id, imageID string }{
		{id: commentAnnotationID, imageID: commentImageID},
		{id: otherCommentAnnotationID, imageID: otherCommentImageID},
		{id: otherProjectCommentAnnotationID, imageID: otherProjectCommentImageID},
	} {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO annotations (id, image_id, type, coordinates)
			VALUES (?, ?, 'bbox', '{"x":0,"y":0,"width":1,"height":1}')
		`, annotation.id, annotation.imageID); err != nil {
			t.Fatalf("insert Annotation %s: %v", annotation.id, err)
		}
	}

	queries := sqlcgen.New(db)
	return commentFixture{
		ctx: ctx,
		db:  db,
		usecase: usecase.NewCommentUsecase(
			sqlite.NewCommentRepository(queries),
			sqlite.NewImageRepository(queries),
			sqlite.NewAnnotationRepository(db, queries),
		),
		graphRepository: sqlite.NewImageGraphRepository(db, queries),
	}
}

func (fixture commentFixture) create(
	t testing.TB,
	imageID string,
	annotationID *string,
	author string,
	body string,
	commentType domain.CommentType,
) domain.Comment {
	t.Helper()
	comment, err := fixture.usecase.Create(fixture.ctx, usecase.CreateCommentInput{
		ImageID: imageID, AnnotationID: annotationID, Author: author, Body: body, Type: commentType,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	return comment
}

func (fixture commentFixture) list(t testing.TB, imageID string) []domain.Comment {
	t.Helper()
	comments, err := fixture.usecase.ListByImage(fixture.ctx, imageID)
	if err != nil {
		t.Fatalf("ListByImage returned error: %v", err)
	}
	return comments
}
