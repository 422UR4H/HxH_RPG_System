package match

import (
	"context"
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
)

func (r *Repository) CreateMatch(ctx context.Context, match *match.Match) error {
	tx, err := r.q.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback(ctx)
			panic(p)
		} else if err != nil {
			tx.Rollback(ctx)
		} else {
			tx.Commit(ctx)
		}
	}()

	const query = `
        INSERT INTO matches (
            uuid, master_uuid, campaign_uuid, title, brief_description, description, 
            story_start_at, story_end_at, created_at, updated_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
        )
    `
	_, err = tx.Exec(ctx, query,
		match.UUID, match.MasterUUID, match.CampaignUUID, match.Title, match.BriefDescription,
		match.Description, match.StoryStartAt, match.StoryEndAt, match.CreatedAt, match.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save match: %w", err)
	}
	return nil
}
