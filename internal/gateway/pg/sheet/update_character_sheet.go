package sheet

import (
	"context"
	"fmt"
	"time"

	sheetEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
)

func (r *Repository) UpdateCharacterSheet(
	ctx context.Context, sheet *sheetEntity.CharacterSheet,
) error {
	m := charSheetToModel(sheet)
	now := time.Now()

	tx, err := r.q.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
		_ = tx.Rollback(ctx)
	}()

	const sheetQuery = `
		UPDATE character_sheets SET
			category_name = $1, curr_hex_value = $2, talent_exp = $3,
			level = $4, points = $5, talent_lvl = $6, physicals_lvl = $7, mentals_lvl = $8, spirituals_lvl = $9, skills_lvl = $10,
			health_min_pts = $11, health_curr_pts = $12, health_max_pts = $13,
			stamina_min_pts = $14, stamina_curr_pts = $15, stamina_max_pts = $16,
			aura_min_pts = $17, aura_curr_pts = $18, aura_max_pts = $19,
			resistance_pts = $20, strength_pts = $21, agility_pts = $22, celerity_pts = $23, flexibility_pts = $24, dexterity_pts = $25, sense_pts = $26, constitution_pts = $27,
			resilience_pts = $28, adaptability_pts = $29, weighting_pts = $30, creativity_pts = $31, resilience_exp = $32, adaptability_exp = $33, weighting_exp = $34, creativity_exp = $35,
			vitality_exp = $36, energy_exp = $37, defense_exp = $38, push_exp = $39, grab_exp = $40, carry_exp = $41, velocity_exp = $42, accelerate_exp = $43, brake_exp = $44,
			legerity_exp = $45, repel_exp = $46, feint_exp = $47, acrobatics_exp = $48, evasion_exp = $49, sneak_exp = $50, reflex_exp = $51, accuracy_exp = $52, stealth_exp = $53,
			vision_exp = $54, hearing_exp = $55, smell_exp = $56, tact_exp = $57, taste_exp = $58, heal_exp = $59, breath_exp = $60, tenacity_exp = $61,
			nen_exp = $62, focus_exp = $63, will_power_exp = $64,
			ten_exp = $65, zetsu_exp = $66, ren_exp = $67, gyo_exp = $68, shu_exp = $69, kou_exp = $70, ken_exp = $71, ryu_exp = $72, in_exp = $73, en_exp = $74, aura_control_exp = $75, aop_exp = $76,
			reinforcement_exp = $77, transmutation_exp = $78, materialization_exp = $79, specialization_exp = $80, manipulation_exp = $81, emission_exp = $82,
			char_exp = $83,
			updated_at = $84
		WHERE uuid = $85 AND (player_uuid = $86 OR master_uuid = $87)
	`

	result, err := tx.Exec(ctx, sheetQuery,
		m.CategoryName, m.CurrHexValue, m.TalentExp,
		m.Level, m.Points, m.TalentLvl, m.PhysicalsLvl, m.MentalsLvl, m.SpiritualsLvl, m.SkillsLvl,
		m.Health.Min, m.Health.Curr, m.Health.Max,
		m.Stamina.Min, m.Stamina.Curr, m.Stamina.Max,
		m.Aura.Min, m.Aura.Curr, m.Aura.Max,
		m.ResistancePts, m.StrengthPts, m.AgilityPts, m.CelerityPts, m.FlexibilityPts, m.DexterityPts, m.SensePts, m.ConstitutionPts,
		m.ResiliencePts, m.AdaptabilityPts, m.WeightingPts, m.CreativityPts, m.ResilienceExp, m.AdaptabilityExp, m.WeightingExp, m.CreativityExp,
		m.VitalityExp, m.EnergyExp, m.DefenseExp, m.PushExp, m.GrabExp, m.CarryExp, m.VelocityExp, m.AccelerateExp, m.BrakeExp,
		m.LegerityExp, m.RepelExp, m.FeintExp, m.AcrobaticsExp, m.EvasionExp, m.SneakExp, m.ReflexExp, m.AccuracyExp, m.StealthExp,
		m.VisionExp, m.HearingExp, m.SmellExp, m.TactExp, m.TasteExp, m.HealExp, m.BreathExp, m.TenacityExp,
		m.NenExp, m.FocusExp, m.WillPowerExp,
		m.TenExp, m.ZetsuExp, m.RenExp, m.GyoExp, m.ShuExp, m.KouExp, m.KenExp, m.RyuExp, m.InExp, m.EnExp, m.AuraControlExp, m.AopExp,
		m.ReinforcementExp, m.TransmutationExp, m.MaterializationExp, m.SpecializationExp, m.ManipulationExp, m.EmissionExp,
		m.CharExp,
		now, m.UUID, m.PlayerUUID, m.MasterUUID,
	)
	if err != nil {
		return fmt.Errorf("failed to update character sheet: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrCharacterSheetNotFound
	}

	// Update profile
	const profileQuery = `
		UPDATE character_profiles SET
			nickname = $1, fullname = $2, alignment = $3, character_class = $4,
			long_description = $5, brief_description = $6, birthday = $7, age = $8,
			updated_at = $9
		WHERE character_sheet_uuid = $10
	`
	_, err = tx.Exec(ctx, profileQuery,
		m.Profile.NickName, m.Profile.FullName, m.Profile.Alignment, m.Profile.CharacterClass,
		m.Profile.Description, m.Profile.BriefDescription, m.Profile.Birthday, m.Profile.Age,
		now,
		m.UUID,
	)
	if err != nil {
		return fmt.Errorf("failed to update character profile: %w", err)
	}

	// Delete and recreate proficiencies
	const deleteProficienciesQuery = `
		DELETE FROM proficiencies WHERE character_sheet_uuid = $1
	`
	_, err = tx.Exec(ctx, deleteProficienciesQuery, m.UUID)
	if err != nil {
		return fmt.Errorf("failed to delete proficiencies: %w", err)
	}

	const proficienciesQuery = `
		INSERT INTO proficiencies (character_sheet_uuid, weapon, exp)
		VALUES ($1, $2, $3)
	`
	for _, proficiency := range m.Proficiencies {
		_, err = tx.Exec(ctx, proficienciesQuery, m.UUID, proficiency.Weapon, proficiency.Exp)
		if err != nil {
			return fmt.Errorf("failed to save proficiency %s: %w", proficiency.Weapon, err)
		}
	}

	// Delete and recreate joint proficiencies
	const deleteJointProficienciesQuery = `
		DELETE FROM joint_proficiencies WHERE character_sheet_uuid = $1
	`
	_, err = tx.Exec(ctx, deleteJointProficienciesQuery, m.UUID)
	if err != nil {
		return fmt.Errorf("failed to delete joint proficiencies: %w", err)
	}

	const jointProficienciesQuery = `
		INSERT INTO joint_proficiencies (character_sheet_uuid, name, weapons, exp)
		VALUES ($1, $2, $3, $4)
	`
	for _, jointProficiency := range m.JointProficiencies {
		_, err = tx.Exec(ctx, jointProficienciesQuery, m.UUID, jointProficiency.Name, jointProficiency.Weapons, jointProficiency.Exp)
		if err != nil {
			return fmt.Errorf("failed to save joint proficiency %s: %w", jointProficiency.Name, err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}
