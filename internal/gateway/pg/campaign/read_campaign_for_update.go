package campaign

import (
	"context"
	"errors"
	"fmt"

	campaignEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetCampaignForUpdate(
	ctx context.Context, campaignUUID uuid.UUID,
) (*campaignEntity.CampaignUpdateContext, error) {
	const query = `
        SELECT
            c.master_uuid,
            c.name,
            COALESCE(c.brief_initial_description, ''),
            COALESCE(c.description, ''),
            c.is_public,
            COALESCE(c.call_link, ''),
            c.story_start_at,
            c.story_current_at,
            c.story_end_at,
            EXISTS(
                SELECT 1 FROM matches m
                WHERE m.campaign_uuid = c.uuid AND m.game_start_at IS NOT NULL
            ) AS has_started_match
        FROM campaigns c
        WHERE c.uuid = $1
    `
	var d campaignEntity.CampaignUpdateContext
	err := r.q.QueryRow(ctx, query, campaignUUID).Scan(
		&d.MasterUUID,
		&d.Name,
		&d.BriefInitialDescription,
		&d.Description,
		&d.IsPublic,
		&d.CallLink,
		&d.StoryStartAt,
		&d.StoryCurrentAt,
		&d.StoryEndAt,
		&d.HasStartedMatch,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCampaignNotFound
		}
		return nil, fmt.Errorf("failed to get campaign for update: %w", err)
	}
	return &d, nil
}
