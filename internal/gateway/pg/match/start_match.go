package match

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (r *Repository) StartMatch(
	ctx context.Context, matchUUID uuid.UUID,
) error {
	now := time.Now()
	const query = `
		UPDATE matches
		SET game_start_at = $1, updated_at = $2
		WHERE uuid = $3 AND game_start_at IS NULL
	`
	result, err := r.q.Exec(ctx, query, now, now, matchUUID)
	if err != nil {
		return fmt.Errorf("failed to start match: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrMatchNotFound
	}
	return nil
}
