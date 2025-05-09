package campaign

import (
	"context"
	"errors"
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetCampaign(ctx context.Context, uuid uuid.UUID) (*campaign.Campaign, error) {
	tx, err := r.q.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
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
        SELECT 
            uuid, user_uuid, scenario_uuid, name, brief_description, description,
            story_start_at, story_current_at, story_end_at, created_at, updated_at
        FROM campaigns
        WHERE uuid = $1
    `
	var c campaign.Campaign
	err = tx.QueryRow(ctx, query, uuid).Scan(
		&c.UUID,
		&c.UserUUID,
		&c.ScenarioUUID,
		&c.Name,
		&c.BriefDescription,
		&c.Description,
		&c.StoryStartAt,
		&c.StoryCurrentAt,
		&c.StoryEndAt,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCampaignNotFound
		}
		return nil, fmt.Errorf("failed to fetch campaign: %w", err)
	}

	return &c, nil
}

func (r *Repository) ListCampaignsByUserUUID(
	ctx context.Context, userUUID uuid.UUID) ([]*campaign.Summary, error) {

	tx, err := r.q.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
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
					SELECT 
							uuid, scenario_uuid, name, brief_description, 
							story_start_at, story_current_at, story_end_at,
							created_at, updated_at
					FROM campaigns
					WHERE user_uuid = $1
					ORDER BY name ASC
	`
	rows, err := tx.Query(ctx, query, userUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch campaigns summary: %w", err)
	}
	defer rows.Close()

	var campaigns []*campaign.Summary
	for rows.Next() {
		var c campaign.Summary
		err := rows.Scan(
			&c.UUID,
			&c.ScenarioUUID,
			&c.Name,
			&c.BriefDescription,
			&c.StoryStartAt,
			&c.StoryCurrentAt,
			&c.StoryEndAt,
			&c.CreatedAt,
			&c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan campaign summary: %w", err)
		}
		campaigns = append(campaigns, &c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over campaigns summary: %w", err)
	}

	return campaigns, nil
}

func (r *Repository) ExistsCampaign(ctx context.Context, campaignUUID uuid.UUID) (bool, error) {
	const query = `
        SELECT EXISTS(SELECT 1 FROM campaigns WHERE uuid = $1)
    `
	var exists bool
	err := r.q.QueryRow(ctx, query, campaignUUID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if campaign exists: %w", err)
	}
	return exists, nil
}
