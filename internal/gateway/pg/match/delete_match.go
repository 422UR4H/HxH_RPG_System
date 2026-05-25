package match

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (r *Repository) DeleteMatch(ctx context.Context, matchUUID uuid.UUID) error {
	const query = `DELETE FROM matches WHERE uuid = $1 AND game_start_at IS NULL`
	result, err := r.q.Exec(ctx, query, matchUUID)
	if err != nil {
		return fmt.Errorf("failed to delete match: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrMatchNotFound
	}
	return nil
}
