package enrollment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (r *Repository) IsUserEnrolledInMatch(
	ctx context.Context, userUUID, matchUUID uuid.UUID,
) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM enrollments e
			JOIN character_sheets cs ON cs.uuid = e.character_sheet_uuid
			WHERE e.match_uuid = $1
			AND (cs.player_uuid = $2 OR cs.master_uuid = $2)
		)
	`
	var exists bool
	err := r.q.QueryRow(ctx, query, matchUUID, userUUID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if user is enrolled in match: %w", err)
	}
	return exists, nil
}
