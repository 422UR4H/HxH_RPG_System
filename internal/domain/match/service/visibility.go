package service

import (
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
)

type Point2D struct{ X, Y float64 }

// VisibilityPolygon is the sweep result for one origin (one piece). World coords.
type VisibilityPolygon struct {
	Origin   Point2D
	Vertices []Point2D
}

// WallSegmentLOS is the minimal projection of a vision-blocking wall.
type WallSegmentLOS struct {
	ID        string
	P1, P2    Point2D
	Direction mapentity.WallDirection
}

// ToLOSWalls keeps only walls that block vision: excludes sense=none, destroyed, open.
func ToLOSWalls(walls []mapentity.WallSegment) []WallSegmentLOS {
	out := make([]WallSegmentLOS, 0, len(walls))
	for _, w := range walls {
		if w.Sense == mapentity.SenseNone || w.Destroyed || w.Open {
			continue
		}
		out = append(out, WallSegmentLOS{
			ID:        w.ID,
			P1:        Point2D{w.P1[0], w.P1[1]},
			P2:        Point2D{w.P2[0], w.P2[1]},
			Direction: w.Direction,
		})
	}
	return out
}

// PointInPolygon does ray casting; O(V).
func PointInPolygon(p Point2D, poly []Point2D) bool {
	inside := false
	n := len(poly)
	for i, j := 0, n-1; i < n; j, i = i, i+1 {
		yi, yj := poly[i].Y, poly[j].Y
		xi, xj := poly[i].X, poly[j].X
		if (yi > p.Y) != (yj > p.Y) &&
			p.X < (xj-xi)*(p.Y-yi)/(yj-yi)+xi {
			inside = !inside
		}
	}
	return inside
}

// IsVisible reports whether target lies in any of the player's polygons.
func IsVisible(target Point2D, polys []VisibilityPolygon) bool {
	for _, poly := range polys {
		if PointInPolygon(target, poly.Vertices) {
			return true
		}
	}
	return false
}
