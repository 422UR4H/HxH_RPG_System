package service_test

import (
	"sort"
	"testing"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/map/service"
)

func squareGrid() entity.GridShape {
	return entity.GridShape{Kind: entity.GridKindSquare, Cols: 10, Rows: 10, CellSize: 40, SkewRatio: 1}
}

func sorted(slots [][2]int) [][2]int {
	sort.Slice(slots, func(i, j int) bool {
		if slots[i][0] != slots[j][0] {
			return slots[i][0] < slots[j][0]
		}
		return slots[i][1] < slots[j][1]
	})
	return slots
}

func TestWallContactSlots(t *testing.T) {
	g := squareGrid()

	t.Run("a one-cell wall on a horizontal edge touches the cell above and the one below", func(t *testing.T) {
		// Edge between (3,4) and (3,5): y = 5 × 40, x from 3 × 40 to 4 × 40.
		w := entity.WallSegment{P1: [2]float64{120, 200}, P2: [2]float64{160, 200}}
		got := sorted(service.WallContactSlots(w, g))
		want := [][2]int{{3, 4}, {3, 5}}
		if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("a two-cell vertical wall touches both columns along it, not the corner cells", func(t *testing.T) {
		w := entity.WallSegment{P1: [2]float64{160, 80}, P2: [2]float64{160, 160}}
		got := sorted(service.WallContactSlots(w, g))
		want := [][2]int{{3, 2}, {3, 3}, {4, 2}, {4, 3}}
		if len(got) != len(want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("got %v, want %v", got, want)
			}
		}
	})

	t.Run("drops cells outside the map", func(t *testing.T) {
		w := entity.WallSegment{P1: [2]float64{0, 0}, P2: [2]float64{40, 0}}
		got := service.WallContactSlots(w, g)
		if len(got) != 1 || got[0] != [2]int{0, 0} {
			t.Errorf("got %v, want [[0 0]]", got)
		}
	})

	t.Run("a zero-length wall has no reach", func(t *testing.T) {
		w := entity.WallSegment{P1: [2]float64{40, 40}, P2: [2]float64{40, 40}}
		if got := service.WallContactSlots(w, g); len(got) != 0 {
			t.Errorf("got %v, want none", got)
		}
	})

	t.Run("reach ignores the grid's rotation and squash: walls and cells share local space", func(t *testing.T) {
		rotated := g
		rotated.Rotation = 45
		rotated.SkewRatio = 0.5
		w := entity.WallSegment{P1: [2]float64{120, 200}, P2: [2]float64{160, 200}}
		if !service.IsWallInReach(3, 4, w, rotated) || service.IsWallInReach(3, 2, w, rotated) {
			t.Error("reach must not depend on the transform")
		}
	})
}
