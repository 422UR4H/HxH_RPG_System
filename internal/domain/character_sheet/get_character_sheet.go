package charactersheet

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/proficiency"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
	sheetPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/sheet"
	"github.com/google/uuid"
)

type IGetCharacterSheet interface {
	GetCharacterSheet(
		ctx context.Context, charSheetId uuid.UUID, playerId uuid.UUID,
	) (*sheet.CharacterSheet, error)
}

type GetCharacterSheetUC struct {
	characterSheets *sync.Map
	factory         *sheet.CharacterSheetFactory
	repo            IRepository
}

func NewGetCharacterSheetUC(
	charSheets *sync.Map,
	factory *sheet.CharacterSheetFactory,
	repo IRepository,
) *GetCharacterSheetUC {
	return &GetCharacterSheetUC{
		characterSheets: charSheets,
		factory:         factory,
		repo:            repo,
	}
}

func (uc *GetCharacterSheetUC) GetCharacterSheet(
	ctx context.Context, charSheetId uuid.UUID, playerId uuid.UUID,
) (*sheet.CharacterSheet, error) {

	if charSheet, ok := uc.characterSheets.Load(charSheetId); ok {
		return charSheet.(*sheet.CharacterSheet), nil
	}

	modelSheet, err := uc.repo.GetCharacterSheetByUUID(ctx, charSheetId.String())
	if err != nil {
		if errors.Is(err, sheetPg.ErrCharacterSheetNotFound) {
			return nil, ErrCharacterSheetNotFound
		}
		return nil, err
	}

	if *modelSheet.PlayerUUID != playerId {
		return nil, auth.ErrInsufficientPermissions
	}

	profile := ModelToProfile(&modelSheet.Profile)

	categoryName := (*enum.CategoryName)(&modelSheet.CategoryName)
	characterSheet, err := uc.factory.Build(
		&playerId, nil, *profile, modelSheet.CurrHexValue, categoryName, nil,
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

	uc.characterSheets.Store(charSheetId, characterSheet)

	return characterSheet, nil
}

func ModelToProfile(profile *model.CharacterProfile) *sheet.CharacterProfile {
	return &sheet.CharacterProfile{
		NickName:         profile.NickName,
		FullName:         profile.FullName,
		Alignment:        profile.Alignment,
		Description:      profile.Description,
		BriefDescription: profile.BriefDescription,
		Birthday:         profile.Birthday,
	}
}

func Wrap(charSheet *sheet.CharacterSheet, modelSheet *model.CharacterSheet) error {
	charSheet.UUID = modelSheet.UUID

	physicalAttrs := map[enum.AttributeName]int{
		enum.Resistance:   modelSheet.ResistancePts,
		enum.Strength:     modelSheet.StrengthPts,
		enum.Agility:      modelSheet.AgilityPts,
		enum.ActionSpeed:  modelSheet.ActionSpeedPts,
		enum.Flexibility:  modelSheet.FlexibilityPts,
		enum.Dexterity:    modelSheet.DexterityPts,
		enum.Sense:        modelSheet.SensePts,
		enum.Constitution: modelSheet.ConstitutionPts,
	}
	for name, points := range physicalAttrs {
		charSheet.IncreasePtsForPhysPrimaryAttr(name, points)
	}

	// TODO: add mental attributes points or remove from modelSheet

	mentalAttrs := map[enum.AttributeName]int{
		enum.Resilience:   modelSheet.ResilienceExp,
		enum.Adaptability: modelSheet.AdaptabilityExp,
		enum.Weighting:    modelSheet.WeightingExp,
		enum.Creativity:   modelSheet.CreativityExp,
	}
	for name, exp := range mentalAttrs {
		charSheet.IncreaseExpForMentals(experience.NewUpgradeCascade(exp), name)
	}

	physicalSkills := map[enum.SkillName]int{
		enum.Vitality:      modelSheet.VitalityExp,
		enum.Energy:        modelSheet.EnergyExp,
		enum.Defense:       modelSheet.DefenseExp,
		enum.Push:          modelSheet.PushExp,
		enum.Grab:          modelSheet.GrabExp,
		enum.CarryCapacity: modelSheet.CarryCapacityExp,
		enum.Velocity:      modelSheet.VelocityExp,
		enum.Accelerate:    modelSheet.AccelerateExp,
		enum.Brake:         modelSheet.BrakeExp,
		enum.AttackSpeed:   modelSheet.AttackSpeedExp,
		enum.Repel:         modelSheet.RepelExp,
		enum.Feint:         modelSheet.FeintExp,
		enum.Acrobatics:    modelSheet.AcrobaticsExp,
		enum.Evasion:       modelSheet.EvasionExp,
		enum.Sneak:         modelSheet.SneakExp,
		enum.Reflex:        modelSheet.ReflexExp,
		enum.Accuracy:      modelSheet.AccuracyExp,
		enum.Stealth:       modelSheet.StealthExp,
		enum.Vision:        modelSheet.VisionExp,
		enum.Hearing:       modelSheet.HearingExp,
		enum.Smell:         modelSheet.SmellExp,
		enum.Tact:          modelSheet.TactExp,
		enum.Taste:         modelSheet.TasteExp,
		enum.Heal:          modelSheet.HealExp,
		enum.Breath:        modelSheet.BreathExp,
		enum.Tenacity:      modelSheet.TenacityExp,
	}
	for name, exp := range physicalSkills {
		charSheet.IncreaseExpForSkill(experience.NewUpgradeCascade(exp), name)
	}

	spiritualPrinciples := map[enum.PrincipleName]int{
		enum.Ten:   modelSheet.TenExp,
		enum.Zetsu: modelSheet.ZetsuExp,
		enum.Ren:   modelSheet.RenExp,
		enum.Gyo:   modelSheet.GyoExp,
		enum.Kou:   modelSheet.KouExp,
		enum.Ken:   modelSheet.KenExp,
		enum.Ryu:   modelSheet.RyuExp,
		enum.In:    modelSheet.InExp,
		enum.En:    modelSheet.EnExp,
	}
	for name, exp := range spiritualPrinciples {
		charSheet.IncreaseExpForPrinciple(experience.NewUpgradeCascade(exp), name)
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
		charSheet.IncreaseExpForCategory(experience.NewUpgradeCascade(exp), name)
	}

	physSkExp, err := charSheet.GetPhysSkillExpReference()
	if err != nil {
		return domain.NewDomainError(
			fmt.Errorf("failed to get physical skills experience reference"),
		)
	}
	expTable := experience.NewExpTable(sheet.PHYSICAL_SKILLS_COEFF)
	newExp := experience.NewExperience(expTable)
	for _, prof := range modelSheet.Proficiencies {
		domainProf := proficiency.NewProficiency(
			enum.WeaponName(prof.Weapon), *newExp, physSkExp,
		)
		charSheet.AddCommonProficiency(enum.WeaponName(prof.Weapon), domainProf)
		charSheet.IncreaseExpForProficiency(
			experience.NewUpgradeCascade(prof.Exp), enum.WeaponName(prof.Weapon),
		)
	}

	for _, jointProf := range modelSheet.JointProficiencies {
		weapons := []enum.WeaponName{}
		for _, weapon := range jointProf.Weapons {
			weapons = append(weapons, enum.WeaponName(weapon))
		}
		domainJointProf := proficiency.NewJointProficiency(
			*newExp, jointProf.Name, weapons,
		)
		charSheet.AddJointProficiency(domainJointProf)
		// TODO: implement for create and add here too
		// charSheet.IncreaseExpForProficiency(
		// 	experience.NewUpgradeCascade(jointProf.Exp), enum.WeaponName(jointProf.Weapon),
		// )
	}

	charSheet.IncreaseExpForTalent(modelSheet.TalentExp)
	return nil
}
