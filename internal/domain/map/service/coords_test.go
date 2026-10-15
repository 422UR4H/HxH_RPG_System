package service

import (
	"math"
	"testing"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func TestSquareSlotToLocal_CellCenter(t *testing.T) {
	g := entity.GridShape{Kind: entity.GridKindSquare, CellSize: 64, SkewRatio: 1, Rotation: 0}
	x, y := SlotCenterToLocal(2, 1, g)
	if !approx(x, 2.5*64) || !approx(y, 1.5*64) {
		t.Fatalf("want (160,96), got (%v,%v)", x, y)
	}
}

func TestSquareLocalToSlot_RoundTrip(t *testing.T) {
	g := entity.GridShape{Kind: entity.GridKindSquare, CellSize: 64, SkewRatio: 1, Rotation: 0}
	x, y := SlotCenterToLocal(5, 3, g)
	a, b := LocalToSlot(x, y, g)
	if a != 5 || b != 3 {
		t.Fatalf("round trip want (5,3), got (%d,%d)", a, b)
	}
}

// The local space ignores rotation and skew: the cell center is the same on a rotated,
// flattened grid as on a flat one. Only the client transforms, and only to draw.
func TestSlotCenterToLocal_IgnoresRotationAndSkew(t *testing.T) {
	flat := entity.GridShape{Kind: entity.GridKindSquare, CellSize: 64, SkewRatio: 1}
	iso := entity.GridShape{Kind: entity.GridKindSquare, CellSize: 64, SkewRatio: 0.5, Rotation: 30}
	fx, fy := SlotCenterToLocal(4, 6, flat)
	ix, iy := SlotCenterToLocal(4, 6, iso)
	if !approx(fx, ix) || !approx(fy, iy) {
		t.Fatalf("local center moved with the transform: flat (%v,%v) vs iso (%v,%v)", fx, fy, ix, iy)
	}
	a, b := LocalToSlot(ix, iy, iso)
	if a != 4 || b != 6 {
		t.Fatalf("iso round trip want (4,6), got (%d,%d)", a, b)
	}
}

func TestHexSlotToLocal_RoundTrip(t *testing.T) {
	g := entity.GridShape{Kind: entity.GridKindHex, CellSize: 64, SkewRatio: 1, Rotation: 0}
	x, y := SlotCenterToLocal(3, -2, g) // (q=3, r=-2)
	a, b := LocalToSlot(x, y, g)
	if a != 3 || b != -2 {
		t.Fatalf("hex round trip want (3,-2), got (%d,%d)", a, b)
	}
}

// The hex numbers are the front's hexToPixel with size = cellSize (center-to-vertex).
func TestHexSlotCenterToLocal_MatchesFrontScale(t *testing.T) {
	g := entity.GridShape{Kind: entity.GridKindHex, CellSize: 64, SkewRatio: 1}
	hexW := 64 * math.Sqrt(3)
	cases := []struct {
		name         string
		q, r         int
		wantX, wantY float64
	}{
		{"origin", 0, 0, 0, 0},
		{"q=1", 1, 0, hexW, 0},
		{"r=1", 0, 1, hexW / 2, 96},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			x, y := SlotCenterToLocal(c.q, c.r, g)
			if !approx(x, c.wantX) || !approx(y, c.wantY) {
				t.Fatalf("(%d,%d) want (%v,%v), got (%v,%v)", c.q, c.r, c.wantX, c.wantY, x, y)
			}
		})
	}
}

// Mirrors gridLocalBounds in the front's tactical-map/utils/coords.ts.
func TestGridLocalBounds_HexMatchesFront(t *testing.T) {
	g := entity.GridShape{Kind: entity.GridKindHex, Cols: 3, Rows: 3, CellSize: 64, SkewRatio: 1}
	hexW := 64 * math.Sqrt(3)
	minX, minY, maxX, maxY := GridLocalBounds(g)
	wantMinX, wantMinY := -hexW/2, -64.0
	wantMaxX, wantMaxY := 2*hexW+hexW/2+hexW/2, 2*96.0+64
	if !approx(minX, wantMinX) || !approx(minY, wantMinY) || !approx(maxX, wantMaxX) || !approx(maxY, wantMaxY) {
		t.Fatalf("bounds want (%v,%v,%v,%v), got (%v,%v,%v,%v)",
			wantMinX, wantMinY, wantMaxX, wantMaxY, minX, minY, maxX, maxY)
	}
}

func TestGridLocalBounds_Square(t *testing.T) {
	g := entity.GridShape{Kind: entity.GridKindSquare, Cols: 4, Rows: 3, CellSize: 50, SkewRatio: 1}
	minX, minY, maxX, maxY := GridLocalBounds(g)
	if minX != 0 || minY != 0 || maxX != 200 || maxY != 150 {
		t.Fatalf("got (%v,%v,%v,%v)", minX, minY, maxX, maxY)
	}
}
