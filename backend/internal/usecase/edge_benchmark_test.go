package usecase_test

import (
	"fmt"
	"testing"

	"github.com/daikichiba9511/goat-cv/backend/internal/domain"
)

// BenchmarkEdgeUsecaseBulkReplaceReadingOrder measures image-level graph validation and persistence.
func BenchmarkEdgeUsecaseBulkReplaceReadingOrder(b *testing.B) {
	fixture := newEdgeFixture(b)

	const annotationCount = 500
	annotationIDs := make([]string, annotationCount)
	for index := range annotationIDs {
		annotationID := fmt.Sprintf("benchmark-ann-%04d", index)
		annotationIDs[index] = annotationID
		execSQL(
			b,
			fixture.ctx,
			fixture.db,
			`INSERT INTO annotations (id, image_id, type, coordinates, label_id) VALUES (?, ?, ?, ?, ?)`,
			annotationID,
			testImageID,
			string(domain.AnnotationTypeBBox),
			`{"x":0,"y":0,"width":0.1,"height":0.1}`,
			"label-object",
		)
	}

	edges := make([]domain.Edge, annotationCount-1)
	for index := range edges {
		edges[index] = domain.Edge{
			SourceAnnotationID: annotationIDs[index],
			TargetAnnotationID: annotationIDs[index+1],
			Type:               domain.EdgeTypeReadingOrder,
		}
	}

	b.ReportAllocs()
	b.ReportMetric(float64(len(edges)), "edges/op")
	b.ResetTimer()
	for b.Loop() {
		if _, err := fixture.usecase.BulkReplace(fixture.ctx, testImageID, edges); err != nil {
			b.Fatalf("BulkReplace returned error: %v", err)
		}
	}
}
