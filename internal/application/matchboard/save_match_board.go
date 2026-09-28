package matchboarduc

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	fogentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchboard"
)

// ISaveMatchBoard is what the game server needs to persist a match's board and every
// player's fog memory in one call (spec §4.3, "Quando persiste", B3). Room.persistBoard is
// the ONLY caller — see its own doc comment for every moment that triggers a save.
type ISaveMatchBoard interface {
	// Save writes the board's line first. A failure there ABORTS before any memory is
	// attempted: a board row that failed to save has nothing for a memory to be consistent
	// WITH. Once the row is written, every memory is upserted regardless of whether an
	// earlier one failed — one player's failure must not cost every other player their fog
	// progress — and every failure collected is returned together via errors.Join.
	Save(ctx context.Context, b *matchboard.Board, memories []fogentity.PlayerMemory) error
}

// IMemoryLoader is what the game server needs to seed per-player fog memory when a session
// is born (spec §4.3; StartMatch and RehydrateSession call this BEFORE the session goes
// live, so a persisted row wins over an empty one). Satisfied by fog.PlayerMemoryRepository.
type IMemoryLoader interface {
	FindByMatchMap(ctx context.Context, matchID, mapID uuid.UUID) ([]fogentity.PlayerMemory, error)
}

// ISaveBoardRepository is the match board's write side. Satisfied by pgmatchboard.Repository
// (its Get and Delete are unused here — see ILoadMatchBoard's IBoardRepository for the read
// side this use case does NOT need).
type ISaveBoardRepository interface {
	Save(ctx context.Context, b *matchboard.Board) error
}

// IMemoryRepository is the write side of player memory persistence, one call per player.
// Satisfied by fog.PlayerMemoryRepository.
type IMemoryRepository interface {
	Upsert(ctx context.Context, m fogentity.PlayerMemory) error
}

// SaveMatchBoardUC implements ISaveMatchBoard.
type SaveMatchBoardUC struct {
	boardRepo  ISaveBoardRepository
	memoryRepo IMemoryRepository
}

func NewSaveMatchBoardUC(boardRepo ISaveBoardRepository, memoryRepo IMemoryRepository) *SaveMatchBoardUC {
	return &SaveMatchBoardUC{boardRepo: boardRepo, memoryRepo: memoryRepo}
}

// Save writes the board row, then upserts every memory — see ISaveMatchBoard's own doc
// comment for the abort/aggregate rule between the two.
func (uc *SaveMatchBoardUC) Save(ctx context.Context, b *matchboard.Board, memories []fogentity.PlayerMemory) error {
	if err := uc.boardRepo.Save(ctx, b); err != nil {
		return fmt.Errorf("save match board: %w", err)
	}

	var errs []error
	for i := range memories {
		if err := uc.memoryRepo.Upsert(ctx, memories[i]); err != nil {
			errs = append(errs, fmt.Errorf(
				"upsert player memory for player %s: %w", memories[i].PlayerID, err))
		}
	}
	return errors.Join(errs...)
}
