package sheet

import (
	"context"
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
)

func (r *Repository) GetCharacterSheetByUUID(ctx context.Context, uuid string) (*model.CharacterSheet, error) {
	const query = `
			SELECT
					cs.id, cs.uuid, cs.curr_hex_value, cs.talent_exp,
					cs.resistance_pts, cs.strength_pts, cs.agility_pts, cs.action_speed_pts, cs.flexibility_pts, cs.dexterity_pts, cs.sense_pts, cs.constitution_pts,
					cs.resilience_pts, cs.adaptability_pts, cs.weighting_pts, cs.creativity_pts, cs.resilience_exp, cs.adaptability_exp, cs.weighting_exp, cs.creativity_exp,
					cs.vitality_exp, cs.energy_exp, cs.defense_exp, cs.push_exp, cs.grab_exp, cs.carry_capacity_exp, cs.velocity_exp, cs.accelerate_exp, cs.brake_exp,
					cs.attack_speed_exp, cs.repel_exp, cs.feint_exp, cs.acrobatics_exp, cs.evasion_exp, cs.sneak_exp, cs.reflex_exp, cs.accuracy_exp, cs.stealth_exp,
					cs.vision_exp, cs.hearing_exp, cs.smell_exp, cs.tact_exp, cs.taste_exp, cs.heal_exp, cs.breath_exp, cs.tenacity_exp,
					cs.nen_exp, cs.focus_exp, cs.will_power_exp,
					cs.ten_exp, cs.zetsu_exp, cs.ren_exp, cs.gyo_exp, cs.kou_exp, cs.ken_exp, cs.ryu_exp, cs.in_exp, cs.en_exp, cs.aura_control_exp, cs.aop_exp,
					cs.reinforcement_exp, cs.transmutation_exp, cs.materialization_exp, cs.specialization_exp, cs.manipulation_exp, cs.emission_exp,
					cs.stamina_curr_pts, cs.health_curr_pts,
					cs.created_at, cs.updated_at,
					cp.uuid, cp.nickname, cp.fullname, cp.alignment, cp.character_class, cp.long_description, cp.brief_description, cp.birthday,
					cp.created_at, cp.updated_at
			FROM character_sheets cs
			JOIN character_profiles cp ON cs.uuid = cp.character_sheet_uuid
			WHERE cs.uuid = $1
	`
	row := r.q.QueryRow(ctx, query, uuid)

	var sheet model.CharacterSheet
	var profile model.CharacterProfile

	err := row.Scan(
		&sheet.ID, &sheet.UUID, &sheet.CurrHexValue, &sheet.TalentExp,
		&sheet.ResistancePts, &sheet.StrengthPts, &sheet.AgilityPts, &sheet.ActionSpeedPts, &sheet.FlexibilityPts, &sheet.DexterityPts, &sheet.SensePts, &sheet.ConstitutionPts,
		&sheet.ResiliencePts, &sheet.AdaptabilityPts, &sheet.WeightingPts, &sheet.CreativityPts, &sheet.ResilienceExp, &sheet.AdaptabilityExp, &sheet.WeightingExp, &sheet.CreativityExp,
		&sheet.VitalityExp, &sheet.EnergyExp, &sheet.DefenseExp, &sheet.PushExp, &sheet.GrabExp, &sheet.CarryCapacityExp, &sheet.VelocityExp, &sheet.AccelerateExp, &sheet.BrakeExp,
		&sheet.AttackSpeedExp, &sheet.RepelExp, &sheet.FeintExp, &sheet.AcrobaticsExp, &sheet.EvasionExp, &sheet.SneakExp, &sheet.ReflexExp, &sheet.AccuracyExp, &sheet.StealthExp,
		&sheet.VisionExp, &sheet.HearingExp, &sheet.SmellExp, &sheet.TactExp, &sheet.TasteExp, &sheet.HealExp, &sheet.BreathExp, &sheet.TenacityExp,
		&sheet.NenExp, &sheet.FocusExp, &sheet.WillPowerExp,
		&sheet.TenExp, &sheet.ZetsuExp, &sheet.RenExp, &sheet.GyoExp, &sheet.KouExp, &sheet.KenExp, &sheet.RyuExp, &sheet.InExp, &sheet.EnExp, &sheet.AuraControlExp, &sheet.AopExp,
		&sheet.ReinforcementExp, &sheet.TransmutationExp, &sheet.MaterializationExp, &sheet.SpecializationExp, &sheet.ManipulationExp, &sheet.EmissionExp,
		&sheet.StaminaCurrPts, &sheet.HealthCurrPts,
		&sheet.CreatedAt, &sheet.UpdatedAt,
		&profile.UUID, &profile.NickName, &profile.FullName, &profile.Alignment, &profile.CharacterClass, &profile.Description, &profile.BriefDescription, &profile.Birthday,
		&profile.CreatedAt, &profile.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch character sheet: %w", err)
	}
	sheet.Profile = profile

	const proficienciesQuery = `
			SELECT weapon, exp
			FROM proficiencies
			WHERE character_sheet_uuid = $1
	`
	rows, err := r.q.Query(ctx, proficienciesQuery, uuid)
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
	rows, err = r.q.Query(ctx, jointProficienciesQuery, uuid)
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
