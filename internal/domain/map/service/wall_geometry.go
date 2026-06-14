package service

import (
	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
)

// IsPathBlocked reports whether the straight path from→to is blocked by any wall
// that has move=true and open=false. WallDirection is respected: "left" blocks
// only movement originating from the left side of the wall vector p1→p2; "right"
// from the right; "both" from either side.
func IsPathBlocked(from, to [2]float64, walls []entity.WallSegment) bool {
	for _, w := range walls {
		if !w.Move || w.Open {
			continue
		}
		if !segmentsIntersect(from, to, w.P1, w.P2) {
			continue
		}
		if w.Direction == entity.WallDirectionBoth {
			return true
		}
		// Cross product of wall vector (p2-p1) with (from-p1).
		// Positive → from is to the LEFT of the wall direction.
		wx := w.P2[0] - w.P1[0]
		wy := w.P2[1] - w.P1[1]
		fx := from[0] - w.P1[0]
		fy := from[1] - w.P1[1]
		cross := wx*fy - wy*fx
		if w.Direction == entity.WallDirectionLeft && cross > 0 {
			return true
		}
		if w.Direction == entity.WallDirectionRight && cross < 0 {
			return true
		}
	}
	return false
}

// segmentsIntersect reports whether segment AB intersects segment CD.
// Uses the cross-product sign test (standard computational geometry).
func segmentsIntersect(a, b, c, d [2]float64) bool {
	d1 := cross2(c, d, a)
	d2 := cross2(c, d, b)
	d3 := cross2(a, b, c)
	d4 := cross2(a, b, d)

	if ((d1 > 0 && d2 < 0) || (d1 < 0 && d2 > 0)) &&
		((d3 > 0 && d4 < 0) || (d3 < 0 && d4 > 0)) {
		return true
	}
	const eps = 1e-9
	if absF(d1) < eps && onSegment(c, d, a) {
		return true
	}
	if absF(d2) < eps && onSegment(c, d, b) {
		return true
	}
	if absF(d3) < eps && onSegment(a, b, c) {
		return true
	}
	if absF(d4) < eps && onSegment(a, b, d) {
		return true
	}
	return false
}

// cross2 returns the z-component of (b-a) × (p-a).
func cross2(a, b, p [2]float64) float64 {
	return (b[0]-a[0])*(p[1]-a[1]) - (b[1]-a[1])*(p[0]-a[0])
}

func onSegment(a, b, p [2]float64) bool {
	const eps = 1e-9
	minX, maxX := a[0], b[0]
	if minX > maxX {
		minX, maxX = maxX, minX
	}
	minY, maxY := a[1], b[1]
	if minY > maxY {
		minY, maxY = maxY, minY
	}
	return p[0] >= minX-eps && p[0] <= maxX+eps &&
		p[1] >= minY-eps && p[1] <= maxY+eps
}

func absF(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
