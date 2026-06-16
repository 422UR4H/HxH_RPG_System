package service

import (
	"math"

	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	mapservice "github.com/422UR4H/HxH_RPG_System/internal/domain/map/service"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
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

const visRadius = 1e7 // effective "infinite" ray length in world units

// blocksFor reports whether a one-way wall occludes for an origin on its given side.
// For Direction "both" it always blocks. For "left"/"right" it blocks only when the
// origin is on that side of the directed segment P1->P2 (sign of the 2D cross product).
func (w WallSegmentLOS) blocksFor(origin Point2D) bool {
	if w.Direction == mapentity.WallDirectionBoth || w.Direction == "" {
		return true
	}
	cross := (w.P2.X-w.P1.X)*(origin.Y-w.P1.Y) - (w.P2.Y-w.P1.Y)*(origin.X-w.P1.X)
	if w.Direction == mapentity.WallDirectionLeft {
		return cross > 0
	}
	return cross < 0 // right
}

// rayHit returns the distance t>=0 along the ray (origin + t*dir, dir unit) at which it
// first crosses segment [a,b], and whether it hits. Uses standard segment intersection.
func rayHit(origin, dir, a, b Point2D) (float64, bool) {
	// Ray: origin + t*dir, t in [0, +inf). Segment: a + u*(b-a), u in [0,1].
	sx, sy := b.X-a.X, b.Y-a.Y
	denom := dir.X*sy - dir.Y*sx
	if math.Abs(denom) < 1e-12 {
		return 0, false // parallel
	}
	ax, ay := a.X-origin.X, a.Y-origin.Y
	t := (ax*sy - ay*sx) / denom
	u := (ax*dir.Y - ay*dir.X) / denom
	if t >= 1e-9 && u >= -1e-9 && u <= 1+1e-9 {
		return t, true
	}
	return 0, false
}

// ComputeVisibilityPolygon performs an angular sweep from origin over the blocking walls.
func ComputeVisibilityPolygon(origin Point2D, walls []WallSegmentLOS) VisibilityPolygon {
	// Active blockers for this origin (respect one-way).
	active := make([]WallSegmentLOS, 0, len(walls))
	for _, w := range walls {
		if w.blocksFor(origin) {
			active = append(active, w)
		}
	}

	// Candidate angles: toward every endpoint, plus +/- epsilon to round corners.
	// Also seed the full circle with compass points so the polygon covers all visible area.
	const eps = 1e-5
	angles := make([]float64, 0, len(active)*6+8)
	// Compass anchors: ensure the polygon covers the entire 360° even when walls only
	// occupy a small angular slice. Without anchors, PointInPolygon finds an open polygon.
	for _, a := range [8]float64{-math.Pi, -3 * math.Pi / 4, -math.Pi / 2, -math.Pi / 4, 0, math.Pi / 4, math.Pi / 2, 3 * math.Pi / 4} {
		angles = append(angles, a)
	}
	for _, w := range active {
		for _, p := range [2]Point2D{w.P1, w.P2} {
			base := math.Atan2(p.Y-origin.Y, p.X-origin.X)
			angles = append(angles, base-eps, base, base+eps)
		}
	}

	sortFloats(angles)

	verts := make([]Point2D, 0, len(angles))
	for _, ang := range angles {
		dir := Point2D{math.Cos(ang), math.Sin(ang)}
		best := visRadius
		for _, w := range active {
			if t, ok := rayHit(origin, dir, w.P1, w.P2); ok && t < best {
				best = t
			}
		}
		verts = append(verts, Point2D{origin.X + dir.X*best, origin.Y + dir.Y*best})
	}
	return VisibilityPolygon{Origin: origin, Vertices: verts}
}

// sortFloats sorts ascending without pulling in sort.Slice closures in a hot path.
func sortFloats(a []float64) {
	for i := 1; i < len(a); i++ {
		v := a[i]
		j := i - 1
		for j >= 0 && a[j] > v {
			a[j+1] = a[j]
			j--
		}
		a[j+1] = v
	}
}

// CellsInPolygon returns the grid cells whose center falls inside poly.
// Bounds are derived from the polygon's world AABB mapped back to slot space.
func CellsInPolygon(poly VisibilityPolygon, grid mapentity.GridShape) []fog.CellCoord {
	if len(poly.Vertices) == 0 {
		return nil
	}
	minX, minY := poly.Vertices[0].X, poly.Vertices[0].Y
	maxX, maxY := minX, minY
	for _, v := range poly.Vertices {
		minX = math.Min(minX, v.X)
		minY = math.Min(minY, v.Y)
		maxX = math.Max(maxX, v.X)
		maxY = math.Max(maxY, v.Y)
	}
	// Convert the four AABB corners to slot space and take the integer envelope.
	aLo, bLo, aHi, bHi := slotEnvelope(minX, minY, maxX, maxY, grid)

	out := make([]fog.CellCoord, 0, 32)
	for a := aLo; a <= aHi; a++ {
		for b := bLo; b <= bHi; b++ {
			cx, cy := mapservice.SlotCenterToWorld(a, b, grid)
			if PointInPolygon(Point2D{cx, cy}, poly.Vertices) {
				out = append(out, fog.CellCoord{A: a, B: b})
			}
		}
	}
	return out
}

func slotEnvelope(minX, minY, maxX, maxY float64, g mapentity.GridShape) (aLo, bLo, aHi, bHi int) {
	corners := [4][2]float64{{minX, minY}, {maxX, minY}, {minX, maxY}, {maxX, maxY}}
	first := true
	for _, c := range corners {
		a, b := mapservice.WorldToSlot(c[0], c[1], g)
		if first {
			aLo, aHi, bLo, bHi, first = a, a, b, b, false
			continue
		}
		aLo, aHi = minInt(aLo, a), maxInt(aHi, a)
		bLo, bHi = minInt(bLo, b), maxInt(bHi, b)
	}
	// Pad by 1 to cover cells whose center lies just inside near the AABB edge.
	return aLo - 1, bLo - 1, aHi + 1, bHi + 1
}

func minInt(a, b int) int { if a < b { return a }; return b }
func maxInt(a, b int) int { if a > b { return a }; return b }
