// internal/gateway/pg/sheet/create_character_sheet.go
package sheet

import (
	"context"
	"fmt"
	"time"

	domainSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
	"github.com/google/uuid"
)

func (r *Repository) CreateCharacterSheet(
	ctx context.Context, sheet *domainSheet.CharacterSheet,
) error {
	m := charSheetToModel(sheet)

	tx, err := r.q.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			// TODO: maybe throws other error
			panic(p)
		}
		_ = tx.Rollback(ctx)
	}()

	const sheetQuery = `
		INSERT INTO character_sheets (
			uuid, player_uuid, category_name, curr_hex_value, talent_exp,
			level, points, talent_lvl, physicals_lvl, mentals_lvl, spirituals_lvl, skills_lvl,
			health_min_pts, health_curr_pts, health_max_pts,
			stamina_min_pts, stamina_curr_pts, stamina_max_pts,
			aura_min_pts, aura_curr_pts, aura_max_pts,
			resistance_pts, strength_pts, agility_pts, celerity_pts, flexibility_pts, dexterity_pts, sense_pts, constitution_pts,
			resilience_pts, adaptability_pts, weighting_pts, creativity_pts, resilience_exp, adaptability_exp, weighting_exp, creativity_exp,
			vitality_exp, energy_exp, defense_exp, push_exp, grab_exp, carry_exp, velocity_exp, accelerate_exp, brake_exp,
			legerity_exp, repel_exp, feint_exp, acrobatics_exp, evasion_exp, sneak_exp, reflex_exp, accuracy_exp, stealth_exp,
			vision_exp, hearing_exp, smell_exp, tact_exp, taste_exp, heal_exp, breath_exp, tenacity_exp,
			nen_exp, focus_exp, will_power_exp,
			ten_exp, zetsu_exp, ren_exp, gyo_exp, shu_exp, kou_exp, ken_exp, ryu_exp, in_exp, en_exp, aura_control_exp, aop_exp,
			reinforcement_exp, transmutation_exp, materialization_exp, specialization_exp, manipulation_exp, emission_exp,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11, $12,
			$13, $14, $15,
			$16, $17, $18,
			$19, $20, $21,
			$22, $23, $24, $25, $26, $27, $28, $29,
			$30, $31, $32, $33, $34, $35, $36, $37,
			$38, $39, $40, $41, $42, $43, $44, $45, $46,
			$47, $48, $49, $50, $51, $52, $53, $54, $55,
			$56, $57, $58, $59, $60, $61, $62, $63,
			$64, $65, $66,
			$67, $68, $69, $70, $71, $72, $73, $74, $75, $76, $77,
			$78, $79, $80, $81, $82, $83,
			$84, $85, $86
		) RETURNING id
	`
	var sheetID int
	err = tx.QueryRow(ctx, sheetQuery,
		m.UUID, m.PlayerUUID, m.CategoryName, m.CurrHexValue, m.TalentExp,
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
		m.CreatedAt, m.UpdatedAt,
	).Scan(&sheetID)
	if err != nil {
		return fmt.Errorf("failed to save character sheet: %w", err)
	}

	const profileQuery = `
		INSERT INTO character_profiles (
			uuid, character_sheet_uuid, nickname, fullname, alignment, character_class, long_description, brief_description, birthday, age, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
	`
	_, err = tx.Exec(ctx, profileQuery,
		m.Profile.UUID, m.UUID, m.Profile.NickName, m.Profile.FullName, m.Profile.Alignment,
		m.Profile.CharacterClass, m.Profile.Description, m.Profile.BriefDescription, m.Profile.Birthday, m.Profile.Age,
		m.Profile.CreatedAt, m.Profile.UpdatedAt,
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
	for _, proficiency := range m.Proficiencies {
		_, err = tx.Exec(ctx, proficienciesQuery,
			m.UUID, proficiency.Weapon, proficiency.Exp,
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
	for _, jointProficiency := range m.JointProficiencies {
		_, err = tx.Exec(ctx, jointProficienciesQuery,
			m.UUID, jointProficiency.Name, jointProficiency.Weapons, jointProficiency.Exp,
		)
		if err != nil {
			return fmt.Errorf("failed to save joint proficiency %s: %w", jointProficiency.Name, err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// charSheetToModel converts a domain CharacterSheet entity to the pg/model used for persistence.
// TODO: refactor these maps (inherited from domain use case)
func charSheetToModel(sheet *domainSheet.CharacterSheet) *model.CharacterSheet {
	now := time.Now()
	profile := sheet.GetProfile()
	physAttrs := sheet.GetPhysicalAttributes()
	mentalAttrs := sheet.GetMentalAttributes()
	physSkills := sheet.GetPhysicalSkills()
	principles := sheet.GetPrinciples()
	categories := sheet.GetCategories()
	statusBars := sheet.GetAllStatusBar()
	profs := sheet.GetCommonProficiencies()
	jointProfs := sheet.GetJointProficiencies()

	categoryName, err := sheet.GetCategoryName()
	categoryString := ""
	if err != nil {
		categoryString = categoryName.String()
	}

	physicalsLvl, _ := sheet.GetLevelOfAbility(enum.Physicals)
	mentalsLvl, _ := sheet.GetLevelOfAbility(enum.Mentals)
	spiritualsLvl, _ := sheet.GetLevelOfAbility(enum.Spirituals)
	skillsLvl, _ := sheet.GetLevelOfAbility(enum.Skills)

	modelProfs := []model.Proficiency{}
	for weapon, prof := range profs {
		modelProfs = append(modelProfs, model.Proficiency{
			UUID:      uuid.New(),
			Weapon:    weapon.String(),
			Exp:       prof.GetExpPoints(),
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	modelJointProfs := []model.JointProficiency{}
	for name, prof := range jointProfs {
		weapons := []string{}
		for _, weapon := range prof.GetWeapons() {
			weapons = append(weapons, weapon.String())
		}
		modelJointProfs = append(modelJointProfs, model.JointProficiency{
			UUID:      uuid.New(),
			Name:      name,
			Exp:       prof.GetExpPoints(),
			Weapons:   weapons,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	playerUUID := sheet.GetPlayerUUID()
	return &model.CharacterSheet{
		UUID:       sheet.UUID,
		PlayerUUID: playerUUID,

		Profile: model.CharacterProfile{
			UUID:             uuid.New(),
			CharacterClass:   sheet.GetCharacterClass().String(),
			NickName:         profile.NickName,
			FullName:         profile.FullName,
			Alignment:        profile.Alignment,
			Description:      profile.Description,
			BriefDescription: profile.BriefDescription,
			Birthday:         profile.Birthday,
			Age:              profile.Age,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		CategoryName: categoryString,
		CurrHexValue: sheet.GetCurrHexValue(),
		TalentExp:    sheet.GetTalentExpPoints(),

		Level:         sheet.GetLevel(),
		Points:        sheet.GetCharacterPoints(),
		TalentLvl:     sheet.GetTalentLevel(),
		PhysicalsLvl:  physicalsLvl,
		MentalsLvl:    mentalsLvl,
		SpiritualsLvl: spiritualsLvl,
		SkillsLvl:     skillsLvl,

		Health: model.StatusBar{
			Min:  statusBars[enum.Health].GetMin(),
			Curr: statusBars[enum.Health].GetCurrent(),
			Max:  statusBars[enum.Health].GetMax(),
		},
		Stamina: model.StatusBar{
			Min:  statusBars[enum.Stamina].GetMin(),
			Curr: statusBars[enum.Stamina].GetCurrent(),
			Max:  statusBars[enum.Stamina].GetMax(),
		},
		// Aura: model.StatusBar{...},

		ResistancePts:   physAttrs[enum.Resistance].GetPoints(),
		StrengthPts:     physAttrs[enum.Strength].GetPoints(),
		AgilityPts:      physAttrs[enum.Agility].GetPoints(),
		CelerityPts:     physAttrs[enum.Celerity].GetPoints(),
		FlexibilityPts:  physAttrs[enum.Flexibility].GetPoints(),
		DexterityPts:    physAttrs[enum.Dexterity].GetPoints(),
		SensePts:        physAttrs[enum.Sense].GetPoints(),
		ConstitutionPts: physAttrs[enum.Constitution].GetPoints(),

		ResiliencePts:   mentalAttrs[enum.Resilience].GetPoints(),
		AdaptabilityPts: mentalAttrs[enum.Adaptability].GetPoints(),
		WeightingPts:    mentalAttrs[enum.Weighting].GetPoints(),
		CreativityPts:   mentalAttrs[enum.Creativity].GetPoints(),
		ResilienceExp:   mentalAttrs[enum.Resilience].GetExpPoints(),
		AdaptabilityExp: mentalAttrs[enum.Adaptability].GetExpPoints(),
		WeightingExp:    mentalAttrs[enum.Weighting].GetExpPoints(),
		CreativityExp:   mentalAttrs[enum.Creativity].GetExpPoints(),

		VitalityExp:   physSkills[enum.Vitality].GetExpPoints(),
		EnergyExp:     physSkills[enum.Energy].GetExpPoints(),
		DefenseExp:    physSkills[enum.Defense].GetExpPoints(),
		PushExp:       physSkills[enum.Push].GetExpPoints(),
		GrabExp:       physSkills[enum.Grab].GetExpPoints(),
		CarryExp:      physSkills[enum.Carry].GetExpPoints(),
		VelocityExp:   physSkills[enum.Velocity].GetExpPoints(),
		AccelerateExp: physSkills[enum.Accelerate].GetExpPoints(),
		BrakeExp:      physSkills[enum.Brake].GetExpPoints(),
		LegerityExp:   physSkills[enum.Legerity].GetExpPoints(),
		RepelExp:      physSkills[enum.Repel].GetExpPoints(),
		FeintExp:      physSkills[enum.Feint].GetExpPoints(),
		AcrobaticsExp: physSkills[enum.Acrobatics].GetExpPoints(),
		EvasionExp:    physSkills[enum.Evasion].GetExpPoints(),
		SneakExp:      physSkills[enum.Sneak].GetExpPoints(),
		ReflexExp:     physSkills[enum.Reflex].GetExpPoints(),
		AccuracyExp:   physSkills[enum.Accuracy].GetExpPoints(),
		StealthExp:    physSkills[enum.Stealth].GetExpPoints(),
		VisionExp:     physSkills[enum.Vision].GetExpPoints(),
		HearingExp:    physSkills[enum.Hearing].GetExpPoints(),
		SmellExp:      physSkills[enum.Smell].GetExpPoints(),
		TactExp:       physSkills[enum.Tact].GetExpPoints(),
		TasteExp:      physSkills[enum.Taste].GetExpPoints(),
		HealExp:       physSkills[enum.Heal].GetExpPoints(),
		BreathExp:     physSkills[enum.Breath].GetExpPoints(),
		TenacityExp:   physSkills[enum.Tenacity].GetExpPoints(),

		TenExp:   principles[enum.Ten].GetExpPoints(),
		ZetsuExp: principles[enum.Zetsu].GetExpPoints(),
		RenExp:   principles[enum.Ren].GetExpPoints(),
		GyoExp:   principles[enum.Gyo].GetExpPoints(),
		ShuExp:   principles[enum.Shu].GetExpPoints(),
		KouExp:   principles[enum.Kou].GetExpPoints(),
		KenExp:   principles[enum.Ken].GetExpPoints(),
		RyuExp:   principles[enum.Ryu].GetExpPoints(),
		InExp:    principles[enum.In].GetExpPoints(),
		EnExp:    principles[enum.En].GetExpPoints(),

		ReinforcementExp:   categories[enum.Reinforcement].GetExpPoints(),
		TransmutationExp:   categories[enum.Transmutation].GetExpPoints(),
		MaterializationExp: categories[enum.Materialization].GetExpPoints(),
		SpecializationExp:  categories[enum.Specialization].GetExpPoints(),
		ManipulationExp:    categories[enum.Manipulation].GetExpPoints(),
		EmissionExp:        categories[enum.Emission].GetExpPoints(),

		Proficiencies:      modelProfs,
		JointProficiencies: modelJointProfs,

		CreatedAt: now,
		UpdatedAt: now,
	}
}
