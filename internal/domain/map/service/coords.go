package service

import (
	"math"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
)

// applyTransform applies the grid's iso skew (Y scaling) then rotation, in that order.
// Mirrors the frontend utils/coords.ts applyTransform.
func applyTransform(x, y float64, g entity.GridShape) (float64, float64) {
	ys := y * g.SkewRatio
	if g.Rotation == 0 {
		return x, ys
	}
	rad := g.Rotation * math.Pi / 180
	cos, sin := math.Cos(rad), math.Sin(rad)
	return x*cos - ys*sin, x*sin + ys*cos
}

// reverseTransform inverts applyTransform.
func reverseTransform(x, y float64, g entity.GridShape) (float64, float64) {
	rx, ry := x, y
	if g.Rotation != 0 {
		rad := g.Rotation * math.Pi / 180
		cos, sin := math.Cos(rad), math.Sin(rad)
		rx = x*cos + y*sin
		ry = -x*sin + y*cos
	}
	if g.SkewRatio != 0 {
		ry /= g.SkewRatio
	}
	return rx, ry
}

// baseCenter returns the untransformed world center of cell (a,b).
func baseCenter(a, b int, g entity.GridShape) (float64, float64) {
	if g.Kind == entity.GridKindHex {
		// pointy-top axial → pixel (RedBlobGames), size = CellSize/2.
		size := g.CellSize / 2
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
		size := g.CellSize / 2
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

// GridWorldCorners returns the grid's bounding quad in world space, wound TL→TR→BR→BL.
// Under rotation the quad is no longer axis-aligned, which is why it is returned as four
// points rather than a rectangle.
func GridWorldCorners(g entity.GridShape) [4][2]float64 {
	minX, minY, maxX, maxY := GridLocalBounds(g)
	local := [4][2]float64{{minX, minY}, {maxX, minY}, {maxX, maxY}, {minX, maxY}}
	var out [4][2]float64
	for i, p := range local {
		x, y := applyTransform(p[0], p[1], g)
		out[i] = [2]float64{x, y}
	}
	return out
}

// SlotCenterToWorld returns the transformed world position of cell (a,b)'s center.
func SlotCenterToWorld(a, b int, g entity.GridShape) (float64, float64) {
	bx, by := baseCenter(a, b, g)
	return applyTransform(bx, by, g)
}

// WorldToSlot returns the cell (a,b) whose region contains world point (x,y).
func WorldToSlot(x, y float64, g entity.GridShape) (int, int) {
	bx, by := reverseTransform(x, y, g)
	if g.Kind == entity.GridKindHex {
		size := g.CellSize / 2
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
