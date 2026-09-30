// internal/application/matchmap/i_match_board_repository.go
package matchmapuc

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchboard"
	"github.com/google/uuid"
)

// IMatchBoardRepository is the slice of pgmatchboard.Repository AttachMatchMapUC needs for
// B16 (spec §4.3): reading the source match's board to validate inheritance, copying it
// into the destination match, or dropping the destination's old board when a DIFFERENT map
// is attached without inheritance.
type IMatchBoardRepository interface {
	// Get returns the match's board row, or (nil, nil) when it has none yet.
	Get(ctx context.Context, matchUUID uuid.UUID) (*matchboard.Board, error)
	// Copy replaces dst's board (and player memories) with src's — see pgmatchboard.Repository.Copy.
	Copy(ctx context.Context, src, dst uuid.UUID) error
	// Delete removes the match's board row, if any.
	Delete(ctx context.Context, matchUUID uuid.UUID) error
}
