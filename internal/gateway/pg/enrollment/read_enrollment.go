package enrollment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (r *Repository) ExistsEnrolledCharacterSheet(
	ctx context.Context, characterSheetUUID uuid.UUID, matchUUID uuid.UUID,
) (bool, error) {
	const query = `
        SELECT EXISTS (
            SELECT 1
            FROM enrollments
            WHERE character_sheet_uuid = $1 AND match_uuid = $2
        )
    `
	var exists bool
	err := r.q.QueryRow(ctx, query, characterSheetUUID, matchUUID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if character is enrolled in this match: %w", err)
	}
	return exists, nil
}
