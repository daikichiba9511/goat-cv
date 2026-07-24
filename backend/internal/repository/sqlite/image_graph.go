package sqlite

import (
	"context"
	"database/sql"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/daikichiba9511/goat-cv/backend/internal/sqlcgen"
)

// ImageGraphRepository atomically persists annotations and edges for one image.
type ImageGraphRepository struct {
	db      *sql.DB
	queries *sqlcgen.Queries
}

// NewImageGraphRepository creates an ImageGraphRepository.
func NewImageGraphRepository(db *sql.DB, queries *sqlcgen.Queries) *ImageGraphRepository {
	return &ImageGraphRepository{db: db, queries: queries}
}

// Replace replaces an image's annotations and edges in one transaction and returns the persisted rows.
func (r *ImageGraphRepository) Replace(
	ctx context.Context,
	imageID string,
	annotations []domain.Annotation,
	edges []domain.Edge,
) ([]domain.Annotation, []domain.Edge, error) {
	// Why: EdgeはAnnotation IDを参照するため、削除・Annotation挿入・Edge挿入を同じTransaction所有者に集約する。
	// Why not: 2つのRepositoryで別々にTransactionを開始すると、片方だけCommitされる部分保存を防げない。
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()

	txQueries := r.queries.WithTx(tx)
	if err := txQueries.DeleteEdgesByImage(ctx, imageID); err != nil {
		return nil, nil, err
	}
	// Why: 同じIDのAnnotationは更新し、削除対象だけを消して永続IDと作成時刻を安定させる。
	persistedAnnotations, err := replaceAnnotations(ctx, txQueries, imageID, annotations)
	if err != nil {
		return nil, nil, err
	}

	persistedEdges := make([]domain.Edge, len(edges))
	for edgeIndex, edge := range edges {
		row, err := txQueries.CreateEdge(ctx, sqlcgen.CreateEdgeParams{
			ID:                 edge.ID,
			ImageID:            imageID,
			SourceAnnotationID: edge.SourceAnnotationID,
			TargetAnnotationID: edge.TargetAnnotationID,
			Type:               string(edge.Type),
		})
		if err != nil {
			return nil, nil, err
		}
		persistedEdges[edgeIndex] = toEdge(row)
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}
	return persistedAnnotations, persistedEdges, nil
}
