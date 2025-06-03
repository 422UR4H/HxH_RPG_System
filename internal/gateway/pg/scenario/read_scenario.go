package scenario

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	campaignEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	scenarioEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/scenario"
	"github.com/google/uuid"
)

func (r *Repository) GetScenario(ctx context.Context, id uuid.UUID) (*scenarioEntity.Scenario, error) {
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
					s.uuid, s.user_uuid, s.name, s.brief_description, s.description,
					s.created_at, s.updated_at,
					c.uuid, c.name, c.brief_description,
					c.story_start_at, c.story_current_at, c.story_end_at,
					c.created_at, c.updated_at
			FROM scenarios s
			LEFT JOIN campaigns c ON s.uuid = c.scenario_uuid
			WHERE s.uuid = $1
	`
	rows, err := tx.Query(ctx, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch scenario: %w", err)
	}
	defer rows.Close()

	var scenario *scenarioEntity.Scenario
	var campaigns []*campaignEntity.Summary

	for rows.Next() {
		var scenarioUUID, userUUID uuid.UUID
		var name, briefDesc, description string
		var scenarioCreatedAt, scenarioUpdatedAt time.Time

		var campaignUUID, campaignName, campaignBriefInitDesc sql.NullString
		var campaignStartAt, campaignCurrentAt, campaignEndAt sql.NullTime
		var campaignCreatedAt, campaignUpdatedAt sql.NullTime

		err := rows.Scan(
			&scenarioUUID, &userUUID, &name, &briefDesc, &description,
			&scenarioCreatedAt, &scenarioUpdatedAt,
			&campaignUUID, &campaignName, &campaignBriefInitDesc,
			&campaignStartAt, &campaignCurrentAt, &campaignEndAt,
			&campaignCreatedAt, &campaignUpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if scenario == nil {
			scenario = &scenarioEntity.Scenario{
				UUID:             scenarioUUID,
				UserUUID:         userUUID,
				Name:             name,
				BriefDescription: briefDesc,
				Description:      description,
				CreatedAt:        scenarioCreatedAt,
				UpdatedAt:        scenarioUpdatedAt,
			}
		}

		if campaignUUID.Valid {
			campaign := &campaignEntity.Summary{
				UUID:                    uuid.MustParse(campaignUUID.String),
				Name:                    campaignName.String,
				BriefInitialDescription: campaignBriefInitDesc.String,
				StoryStartAt:            campaignStartAt.Time,
			}

			if campaignCurrentAt.Valid {
				campaign.StoryCurrentAt = &campaignCurrentAt.Time
			}

			if campaignEndAt.Valid {
				campaign.StoryEndAt = &campaignEndAt.Time
			}

			campaign.CreatedAt = campaignCreatedAt.Time
			campaign.UpdatedAt = campaignUpdatedAt.Time

			campaigns = append(campaigns, campaign)
		}
	}
	if scenario == nil {
		return nil, ErrScenarioNotFound
	}
	scenario.Campaigns = campaigns
	return scenario, nil
}

func (r *Repository) ListScenariosByUserUUID(
	ctx context.Context, userUUID uuid.UUID) ([]*scenarioEntity.Summary, error) {

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
					uuid, name, brief_description, created_at, updated_at
			FROM scenarios
			WHERE user_uuid = $1
			ORDER BY name ASC
	`
	rows, err := tx.Query(ctx, query, userUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch scenarios summary: %w", err)
	}
	defer rows.Close()

	var scenarios []*scenarioEntity.Summary
	for rows.Next() {
		var s scenarioEntity.Summary
		err := rows.Scan(
			&s.UUID,
			&s.Name,
			&s.BriefDescription,
			&s.CreatedAt,
			&s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan scenario summary: %w", err)
		}
		scenarios = append(scenarios, &s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over scenarios summary: %w", err)
	}
	return scenarios, nil
}

func (r *Repository) ExistsScenarioWithName(ctx context.Context, name string) (bool, error) {
	const query = `
        SELECT EXISTS(SELECT 1 FROM scenarios WHERE name = $1)
    `
	var exists bool
	err := r.q.QueryRow(ctx, query, name).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if scenario exists by name: %w", err)
	}
	return exists, nil
}

func (r *Repository) ExistsScenario(ctx context.Context, uuid uuid.UUID) (bool, error) {
	const query = `
        SELECT EXISTS(SELECT 1 FROM scenarios WHERE uuid = $1)
    `
	var exists bool
	err := r.q.QueryRow(ctx, query, uuid).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if scenario exists: %w", err)
	}
	return exists, nil
}
