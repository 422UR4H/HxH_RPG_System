package service_test

import (
	"testing"

	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	mapservice "github.com/422UR4H/HxH_RPG_System/internal/domain/map/service"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
)

// The board edge is the LOCAL rectangle (TL→TR→BR→BL) whatever the grid's kind, rotation or
// skew: the server never leaves LOCAL space, the client transforms to draw.
func TestBoundaryLOSWalls_IsTheLocalRectangle(t *testing.T) {
	cases := []struct {
		name string
		grid mapentity.GridShape
	}{
		{"square", mapentity.GridShape{Kind: mapentity.GridKindSquare, Cols: 5, Rows: 4, CellSize: 64, SkewRatio: 1}},
		{"hex", mapentity.GridShape{Kind: mapentity.GridKindHex, Cols: 5, Rows: 4, CellSize: 64, SkewRatio: 1}},
		{"rotated and flattened", mapentity.GridShape{Kind: mapentity.GridKindSquare, Cols: 5, Rows: 4, CellSize: 64, SkewRatio: 0.6, Rotation: 30}},
		{"rotated hex", mapentity.GridShape{Kind: mapentity.GridKindHex, Cols: 5, Rows: 4, CellSize: 64, SkewRatio: 0.6, Rotation: 30}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			minX, minY, maxX, maxY := mapservice.GridLocalBounds(c.grid)
			want := [4][2]float64{{minX, minY}, {maxX, minY}, {maxX, maxY}, {minX, maxY}}
			walls := service.BoundaryLOSWalls(c.grid)
			if len(walls) != 4 {
				t.Fatalf("want 4 edges, got %d", len(walls))
			}
			for i, w := range walls {
				a, b := want[i], want[(i+1)%4]
				if !near(w.P1.X, a[0]) || !near(w.P1.Y, a[1]) || !near(w.P2.X, b[0]) || !near(w.P2.Y, b[1]) {
					t.Errorf("edge %d = (%v,%v)→(%v,%v), want (%v,%v)→(%v,%v)",
						i, w.P1.X, w.P1.Y, w.P2.X, w.P2.Y, a[0], a[1], b[0], b[1])
				}
			}
		})
	}
}

func near(a, b float64) bool { d := a - b; return d < 1e-6 && d > -1e-6 }
