package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
	"github.com/google/uuid"
)

var (
	// ErrInvalidImageGraph indicates malformed client identities or annotation references in an image graph.
	ErrInvalidImageGraph = errors.New("invalid image graph")
	// ErrIncompleteImageGraphSave indicates a repository result missing a requested graph resource.
	ErrIncompleteImageGraphSave = errors.New("incomplete image graph save result")
)

// ImageGraphAnnotationInput describes an annotation and its request-local identity.
type ImageGraphAnnotationInput struct {
	ClientID    string
	ID          string
	Type        domain.AnnotationType
	Coordinates domain.Coordinates
	LabelID     *string
}

// ImageGraphEdgeInput describes an edge whose endpoints use request-local annotation identities.
type ImageGraphEdgeInput struct {
	ClientID                 string
	ID                       string
	SourceAnnotationClientID string
	TargetAnnotationClientID string
	Type                     domain.EdgeType
}

// ImageGraphInput contains the complete annotation graph for one image.
type ImageGraphInput struct {
	Annotations []ImageGraphAnnotationInput
	Edges       []ImageGraphEdgeInput
}

// SavedImageGraphAnnotation pairs a persisted annotation with its request-local identity.
type SavedImageGraphAnnotation struct {
	ClientID   string
	Annotation domain.Annotation
}

// SavedImageGraphEdge pairs a persisted edge with its request-local identity.
type SavedImageGraphEdge struct {
	ClientID string
	Edge     domain.Edge
}

// SavedImageGraph contains the persisted graph and explicit client-to-server identity mappings.
type SavedImageGraph struct {
	Annotations []SavedImageGraphAnnotation
	Edges       []SavedImageGraphEdge
}

type imageGraphRepository interface {
	Replace(ctx context.Context, imageID string, annotations []domain.Annotation, edges []domain.Edge) ([]domain.Annotation, []domain.Edge, error)
}

// ImageGraphUsecase validates and atomically saves the complete annotation graph for an image.
type ImageGraphUsecase struct {
	repository imageGraphRepository
	labelRepo  edgeLabelRepository
}

// NewImageGraphUsecase creates an ImageGraphUsecase.
func NewImageGraphUsecase(repository imageGraphRepository, labelRepo edgeLabelRepository) *ImageGraphUsecase {
	return &ImageGraphUsecase{repository: repository, labelRepo: labelRepo}
}

// Save resolves request-local identities, validates the complete graph, and persists it atomically.
func (u *ImageGraphUsecase) Save(ctx context.Context, imageID string, graph ImageGraphInput) (SavedImageGraph, error) {
	annotations, annotationClientIDs, annotationIDByClientID, err := prepareGraphAnnotations(imageID, graph.Annotations)
	if err != nil {
		return SavedImageGraph{}, err
	}
	edges, edgeClientIDs, err := prepareGraphEdges(imageID, graph.Edges, annotationIDByClientID)
	if err != nil {
		return SavedImageGraph{}, err
	}
	if err := validateEdgeSetAgainstAnnotations(ctx, imageID, edges, annotations, nil, u.labelRepo); err != nil {
		return SavedImageGraph{}, err
	}

	persistedAnnotations, persistedEdges, err := u.repository.Replace(ctx, imageID, annotations, edges)
	if err != nil {
		return SavedImageGraph{}, err
	}
	return pairSavedImageGraph(annotations, annotationClientIDs, edges, edgeClientIDs, persistedAnnotations, persistedEdges)
}

// prepareGraphAnnotations validates annotations and assigns persistent IDs without mutating the request.
func prepareGraphAnnotations(
	imageID string,
	inputs []ImageGraphAnnotationInput,
) ([]domain.Annotation, []string, map[string]string, error) {
	annotations := make([]domain.Annotation, len(inputs))
	clientIDs := make([]string, len(inputs))
	annotationIDByClientID := make(map[string]string, len(inputs))
	usedAnnotationIDs := make(map[string]struct{}, len(inputs))

	for inputIndex, input := range inputs {
		if input.ClientID == "" {
			return nil, nil, nil, fmt.Errorf("%w: annotations[%d].client_id is required", ErrInvalidImageGraph, inputIndex)
		}
		if _, exists := annotationIDByClientID[input.ClientID]; exists {
			return nil, nil, nil, fmt.Errorf("%w: duplicate annotation client_id %q", ErrInvalidImageGraph, input.ClientID)
		}
		if err := validateAnnotationCoordinates(input.Type, input.Coordinates); err != nil {
			return nil, nil, nil, fmt.Errorf("annotations[%d]: %w", inputIndex, err)
		}

		annotationID := input.ID
		if annotationID == "" {
			annotationID = uuid.Must(uuid.NewV7()).String()
		}
		if _, exists := usedAnnotationIDs[annotationID]; exists {
			return nil, nil, nil, fmt.Errorf("%w: duplicate annotation id %q", ErrInvalidImageGraph, annotationID)
		}

		annotationIDByClientID[input.ClientID] = annotationID
		usedAnnotationIDs[annotationID] = struct{}{}
		clientIDs[inputIndex] = input.ClientID
		annotations[inputIndex] = domain.Annotation{
			ID:          annotationID,
			ImageID:     imageID,
			Type:        input.Type,
			Coordinates: input.Coordinates,
			LabelID:     input.LabelID,
		}
	}
	return annotations, clientIDs, annotationIDByClientID, nil
}

// prepareGraphEdges resolves annotation client IDs and assigns persistent edge IDs.
func prepareGraphEdges(
	imageID string,
	inputs []ImageGraphEdgeInput,
	annotationIDByClientID map[string]string,
) ([]domain.Edge, []string, error) {
	edges := make([]domain.Edge, len(inputs))
	clientIDs := make([]string, len(inputs))
	usedClientIDs := make(map[string]struct{}, len(inputs))
	usedEdgeIDs := make(map[string]struct{}, len(inputs))

	for inputIndex, input := range inputs {
		if input.ClientID == "" {
			return nil, nil, fmt.Errorf("%w: edges[%d].client_id is required", ErrInvalidImageGraph, inputIndex)
		}
		if _, exists := usedClientIDs[input.ClientID]; exists {
			return nil, nil, fmt.Errorf("%w: duplicate edge client_id %q", ErrInvalidImageGraph, input.ClientID)
		}

		sourceAnnotationID, sourceExists := annotationIDByClientID[input.SourceAnnotationClientID]
		if !sourceExists {
			return nil, nil, fmt.Errorf("%w: edges[%d].source_annotation_client_id %q not found", ErrInvalidImageGraph, inputIndex, input.SourceAnnotationClientID)
		}
		targetAnnotationID, targetExists := annotationIDByClientID[input.TargetAnnotationClientID]
		if !targetExists {
			return nil, nil, fmt.Errorf("%w: edges[%d].target_annotation_client_id %q not found", ErrInvalidImageGraph, inputIndex, input.TargetAnnotationClientID)
		}

		edgeID := input.ID
		if edgeID == "" {
			edgeID = uuid.Must(uuid.NewV7()).String()
		}
		if _, exists := usedEdgeIDs[edgeID]; exists {
			return nil, nil, fmt.Errorf("%w: duplicate edge id %q", ErrInvalidImageGraph, edgeID)
		}

		usedClientIDs[input.ClientID] = struct{}{}
		usedEdgeIDs[edgeID] = struct{}{}
		clientIDs[inputIndex] = input.ClientID
		edges[inputIndex] = domain.Edge{
			ID:                 edgeID,
			ImageID:            imageID,
			SourceAnnotationID: sourceAnnotationID,
			TargetAnnotationID: targetAnnotationID,
			Type:               input.Type,
		}
	}
	return edges, clientIDs, nil
}

// pairSavedImageGraph joins repository results to client IDs through persistent IDs rather than response order.
func pairSavedImageGraph(
	requestedAnnotations []domain.Annotation,
	annotationClientIDs []string,
	requestedEdges []domain.Edge,
	edgeClientIDs []string,
	persistedAnnotations []domain.Annotation,
	persistedEdges []domain.Edge,
) (SavedImageGraph, error) {
	annotationByID := make(map[string]domain.Annotation, len(persistedAnnotations))
	for _, annotation := range persistedAnnotations {
		annotationByID[annotation.ID] = annotation
	}
	edgeByID := make(map[string]domain.Edge, len(persistedEdges))
	for _, edge := range persistedEdges {
		edgeByID[edge.ID] = edge
	}

	savedGraph := SavedImageGraph{
		Annotations: make([]SavedImageGraphAnnotation, len(requestedAnnotations)),
		Edges:       make([]SavedImageGraphEdge, len(requestedEdges)),
	}
	for annotationIndex, requestedAnnotation := range requestedAnnotations {
		persistedAnnotation, exists := annotationByID[requestedAnnotation.ID]
		if !exists {
			return SavedImageGraph{}, fmt.Errorf("%w: annotation %q", ErrIncompleteImageGraphSave, requestedAnnotation.ID)
		}
		savedGraph.Annotations[annotationIndex] = SavedImageGraphAnnotation{
			ClientID:   annotationClientIDs[annotationIndex],
			Annotation: persistedAnnotation,
		}
	}
	for edgeIndex, requestedEdge := range requestedEdges {
		persistedEdge, exists := edgeByID[requestedEdge.ID]
		if !exists {
			return SavedImageGraph{}, fmt.Errorf("%w: edge %q", ErrIncompleteImageGraphSave, requestedEdge.ID)
		}
		savedGraph.Edges[edgeIndex] = SavedImageGraphEdge{
			ClientID: edgeClientIDs[edgeIndex],
			Edge:     persistedEdge,
		}
	}
	return savedGraph, nil
}
