package campaign

import (
	"context"
	"errors"
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// TODO: optimize db calls
func (r *Repository) GetCampaign(
	ctx context.Context, uuid uuid.UUID) (*campaign.Campaign, error) {

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

	const campaignQuery = `
        SELECT 
            uuid, master_uuid, scenario_uuid,
						name, brief_initial_description, brief_final_description, description,
						is_public, call_link,
            story_start_at, story_current_at, story_end_at,
						created_at, updated_at
        FROM campaigns
        WHERE uuid = $1
    `
	var c campaign.Campaign
	err = tx.QueryRow(ctx, campaignQuery, uuid).Scan(
		&c.UUID,
		&c.MasterUUID,
		&c.ScenarioUUID,
		&c.Name,
		&c.BriefInitialDescription,
		&c.BriefFinalDescription,
		&c.Description,
		&c.IsPublic,
		&c.CallLink,
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

	const pendingSheetsQuery = `
        SELECT 
            cs.id, cs.uuid, cs.player_uuid, cs.master_uuid, cs.campaign_uuid,
            cs.category_name, cs.curr_hex_value,
            cs.level, cs.points, cs.talent_lvl, cs.skills_lvl,
            cs.physicals_lvl, cs.mentals_lvl, cs.spirituals_lvl,
            cs.health_min_pts, cs.health_curr_pts, cs.health_max_pts,
            cs.stamina_min_pts, cs.stamina_curr_pts, cs.stamina_max_pts,
            cs.aura_min_pts, cs.aura_curr_pts, cs.aura_max_pts,
						cs.story_start_at, cs.story_current_at, cs.dead_at,
            cs.created_at, cs.updated_at,
            cp.nickname, cp.fullname, cp.alignment, cp.character_class, cp.birthday
        FROM submit_character_sheets scs
        JOIN character_sheets cs ON scs.character_sheet_uuid = cs.uuid
        JOIN character_profiles cp ON cs.uuid = cp.character_sheet_uuid
        WHERE scs.campaign_uuid = $1
        ORDER BY cp.nickname ASC
    `
	rows, err := tx.Query(ctx, pendingSheetsQuery, uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch submitted pending character sheets: %w", err)
	}
	defer rows.Close()

	var pendingSheets []model.CharacterSheetSummary
	for rows.Next() {
		var sheet model.CharacterSheetSummary
		err := rows.Scan(
			&sheet.ID, &sheet.UUID, &sheet.PlayerUUID, &sheet.MasterUUID, &sheet.CampaignUUID,
			&sheet.CategoryName, &sheet.CurrHexValue,
			&sheet.Level, &sheet.Points, &sheet.TalentLvl, &sheet.SkillsLvl,
			&sheet.PhysicalsLvl, &sheet.MentalsLvl, &sheet.SpiritualsLvl,
			&sheet.Health.Min, &sheet.Health.Curr, &sheet.Health.Max,
			&sheet.Stamina.Min, &sheet.Stamina.Curr, &sheet.Stamina.Max,
			&sheet.Aura.Min, &sheet.Aura.Curr, &sheet.Aura.Max,
			&sheet.StoryStartAt, &sheet.StoryCurrentAt, &sheet.DeadAt,
			&sheet.CreatedAt, &sheet.UpdatedAt,
			&sheet.NickName, &sheet.FullName, &sheet.Alignment, &sheet.CharacterClass, &sheet.Birthday,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan character sheet summary: %w", err)
		}
		pendingSheets = append(pendingSheets, sheet)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over character sheets: %w", err)
	}
	c.PendingSheets = pendingSheets

	const characterSheetsQuery = `
        SELECT 
            cs.id, cs.uuid, cs.player_uuid, cs.master_uuid, cs.campaign_uuid,
            cs.category_name, cs.curr_hex_value,
            cs.level, cs.points, cs.talent_lvl, cs.skills_lvl,
            cs.physicals_lvl, cs.mentals_lvl, cs.spirituals_lvl,
            cs.health_min_pts, cs.health_curr_pts, cs.health_max_pts,
            cs.stamina_min_pts, cs.stamina_curr_pts, cs.stamina_max_pts,
            cs.aura_min_pts, cs.aura_curr_pts, cs.aura_max_pts,
            cs.story_start_at, cs.story_current_at, cs.dead_at,
            cs.created_at, cs.updated_at,
            cp.nickname, cp.fullname, cp.alignment, cp.character_class, cp.birthday
        FROM character_sheets cs
        JOIN character_profiles cp ON cs.uuid = cp.character_sheet_uuid
        WHERE cs.campaign_uuid = $1
        ORDER BY cp.nickname ASC
    `
	rows, err = tx.Query(ctx, characterSheetsQuery, uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch character sheets: %w", err)
	}
	defer rows.Close()

	var characterSheets []model.CharacterSheetSummary
	for rows.Next() {
		var sheet model.CharacterSheetSummary
		err := rows.Scan(
			&sheet.ID, &sheet.UUID, &sheet.PlayerUUID, &sheet.MasterUUID, &sheet.CampaignUUID,
			&sheet.CategoryName, &sheet.CurrHexValue,
			&sheet.Level, &sheet.Points, &sheet.TalentLvl, &sheet.SkillsLvl,
			&sheet.PhysicalsLvl, &sheet.MentalsLvl, &sheet.SpiritualsLvl,
			&sheet.Health.Min, &sheet.Health.Curr, &sheet.Health.Max,
			&sheet.Stamina.Min, &sheet.Stamina.Curr, &sheet.Stamina.Max,
			&sheet.Aura.Min, &sheet.Aura.Curr, &sheet.Aura.Max,
			&sheet.StoryStartAt, &sheet.StoryCurrentAt, &sheet.DeadAt,
			&sheet.CreatedAt, &sheet.UpdatedAt,
			&sheet.NickName, &sheet.FullName, &sheet.Alignment, &sheet.CharacterClass, &sheet.Birthday,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan character sheet summary: %w", err)
		}
		characterSheets = append(characterSheets, sheet)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over character sheets: %w", err)
	}
	c.CharacterSheets = characterSheets

	const matchesQuery = `
        SELECT 
            uuid, campaign_uuid,
						title, brief_initial_description, brief_final_description,
						is_public, game_start_at,
            story_start_at, story_end_at,
            created_at, updated_at
        FROM matches
        WHERE campaign_uuid = $1
        ORDER BY story_start_at DESC
    `
	rows, err = tx.Query(ctx, matchesQuery, uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch matches: %w", err)
	}
	defer rows.Close()

	var matches []match.Summary
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
		matches = append(matches, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over matches: %w", err)
	}
	c.Matches = matches

	return &c, nil
}

func (r *Repository) ListCampaignsByMasterUUID(
	ctx context.Context, masterUUID uuid.UUID) ([]*campaign.Summary, error) {

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
							uuid, scenario_uuid,
							name, brief_initial_description, brief_final_description,
							is_public, call_link,
							story_start_at, story_current_at, story_end_at,
							created_at, updated_at
					FROM campaigns
					WHERE master_uuid = $1
					ORDER BY name ASC
	`
	rows, err := tx.Query(ctx, query, masterUUID)
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
			&c.BriefInitialDescription,
			&c.BriefFinalDescription,
			&c.IsPublic,
			&c.CallLink,
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

// GetCampaignMasterUUID retrieves the master UUID associated with a campaign.
func (r *Repository) GetCampaignMasterUUID(
	ctx context.Context, campaignUUID uuid.UUID) (uuid.UUID, error) {

	const query = `
        SELECT master_uuid FROM campaigns WHERE uuid = $1
    `
	var masterUUID uuid.UUID
	err := r.q.QueryRow(ctx, query, campaignUUID).Scan(&masterUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrCampaignNotFound
		}
		return uuid.Nil, fmt.Errorf("failed to get campaign master UUID: %w", err)
	}
	return masterUUID, nil
}

func (r *Repository) GetCampaignStoryDates(
	ctx context.Context, uuid uuid.UUID) (*campaign.Campaign, error) {

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

	const campaignQuery = `
        SELECT 
            uuid, master_uuid,
            story_start_at, story_current_at, story_end_at
        FROM campaigns
        WHERE uuid = $1
    `
	var c campaign.Campaign
	err = tx.QueryRow(ctx, campaignQuery, uuid).Scan(
		&c.UUID,
		&c.MasterUUID,
		&c.StoryStartAt,
		&c.StoryCurrentAt,
		&c.StoryEndAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCampaignNotFound
		}
		return nil, fmt.Errorf("failed to fetch campaign story dates: %w", err)
	}
	return &c, nil
}

func (r *Repository) CountCampaignsByMasterUUID(
	ctx context.Context, masterUUID uuid.UUID) (int, error) {

	tx, err := r.q.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
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
        SELECT COUNT(*) 
        FROM campaigns
        WHERE master_uuid = $1
    `
	var count int
	err = tx.QueryRow(ctx, query, masterUUID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count user campaigns: %w", err)
	}
	return count, nil
}
