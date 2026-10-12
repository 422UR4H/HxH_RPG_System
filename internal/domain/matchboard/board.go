// internal/domain/matchboard/board.go
package matchboard

import (
	"time"

	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/google/uuid"
)

// Board is the match's own board: a SNAPSHOT that starts as a copy of the campaign map it
// was attached to and then belongs to the match alone. Editing the campaign map never
// reaches it, and two matches on one map never touch each other's pieces
// (front-combat-phases.md §6A.5, B3).
type Board struct {
	MatchUUID uuid.UUID
	MapUUID   uuid.UUID
	Grid      mapentity.GridShape
	// Bg is part of the portrait, like Grid: a fresh snapshot copies the map's. nil means
	// the board has no background, frozen — never "inherit the map's".
	Bg        *mapentity.BgImage
	Pieces    []mapentity.Piece
	Walls     []mapentity.WallSegment
	UpdatedAt time.Time
}
