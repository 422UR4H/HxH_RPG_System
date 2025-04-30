package sheet

import (
	"context"
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
)

func (r *Repository) CreateCharacterSheet(
	ctx context.Context, sheet *model.CharacterSheet,
) error {
	tx, err := r.q.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
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

	const sheetQuery = `
		INSERT INTO character_sheets (
			uuid, player_id, category_name, curr_hex_value, talent_exp,
			resistance_pts, strength_pts, agility_pts, action_speed_pts, flexibility_pts, dexterity_pts, sense_pts, constitution_pts,
			resilience_pts, adaptability_pts, weighting_pts, creativity_pts, resilience_exp, adaptability_exp, weighting_exp, creativity_exp,
			vitality_exp, energy_exp, defense_exp, push_exp, grab_exp, carry_capacity_exp, velocity_exp, accelerate_exp, brake_exp,
			attack_speed_exp, repel_exp, feint_exp, acrobatics_exp, evasion_exp, sneak_exp, reflex_exp, accuracy_exp, stealth_exp,
			vision_exp, hearing_exp, smell_exp, tact_exp, taste_exp, heal_exp, breath_exp, tenacity_exp,
			nen_exp, focus_exp, will_power_exp,
			ten_exp, zetsu_exp, ren_exp, gyo_exp, kou_exp, ken_exp, ryu_exp, in_exp, en_exp, aura_control_exp, aop_exp,
			reinforcement_exp, transmutation_exp, materialization_exp, specialization_exp, manipulation_exp, emission_exp,
			stamina_curr_pts, health_curr_pts, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11, $12, $13,
			$14, $15, $16, $17, $18, $19, $20, $21,
			$22, $23, $24, $25, $26, $27, $28, $29, $30,
			$31, $32, $33, $34, $35, $36, $37, $38, $39,
			$40, $41, $42, $43, $44, $45, $46, $47,
			$48, $49, $50,
			$51, $52, $53, $54, $55, $56, $57, $58, $59, $60,
			$61, $62, $63, $64, $65, $66,
			$67, $68, $69, $70, $71
		) RETURNING id
	`
	var sheetID int
	err = tx.QueryRow(ctx, sheetQuery,
		sheet.UUID, sheet.PlayerUUID, sheet.CategoryName, sheet.CurrHexValue, sheet.TalentExp,
		sheet.ResistancePts, sheet.StrengthPts, sheet.AgilityPts, sheet.ActionSpeedPts, sheet.FlexibilityPts, sheet.DexterityPts, sheet.SensePts, sheet.ConstitutionPts,
		sheet.ResiliencePts, sheet.AdaptabilityPts, sheet.WeightingPts, sheet.CreativityPts, sheet.ResilienceExp, sheet.AdaptabilityExp, sheet.WeightingExp, sheet.CreativityExp,
		sheet.VitalityExp, sheet.EnergyExp, sheet.DefenseExp, sheet.PushExp, sheet.GrabExp, sheet.CarryCapacityExp, sheet.VelocityExp, sheet.AccelerateExp, sheet.BrakeExp,
		sheet.AttackSpeedExp, sheet.RepelExp, sheet.FeintExp, sheet.AcrobaticsExp, sheet.EvasionExp, sheet.SneakExp, sheet.ReflexExp, sheet.AccuracyExp, sheet.StealthExp,
		sheet.VisionExp, sheet.HearingExp, sheet.SmellExp, sheet.TactExp, sheet.TasteExp, sheet.HealExp, sheet.BreathExp, sheet.TenacityExp,
		sheet.NenExp, sheet.FocusExp, sheet.WillPowerExp,
		sheet.TenExp, sheet.ZetsuExp, sheet.RenExp, sheet.GyoExp, sheet.KouExp, sheet.KenExp, sheet.RyuExp, sheet.InExp, sheet.EnExp, sheet.AuraControlExp, sheet.AopExp,
		sheet.ReinforcementExp, sheet.TransmutationExp, sheet.MaterializationExp, sheet.SpecializationExp, sheet.ManipulationExp, sheet.EmissionExp,
		sheet.StaminaCurrPts, sheet.HealthCurrPts, sheet.CreatedAt, sheet.UpdatedAt,
	).Scan(&sheetID)
	if err != nil {
		return fmt.Errorf("failed to save character sheet: %w", err)
	}
	sheet.ID = sheetID

	const profileQuery = `
		INSERT INTO character_profiles (
			uuid, character_sheet_uuid, nickname, fullname, alignment, character_class, long_description, brief_description, birthday, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
	`
	_, err = tx.Exec(ctx, profileQuery,
		sheet.Profile.UUID, sheet.UUID, sheet.Profile.NickName, sheet.Profile.FullName, sheet.Profile.Alignment,
		sheet.Profile.CharacterClass, sheet.Profile.Description, sheet.Profile.BriefDescription, sheet.Profile.Birthday,
		sheet.Profile.CreatedAt, sheet.Profile.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save character profile: %w", err)
	}

	const proficienciesQuery = `
		INSERT INTO proficiencies (
			character_sheet_uuid, weapon, exp
		) VALUES (
			$1, $2, $3
		)
	`
	for _, proficiency := range sheet.Proficiencies {
		_, err = tx.Exec(ctx, proficienciesQuery,
			sheet.UUID, proficiency.Weapon, proficiency.Exp,
		)
		if err != nil {
			return fmt.Errorf("failed to save proficiency %s: %w", proficiency.Weapon, err)
		}
	}

	const jointProficienciesQuery = `
		INSERT INTO joint_proficiencies (
			character_sheet_uuid, name, weapons, exp
		) VALUES (
			$1, $2, $3, $4
		)
	`
	for _, jointProficiency := range sheet.JointProficiencies {
		_, err = tx.Exec(ctx, jointProficienciesQuery,
			sheet.UUID, jointProficiency.Name, jointProficiency.Weapons, jointProficiency.Exp,
		)
		if err != nil {
			return fmt.Errorf("failed to save joint proficiency %s: %w", jointProficiency.Name, err)
		}
	}

	return nil
}
