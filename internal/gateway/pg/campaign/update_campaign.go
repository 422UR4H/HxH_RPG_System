package campaign

import (
	"context"
	"fmt"

	campaignEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
)

func (r *Repository) UpdateCampaign(ctx context.Context, c *campaignEntity.Campaign) error {
	tx, err := r.q.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
		_ = tx.Rollback(ctx)
	}()

	const query = `
        UPDATE campaigns SET
            name = $1,
            brief_initial_description = $2,
            description = $3,
            is_public = $4,
            call_link = $5,
            story_start_at = $6,
            story_current_at = $7,
            updated_at = $8
        WHERE uuid = $9
    `
	result, err := tx.Exec(ctx, query,
		c.Name, c.BriefInitialDescription, c.Description,
		c.IsPublic, c.CallLink, c.StoryStartAt, c.StoryCurrentAt,
		c.UpdatedAt, c.UUID,
	)
	if err != nil {
		return fmt.Errorf("failed to update campaign: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrCampaignNotFound
	}
	return tx.Commit(ctx)
}
