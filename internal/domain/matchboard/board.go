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
	// Bg nil means "inherit the map's background"; the future in-match map editor writes
	// here.
	Bg        *mapentity.BgImage
	Pieces    []mapentity.Piece
	Walls     []mapentity.WallSegment
	UpdatedAt time.Time
}
