package usecase

import (
	"context"
	"errors"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/google/uuid"
)

var (
	// ErrInvalidEdgeType indicates an unsupported edge type.
	ErrInvalidEdgeType = errors.New("invalid edge type")
	// ErrSelfEdge indicates an edge that points an annotation to itself.
	ErrSelfEdge = errors.New("edge source and target must differ")
	// ErrEdgeAnnotationNotFound indicates a missing source or target annotation.
	ErrEdgeAnnotationNotFound = errors.New("edge annotation not found")
	// ErrEdgeImageMismatch indicates an edge whose annotations are outside the route image.
	ErrEdgeImageMismatch = errors.New("edge annotations must belong to the route image")
	// ErrDuplicateEdge indicates an existing source-target-type relation.
	ErrDuplicateEdge = errors.New("duplicate edge")
	// ErrReadingOrderCycle indicates a reading-order edge that would create a cycle.
	ErrReadingOrderCycle = errors.New("reading order edges must be acyclic")
	// ErrInvalidEdgeCategory indicates source or target label categories do not match the edge type.
	ErrInvalidEdgeCategory = errors.New("edge label categories do not match edge type")
	// ErrEdgeCardinalityViolation indicates a key-value or table-cell cardinality rule was broken.
	ErrEdgeCardinalityViolation = errors.New("edge cardinality rule violation")
)

type edgeRepository interface {
	Create(ctx context.Context, edge domain.Edge) (domain.Edge, error)
	ListByImage(ctx context.Context, imageID string) ([]domain.Edge, error)
	Delete(ctx context.Context, id string) error
	BulkReplace(ctx context.Context, imageID string, edges []domain.Edge) ([]domain.Edge, error)
}

type edgeAnnotationRepository interface {
	Get(ctx context.Context, id string) (domain.Annotation, error)
	ListByImage(ctx context.Context, imageID string) ([]domain.Annotation, error)
}

type edgeLabelRepository interface {
	Get(ctx context.Context, id string) (domain.LabelDefinition, error)
}

// EdgeUsecase coordinates edge operations and graph validation.
type EdgeUsecase struct {
	edgeRepo       edgeRepository
	annotationRepo edgeAnnotationRepository
	labelRepo      edgeLabelRepository
}

// NewEdgeUsecase creates an EdgeUsecase.
func NewEdgeUsecase(edgeRepo edgeRepository, annotationRepo edgeAnnotationRepository, labelRepo edgeLabelRepository) *EdgeUsecase {
	return &EdgeUsecase{
		edgeRepo:       edgeRepo,
		annotationRepo: annotationRepo,
		labelRepo:      labelRepo,
	}
}

// Create creates a validated edge for an image.
func (u *EdgeUsecase) Create(ctx context.Context, imageID, sourceAnnotationID, targetAnnotationID string, edgeType domain.EdgeType) (domain.Edge, error) {
	edge := domain.Edge{
		ID:                 uuid.Must(uuid.NewV7()).String(),
		ImageID:            imageID,
		SourceAnnotationID: sourceAnnotationID,
		TargetAnnotationID: targetAnnotationID,
		Type:               edgeType,
	}

	existingEdges, err := u.edgeRepo.ListByImage(ctx, imageID)
	if err != nil {
		return domain.Edge{}, err
	}
	candidateEdges := append(existingEdges, edge)
	if err := u.validateEdgeSet(ctx, imageID, candidateEdges); err != nil {
		return domain.Edge{}, err
	}

	return u.edgeRepo.Create(ctx, edge)
}

// ListByImage returns edges for an image.
func (u *EdgeUsecase) ListByImage(ctx context.Context, imageID string) ([]domain.Edge, error) {
	return u.edgeRepo.ListByImage(ctx, imageID)
}

// Delete removes an edge by ID.
func (u *EdgeUsecase) Delete(ctx context.Context, id string) error {
	return u.edgeRepo.Delete(ctx, id)
}

// BulkReplace validates and replaces all edges for an image.
// Edges with an empty ID are treated as new records and receive UUID v7 IDs.
func (u *EdgeUsecase) BulkReplace(ctx context.Context, imageID string, edges []domain.Edge) ([]domain.Edge, error) {
	// Why: Edge編集UIは画像単位のグラフ全体を保存するため、永続化前に候補グラフ全体を検証する。
	// Why not: 不正な候補が混じる場合に既存グラフを一度消すと、ユーザーの作業状態を壊してしまう。
	candidateEdges := make([]domain.Edge, len(edges))
	for i, edge := range edges {
		if edge.ID == "" {
			edge.ID = uuid.Must(uuid.NewV7()).String()
		}
		edge.ImageID = imageID
		candidateEdges[i] = edge
	}

	if err := u.validateEdgeSet(ctx, imageID, candidateEdges); err != nil {
		return nil, err
	}

	return u.edgeRepo.BulkReplace(ctx, imageID, candidateEdges)
}
