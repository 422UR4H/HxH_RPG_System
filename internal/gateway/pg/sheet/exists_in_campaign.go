package sheet

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (r *Repository) ExistsSheetInCampaign(
	ctx context.Context, playerUUID uuid.UUID, campaignUUID uuid.UUID,
) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM character_sheets
			WHERE player_uuid = $1 AND campaign_uuid = $2
		)
	`
	var exists bool
	err := r.q.QueryRow(ctx, query, playerUUID, campaignUUID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check sheet in campaign: %w", err)
	}
	return exists, nil
}
