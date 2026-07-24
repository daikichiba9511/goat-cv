package sqlite

import (
	"context"
	"database/sql"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/daikichiba9511/goat-cv/backend/internal/sqlcgen"
)

// EdgeRepository persists annotation edges in SQLite.
type EdgeRepository struct {
	db      *sql.DB
	queries *sqlcgen.Queries
}

// NewEdgeRepository creates an EdgeRepository.
func NewEdgeRepository(db *sql.DB, queries *sqlcgen.Queries) *EdgeRepository {
	return &EdgeRepository{db: db, queries: queries}
}

// Create inserts an edge.
func (r *EdgeRepository) Create(ctx context.Context, edge domain.Edge) (domain.Edge, error) {
	row, err := r.queries.CreateEdge(ctx, sqlcgen.CreateEdgeParams{
		ID:                 edge.ID,
		ImageID:            edge.ImageID,
		SourceAnnotationID: edge.SourceAnnotationID,
		TargetAnnotationID: edge.TargetAnnotationID,
		Type:               string(edge.Type),
	})
	if err != nil {
		return domain.Edge{}, err
	}
	return toEdge(row), nil
}

// Get returns an edge by ID.
func (r *EdgeRepository) Get(ctx context.Context, id string) (domain.Edge, error) {
	row, err := r.queries.GetEdge(ctx, id)
	if err != nil {
		return domain.Edge{}, err
	}
	return toEdge(row), nil
}

// ListByImage returns edges for an image.
func (r *EdgeRepository) ListByImage(ctx context.Context, imageID string) ([]domain.Edge, error) {
	rows, err := r.queries.ListEdgesByImage(ctx, imageID)
	if err != nil {
		return nil, err
	}
	edges := make([]domain.Edge, len(rows))
	for i, row := range rows {
		edges[i] = toEdge(row)
	}
	return edges, nil
}

// Delete removes an edge by ID.
func (r *EdgeRepository) Delete(ctx context.Context, id string) error {
	return r.queries.DeleteEdge(ctx, id)
}

// BulkReplace replaces all edges for an image and returns the persisted rows.
// The operation is atomic: if any insert fails, the previous edge set remains in place.
func (r *EdgeRepository) BulkReplace(ctx context.Context, imageID string, edges []domain.Edge) ([]domain.Edge, error) {
	// Why: EdgeはAnnotation集合と同じく画像単位で編集されるため、一括保存に合わせて全置換にする。
	// Why not: 途中失敗時にグラフだけ半端に消えないよう、DeleteとInsertを分けてコミットしない。
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	txQueries := r.queries.WithTx(tx)

	if err := txQueries.DeleteEdgesByImage(ctx, imageID); err != nil {
		return nil, err
	}

	result := make([]domain.Edge, len(edges))
	for i, edge := range edges {
		row, err := txQueries.CreateEdge(ctx, sqlcgen.CreateEdgeParams{
			ID:                 edge.ID,
			ImageID:            imageID,
			SourceAnnotationID: edge.SourceAnnotationID,
			TargetAnnotationID: edge.TargetAnnotationID,
			Type:               string(edge.Type),
		})
		if err != nil {
			return nil, err
		}
		result[i] = toEdge(row)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}
