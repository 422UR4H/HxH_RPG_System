package service_test

import (
	"testing"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/map/service"
)

func wall(p1, p2 [2]float64) entity.WallSegment {
	return entity.WallSegment{
		ID:        "w",
		P1:        p1,
		P2:        p2,
		WallType:  entity.WallTypeWall,
		Move:      true,
		Direction: entity.WallDirectionBoth,
	}
}

func TestIsPathBlocked_NoWalls(t *testing.T) {
	if service.IsPathBlocked([2]float64{0, 0}, [2]float64{100, 0}, nil) {
		t.Error("expected not blocked with no walls")
	}
}

func TestIsPathBlocked_CrossingWall(t *testing.T) {
	// Vertical wall at x=50, y=0..100; path goes horizontally through it.
	w := wall([2]float64{50, 0}, [2]float64{50, 100})
	if !service.IsPathBlocked([2]float64{0, 50}, [2]float64{100, 50}, []entity.WallSegment{w}) {
		t.Error("expected path to be blocked by crossing wall")
	}
}

func TestIsPathBlocked_ParallelWall(t *testing.T) {
	w := wall([2]float64{0, 20}, [2]float64{100, 20})
	if service.IsPathBlocked([2]float64{0, 50}, [2]float64{100, 50}, []entity.WallSegment{w}) {
		t.Error("expected parallel wall not to block")
	}
}

func TestIsPathBlocked_OpenDoor(t *testing.T) {
	w := wall([2]float64{50, 0}, [2]float64{50, 100})
	w.WallType = entity.WallTypeDoor
	w.Open = true
	if service.IsPathBlocked([2]float64{0, 50}, [2]float64{100, 50}, []entity.WallSegment{w}) {
		t.Error("expected open door not to block movement")
	}
}

func TestIsPathBlocked_MoveFalse(t *testing.T) {
	w := wall([2]float64{50, 0}, [2]float64{50, 100})
	w.Move = false
	if service.IsPathBlocked([2]float64{0, 50}, [2]float64{100, 50}, []entity.WallSegment{w}) {
		t.Error("expected wall with move=false not to block")
	}
}

// Wall vector p1→p2 is (50,0)→(50,100), i.e. pointing downward.
// Cross product of wall vector with (from - p1): positive → from is to the LEFT of p1→p2.
// direction=left means it only blocks movement originating from the left side (cross > 0).

func TestIsPathBlocked_DirectionLeft_FromRight(t *testing.T) {
	// from=(100,50) is to the RIGHT of the wall vector → NOT blocked
	w := wall([2]float64{50, 0}, [2]float64{50, 100})
	w.Direction = entity.WallDirectionLeft
	if service.IsPathBlocked([2]float64{100, 50}, [2]float64{0, 50}, []entity.WallSegment{w}) {
		t.Error("direction=left wall should not block from the right side")
	}
}

func TestIsPathBlocked_DirectionLeft_FromLeft(t *testing.T) {
	// from=(0,50) is to the LEFT of the wall vector → blocked
	w := wall([2]float64{50, 0}, [2]float64{50, 100})
	w.Direction = entity.WallDirectionLeft
	if !service.IsPathBlocked([2]float64{0, 50}, [2]float64{100, 50}, []entity.WallSegment{w}) {
		t.Error("direction=left wall should block from the left side")
	}
}
