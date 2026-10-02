package round

import (
	"context"
	"fmt"

	roundentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	sceneentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
	"github.com/google/uuid"
)

// PersistRoundClose writes a round's end and its successor's birth in one transaction — the
// round that ends with no turn closing in the same command: an open_next_action with nothing open
// and nothing pending that can still pay its price. When a turn does close in that command, its
// own PersistTurnClose writes both (TurnCloseData.NextRound) instead — one master command, one
// transaction (owner decision, 2026-10-02).
//
// Both rows go through ensureSceneAndRound, the same idempotent upsert every scene/round write
// uses: on the closed round it IS the round-close SQL (a missing finished_at is filled, one
// already there never moves), and it also writes a round that was never a row — its birth write
// failed — already closed, instead of an UPDATE that would find nothing. Never one without the
// other: the scene never holds two open rounds, nor none.
func (r *Repository) PersistRoundClose(
	ctx context.Context, matchUUID uuid.UUID, sc *sceneentity.Scene, closed, next *roundentity.Round,
) error {
	if closed == nil || closed.GetFinishedAt() == nil || next == nil {
		return fmt.Errorf("PersistRoundClose: a finished round and its successor are required")
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

	if err := ensureSceneAndRound(ctx, tx, matchUUID, sc, closed); err != nil {
		return fmt.Errorf("PersistRoundClose closed round: %w", err)
	}
	if err := ensureSceneAndRound(ctx, tx, matchUUID, sc, next); err != nil {
		return fmt.Errorf("PersistRoundClose next round: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("PersistRoundClose commit: %w", err)
	}
	return nil
}
