package service

import (
	"testing"

	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
)

func wall(p1, p2 [2]float64, sense mapentity.SenseKind, open, destroyed bool) mapentity.WallSegment {
	return mapentity.WallSegment{
		ID: "w", P1: p1, P2: p2, Sense: sense, Open: open, Destroyed: destroyed,
		Direction: mapentity.WallDirectionBoth,
	}
}

func TestToLOSWalls_Excludes(t *testing.T) {
	walls := []mapentity.WallSegment{
		wall([2]float64{0, 0}, [2]float64{1, 0}, mapentity.SenseFull, false, false),  // keep
		wall([2]float64{0, 1}, [2]float64{1, 1}, mapentity.SenseSight, false, false), // keep
		wall([2]float64{0, 2}, [2]float64{1, 2}, mapentity.SenseNone, false, false),  // drop: none
		wall([2]float64{0, 3}, [2]float64{1, 3}, mapentity.SenseFull, true, false),   // drop: open
		wall([2]float64{0, 4}, [2]float64{1, 4}, mapentity.SenseFull, false, true),   // drop: destroyed
	}
	got := ToLOSWalls(walls)
	if len(got) != 2 {
		t.Fatalf("want 2 LOS walls, got %d", len(got))
	}
}

func TestPointInPolygon(t *testing.T) {
	square := []Point2D{{0, 0}, {10, 0}, {10, 10}, {0, 10}}
	if !PointInPolygon(Point2D{5, 5}, square) {
		t.Fatal("center should be inside")
	}
	if PointInPolygon(Point2D{15, 5}, square) {
		t.Fatal("outside point should be outside")
	}
}

func TestIsVisible_AnyPolygon(t *testing.T) {
	polys := []VisibilityPolygon{
		{Vertices: []Point2D{{0, 0}, {10, 0}, {10, 10}, {0, 10}}},
	}
	if !IsVisible(Point2D{5, 5}, polys) {
		t.Fatal("should be visible")
	}
	if IsVisible(Point2D{50, 50}, polys) {
		t.Fatal("should not be visible")
	}
}

func TestComputeVisibility_WallOccludes(t *testing.T) {
	// Vertical wall at x=10 from y=-5..5. Origin at (0,0).
	// A point behind the wall (20,0) must NOT be visible; (5,0) in front must be.
	walls := []WallSegmentLOS{
		{ID: "w", P1: Point2D{10, -5}, P2: Point2D{10, 5}, Direction: "both"},
	}
	poly := ComputeVisibilityPolygon(Point2D{0, 0}, walls, 1e4)
	if PointInPolygon(Point2D{20, 0}, poly.Vertices) {
		t.Fatal("point behind wall must be occluded")
	}
	if !PointInPolygon(Point2D{5, 0}, poly.Vertices) {
		t.Fatal("point in front of wall must be visible")
	}
}

func TestComputeVisibility_NoWalls_SeesFar(t *testing.T) {
	// maxRadius 1e4 is large enough to include (100,100) at distance ~141.
	poly := ComputeVisibilityPolygon(Point2D{0, 0}, nil, 1e4)
	if !PointInPolygon(Point2D{100, 100}, poly.Vertices) {
		t.Fatal("with no walls, far point should be visible")
	}
}

func TestComputeVisibility_OneWay_BlocksFromOneSide(t *testing.T) {
	// One-way wall blocking only its "left" side relative to P1->P2.
	// P1->P2 points +Y; left side (by (P2-P1) x (origin-P1) sign) is -X.
	// Origin on the blocking side should be occluded; on the other side, not.
	wallLeft := []WallSegmentLOS{
		{ID: "w", P1: Point2D{0, -5}, P2: Point2D{0, 5}, Direction: "left"},
	}
	// Origin at +X looking at the wall: depending on side it blocks or not.
	polyA := ComputeVisibilityPolygon(Point2D{10, 0}, wallLeft, 1e4)
	polyB := ComputeVisibilityPolygon(Point2D{-10, 0}, wallLeft, 1e4)
	behindFromA := PointInPolygon(Point2D{-1, 0}, polyA.Vertices)
	behindFromB := PointInPolygon(Point2D{1, 0}, polyB.Vertices)
	// Exactly one side must be blocked (XOR): the wall is one-way.
	if behindFromA == behindFromB {
		t.Fatalf("one-way wall must block from exactly one side (A=%v B=%v)", behindFromA, behindFromB)
	}
}

