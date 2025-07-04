package match

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetMatch(
	ctx context.Context, uuid uuid.UUID,
) (*match.Match, error) {
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
            uuid, master_uuid, campaign_uuid,
						title, brief_initial_description, brief_final_description, description,
						is_public, game_start_at,
            story_start_at, story_end_at,
						created_at, updated_at
        FROM matches
        WHERE uuid = $1
    `
	var m match.Match
	err = tx.QueryRow(ctx, query, uuid).Scan(
		&m.UUID,
		&m.MasterUUID,
		&m.CampaignUUID,
		&m.Title,
		&m.BriefInitialDescription,
		&m.BriefFinalDescription,
		&m.Description,
		&m.IsPublic,
		&m.GameStartAt,
		&m.StoryStartAt,
		&m.StoryEndAt,
		&m.CreatedAt,
		&m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMatchNotFound
		}
		return nil, fmt.Errorf("failed to fetch match: %w", err)
	}
	return &m, nil
}

func (r *Repository) GetMatchCampaignUUID(
	ctx context.Context, matchUUID uuid.UUID,
) (uuid.UUID, error) {
	const query = `
        SELECT campaign_uuid
        FROM matches
        WHERE uuid = $1
    `
	var campaignUUID uuid.UUID
	err := r.q.QueryRow(ctx, query, matchUUID).Scan(&campaignUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrMatchNotFound
		}
		return uuid.Nil, fmt.Errorf("failed to get match campaign UUID: %w", err)
	}
	return campaignUUID, nil
}

func (r *Repository) ListMatchesByMasterUUID(
	ctx context.Context, masterUUID uuid.UUID,
) ([]*match.Summary, error) {
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
            uuid, campaign_uuid, title,
						brief_initial_description, brief_final_description,
						is_public, game_start_at,
            story_start_at, story_end_at,
            created_at, updated_at
        FROM matches
        WHERE master_uuid = $1
        ORDER BY story_start_at ASC
    `

	rows, err := tx.Query(ctx, query, masterUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch match summaries: %w", err)
	}
	defer rows.Close()

	var matches []*match.Summary
	for rows.Next() {
		var m match.Summary
		err := rows.Scan(
			&m.UUID,
			&m.CampaignUUID,
			&m.Title,
			&m.BriefInitialDescription,
			&m.BriefFinalDescription,
			&m.IsPublic,
			&m.GameStartAt,
			&m.StoryStartAt,
			&m.StoryEndAt,
			&m.CreatedAt,
			&m.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan match summary: %w", err)
		}
		matches = append(matches, &m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over match summaries: %w", err)
	}
	return matches, nil
}

func (r *Repository) ListPublicUpcomingMatches(
	ctx context.Context, after time.Time, masterUUID uuid.UUID,
) ([]*match.Summary, error) {
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
            uuid, campaign_uuid, title,
            brief_initial_description, brief_final_description,
            is_public, game_start_at,
            story_start_at, story_end_at,
            created_at, updated_at
        FROM matches
        WHERE is_public = true
        AND game_start_at > $1
        AND master_uuid != $2
        ORDER BY game_start_at ASC
    `
	rows, err := tx.Query(ctx, query, after, masterUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch public upcoming match summaries: %w", err)
	}
	defer rows.Close()

	var matches []*match.Summary
	for rows.Next() {
		var m match.Summary
		err := rows.Scan(
			&m.UUID,
			&m.CampaignUUID,
			&m.Title,
			&m.BriefInitialDescription,
			&m.BriefFinalDescription,
			&m.IsPublic,
			&m.GameStartAt,
			&m.StoryStartAt,
			&m.StoryEndAt,
			&m.CreatedAt,
			&m.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan match summary: %w", err)
		}
		matches = append(matches, &m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over public match summaries: %w", err)
	}
	return matches, nil
}
