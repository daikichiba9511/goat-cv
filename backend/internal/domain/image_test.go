package domain_test

import (
	"testing"

	"github.com/chibadaimare/goat/backend/internal/domain"
)

func TestEffectiveDimensions(t *testing.T) {
	tests := []struct {
		name           string
		originalWidth  int
		originalHeight int
		rotation       domain.Rotation
		wantWidth      int
		wantHeight     int
	}{
		{
			name:           "0 degrees keeps original",
			originalWidth:  2480,
			originalHeight: 3508,
			rotation:       domain.Rotation0,
			wantWidth:      2480,
			wantHeight:     3508,
		},
		{
			name:           "90 degrees swaps width and height",
			originalWidth:  2480,
			originalHeight: 3508,
			rotation:       domain.Rotation90,
			wantWidth:      3508,
			wantHeight:     2480,
		},
		{
			name:           "180 degrees keeps original",
			originalWidth:  2480,
			originalHeight: 3508,
			rotation:       domain.Rotation180,
			wantWidth:      2480,
			wantHeight:     3508,
		},
		{
			name:           "270 degrees swaps width and height",
			originalWidth:  2480,
			originalHeight: 3508,
			rotation:       domain.Rotation270,
			wantWidth:      3508,
			wantHeight:     2480,
		},
		{
			name:           "square image unchanged by rotation",
			originalWidth:  1000,
			originalHeight: 1000,
			rotation:       domain.Rotation90,
			wantWidth:      1000,
			wantHeight:     1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotW, gotH := domain.EffectiveDimensions(tt.originalWidth, tt.originalHeight, tt.rotation)
			if gotW != tt.wantWidth || gotH != tt.wantHeight {
				t.Errorf("EffectiveDimensions(%d, %d, %d) = (%d, %d), want (%d, %d)",
					tt.originalWidth, tt.originalHeight, tt.rotation,
					gotW, gotH, tt.wantWidth, tt.wantHeight)
			}
		})
	}
}
