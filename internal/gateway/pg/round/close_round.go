package round

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CloseRound sets finished_at on a single round row.
func (r *Repository) CloseRound(ctx context.Context, roundUUID uuid.UUID, at time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE rounds SET finished_at = $1 WHERE uuid = $2`,
		at, roundUUID,
	)
	if err != nil {
		return fmt.Errorf("CloseRound: %w", err)
	}
	return nil
}
