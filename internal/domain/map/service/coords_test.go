package service

import (
	"math"
	"testing"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func TestSquareSlotToWorld_CellCenter(t *testing.T) {
	g := entity.GridShape{Kind: entity.GridKindSquare, CellSize: 64, SkewRatio: 1, Rotation: 0}
	x, y := SlotCenterToWorld(2, 1, g)
	if !approx(x, 2.5*64) || !approx(y, 1.5*64) {
		t.Fatalf("want (160,96), got (%v,%v)", x, y)
	}
}

func TestSquareWorldToSlot_RoundTrip(t *testing.T) {
	g := entity.GridShape{Kind: entity.GridKindSquare, CellSize: 64, SkewRatio: 1, Rotation: 0}
	x, y := SlotCenterToWorld(5, 3, g)
	a, b := WorldToSlot(x, y, g)
	if a != 5 || b != 3 {
		t.Fatalf("round trip want (5,3), got (%d,%d)", a, b)
	}
}

func TestIsoSkew_RoundTrip(t *testing.T) {
	g := entity.GridShape{Kind: entity.GridKindSquare, CellSize: 64, SkewRatio: 0.5, Rotation: 30}
	x, y := SlotCenterToWorld(4, 6, g)
	a, b := WorldToSlot(x, y, g)
	if a != 4 || b != 6 {
		t.Fatalf("iso round trip want (4,6), got (%d,%d)", a, b)
	}
}

func TestHexSlotToWorld_RoundTrip(t *testing.T) {
	g := entity.GridShape{Kind: entity.GridKindHex, CellSize: 64, SkewRatio: 1, Rotation: 0}
	x, y := SlotCenterToWorld(3, -2, g) // (q=3, r=-2)
	a, b := WorldToSlot(x, y, g)
	if a != 3 || b != -2 {
		t.Fatalf("hex round trip want (3,-2), got (%d,%d)", a, b)
	}
}
