package match

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (r *Repository) StartMatch(
	ctx context.Context, matchUUID uuid.UUID, gameStartAt time.Time,
) error {
	tx, err := r.q.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
		_ = tx.Rollback(ctx) // no-op after Commit
	}()

	now := time.Now()

	result, err := tx.Exec(ctx, `
		UPDATE matches
		SET game_start_at = $1, updated_at = $2
		WHERE uuid = $3 AND game_start_at IS NULL
	`, gameStartAt, now, matchUUID)
	if err != nil {
		return fmt.Errorf("failed to start match: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrMatchNotFound
	}

	if _, err = tx.Exec(ctx, `
		UPDATE enrollments
		SET status = 'rejected'
		WHERE match_uuid = $1 AND status = 'pending'
	`, matchUUID); err != nil {
		return fmt.Errorf("failed to reject pending enrollments: %w", err)
	}

	if _, err = tx.Exec(ctx, `
		INSERT INTO match_participants
			(uuid, match_uuid, character_sheet_uuid, joined_at, created_at, updated_at)
		SELECT gen_random_uuid(), match_uuid, character_sheet_uuid, $2, $3, $3
		FROM enrollments
		WHERE match_uuid = $1 AND status = 'accepted'
		ON CONFLICT (match_uuid, character_sheet_uuid) DO NOTHING
	`, matchUUID, gameStartAt, now); err != nil {
		return fmt.Errorf("failed to register match participants: %w", err)
	}

	return tx.Commit(ctx)
}
