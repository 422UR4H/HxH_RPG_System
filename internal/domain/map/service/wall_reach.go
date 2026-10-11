package service

import (
	"math"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
)

// WallContactSlots returns the cells that TOUCH a wall — where a character stands to reach it,
// the way a piece reaches the cell next to it. Mirrors the front's wallContactSlots
// (tactical-map/utils/wallReach.ts), so the approach the front proposes is the reach the server
// accepts.
//
// It walks the segment in the grid's LOCAL space (where P1/P2 live) and, at each sample, looks a
// quarter cell to either side: a wall on a grid edge touches the cells on both sides of it; a
// wall cutting through cells touches those and their neighbours. The ends are inset a little:
// a wall ending on a grid vertex does not reach the cells that only share that corner.
func WallContactSlots(w entity.WallSegment, g entity.GridShape) [][2]int {
	x1, y1 := w.P1[0], w.P1[1]
	x2, y2 := w.P2[0], w.P2[1]
	length := math.Hypot(x2-x1, y2-y1)
	if length == 0 || g.CellSize <= 0 {
		return nil
	}
	nx, ny := -(y2-y1)/length, (x2-x1)/length
	side := g.CellSize / 4
	samples := int(math.Max(1, math.Ceil(length/(g.CellSize/4))))
	inset := math.Min(0.49, (g.CellSize*0.05)/length)

	var out [][2]int
	seen := map[[2]int]bool{}
	for i := 0; i <= samples; i++ {
		t := inset + (1-2*inset)*(float64(i)/float64(samples))
		px, py := x1+(x2-x1)*t, y1+(y2-y1)*t
		for _, s := range [2]float64{1, -1} {
			a, b := LocalToSlot(px+nx*side*s, py+ny*side*s, g)
			slot := [2]int{a, b}
			if SlotInGrid(a, b, g) && !seen[slot] {
				seen[slot] = true
				out = append(out, slot)
			}
		}
	}
	return out
}

// IsWallInReach reports whether a character standing on cell (a,b) touches the wall.
func IsWallInReach(a, b int, w entity.WallSegment, g entity.GridShape) bool {
	for _, s := range WallContactSlots(w, g) {
		if s[0] == a && s[1] == b {
			return true
		}
	}
	return false
}
