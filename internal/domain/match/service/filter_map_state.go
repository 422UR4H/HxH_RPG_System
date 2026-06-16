package service

import (
	"github.com/google/uuid"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	mapservice "github.com/422UR4H/HxH_RPG_System/internal/domain/map/service"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
)

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

	// Walls: in-LOS or (explored mode and midpoint cell explored).
	for _, w := range allWalls {
		mid := Point2D{(w.P1[0] + w.P2[0]) / 2, (w.P1[1] + w.P2[1]) / 2}
		seen := IsVisible(mid, polys)
		if !seen && fogMode == fog.FogModeExplored && explored != nil {
			a, b := mapservice.WorldToSlot(mid.X, mid.Y, grid)
			if _, ok := explored[fog.CellCoord{A: a, B: b}]; ok {
				seen = true
			}
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
