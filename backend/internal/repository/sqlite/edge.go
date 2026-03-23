package sqlite

import (
	"context"
	"database/sql"

	"github.com/chibadaimare/goat/backend/internal/domain"
	"github.com/chibadaimare/goat/backend/internal/sqlcgen"
)

type EdgeRepository struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewEdgeRepository(db *sql.DB, q *sqlcgen.Queries) *EdgeRepository {
	return &EdgeRepository{db: db, q: q}
}

func (r *EdgeRepository) Create(ctx context.Context, edge domain.Edge) (domain.Edge, error) {
	row, err := r.q.CreateEdge(ctx, sqlcgen.CreateEdgeParams{
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

func (r *EdgeRepository) ListByImage(ctx context.Context, imageID string) ([]domain.Edge, error) {
	rows, err := r.q.ListEdgesByImage(ctx, imageID)
	if err != nil {
		return nil, err
	}
	edges := make([]domain.Edge, len(rows))
	for i, row := range rows {
		edges[i] = toEdge(row)
	}
	return edges, nil
}

func (r *EdgeRepository) Delete(ctx context.Context, id string) error {
	return r.q.DeleteEdge(ctx, id)
}

// BulkReplace deletes all edges for an image and inserts new ones in a transaction.
func (r *EdgeRepository) BulkReplace(ctx context.Context, imageID string, edges []domain.Edge) ([]domain.Edge, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	qtx := r.q.WithTx(tx)

	if err := qtx.DeleteEdgesByImage(ctx, imageID); err != nil {
		return nil, err
	}

	result := make([]domain.Edge, len(edges))
	for i, edge := range edges {
		row, err := qtx.CreateEdge(ctx, sqlcgen.CreateEdgeParams{
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
