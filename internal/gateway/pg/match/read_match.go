package match

import (
	"context"
	"errors"
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetMatch(ctx context.Context, uuid uuid.UUID) (*match.Match, error) {
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
						title, brief_description, description,
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
		&m.BriefDescription,
		&m.Description,
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
            uuid, campaign_uuid, title, brief_description,
            story_start_at, story_end_at,
            created_at, updated_at
        FROM matches
        WHERE master_uuid = $1
        ORDER BY title ASC
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
			&m.BriefDescription,
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
