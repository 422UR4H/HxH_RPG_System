package service

import (
	"math"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
)

// baseCenter returns the center of cell (a,b) in LOCAL (pre-transform) space. The server
// works only in this space: the grid's rotation/skew/origin are applied by the client, and
// only to draw.
func baseCenter(a, b int, g entity.GridShape) (float64, float64) {
	if g.Kind == entity.GridKindHex {
		// pointy-top axial → pixel (RedBlobGames), size = CellSize (center-to-vertex
		// radius), as the front draws it (hexToPixel(…, grid.cellSize)).
		size := g.CellSize
		x := size * math.Sqrt(3) * (float64(a) + float64(b)/2)
		y := size * 1.5 * float64(b)
		return x, y
	}
	return (float64(a) + 0.5) * g.CellSize, (float64(b) + 0.5) * g.CellSize
}

// GridLocalBounds returns the axis-aligned extent of the grid's cells in LOCAL
// (pre-transform) space. Square cells tile from the origin; hex cells are centered on
// their grid points, so the grid reaches half a hex-width left of x=0 and one hex
// radius above y=0. Derived from the same geometry as baseCenter.
func GridLocalBounds(g entity.GridShape) (minX, minY, maxX, maxY float64) {
	if g.Kind == entity.GridKindHex {
		size := g.CellSize
		hexW := size * math.Sqrt(3)
		hexH := size * 1.5
		maxCx := float64(g.Cols-1) * hexW
		if g.Rows >= 2 {
			maxCx += hexW / 2 // odd rows are shifted right by half a hex
		}
		return -hexW / 2, -size, maxCx + hexW/2, float64(g.Rows-1)*hexH + size
	}
	return 0, 0, float64(g.Cols) * g.CellSize, float64(g.Rows) * g.CellSize
}

// SlotCenterToLocal returns cell (a,b)'s center in the grid's LOCAL (pre-transform) space —
// the space WallSegment.P1/P2 live in. Anything measured against a wall measures here.
func SlotCenterToLocal(a, b int, g entity.GridShape) (float64, float64) {
	return baseCenter(a, b, g)
}

// LocalToSlot returns the cell (a,b) whose region contains LOCAL point (x,y).
func LocalToSlot(bx, by float64, g entity.GridShape) (int, int) {
	if g.Kind == entity.GridKindHex {
		size := g.CellSize
		q := (math.Sqrt(3)/3*bx - 1.0/3*by) / size
		r := (2.0 / 3 * by) / size
		return hexRound(q, r)
	}
	return int(math.Floor(bx / g.CellSize)), int(math.Floor(by / g.CellSize))
}

// hexRound rounds fractional axial coords to the nearest hex (cube rounding).
func hexRound(q, r float64) (int, int) {
	s := -q - r
	rq, rr, rs := math.Round(q), math.Round(r), math.Round(s)
	dq, dr, ds := math.Abs(rq-q), math.Abs(rr-r), math.Abs(rs-s)
	if dq > dr && dq > ds {
		rq = -rr - rs
	} else if dr > ds {
		rr = -rq - rs
	}
	return int(rq), int(rr)
}

// SlotInGrid reports whether (a,b) is a cell of the grid. Hex cells travel axial (q, r) while
// Cols/Rows count odd-r offset cells, so the column is q + floor(r/2) — the front's
// isSlotInBounds.
func SlotInGrid(a, b int, g entity.GridShape) bool {
	if b < 0 || b >= g.Rows {
		return false
	}
	col := a
	if g.Kind == entity.GridKindHex {
		col += b / 2 // b >= 0 here, so the truncation is the floor
	}
	return col >= 0 && col < g.Cols
}
