package fog

import (
	"time"

	"github.com/google/uuid"
)

// PlayerFogState is the accumulated explored area for one (player, match, map).
// GridKind is stored as a plain string to avoid an import cycle: map/entity imports fog,
// so fog must not import map/entity. Use the string literals from map/entity (e.g. "square").
type PlayerFogState struct {
	PlayerID      uuid.UUID
	MatchID       uuid.UUID
	MapID         uuid.UUID
	GridKind      string
	ExploredCells map[CellCoord]struct{}
	UpdatedAt     time.Time
}

func NewPlayerFogState(playerID, matchID, mapID uuid.UUID, gridKind string) *PlayerFogState {
	return &PlayerFogState{
		PlayerID:      playerID,
		MatchID:       matchID,
		MapID:         mapID,
		GridKind:      gridKind,
		ExploredCells: make(map[CellCoord]struct{}),
		UpdatedAt:     time.Now(),
	}
}

// AddExplored unions cells into the explored set and returns only the newly added
// cells (the delta). Order of the delta follows the input order.
func (s *PlayerFogState) AddExplored(cells []CellCoord) []CellCoord {
	delta := make([]CellCoord, 0, len(cells))
	for _, c := range cells {
		if _, ok := s.ExploredCells[c]; ok {
			continue
		}
		s.ExploredCells[c] = struct{}{}
		delta = append(delta, c)
	}
	if len(delta) > 0 {
		s.UpdatedAt = time.Now()
	}
	return delta
}
