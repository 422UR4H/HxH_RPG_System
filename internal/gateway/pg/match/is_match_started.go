package match

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (r *Repository) IsStarted(ctx context.Context, matchUUID uuid.UUID) (bool, error) {
	const query = `
		SELECT EXISTS(
			SELECT 1 FROM matches
			WHERE uuid = $1 AND game_start_at IS NOT NULL
		)
	`
	var started bool
	if err := r.q.QueryRow(ctx, query, matchUUID).Scan(&started); err != nil {
		return false, fmt.Errorf("failed to check if match started: %w", err)
	}
	return started, nil
}
