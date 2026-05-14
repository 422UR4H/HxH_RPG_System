package round

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CloseSceneAndRound sets finished_at on both scene and round atomically.
func (r *Repository) CloseSceneAndRound(ctx context.Context, sceneUUID, roundUUID uuid.UUID, at time.Time) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("CloseSceneAndRound begin tx: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx,
		`UPDATE scenes SET finished_at = $1 WHERE uuid = $2`,
		at, sceneUUID,
	); err != nil {
		return fmt.Errorf("CloseSceneAndRound update scene: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`UPDATE rounds SET finished_at = $1 WHERE uuid = $2`,
		at, roundUUID,
	); err != nil {
		return fmt.Errorf("CloseSceneAndRound update round: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("CloseSceneAndRound commit: %w", err)
	}
	return nil
}
