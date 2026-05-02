package enrollment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (r *Repository) RejectPendingEnrollments(
	ctx context.Context, matchUUID uuid.UUID,
) error {
	const query = `
		UPDATE enrollments
		SET status = 'rejected'
		WHERE match_uuid = $1 AND status = 'pending'
	`
	_, err := r.q.Exec(ctx, query, matchUUID)
	if err != nil {
		return fmt.Errorf("failed to reject pending enrollments: %w", err)
	}
	return nil
}
