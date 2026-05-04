// internal/gateway/pg/sheet/read_character_sheet.go
package sheet

import (
	"context"
	"errors"
	"fmt"
	"math"

	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	domainSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/proficiency"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetCharacterSheetByUUID(
	ctx context.Context, uuid string,
) (*domainSheet.CharacterSheet, bool, error) {
	tx, err := r.q.Begin(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			// TODO: maybe throws other error
			panic(p)
		}
		_ = tx.Rollback(ctx)
	}()

	const query = `
		SELECT
			cs.id, cs.uuid, cs.player_uuid, cs.master_uuid, cs.campaign_uuid,
			cs.category_name, cs.curr_hex_value, cs.talent_exp,
			cs.health_min_pts, cs.health_curr_pts, cs.health_max_pts,
			cs.stamina_min_pts, cs.stamina_curr_pts, cs.stamina_max_pts,
			cs.aura_min_pts, cs.aura_curr_pts, cs.aura_max_pts,
			cs.resistance_pts, cs.strength_pts, cs.agility_pts, cs.celerity_pts, cs.flexibility_pts, cs.dexterity_pts, cs.sense_pts, cs.constitution_pts,
			cs.resilience_pts, cs.adaptability_pts, cs.weighting_pts, cs.creativity_pts, cs.resilience_exp, cs.adaptability_exp, cs.weighting_exp, cs.creativity_exp,
			cs.vitality_exp, cs.energy_exp, cs.defense_exp, cs.push_exp, cs.grab_exp, cs.carry_exp, cs.velocity_exp, cs.accelerate_exp, cs.brake_exp,
			cs.legerity_exp, cs.repel_exp, cs.feint_exp, cs.acrobatics_exp, cs.evasion_exp, cs.sneak_exp, cs.reflex_exp, cs.accuracy_exp, cs.stealth_exp,
			cs.vision_exp, cs.hearing_exp, cs.smell_exp, cs.tact_exp, cs.taste_exp, cs.heal_exp, cs.breath_exp, cs.tenacity_exp,
			cs.nen_exp, cs.focus_exp, cs.will_power_exp,
			cs.ten_exp, cs.zetsu_exp, cs.ren_exp, cs.gyo_exp, cs.shu_exp, cs.kou_exp, cs.ken_exp, cs.ryu_exp, cs.in_exp, cs.en_exp, cs.aura_control_exp, cs.aop_exp,
			cs.reinforcement_exp, cs.transmutation_exp, cs.materialization_exp, cs.specialization_exp, cs.manipulation_exp, cs.emission_exp,
			cs.created_at, cs.updated_at,
			cp.uuid, cp.nickname, cp.fullname, cp.alignment, cp.character_class, cp.long_description, cp.brief_description, cp.birthday,
			cp.created_at, cp.updated_at
		FROM character_sheets cs
		JOIN character_profiles cp ON cs.uuid = cp.character_sheet_uuid
		WHERE cs.uuid = $1
	`
	row := tx.QueryRow(ctx, query, uuid)

	var m model.CharacterSheet
	var profile model.CharacterProfile

	err = row.Scan(
		&m.ID, &m.UUID, &m.PlayerUUID, &m.MasterUUID, &m.CampaignUUID,
		&m.CategoryName, &m.CurrHexValue, &m.TalentExp,
		&m.Health.Min, &m.Health.Curr, &m.Health.Max,
		&m.Stamina.Min, &m.Stamina.Curr, &m.Stamina.Max,
		&m.Aura.Min, &m.Aura.Curr, &m.Aura.Max,
		&m.ResistancePts, &m.StrengthPts, &m.AgilityPts, &m.CelerityPts, &m.FlexibilityPts, &m.DexterityPts, &m.SensePts, &m.ConstitutionPts,
		&m.ResiliencePts, &m.AdaptabilityPts, &m.WeightingPts, &m.CreativityPts, &m.ResilienceExp, &m.AdaptabilityExp, &m.WeightingExp, &m.CreativityExp,
		&m.VitalityExp, &m.EnergyExp, &m.DefenseExp, &m.PushExp, &m.GrabExp, &m.CarryExp, &m.VelocityExp, &m.AccelerateExp, &m.BrakeExp,
		&m.LegerityExp, &m.RepelExp, &m.FeintExp, &m.AcrobaticsExp, &m.EvasionExp, &m.SneakExp, &m.ReflexExp, &m.AccuracyExp, &m.StealthExp,
		&m.VisionExp, &m.HearingExp, &m.SmellExp, &m.TactExp, &m.TasteExp, &m.HealExp, &m.BreathExp, &m.TenacityExp,
		&m.NenExp, &m.FocusExp, &m.WillPowerExp,
		&m.TenExp, &m.ZetsuExp, &m.RenExp, &m.GyoExp, &m.ShuExp, &m.KouExp, &m.KenExp, &m.RyuExp, &m.InExp, &m.EnExp, &m.AuraControlExp, &m.AopExp,
		&m.ReinforcementExp, &m.TransmutationExp, &m.MaterializationExp, &m.SpecializationExp, &m.ManipulationExp, &m.EmissionExp,
		&m.CreatedAt, &m.UpdatedAt,
		&profile.UUID, &profile.NickName, &profile.FullName, &profile.Alignment, &profile.CharacterClass, &profile.Description, &profile.BriefDescription, &profile.Birthday,
		&profile.CreatedAt, &profile.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, charactersheet.ErrCharacterSheetNotFound
		}
		return nil, false, fmt.Errorf("failed to fetch character sheet: %w", err)
	}
	m.Profile = profile

	const proficienciesQuery = `
		SELECT weapon, exp
		FROM proficiencies
		WHERE character_sheet_uuid = $1
	`
	rows, err := tx.Query(ctx, proficienciesQuery, uuid)
	if err != nil {
		return nil, false, fmt.Errorf("failed to fetch proficiencies: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var prof model.Proficiency
		if err := rows.Scan(&prof.Weapon, &prof.Exp); err != nil {
			return nil, false, fmt.Errorf("failed to scan proficiency: %w", err)
		}
		m.Proficiencies = append(m.Proficiencies, prof)
	}

	const jointProficienciesQuery = `
		SELECT name, weapons, exp
		FROM joint_proficiencies
		WHERE character_sheet_uuid = $1
	`
	rows, err = tx.Query(ctx, jointProficienciesQuery, uuid)
	if err != nil {
		return nil, false, fmt.Errorf("failed to fetch joint proficiencies: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var jointProf model.JointProficiency
		if err := rows.Scan(&jointProf.Name, &jointProf.Weapons, &jointProf.Exp); err != nil {
			return nil, false, fmt.Errorf("failed to scan joint proficiency: %w", err)
		}
		m.JointProficiencies = append(m.JointProficiencies, jointProf)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, false, fmt.Errorf("failed to commit transaction: %w", err)
	}

	domainProfile := modelToProfile(&m.Profile)
	categoryName := (*enum.CategoryName)(&m.CategoryName)
	factory := domainSheet.NewCharacterSheetFactory()
	charSheet, err := factory.Build(
		m.PlayerUUID,
		m.MasterUUID,
		m.CampaignUUID,
		*domainProfile,
		m.CurrHexValue,
		categoryName,
		nil,
	)
	if err != nil {
		return nil, false, fmt.Errorf("failed to build character sheet entity: %w", err)
	}

	if m.Profile.CharacterClass != "" {
		charClass, err := enum.CharacterClassNameFrom(m.Profile.CharacterClass)
		if err != nil {
			return nil, false, err
		}
		if err := charSheet.AddDryCharacterClass(&charClass); err != nil {
			return nil, false, err
		}
	}

	wasCorrected, err := wrap(charSheet, &m)
	if err != nil {
		return nil, false, err
	}
	return charSheet, wasCorrected, nil
}

func (r *Repository) GetCharacterSheetPlayerUUID(
	ctx context.Context, sheet_uuid uuid.UUID,
) (uuid.UUID, error) {
	const query = `
		SELECT player_uuid
		FROM character_sheets
		WHERE uuid = $1
	`
	var playerUUID uuid.UUID
	err := r.q.QueryRow(ctx, query, sheet_uuid).Scan(&playerUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrCharacterSheetNotFound
		}
		return uuid.Nil, fmt.Errorf("failed to fetch character sheet player UUID: %w", err)
	}
	return playerUUID, nil
}

func (r *Repository) ListCharacterSheetsByPlayerUUID(
	ctx context.Context, playerUUID string,
) ([]csEntity.Summary, error) {
	const query = `
		SELECT
			cs.id, cs.uuid, cs.player_uuid, cs.master_uuid, cs.campaign_uuid,
			cs.category_name, cs.curr_hex_value,
			COALESCE(cs.level, 0), COALESCE(cs.points, 0),
			COALESCE(cs.talent_lvl, 0), COALESCE(cs.skills_lvl, 0),
			COALESCE(cs.physicals_lvl, 0), COALESCE(cs.mentals_lvl, 0), COALESCE(cs.spirituals_lvl, 0),
			COALESCE(cs.health_min_pts, 0), COALESCE(cs.health_curr_pts, 0), COALESCE(cs.health_max_pts, 0),
			COALESCE(cs.stamina_min_pts, 0), COALESCE(cs.stamina_curr_pts, 0), COALESCE(cs.stamina_max_pts, 0),
			COALESCE(cs.aura_min_pts, 0), COALESCE(cs.aura_curr_pts, 0), COALESCE(cs.aura_max_pts, 0),
			cs.story_start_at, cs.story_current_at, cs.dead_at,
			cs.created_at, cs.updated_at,
			cp.nickname, cp.fullname, cp.alignment, cp.character_class, cp.birthday
		FROM character_sheets cs
		JOIN character_profiles cp ON cs.uuid = cp.character_sheet_uuid
		WHERE cs.player_uuid = $1
		ORDER BY cp.nickname ASC
	`
	rows, err := r.q.Query(ctx, query, playerUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to list character sheets: %w", err)
	}
	defer rows.Close()

	var out []csEntity.Summary
	for rows.Next() {
		var s csEntity.Summary
		err := rows.Scan(
			&s.ID, &s.UUID, &s.PlayerUUID, &s.MasterUUID, &s.CampaignUUID,
			&s.CategoryName, &s.CurrHexValue,
			&s.Level, &s.Points, &s.TalentLvl, &s.SkillsLvl,
			&s.PhysicalsLvl, &s.MentalsLvl, &s.SpiritualsLvl,
			&s.Health.Min, &s.Health.Curr, &s.Health.Max,
			&s.Stamina.Min, &s.Stamina.Curr, &s.Stamina.Max,
			&s.Aura.Min, &s.Aura.Curr, &s.Aura.Max,
			&s.StoryStartAt, &s.StoryCurrentAt, &s.DeadAt,
			&s.CreatedAt, &s.UpdatedAt,
			&s.NickName, &s.FullName, &s.Alignment, &s.CharacterClass, &s.Birthday,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan character sheet summary: %w", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	return out, nil
}

func (r *Repository) GetCharacterSheetRelationshipUUIDs(
	ctx context.Context, sheet_uuid uuid.UUID,
) (csEntity.RelationshipUUIDs, error) {
	const query = `
		SELECT player_uuid, master_uuid, campaign_uuid
		FROM character_sheets
		WHERE uuid = $1
	`
	var rel csEntity.RelationshipUUIDs
	err := r.q.QueryRow(ctx, query, sheet_uuid).Scan(
		&rel.PlayerUUID,
		&rel.MasterUUID,
		&rel.CampaignUUID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return csEntity.RelationshipUUIDs{}, charactersheet.ErrCharacterSheetNotFound
		}
		return csEntity.RelationshipUUIDs{},
			fmt.Errorf("failed to fetch character sheet relationship UUIDs: %w", err)
	}
	return rel, nil
}

// modelToProfile converts a pg/model CharacterProfile to the domain entity profile.
func modelToProfile(profile *model.CharacterProfile) *domainSheet.CharacterProfile {
	return &domainSheet.CharacterProfile{
		NickName:         profile.NickName,
		FullName:         profile.FullName,
		Alignment:        profile.Alignment,
		Description:      profile.Description,
		BriefDescription: profile.BriefDescription,
		Birthday:         profile.Birthday,
	}
}

// wrap populates charSheet with experience and status values from the DB model.
func wrap(charSheet *domainSheet.CharacterSheet, m *model.CharacterSheet) (wasCorrected bool, err error) {
	charSheet.UUID = m.UUID

	physicalAttrs := map[enum.AttributeName]int{
		enum.Resistance:   m.ResistancePts,
		enum.Strength:     m.StrengthPts,
		enum.Agility:      m.AgilityPts,
		enum.Celerity:     m.CelerityPts,
		enum.Flexibility:  m.FlexibilityPts,
		enum.Dexterity:    m.DexterityPts,
		enum.Sense:        m.SensePts,
		enum.Constitution: m.ConstitutionPts,
	}
	for name, points := range physicalAttrs {
		if points == 0 {
			continue
		}
		if _, _, err := charSheet.IncreasePtsForPhysPrimaryAttr(name, points); err != nil {
			return false, fmt.Errorf("%w %s: %v", domainSheet.ErrFailedToIncreasePhysAttrPts, name, err)
		}
	}

	// TODO: add mental attributes points or remove from modelSheet
	mentalAttrs := map[enum.AttributeName]int{
		enum.Resilience:   m.ResilienceExp,
		enum.Adaptability: m.AdaptabilityExp,
		enum.Weighting:    m.WeightingExp,
		enum.Creativity:   m.CreativityExp,
	}
	for name, exp := range mentalAttrs {
		if exp == 0 {
			continue
		}
		if err := charSheet.IncreaseExpForMentals(experience.NewUpgradeCascade(exp), name); err != nil {
			return false, fmt.Errorf("%w %s: %v", domainSheet.ErrFailedToIncreaseMentalExp, name, err)
		}
	}

	physicalSkills := map[enum.SkillName]int{
		enum.Vitality:   m.VitalityExp,
		enum.Energy:     m.EnergyExp,
		enum.Defense:    m.DefenseExp,
		enum.Push:       m.PushExp,
		enum.Grab:       m.GrabExp,
		enum.Carry:      m.CarryExp,
		enum.Velocity:   m.VelocityExp,
		enum.Accelerate: m.AccelerateExp,
		enum.Brake:      m.BrakeExp,
		enum.Legerity:   m.LegerityExp,
		enum.Repel:      m.RepelExp,
		enum.Feint:      m.FeintExp,
		enum.Acrobatics: m.AcrobaticsExp,
		enum.Evasion:    m.EvasionExp,
		enum.Sneak:      m.SneakExp,
		enum.Reflex:     m.ReflexExp,
		enum.Accuracy:   m.AccuracyExp,
		enum.Stealth:    m.StealthExp,
		enum.Vision:     m.VisionExp,
		enum.Hearing:    m.HearingExp,
		enum.Smell:      m.SmellExp,
		enum.Tact:       m.TactExp,
		enum.Taste:      m.TasteExp,
		enum.Heal:       m.HealExp,
		enum.Breath:     m.BreathExp,
		enum.Tenacity:   m.TenacityExp,
	}
	for name, exp := range physicalSkills {
		if exp == 0 {
			continue
		}
		if err := charSheet.IncreaseExpForSkill(experience.NewUpgradeCascade(exp), name); err != nil {
			return false, fmt.Errorf("%w %s: %v", domainSheet.ErrFailedToIncreaseSkillExp, name, err)
		}
	}

	spiritualPrinciples := map[enum.PrincipleName]int{
		enum.Ten:   m.TenExp,
		enum.Zetsu: m.ZetsuExp,
		enum.Ren:   m.RenExp,
		enum.Gyo:   m.GyoExp,
		enum.Shu:   m.ShuExp,
		enum.Kou:   m.KouExp,
		enum.Ken:   m.KenExp,
		enum.Ryu:   m.RyuExp,
		enum.In:    m.InExp,
		enum.En:    m.EnExp,
	}
	for name, exp := range spiritualPrinciples {
		if exp == 0 {
			continue
		}
		if err := charSheet.IncreaseExpForPrinciple(experience.NewUpgradeCascade(exp), name); err != nil {
			return false, fmt.Errorf("%w %s: %v", domainSheet.ErrFailedToIncreasePrincipleExp, name, err)
		}
	}

	spiritualCategories := map[enum.CategoryName]int{
		enum.Reinforcement:   m.ReinforcementExp,
		enum.Transmutation:   m.TransmutationExp,
		enum.Materialization: m.MaterializationExp,
		enum.Specialization:  m.SpecializationExp,
		enum.Manipulation:    m.ManipulationExp,
		enum.Emission:        m.EmissionExp,
	}
	for name, exp := range spiritualCategories {
		if exp == 0 {
			continue
		}
		if err := charSheet.IncreaseExpForCategory(experience.NewUpgradeCascade(exp), name); err != nil {
			return false, fmt.Errorf("%w %s: %v", domainSheet.ErrFailedToIncreaseCategoryExp, name, err)
		}
	}

	type statusEntry struct {
		name   enum.StatusName
		curr   int
		oldMax int
	}
	for _, e := range []statusEntry{
		{enum.Health, m.Health.Curr, m.Health.Max},
		{enum.Stamina, m.Stamina.Curr, m.Stamina.Max},
		{enum.Aura, m.Aura.Curr, m.Aura.Max},
	} {
		newMax, err := charSheet.GetMaxOfStatus(e.name)
		if err != nil {
			return false, fmt.Errorf("%w (%s): %v", domainSheet.ErrFailedToGetStatus, e.name, err)
		}
		minVal, err := charSheet.GetMinOfStatus(e.name)
		if err != nil {
			return false, fmt.Errorf("%w (%s): %v", domainSheet.ErrFailedToGetStatus, e.name, err)
		}
		corrected, correctionApplied := normalizeStatus(e.curr, e.oldMax, newMax, minVal)
		if correctionApplied {
			wasCorrected = true
		}
		if newMax == 0 {
			continue
		}
		if err := charSheet.SetCurrStatus(e.name, corrected); err != nil {
			return false, fmt.Errorf("%w (%s): %v", domainSheet.ErrFailedToSetStatus, e.name, err)
		}
	}

	physSkExp, err := charSheet.GetPhysSkillExpReference()
	if err != nil {
		return false, domainSheet.ErrFailedToGetPhysSkillExpRef
	}
	expTable := experience.NewExpTable(domainSheet.PHYSICAL_SKILLS_COEFF)
	newExp := experience.NewExperience(expTable)
	for _, prof := range m.Proficiencies {
		domainProf := proficiency.NewProficiency(
			enum.WeaponName(prof.Weapon), *newExp, physSkExp,
		)
		if err := charSheet.AddCommonProficiency(enum.WeaponName(prof.Weapon), domainProf); err != nil {
			return false, fmt.Errorf("%w: %v", domainSheet.ErrFailedToAddCommonProficiency, err)
		}
		if err := charSheet.IncreaseExpForProficiency(
			experience.NewUpgradeCascade(prof.Exp), enum.WeaponName(prof.Weapon),
		); err != nil {
			return false, fmt.Errorf("%w: %v", domainSheet.ErrFailedToIncreaseProficiencyExp, err)
		}
	}

	for _, jointProf := range m.JointProficiencies {
		weapons := []enum.WeaponName{}
		for _, weapon := range jointProf.Weapons {
			weapons = append(weapons, enum.WeaponName(weapon))
		}
		domainJointProf := proficiency.NewJointProficiency(
			*newExp, jointProf.Name, weapons,
		)
		if err := charSheet.AddJointProficiency(domainJointProf); err != nil {
			return false, fmt.Errorf("%w: %v", domainSheet.ErrFailedToAddJointProficiency, err)
		}
		// TODO: implement for create and add here too
	}

	charSheet.IncreaseExpForTalent(m.TalentExp)
	return wasCorrected, nil
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
			_ = tx.Rollback(ctx)
			panic(p)
		}
		_ = tx.Rollback(ctx)
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
	if err = tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}
	return count, nil
}

// normalizeStatus corrects curr when it exceeds the recalculated newMax.
func normalizeStatus(curr, oldMax, newMax, minVal int) (int, bool) {
	if newMax == 0 {
		fmt.Printf("TODO(logger): status normalized anomaly: newMax is 0, curr %d not corrected\n", curr)
		return curr, false
	}
	if curr <= newMax {
		return curr, false
	}
	if oldMax <= 0 {
		fmt.Printf("TODO(logger): status normalized (fallback): curr %d → new_max %d\n", curr, newMax)
		return newMax, true
	}
	corrected := int(math.Round(float64(newMax) * float64(curr) / float64(oldMax)))
	corrected = max(minVal, min(newMax, corrected))
	fmt.Printf("TODO(logger): status normalized: curr %d → %d (old_max: %d, new_max: %d)\n", curr, corrected, oldMax, newMax)
	return corrected, true
}
