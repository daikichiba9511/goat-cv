package usecase

import (
	"context"
	"database/sql"
	"errors"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
)

func (u *EdgeUsecase) validateEdgeSet(ctx context.Context, imageID string, edges []domain.Edge) error {
	if len(edges) == 0 {
		return nil
	}

	annotations, err := u.annotationRepo.ListByImage(ctx, imageID)
	if err != nil {
		return err
	}
	return validateEdgeSetAgainstAnnotations(ctx, imageID, edges, annotations, u.annotationRepo, u.labelRepo)
}

// validateEdgeSetAgainstAnnotations validates candidate edges against the annotation set saved with them.
func validateEdgeSetAgainstAnnotations(
	ctx context.Context,
	imageID string,
	edges []domain.Edge,
	annotations []domain.Annotation,
	annotationRepo edgeAnnotationRepository,
	labelRepo edgeLabelRepository,
) error {
	if len(edges) == 0 {
		return nil
	}

	annotationByID := make(map[string]domain.Annotation, len(annotations))
	for _, annotation := range annotations {
		annotationByID[annotation.ID] = annotation
	}

	validator := edgeSetValidator{
		annotationRepo:    annotationRepo,
		labelRepo:         labelRepo,
		imageID:           imageID,
		annotationByID:    annotationByID,
		relations:         make(map[edgeRelation]struct{}, len(edges)),
		readingOrderNext:  make(map[string][]string),
		keySources:        make(map[string]string),
		keyTargets:        make(map[string]string),
		tableCellTargets:  make(map[string]string),
		categoryByLabelID: make(map[string]domain.LabelCategory),
	}

	for _, edge := range edges {
		if err := validator.validate(ctx, edge); err != nil {
			return err
		}
	}
	return rejectReadingOrderCycle(validator.readingOrderNext)
}

type edgeSetValidator struct {
	annotationRepo    edgeAnnotationRepository
	labelRepo         edgeLabelRepository
	imageID           string
	annotationByID    map[string]domain.Annotation
	relations         map[edgeRelation]struct{}
	readingOrderNext  map[string][]string
	keySources        map[string]string
	keyTargets        map[string]string
	tableCellTargets  map[string]string
	categoryByLabelID map[string]domain.LabelCategory
}

func (v *edgeSetValidator) validate(ctx context.Context, edge domain.Edge) error {
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
	if _, exists := v.relations[relation]; exists {
		return ErrDuplicateEdge
	}
	v.relations[relation] = struct{}{}

	sourceAnnotation, err := v.annotation(ctx, edge.SourceAnnotationID)
	if err != nil {
		return err
	}
	targetAnnotation, err := v.annotation(ctx, edge.TargetAnnotationID)
	if err != nil {
		return err
	}

	switch edge.Type {
	case domain.EdgeTypeReadingOrder:
		v.readingOrderNext[edge.SourceAnnotationID] = append(v.readingOrderNext[edge.SourceAnnotationID], edge.TargetAnnotationID)
	case domain.EdgeTypeKeyValue:
		return v.validateKeyValue(ctx, sourceAnnotation, targetAnnotation)
	case domain.EdgeTypeTableCell:
		return v.validateTableCell(ctx, sourceAnnotation, targetAnnotation)
	}
	return nil
}

func (v *edgeSetValidator) annotation(ctx context.Context, annotationID string) (domain.Annotation, error) {
	if annotation, exists := v.annotationByID[annotationID]; exists {
		return annotation, nil
	}
	if v.annotationRepo == nil {
		return domain.Annotation{}, ErrEdgeAnnotationNotFound
	}

	// Why: 画像内一覧にないIDだけ個別検索し、存在しない参照と別画像への参照を区別する。
	annotation, err := v.annotationRepo.Get(ctx, annotationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Annotation{}, ErrEdgeAnnotationNotFound
		}
		return domain.Annotation{}, err
	}
	if annotation.ImageID != v.imageID {
		return domain.Annotation{}, ErrEdgeImageMismatch
	}
	v.annotationByID[annotationID] = annotation
	return annotation, nil
}

func (v *edgeSetValidator) validateKeyValue(ctx context.Context, sourceAnnotation, targetAnnotation domain.Annotation) error {
	sourceCategory, ok, err := v.annotationCategory(ctx, sourceAnnotation)
	if err != nil {
		return err
	}
	if !ok || sourceCategory != domain.LabelCategoryKey {
		return ErrInvalidEdgeCategory
	}
	targetCategory, ok, err := v.annotationCategory(ctx, targetAnnotation)
	if err != nil {
		return err
	}
	if !ok || targetCategory != domain.LabelCategoryValue {
		return ErrInvalidEdgeCategory
	}

	if existingTargetID, exists := v.keySources[sourceAnnotation.ID]; exists && existingTargetID != targetAnnotation.ID {
		return ErrEdgeCardinalityViolation
	}
	if existingSourceID, exists := v.keyTargets[targetAnnotation.ID]; exists && existingSourceID != sourceAnnotation.ID {
		return ErrEdgeCardinalityViolation
	}
	v.keySources[sourceAnnotation.ID] = targetAnnotation.ID
	v.keyTargets[targetAnnotation.ID] = sourceAnnotation.ID
	return nil
}

func (v *edgeSetValidator) validateTableCell(ctx context.Context, sourceAnnotation, targetAnnotation domain.Annotation) error {
	sourceCategory, ok, err := v.annotationCategory(ctx, sourceAnnotation)
	if err != nil {
		return err
	}
	if !ok || sourceCategory != domain.LabelCategoryTable {
		return ErrInvalidEdgeCategory
	}
	targetCategory, ok, err := v.annotationCategory(ctx, targetAnnotation)
	if err != nil {
		return err
	}
	if !ok || targetCategory != domain.LabelCategoryCell {
		return ErrInvalidEdgeCategory
	}

	if existingTableID, exists := v.tableCellTargets[targetAnnotation.ID]; exists && existingTableID != sourceAnnotation.ID {
		return ErrEdgeCardinalityViolation
	}
	v.tableCellTargets[targetAnnotation.ID] = sourceAnnotation.ID
	return nil
}

func (v *edgeSetValidator) annotationCategory(ctx context.Context, annotation domain.Annotation) (domain.LabelCategory, bool, error) {
	if annotation.LabelID == nil {
		return "", false, nil
	}
	if category, exists := v.categoryByLabelID[*annotation.LabelID]; exists {
		return category, true, nil
	}

	label, err := v.labelRepo.Get(ctx, *annotation.LabelID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}
		return "", false, err
	}
	v.categoryByLabelID[*annotation.LabelID] = label.Category
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
