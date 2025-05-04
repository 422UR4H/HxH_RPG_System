package campaign

import (
	"context"
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
)

func (r *Repository) CreateCampaign(ctx context.Context, campaign *campaign.Campaign) error {
	tx, err := r.q.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback(ctx)
			// TODO: improve error handling
			panic(p)
		} else if err != nil {
			tx.Rollback(ctx)
		} else {
			tx.Commit(ctx)
		}
	}()

	const query = `
        INSERT INTO campaigns (
            uuid, user_uuid, scenario_uuid, name, brief_description, description, 
            story_start_at, story_current_at, story_end_at, created_at, updated_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
        )
    `
	_, err = tx.Exec(ctx, query,
		campaign.UUID, campaign.UserUUID, campaign.ScenarioUUID, campaign.Name, campaign.BriefDescription,
		campaign.Description, campaign.StoryStartAt, campaign.StoryCurrentAt, campaign.StoryEndAt,
		campaign.CreatedAt, campaign.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save campaign: %w", err)
	}

	return nil
}
