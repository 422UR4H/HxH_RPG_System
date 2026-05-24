package match

import (
	"context"
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
)

func (r *Repository) UpdateMatch(ctx context.Context, m *match.Match) error {
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

	const query = `
		UPDATE matches SET
			title = $1,
			brief_initial_description = $2,
			description = $3,
			is_public = $4,
			game_scheduled_at = $5,
			story_start_at = $6,
			updated_at = $7
		WHERE uuid = $8 AND game_start_at IS NULL
	`
	result, err := tx.Exec(ctx, query,
		m.Title, m.BriefInitialDescription, m.Description,
		m.IsPublic, m.GameScheduledAt, m.StoryStartAt,
		m.UpdatedAt, m.UUID,
	)
	if err != nil {
		return fmt.Errorf("failed to update match: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrMatchNotFound
	}
	return tx.Commit(ctx)
}
