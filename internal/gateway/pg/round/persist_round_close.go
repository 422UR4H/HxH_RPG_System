package round

import (
	"context"
	"fmt"

	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	roundentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	sceneentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
	"github.com/google/uuid"
)

// PersistRoundClose writes round ends and the round the session is in now, in one transaction:
// every end closed, then sc/next. Two callers in the room: a round that ends with no turn closing
// in the same command — an open_next_action with nothing open and no action pending that can still
// pay its price (ends = that round, next = its successor) — and any write of a round while an
// earlier round end is still unwritten (ends = those, next = the round being written). When a turn
// closes in the command, its own PersistTurnClose writes the same rows instead
// (TurnCloseData.NextRound, UnwrittenRoundEnds) — one master command, one transaction (owner
// decision, 2026-10-02).
//
// Every row goes through ensureSceneAndRound, the same idempotent upsert every scene/round write
// uses: on a closed round it IS the round-close SQL (a missing finished_at is filled, one already
// there never moves), and it also writes a round that was never a row — its birth write failed —
// already closed, instead of an UPDATE that would find nothing. Never a round born without its
// predecessors' ends: the scene never holds two open rounds.
func (r *Repository) PersistRoundClose(
	ctx context.Context, matchUUID uuid.UUID, ends []appmatch.RoundEnd, sc *sceneentity.Scene, next *roundentity.Round,
) error {
	if next == nil {
		return fmt.Errorf("PersistRoundClose: the round being written is required")
	}
	if err := validateRoundEnds(ends); err != nil {
		return fmt.Errorf("PersistRoundClose: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("PersistRoundClose begin tx: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
		_ = tx.Rollback(ctx) // no-op after Commit
	}()

	if err := writeRoundEnds(ctx, tx, matchUUID, ends); err != nil {
		return fmt.Errorf("PersistRoundClose %w", err)
	}
	if err := ensureSceneAndRound(ctx, tx, matchUUID, sc, next); err != nil {
		return fmt.Errorf("PersistRoundClose next round: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("PersistRoundClose commit: %w", err)
	}
	return nil
}

// validateRoundEnds refuses an end that is not one: a round with no finished_at written as an
// "end" would be one more open round, the very thing the ends are there to prevent.
func validateRoundEnds(ends []appmatch.RoundEnd) error {
	for i, e := range ends {
		if e.Scene == nil || e.Round == nil || e.Round.GetFinishedAt() == nil {
			return fmt.Errorf("round end %d needs its scene and a finished round", i)
		}
	}
	return nil
}

// writeRoundEnds writes each end closed, with its own scene, on q (a transaction).
func writeRoundEnds(ctx context.Context, q execer, matchUUID uuid.UUID, ends []appmatch.RoundEnd) error {
	for i, e := range ends {
		if err := ensureSceneAndRound(ctx, q, matchUUID, e.Scene, e.Round); err != nil {
			return fmt.Errorf("round end %d: %w", i, err)
		}
	}
	return nil
}
