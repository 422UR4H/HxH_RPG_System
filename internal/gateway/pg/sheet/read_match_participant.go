package sheet

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (r *Repository) ExistsMatchParticipantForSheet(
	ctx context.Context, sheetUUID uuid.UUID,
) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM match_participants
			WHERE character_sheet_uuid = $1
		)
	`
	var exists bool
	err := r.q.QueryRow(ctx, query, sheetUUID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check match participation for sheet: %w", err)
	}
	return exists, nil
}
