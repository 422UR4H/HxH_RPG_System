package matchboarduc

import (
	"context"

	"github.com/google/uuid"

	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchboard"
	matchmapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/matchmap/entity"
)

// IBoardRepository is the match board's own store: the line, if the match has already saved
// one (spec §4.3, "Ciclo de vida"). Satisfied by pgmatchboard.Repository.
type IBoardRepository interface {
	Get(ctx context.Context, matchUUID uuid.UUID) (*matchboard.Board, error)
}

// IMatchMapRepository tells which campaign map a match is attached to, if any. Satisfied by
// pgmatchmap.Repository.
type IMatchMapRepository interface {
	GetMatchMap(ctx context.Context, matchUUID uuid.UUID) (*matchmapentity.MatchMap, error)
}

// IMapRepository reads the campaign map a board with no saved line yet is a snapshot of.
// Satisfied by pgmap.Repository.
type IMapRepository interface {
	GetMap(ctx context.Context, mapUUID uuid.UUID) (*mapentity.TacticalMap, error)
}
