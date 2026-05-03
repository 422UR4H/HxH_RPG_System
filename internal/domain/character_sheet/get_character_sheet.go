package charactersheet

import (
	"context"
	"fmt"
	"math"
	"sync"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	domainCampaign "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/proficiency"
	domainSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	pgCampaign "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
	pgSheet "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/sheet"
	"github.com/google/uuid"
)

type IGetCharacterSheet interface {
	GetCharacterSheet(
		ctx context.Context, charSheetId uuid.UUID, playerId uuid.UUID,
	) (*domainSheet.CharacterSheet, error)
}

type GetCharacterSheetUC struct {
	characterSheets *sync.Map
	factory         *domainSheet.CharacterSheetFactory
	repo            IRepository
	campaignRepo    domainCampaign.IRepository
}

func NewGetCharacterSheetUC(
	charSheets *sync.Map,
	factory *domainSheet.CharacterSheetFactory,
	repo IRepository,
	campaignRepo domainCampaign.IRepository,
) *GetCharacterSheetUC {
	return &GetCharacterSheetUC{
		characterSheets: charSheets,
		factory:         factory,
		repo:            repo,
		campaignRepo:    campaignRepo,
	}
}

func (uc *GetCharacterSheetUC) GetCharacterSheet(
	ctx context.Context, sheetUUID uuid.UUID, userUUID uuid.UUID,
) (*domainSheet.CharacterSheet, error) {

	// TODO: fix, move after auth validations or remove
	// if charSheet, ok := uc.characterSheets.Load(sheetUUID); ok {
	// 	return charSheet.(*sheet.CharacterSheet), nil
	// }

	modelSheet, err := uc.repo.GetCharacterSheetByUUID(ctx, sheetUUID.String())
	if err == pgSheet.ErrCharacterSheetNotFound {
		return nil, ErrCharacterSheetNotFound
	}
	if err != nil {
		return nil, err
	}
	masterUUID := modelSheet.MasterUUID
	playerUUID := modelSheet.PlayerUUID

	// Check if the user is the owner of the character sheet
	if masterUUID != nil && *masterUUID == userUUID {
		return uc.hydrateCharacterSheet(playerUUID, masterUUID, modelSheet)
	}
	if playerUUID != nil && *playerUUID == userUUID {
		return uc.hydrateCharacterSheet(playerUUID, masterUUID, modelSheet)
	}

	campaignUUID := modelSheet.CampaignUUID
	if campaignUUID == nil {
		return nil, auth.ErrInsufficientPermissions
	}

	// Check if the user is the owner of the character sheet campaign
	campaignMasterUUID, err := uc.campaignRepo.GetCampaignMasterUUID(ctx, *campaignUUID)
	if err == pgCampaign.ErrCampaignNotFound {
		return nil, domainCampaign.ErrCampaignNotFound
	}
	if err != nil {
		return nil, err
	}
	if campaignMasterUUID == userUUID {
		return uc.hydrateCharacterSheet(playerUUID, masterUUID, modelSheet)
	}
	return nil, auth.ErrInsufficientPermissions
}

func (uc *GetCharacterSheetUC) hydrateCharacterSheet(
	playerUUID *uuid.UUID,
	masterUUID *uuid.UUID,
	modelSheet *model.CharacterSheet,
) (*domainSheet.CharacterSheet, error) {
	profile := ModelToProfile(&modelSheet.Profile)

	categoryName := (*enum.CategoryName)(&modelSheet.CategoryName)
	characterSheet, err := uc.factory.Build(
		playerUUID,
		masterUUID,
		modelSheet.CampaignUUID,
		*profile,
		modelSheet.CurrHexValue,
		categoryName,
		nil,
	)
	if err != nil {
		return nil, err
	}

	charClass, err := enum.CharacterClassNameFrom(modelSheet.Profile.CharacterClass)
	if err != nil {
		return nil, err
	}

	err = characterSheet.AddDryCharacterClass(&charClass)
	if err != nil {
		return nil, err
	}

	err = Wrap(characterSheet, modelSheet)
	if err != nil {
		return nil, err
	}

	// uc.characterSheets.Store(sheetUUID, characterSheet)

	return characterSheet, nil
}

func ModelToProfile(profile *model.CharacterProfile) *domainSheet.CharacterProfile {
	return &domainSheet.CharacterProfile{
		NickName:         profile.NickName,
		FullName:         profile.FullName,
		Alignment:        profile.Alignment,
		Description:      profile.Description,
		BriefDescription: profile.BriefDescription,
		Birthday:         profile.Birthday,
	}
}

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

func Wrap(charSheet *domainSheet.CharacterSheet, modelSheet *model.CharacterSheet) error {
	charSheet.UUID = modelSheet.UUID

	physicalAttrs := map[enum.AttributeName]int{
		enum.Resistance:   modelSheet.ResistancePts,
		enum.Strength:     modelSheet.StrengthPts,
		enum.Agility:      modelSheet.AgilityPts,
		enum.Celerity:     modelSheet.CelerityPts,
		enum.Flexibility:  modelSheet.FlexibilityPts,
		enum.Dexterity:    modelSheet.DexterityPts,
		enum.Sense:        modelSheet.SensePts,
		enum.Constitution: modelSheet.ConstitutionPts,
	}
	for name, points := range physicalAttrs {
		if points == 0 {
			continue
		}
		if _, _, err := charSheet.IncreasePtsForPhysPrimaryAttr(name, points); err != nil {
			return fmt.Errorf("%w %s: %v", domainSheet.ErrFailedToIncreasePhysAttrPts, name, err)
		}
	}

	// TODO: add mental attributes points or remove from modelSheet

	mentalAttrs := map[enum.AttributeName]int{
		enum.Resilience:   modelSheet.ResilienceExp,
		enum.Adaptability: modelSheet.AdaptabilityExp,
		enum.Weighting:    modelSheet.WeightingExp,
		enum.Creativity:   modelSheet.CreativityExp,
	}
	for name, exp := range mentalAttrs {
		if exp == 0 {
			continue
		}
		if err := charSheet.IncreaseExpForMentals(experience.NewUpgradeCascade(exp), name); err != nil {
			return fmt.Errorf("%w %s: %v", domainSheet.ErrFailedToIncreaseMentalExp, name, err)
		}
	}

	physicalSkills := map[enum.SkillName]int{
		enum.Vitality:   modelSheet.VitalityExp,
		enum.Energy:     modelSheet.EnergyExp,
		enum.Defense:    modelSheet.DefenseExp,
		enum.Push:       modelSheet.PushExp,
		enum.Grab:       modelSheet.GrabExp,
		enum.Carry:      modelSheet.CarryExp,
		enum.Velocity:   modelSheet.VelocityExp,
		enum.Accelerate: modelSheet.AccelerateExp,
		enum.Brake:      modelSheet.BrakeExp,
		enum.Legerity:   modelSheet.LegerityExp,
		enum.Repel:      modelSheet.RepelExp,
		enum.Feint:      modelSheet.FeintExp,
		enum.Acrobatics: modelSheet.AcrobaticsExp,
		enum.Evasion:    modelSheet.EvasionExp,
		enum.Sneak:      modelSheet.SneakExp,
		enum.Reflex:     modelSheet.ReflexExp,
		enum.Accuracy:   modelSheet.AccuracyExp,
		enum.Stealth:    modelSheet.StealthExp,
		enum.Vision:     modelSheet.VisionExp,
		enum.Hearing:    modelSheet.HearingExp,
		enum.Smell:      modelSheet.SmellExp,
		enum.Tact:       modelSheet.TactExp,
		enum.Taste:      modelSheet.TasteExp,
		enum.Heal:       modelSheet.HealExp,
		enum.Breath:     modelSheet.BreathExp,
		enum.Tenacity:   modelSheet.TenacityExp,
	}
	for name, exp := range physicalSkills {
		if exp == 0 {
			continue
		}
		if err := charSheet.IncreaseExpForSkill(experience.NewUpgradeCascade(exp), name); err != nil {
			return fmt.Errorf("%w %s: %v", domainSheet.ErrFailedToIncreaseSkillExp, name, err)
		}
	}

	spiritualPrinciples := map[enum.PrincipleName]int{
		enum.Ten:   modelSheet.TenExp,
		enum.Zetsu: modelSheet.ZetsuExp,
		enum.Ren:   modelSheet.RenExp,
		enum.Gyo:   modelSheet.GyoExp,
		enum.Shu:   modelSheet.ShuExp,
		enum.Kou:   modelSheet.KouExp,
		enum.Ken:   modelSheet.KenExp,
		enum.Ryu:   modelSheet.RyuExp,
		enum.In:    modelSheet.InExp,
		enum.En:    modelSheet.EnExp,
	}
	for name, exp := range spiritualPrinciples {
		if exp == 0 {
			continue
		}
		if err := charSheet.IncreaseExpForPrinciple(experience.NewUpgradeCascade(exp), name); err != nil {
			return fmt.Errorf("%w %s: %v", domainSheet.ErrFailedToIncreasePrincipleExp, name, err)
		}
	}

	spiritualCategories := map[enum.CategoryName]int{
		enum.Reinforcement:   modelSheet.ReinforcementExp,
		enum.Transmutation:   modelSheet.TransmutationExp,
		enum.Materialization: modelSheet.MaterializationExp,
		enum.Specialization:  modelSheet.SpecializationExp,
		enum.Manipulation:    modelSheet.ManipulationExp,
		enum.Emission:        modelSheet.EmissionExp,
	}
	for name, exp := range spiritualCategories {
		if exp == 0 {
			continue
		}
		if err := charSheet.IncreaseExpForCategory(experience.NewUpgradeCascade(exp), name); err != nil {
			return fmt.Errorf("%w %s: %v", domainSheet.ErrFailedToIncreaseCategoryExp, name, err)
		}
	}

	if err := charSheet.SetCurrStatus(enum.Health, modelSheet.Health.Curr); err != nil {
		return fmt.Errorf("%w (health): %v", domainSheet.ErrFailedToSetStatus, err)
	}
	if err := charSheet.SetCurrStatus(enum.Stamina, modelSheet.Stamina.Curr); err != nil {
		return fmt.Errorf("%w (stamina): %v", domainSheet.ErrFailedToSetStatus, err)
	}
	if err := charSheet.SetCurrStatus(enum.Aura, modelSheet.Aura.Curr); err != nil {
		return fmt.Errorf("%w (aura): %v", domainSheet.ErrFailedToSetStatus, err)
	}

	physSkExp, err := charSheet.GetPhysSkillExpReference()
	if err != nil {
		return domainSheet.ErrFailedToGetPhysSkillExpRef
	}
	expTable := experience.NewExpTable(domainSheet.PHYSICAL_SKILLS_COEFF)
	newExp := experience.NewExperience(expTable)
	for _, prof := range modelSheet.Proficiencies {
		domainProf := proficiency.NewProficiency(
			enum.WeaponName(prof.Weapon), *newExp, physSkExp,
		)
		if err := charSheet.AddCommonProficiency(enum.WeaponName(prof.Weapon), domainProf); err != nil {
			return fmt.Errorf("%w: %v", domainSheet.ErrFailedToAddCommonProficiency, err)
		}
		if err := charSheet.IncreaseExpForProficiency(
			experience.NewUpgradeCascade(prof.Exp), enum.WeaponName(prof.Weapon),
		); err != nil {
			return fmt.Errorf("%w: %v", domainSheet.ErrFailedToIncreaseProficiencyExp, err)
		}
	}

	for _, jointProf := range modelSheet.JointProficiencies {
		weapons := []enum.WeaponName{}
		for _, weapon := range jointProf.Weapons {
			weapons = append(weapons, enum.WeaponName(weapon))
		}
		domainJointProf := proficiency.NewJointProficiency(
			*newExp, jointProf.Name, weapons,
		)
		if err := charSheet.AddJointProficiency(domainJointProf); err != nil {
			return fmt.Errorf("%w: %v", domainSheet.ErrFailedToAddJointProficiency, err)
		}
		// TODO: implement for create and add here too
		// charSheet.IncreaseExpForProficiency(
		// 	experience.NewUpgradeCascade(jointProf.Exp), enum.WeaponName(jointProf.Weapon),
		// )
	}

	charSheet.IncreaseExpForTalent(modelSheet.TalentExp)
	return nil
}
