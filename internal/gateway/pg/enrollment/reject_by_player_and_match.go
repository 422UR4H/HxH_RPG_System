package enrollment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (r *Repository) RejectEnrollmentByPlayerAndMatch(
	ctx context.Context, playerUUID uuid.UUID, matchUUID uuid.UUID,
) error {
	const query = `
		UPDATE enrollments
		SET status = 'rejected'
		WHERE match_uuid = $1
		AND status = 'accepted'
		AND character_sheet_uuid IN (
			SELECT uuid FROM character_sheets
			WHERE player_uuid = $2 OR master_uuid = $2
		)
	`
	result, err := r.q.Exec(ctx, query, matchUUID, playerUUID)
	if err != nil {
		return fmt.Errorf("failed to reject enrollment by player and match: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrEnrollmentNotFound
	}
	return nil
}
