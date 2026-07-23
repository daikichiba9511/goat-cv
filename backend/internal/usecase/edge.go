package usecase

import (
	"context"
	"database/sql"
	"errors"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/daikichiba9511/goat-cv/backend/internal/repository/sqlite"
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

// EdgeUsecase coordinates edge operations and graph validation.
type EdgeUsecase struct {
	edgeRepo       *sqlite.EdgeRepository
	annotationRepo *sqlite.AnnotationRepository
	labelRepo      *sqlite.LabelRepository
}

// NewEdgeUsecase creates an EdgeUsecase.
func NewEdgeUsecase(edgeRepo *sqlite.EdgeRepository, annotationRepo *sqlite.AnnotationRepository, labelRepo *sqlite.LabelRepository) *EdgeUsecase {
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

func (u *EdgeUsecase) validateEdgeSet(ctx context.Context, imageID string, edges []domain.Edge) error {
	relations := make(map[edgeRelation]struct{}, len(edges))
	readingOrderNext := make(map[string][]string)
	keySources := make(map[string]string)
	keyTargets := make(map[string]string)
	tableCellTargets := make(map[string]string)
	categoryByAnnotationID := make(map[string]domain.LabelCategory)

	for _, edge := range edges {
		if !isValidEdgeType(edge.Type) {
			return ErrInvalidEdgeType
		}
		if edge.SourceAnnotationID == edge.TargetAnnotationID {
			return ErrSelfEdge
		}

		relation := edgeRelation{
			sourceAnnotationID: edge.SourceAnnotationID,
			targetAnnotationID: edge.TargetAnnotationID,
			edgeType:           edge.Type,
		}
		if _, ok := relations[relation]; ok {
			return ErrDuplicateEdge
		}
		relations[relation] = struct{}{}

		sourceAnnotation, err := u.annotationRepo.Get(ctx, edge.SourceAnnotationID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrEdgeAnnotationNotFound
			}
			return err
		}
		targetAnnotation, err := u.annotationRepo.Get(ctx, edge.TargetAnnotationID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrEdgeAnnotationNotFound
			}
			return err
		}
		if sourceAnnotation.ImageID != imageID || targetAnnotation.ImageID != imageID {
			return ErrEdgeImageMismatch
		}

		switch edge.Type {
		case domain.EdgeTypeReadingOrder:
			readingOrderNext[edge.SourceAnnotationID] = append(readingOrderNext[edge.SourceAnnotationID], edge.TargetAnnotationID)
		case domain.EdgeTypeKeyValue:
			if err := u.validateKeyValueEdge(ctx, sourceAnnotation, targetAnnotation, categoryByAnnotationID, keySources, keyTargets); err != nil {
				return err
			}
		case domain.EdgeTypeTableCell:
			if err := u.validateTableCellEdge(ctx, sourceAnnotation, targetAnnotation, categoryByAnnotationID, tableCellTargets); err != nil {
				return err
			}
		}
	}

	return rejectReadingOrderCycle(readingOrderNext)
}

func (u *EdgeUsecase) validateKeyValueEdge(
	ctx context.Context,
	sourceAnnotation domain.Annotation,
	targetAnnotation domain.Annotation,
	categoryByAnnotationID map[string]domain.LabelCategory,
	keySources map[string]string,
	keyTargets map[string]string,
) error {
	sourceCategory, ok, err := u.annotationCategory(ctx, sourceAnnotation, categoryByAnnotationID)
	if err != nil {
		return err
	}
	if !ok || sourceCategory != domain.LabelCategoryKey {
		return ErrInvalidEdgeCategory
	}
	targetCategory, ok, err := u.annotationCategory(ctx, targetAnnotation, categoryByAnnotationID)
	if err != nil {
		return err
	}
	if !ok || targetCategory != domain.LabelCategoryValue {
		return ErrInvalidEdgeCategory
	}

	if existingTargetID, ok := keySources[sourceAnnotation.ID]; ok && existingTargetID != targetAnnotation.ID {
		return ErrEdgeCardinalityViolation
	}
	if existingSourceID, ok := keyTargets[targetAnnotation.ID]; ok && existingSourceID != sourceAnnotation.ID {
		return ErrEdgeCardinalityViolation
	}
	keySources[sourceAnnotation.ID] = targetAnnotation.ID
	keyTargets[targetAnnotation.ID] = sourceAnnotation.ID
	return nil
}

func (u *EdgeUsecase) validateTableCellEdge(
	ctx context.Context,
	sourceAnnotation domain.Annotation,
	targetAnnotation domain.Annotation,
	categoryByAnnotationID map[string]domain.LabelCategory,
	tableCellTargets map[string]string,
) error {
	sourceCategory, ok, err := u.annotationCategory(ctx, sourceAnnotation, categoryByAnnotationID)
	if err != nil {
		return err
	}
	if !ok || sourceCategory != domain.LabelCategoryTable {
		return ErrInvalidEdgeCategory
	}
	targetCategory, ok, err := u.annotationCategory(ctx, targetAnnotation, categoryByAnnotationID)
	if err != nil {
		return err
	}
	if !ok || targetCategory != domain.LabelCategoryCell {
		return ErrInvalidEdgeCategory
	}

	if existingTableID, ok := tableCellTargets[targetAnnotation.ID]; ok && existingTableID != sourceAnnotation.ID {
		return ErrEdgeCardinalityViolation
	}
	tableCellTargets[targetAnnotation.ID] = sourceAnnotation.ID
	return nil
}

func (u *EdgeUsecase) annotationCategory(ctx context.Context, annotation domain.Annotation, categoryByAnnotationID map[string]domain.LabelCategory) (domain.LabelCategory, bool, error) {
	if annotation.LabelID == nil {
		return "", false, nil
	}
	if category, ok := categoryByAnnotationID[annotation.ID]; ok {
		return category, true, nil
	}
	label, err := u.labelRepo.Get(ctx, *annotation.LabelID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}
		return "", false, err
	}
	categoryByAnnotationID[annotation.ID] = label.Category
	return label.Category, true, nil
}

func rejectReadingOrderCycle(readingOrderNext map[string][]string) error {
	const (
		unvisited = 0
		visiting  = 1
		visited   = 2
	)
	visitStateByAnnotationID := make(map[string]int, len(readingOrderNext))

	var visit func(annotationID string) bool
	visit = func(annotationID string) bool {
		switch visitStateByAnnotationID[annotationID] {
		case visiting:
			return true
		case visited:
			return false
		}

		visitStateByAnnotationID[annotationID] = visiting
		for _, nextAnnotationID := range readingOrderNext[annotationID] {
			if visit(nextAnnotationID) {
				return true
			}
		}
		visitStateByAnnotationID[annotationID] = visited
		return false
	}

	for annotationID := range readingOrderNext {
		if visitStateByAnnotationID[annotationID] == unvisited && visit(annotationID) {
			return ErrReadingOrderCycle
		}
	}
	return nil
}

func isValidEdgeType(edgeType domain.EdgeType) bool {
	switch edgeType {
	case domain.EdgeTypeReadingOrder, domain.EdgeTypeKeyValue, domain.EdgeTypeTableCell:
		return true
	default:
		return false
	}
}

type edgeRelation struct {
	sourceAnnotationID string
	targetAnnotationID string
	edgeType           domain.EdgeType
}
