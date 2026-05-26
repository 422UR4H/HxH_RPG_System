package campaign

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (r *Repository) DeleteCampaign(ctx context.Context, campaignUUID uuid.UUID) error {
	const query = `
		DELETE FROM campaigns WHERE uuid = $1
		AND NOT EXISTS (
			SELECT 1 FROM matches
			WHERE campaign_uuid = $1 AND game_start_at IS NOT NULL
		)`
	result, err := r.q.Exec(ctx, query, campaignUUID)
	if err != nil {
		return fmt.Errorf("failed to delete campaign: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrCampaignNotFound
	}
	return nil
}
