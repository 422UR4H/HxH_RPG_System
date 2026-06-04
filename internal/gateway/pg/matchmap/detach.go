package pgmatchmap

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (r *Repository) DetachMap(ctx context.Context, matchUUID uuid.UUID) error {
	const query = `DELETE FROM match_maps WHERE match_uuid = $1`
	tag, err := r.q.Exec(ctx, query, matchUUID)
	if err != nil {
		return fmt.Errorf("detach match map: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrMatchMapNotFound
	}
	return nil
}
