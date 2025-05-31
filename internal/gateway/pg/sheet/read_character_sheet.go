package sheet

import (
	"context"
	"errors"
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetCharacterSheetByUUID(ctx context.Context, uuid string) (*model.CharacterSheet, error) {
	tx, err := r.q.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback(ctx)
			// TODO: maybe throws other error
			panic(p)
		} else if err != nil {
			tx.Rollback(ctx)
		} else {
			tx.Commit(ctx)
		}
	}()

	const query = `
			SELECT
					cs.id, cs.category_name, cs.uuid, cs.curr_hex_value, cs.talent_exp,
					cs.health_curr_pts, cs.stamina_curr_pts, cs.aura_curr_pts,
					cs.resistance_pts, cs.strength_pts, cs.agility_pts, cs.action_speed_pts, cs.flexibility_pts, cs.dexterity_pts, cs.sense_pts, cs.constitution_pts,
					cs.resilience_pts, cs.adaptability_pts, cs.weighting_pts, cs.creativity_pts, cs.resilience_exp, cs.adaptability_exp, cs.weighting_exp, cs.creativity_exp,
					cs.vitality_exp, cs.energy_exp, cs.defense_exp, cs.push_exp, cs.grab_exp, cs.carry_capacity_exp, cs.velocity_exp, cs.accelerate_exp, cs.brake_exp,
					cs.attack_speed_exp, cs.repel_exp, cs.feint_exp, cs.acrobatics_exp, cs.evasion_exp, cs.sneak_exp, cs.reflex_exp, cs.accuracy_exp, cs.stealth_exp,
					cs.vision_exp, cs.hearing_exp, cs.smell_exp, cs.tact_exp, cs.taste_exp, cs.heal_exp, cs.breath_exp, cs.tenacity_exp,
					cs.nen_exp, cs.focus_exp, cs.will_power_exp,
					cs.ten_exp, cs.zetsu_exp, cs.ren_exp, cs.gyo_exp, cs.kou_exp, cs.ken_exp, cs.ryu_exp, cs.in_exp, cs.en_exp, cs.aura_control_exp, cs.aop_exp,
					cs.reinforcement_exp, cs.transmutation_exp, cs.materialization_exp, cs.specialization_exp, cs.manipulation_exp, cs.emission_exp,
					cs.created_at, cs.updated_at,
					cp.uuid, cp.nickname, cp.fullname, cp.alignment, cp.character_class, cp.long_description, cp.brief_description, cp.birthday,
					cp.created_at, cp.updated_at
			FROM character_sheets cs
			JOIN character_profiles cp ON cs.uuid = cp.character_sheet_uuid
			WHERE cs.uuid = $1
	`
	row := tx.QueryRow(ctx, query, uuid)

	var sheet model.CharacterSheet
	var profile model.CharacterProfile

	err = row.Scan(
		&sheet.ID, &sheet.CategoryName, &sheet.UUID, &sheet.CurrHexValue, &sheet.TalentExp,
		&sheet.Health.Curr, &sheet.Stamina.Curr, &sheet.Aura.Curr,
		&sheet.ResistancePts, &sheet.StrengthPts, &sheet.AgilityPts, &sheet.ActionSpeedPts, &sheet.FlexibilityPts, &sheet.DexterityPts, &sheet.SensePts, &sheet.ConstitutionPts,
		&sheet.ResiliencePts, &sheet.AdaptabilityPts, &sheet.WeightingPts, &sheet.CreativityPts, &sheet.ResilienceExp, &sheet.AdaptabilityExp, &sheet.WeightingExp, &sheet.CreativityExp,
		&sheet.VitalityExp, &sheet.EnergyExp, &sheet.DefenseExp, &sheet.PushExp, &sheet.GrabExp, &sheet.CarryCapacityExp, &sheet.VelocityExp, &sheet.AccelerateExp, &sheet.BrakeExp,
		&sheet.AttackSpeedExp, &sheet.RepelExp, &sheet.FeintExp, &sheet.AcrobaticsExp, &sheet.EvasionExp, &sheet.SneakExp, &sheet.ReflexExp, &sheet.AccuracyExp, &sheet.StealthExp,
		&sheet.VisionExp, &sheet.HearingExp, &sheet.SmellExp, &sheet.TactExp, &sheet.TasteExp, &sheet.HealExp, &sheet.BreathExp, &sheet.TenacityExp,
		&sheet.NenExp, &sheet.FocusExp, &sheet.WillPowerExp,
		&sheet.TenExp, &sheet.ZetsuExp, &sheet.RenExp, &sheet.GyoExp, &sheet.KouExp, &sheet.KenExp, &sheet.RyuExp, &sheet.InExp, &sheet.EnExp, &sheet.AuraControlExp, &sheet.AopExp,
		&sheet.ReinforcementExp, &sheet.TransmutationExp, &sheet.MaterializationExp, &sheet.SpecializationExp, &sheet.ManipulationExp, &sheet.EmissionExp,
		&sheet.CreatedAt, &sheet.UpdatedAt,
		&profile.UUID, &profile.NickName, &profile.FullName, &profile.Alignment, &profile.CharacterClass, &profile.Description, &profile.BriefDescription, &profile.Birthday,
		&profile.CreatedAt, &profile.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCharacterSheetNotFound
		}
		return nil, fmt.Errorf("failed to fetch character sheet: %w", err)
	}
	sheet.Profile = profile

	const proficienciesQuery = `
			SELECT weapon, exp
			FROM proficiencies
			WHERE character_sheet_uuid = $1
	`
	rows, err := tx.Query(ctx, proficienciesQuery, uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch proficiencies: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var prof model.Proficiency
		if err := rows.Scan(&prof.Weapon, &prof.Exp); err != nil {
			return nil, fmt.Errorf("failed to scan proficiency: %w", err)
		}
		sheet.Proficiencies = append(sheet.Proficiencies, prof)
	}

	const jointProficienciesQuery = `
			SELECT name, weapons, exp
			FROM joint_proficiencies
			WHERE character_sheet_uuid = $1
	`
	rows, err = tx.Query(ctx, jointProficienciesQuery, uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch joint proficiencies: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var jointProf model.JointProficiency
		if err := rows.Scan(&jointProf.Name, &jointProf.Weapons, &jointProf.Exp); err != nil {
			return nil, fmt.Errorf("failed to scan joint proficiency: %w", err)
		}
		sheet.JointProficiencies = append(sheet.JointProficiencies, jointProf)
	}

	return &sheet, nil
}

func (r *Repository) ExistsCharacterWithNick(ctx context.Context, nick string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM character_profiles
			WHERE nickname = $1
		)
	`
	var exists bool
	err := r.q.QueryRow(ctx, query, nick).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if character profile exists by nickname: %w", err)
	}
	return exists, nil
}

func (r *Repository) CountCharactersByPlayerUUID(ctx context.Context, playerUUID uuid.UUID) (int, error) {
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
        FROM character_sheets
        WHERE player_uuid = $1
    `
	var count int
	err = tx.QueryRow(ctx, query, playerUUID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count player character sheets: %w", err)
	}
	return count, nil
}

func (r *Repository) ListCharacterSheetsByPlayerUUID(
	ctx context.Context, playerUUID string) ([]model.CharacterSheetSummary, error) {

	tx, err := r.q.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback(ctx)
			// TODO: maybe throws other error
			panic(p)
		} else if err != nil {
			tx.Rollback(ctx)
		} else {
			tx.Commit(ctx)
		}
	}()

	const query = `
			SELECT
					cs.id, cs.uuid, cs.category_name, cs.curr_hex_value,
					cs.level, cs.points, cs.talent_lvl, cs.skills_lvl,
					cs.health_min_pts, cs.health_curr_pts, cs.health_max_pts,
					cs.stamina_min_pts, cs.stamina_curr_pts, cs.stamina_max_pts,
					cs.physicals_lvl, cs.mentals_lvl, cs.spirituals_lvl,
					cs.aura_min_pts, cs.aura_curr_pts, cs.aura_max_pts,
					cs.created_at, cs.updated_at,
					cp.nickname, cp.fullname, cp.alignment, cp.character_class, cp.birthday
			FROM character_sheets cs
			JOIN character_profiles cp ON cs.uuid = cp.character_sheet_uuid
			WHERE cs.player_uuid = $1
			ORDER BY cp.nickname ASC
	`
	rows, err := tx.Query(ctx, query, playerUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch character sheets: %w", err)
	}
	defer rows.Close()

	var sheets []model.CharacterSheetSummary
	for rows.Next() {
		var sheet model.CharacterSheetSummary
		err := rows.Scan(
			&sheet.ID, &sheet.UUID, &sheet.CategoryName, &sheet.CurrHexValue,
			&sheet.Level, &sheet.Points, &sheet.TalentLvl, &sheet.SkillsLvl,
			&sheet.Health.Min, &sheet.Health.Curr, &sheet.Health.Max,
			&sheet.Stamina.Min, &sheet.Stamina.Curr, &sheet.Stamina.Max,
			&sheet.PhysicalsLvl, &sheet.MentalsLvl, &sheet.SpiritualsLvl,
			&sheet.Aura.Min, &sheet.Aura.Curr, &sheet.Aura.Max,
			&sheet.CreatedAt, &sheet.UpdatedAt,
			&sheet.NickName, &sheet.FullName, &sheet.Alignment, &sheet.CharacterClass, &sheet.Birthday,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan character sheet summary: %w", err)
		}
		sheets = append(sheets, sheet)
	}
	return sheets, nil
}
