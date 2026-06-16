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
