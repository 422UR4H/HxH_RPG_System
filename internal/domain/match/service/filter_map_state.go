package service

import (
	"math"

	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	mapservice "github.com/422UR4H/HxH_RPG_System/internal/domain/map/service"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
	"github.com/google/uuid"
)

// Walls are tested by sampling along the segment rather than at a single point: a long
// wall is often only partly in view, and the midpoint may be the occluded part.
const wallSampleCount = 9

// wallNudge lifts a sample off the wall surface, toward the viewer, in world units.
// A vision-blocking wall lies exactly ON the boundary of the visibility polygon, where
// point-in-polygon is ambiguous — which is why testing the wall directly reports "not
// visible" and hides from the player the very wall that is blocking them. Offsetting
// toward the origin puts the sample just inside the lit region when that stretch of wall
// faces the viewer, and leaves it outside when the wall is occluded or facing away.
const wallNudge = 0.5

func wallSample(w mapentity.WallSegment, i int) (float64, float64) {
	t := float64(i) / float64(wallSampleCount-1)
	return w.P1[0] + (w.P2[0]-w.P1[0])*t, w.P1[1] + (w.P2[1]-w.P1[1])*t
}

// wallInLOS reports whether any stretch of the wall is directly visible from any of the
// viewer's vantage points.
func wallInLOS(w mapentity.WallSegment, polys []VisibilityPolygon) bool {
	for _, poly := range polys {
		for i := range wallSampleCount {
			px, py := wallSample(w, i)
			dx, dy := poly.Origin.X-px, poly.Origin.Y-py
			d := math.Hypot(dx, dy)
			if d < 1e-9 {
				return true // the viewer is standing on the wall
			}
			sx := px + dx/d*wallNudge
			sy := py + dy/d*wallNudge
			if PointInPolygon(Point2D{X: sx, Y: sy}, poly.Vertices) {
				return true
			}
		}
	}
	return false
}

// wallInExploredCells reports whether any stretch of the wall lies in a cell the viewer
// has already explored, so walls stay on screen after the player moves away.
func wallInExploredCells(
	w mapentity.WallSegment,
	explored map[fog.CellCoord]struct{},
	grid mapentity.GridShape,
) bool {
	for i := range wallSampleCount {
		px, py := wallSample(w, i)
		a, b := mapservice.WorldToSlot(px, py, grid)
		if _, ok := explored[fog.CellCoord{A: a, B: b}]; ok {
			return true
		}
	}
	return false
}

// PieceVisibility is the domain projection of a piece for filtering (no delivery types).
type PieceVisibility struct {
	ID          string
	CharacterID string
	Pos         Point2D
	Visible     bool
}

// FilterMapState applies visibility policy. Returns walls filtered/masked for the viewer
// and the set of piece IDs the viewer may see. Master gets everything unmasked.
func FilterMapState(
	allWalls []mapentity.WallSegment,
	pieces []PieceVisibility,
	polys []VisibilityPolygon,
	explored map[fog.CellCoord]struct{},
	fogMode fog.FogMode,
	grid mapentity.GridShape,
	playerID uuid.UUID,
	charToPlayer map[string]uuid.UUID,
	isMaster bool,
) (walls []mapentity.WallSegment, visiblePieceIDs map[string]bool) {
	visiblePieceIDs = make(map[string]bool, len(pieces))

	if isMaster {
		walls = append(walls, allWalls...)
		for _, p := range pieces {
			visiblePieceIDs[p.ID] = true
		}
		return walls, visiblePieceIDs
	}

	// Pieces: visible=false never; else in-LOS or own.
	for _, p := range pieces {
		if !p.Visible {
			continue
		}
		own := charToPlayer[p.CharacterID] == playerID && charToPlayer[p.CharacterID] != uuid.Nil
		if own || IsVisible(p.Pos, polys) {
			visiblePieceIDs[p.ID] = true
		}
	}

	// Walls: any part in LOS, or (explored mode) any part in an explored cell.
	for _, w := range allWalls {
		seen := wallInLOS(w, polys)
		if !seen && fogMode == fog.FogModeExplored && explored != nil {
			seen = wallInExploredCells(w, explored, grid)
		}
		if !seen {
			continue
		}
		if w.WallType == mapentity.WallTypeSecretDoor && !w.Revealed {
			walls = append(walls, MaskSecretDoorForPlayer(w))
		} else {
			walls = append(walls, w)
		}
	}
	return walls, visiblePieceIDs
}
