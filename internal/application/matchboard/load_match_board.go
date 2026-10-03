package matchboarduc

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchboard"
	pgmatchmap "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/matchmap"
)

// ILoadMatchBoard is what the game server needs to load a match's board (spec §4.3, "Quem
// carrega", B14): the Room calls this when it is born and, while still a lobby, on every
// master (re)connect.
type ILoadMatchBoard interface {
	// Load returns the match's board, or (nil, nil) when the match has no map attached at all.
	Load(ctx context.Context, matchUUID uuid.UUID) (*matchboard.Board, error)
}

// LoadMatchBoardUC implements ILoadMatchBoard. It imports the gateway package ONLY for its
// sentinel error (pgmatchmap.ErrMatchMapNotFound), the same shape matchmapuc.GetMatchMapUC
// already uses — not for its Repository type, which IMatchMapRepository stands in for.
type LoadMatchBoardUC struct {
	boardRepo    IBoardRepository
	matchMapRepo IMatchMapRepository
	mapRepo      IMapRepository
}

func NewLoadMatchBoardUC(
	boardRepo IBoardRepository, matchMapRepo IMatchMapRepository, mapRepo IMapRepository,
) *LoadMatchBoardUC {
	return &LoadMatchBoardUC{boardRepo: boardRepo, matchMapRepo: matchMapRepo, mapRepo: mapRepo}
}

// Load returns the match's board: the saved line if one exists for the attached map, else a
// fresh snapshot of the attached map (read, never written — spec §4.3, "Ciclo de vida": "a linha nasce no primeiro
// salvamento"). (nil, nil) when the match has no map attached at all.
//
// The attach is checked FIRST, and the saved line is checked before ever reading the map: a
// saved row is what the match diverged into, and re-reading the map over it would erase that
// divergence — the map is a fallback for a board that was never saved, not a refresh of one
// that was.
func (uc *LoadMatchBoardUC) Load(ctx context.Context, matchUUID uuid.UUID) (*matchboard.Board, error) {
	mm, err := uc.matchMapRepo.GetMatchMap(ctx, matchUUID)
	if err != nil {
		if errors.Is(err, pgmatchmap.ErrMatchMapNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("load match board: get match map: %w", err)
	}

	mapUUID, err := uuid.Parse(mm.MapUUID)
	if err != nil {
		return nil, fmt.Errorf("load match board: parse attached map uuid %q: %w", mm.MapUUID, err)
	}

	board, err := uc.boardRepo.Get(ctx, matchUUID)
	if err != nil {
		return nil, fmt.Errorf("load match board: get board: %w", err)
	}
	// A saved row only counts if it is a portrait of the map attached NOW. Detach + attach
	// another map deletes nothing (AttachMatchMapUC only clears the board when the previous
	// attachment was a different map, and a detach leaves no previous one), so a row of the
	// old map can outlive its attachment; it reads as absent, and the new map's snapshot wins.
	if board != nil && board.MapUUID == mapUUID {
		return board, nil
	}

	m, err := uc.mapRepo.GetMap(ctx, mapUUID)
	if err != nil {
		return nil, fmt.Errorf("load match board: get map: %w", err)
	}
	return &matchboard.Board{
		MatchUUID: matchUUID,
		MapUUID:   mapUUID,
		Grid:      m.Grid,
		// Bg nil: a fresh snapshot inherits the map's background rather than copying it —
		// matchboard.Board.Bg's own doc comment (spec §4.3, "Estrutura").
		Bg:     nil,
		Pieces: m.Pieces,
		Walls:  m.Walls,
	}, nil
}
