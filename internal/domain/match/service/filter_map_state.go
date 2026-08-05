package service

import (
	"math"

	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
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

// SeenWalls returns a memory reference for every wall currently in the viewer's line of
// sight, so the caller can union them into the player's PlayerMemory.
//
// INVARIANT — the predicate that reveals is the predicate that records.
// This uses the exact same wallInLOS as FilterMapState, in the same package, on purpose.
// If the two ever diverge, a player can see a wall that never reaches memory, and it
// vanishes the instant they step away — the false negative that the cell-based model
// suffered from. Guarded by TestSeenWallsAgreesWithFilterMapState.
//
// Pass the REAL walls here, never the LOS wall set: RecomputeVisibility appends
// BoundaryLOSWalls (ID == BoundaryWallID) to bound the sweep, and those are phantom
// board edges that must never enter a player's memory.
func SeenWalls(allWalls []mapentity.WallSegment, polys []VisibilityPolygon) []fog.FeatureRef {
	refs := make([]fog.FeatureRef, 0, len(allWalls))
	for _, w := range allWalls {
		if wallInLOS(w, polys) {
			refs = append(refs, fog.FeatureRef{Kind: fog.FeatureWall, ID: w.ID})
		}
	}
	return refs
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
	memory *fog.PlayerMemory,
	fogMode fog.FogMode,
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

	// Walls: any part in LOS now, or (explored mode) observed at some point in the past.
	for _, w := range allWalls {
		seen := wallInLOS(w, polys)
		if !seen && fogMode == fog.FogModeExplored {
			seen = memory.Has(fog.FeatureWall, w.ID)
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
