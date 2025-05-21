package charactersheet

import (
	"context"
	"sync"
	"time"

	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
	"github.com/google/uuid"
)

type ICreateCharacterSheet interface {
	CreateCharacterSheet(
		ctx context.Context, input *CreateCharacterSheetInput,
	) (*sheet.CharacterSheet, error)
}

type CreateCharacterSheetUC struct {
	characterClasses *sync.Map
	characterSheets  *sync.Map
	factory          *sheet.CharacterSheetFactory
	repo             IRepository
}

func NewCreateCharacterSheetUC(
	charClasses *sync.Map,
	charSheets *sync.Map,
	factory *sheet.CharacterSheetFactory,
	repo IRepository,
) *CreateCharacterSheetUC {
	return &CreateCharacterSheetUC{
		characterClasses: charClasses,
		characterSheets:  charSheets,
		factory:          factory,
		repo:             repo,
	}
}

type DistributionInput struct {
}

type CreateCharacterSheetInput struct {
	PlayerUUID        *uuid.UUID
	Profile           sheet.CharacterProfile
	CharacterClass    enum.CharacterClassName
	CategorySet       sheet.TalentByCategorySet
	SkillsExps        map[enum.SkillName]int
	ProficienciesExps map[enum.WeaponName]int
}

func (uc *CreateCharacterSheetUC) CreateCharacterSheet(
	ctx context.Context, input *CreateCharacterSheetInput,
) (*sheet.CharacterSheet, error) {

	class, exists := uc.characterClasses.Load(input.CharacterClass)
	if !exists {
		return nil, NewCharacterClassNotFoundError(input.CharacterClass.String())
	}
	charClass := class.(cc.CharacterClass)

	skillsExps := input.SkillsExps
	if err := charClass.ValidateSkills(skillsExps); err != nil {
		return nil, err
	}
	profExps := input.ProficienciesExps
	if err := charClass.ValidateProficiencies(profExps); err != nil {
		return nil, err
	}
	charClass.ApplySkills(skillsExps)
	charClass.ApplyProficiencies(profExps)

	if err := uc.validateNickName(ctx, input.Profile.NickName); err != nil {
		return nil, err
	}

	characterSheetsCount, err := uc.repo.CountCharactersByPlayerUUID(
		ctx, *input.PlayerUUID,
	)
	if err != nil {
		return nil, err
	}
	if characterSheetsCount >= 10 {
		return nil, ErrMaxCharacterSheetsLimit
	}

	set := input.CategorySet
	characterSheet, err := uc.factory.Build(
		input.PlayerUUID,
		nil,
		input.Profile,
		set.GetInitialHexValue(),
		nil,
		&charClass,
	)
	if err != nil {
		return nil, err
	}
	talentLvl := set.GetTalentLvl()
	characterSheet.InitTalentWithLvl(talentLvl)

	characterSheet.UUID = uuid.New()
	uc.characterSheets.Store(characterSheet.UUID, characterSheet)

	model := CharacterSheetToModel(characterSheet)
	err = uc.repo.CreateCharacterSheet(ctx, model)
	if err != nil {
		return nil, err
	}
	return characterSheet, err
}

func (uc *CreateCharacterSheetUC) validateNickName(
	ctx context.Context, nick string,
) error {
	var allowedNickName = true
	uc.characterClasses.Range(func(_, value any) bool {
		charClass := value.(cc.CharacterClass)
		if charClass.GetNameString() == nick {
			allowedNickName = false
			return false
		}
		return true
	})
	if !allowedNickName {
		return NewNicknameNotAllowedError(nick)
	}

	exists, err := uc.repo.ExistsCharacterWithNick(ctx, nick)
	if err != nil {
		return err
	}
	if exists {
		return NewNicknameAlreadyExistsError(nick)
	}
	return nil
}

func CharacterSheetToModel(sheet *sheet.CharacterSheet) *model.CharacterSheet {
	// TODO: refactor these maps
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
	charSheetModel := &model.CharacterSheet{
		UUID:       sheet.UUID,
		PlayerUUID: &playerUUID,

		Profile: model.CharacterProfile{
			UUID:             uuid.New(),
			CharacterClass:   sheet.GetCharacterClass().String(),
			NickName:         profile.NickName,
			FullName:         profile.FullName,
			Alignment:        profile.Alignment,
			Description:      profile.Description,
			BriefDescription: profile.BriefDescription,
			Birthday:         profile.Birthday,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		CategoryName: categoryString,
		CurrHexValue: sheet.GetCurrHexValue(),
		TalentExp:    sheet.GetTalentExpPoints(),

		// Physical Attributes
		ResistancePts:   physAttrs[enum.Resistance].GetPoints(),
		StrengthPts:     physAttrs[enum.Strength].GetPoints(),
		AgilityPts:      physAttrs[enum.Agility].GetPoints(),
		ActionSpeedPts:  physAttrs[enum.ActionSpeed].GetPoints(),
		FlexibilityPts:  physAttrs[enum.Flexibility].GetPoints(),
		DexterityPts:    physAttrs[enum.Dexterity].GetPoints(),
		SensePts:        physAttrs[enum.Sense].GetPoints(),
		ConstitutionPts: physAttrs[enum.Constitution].GetPoints(),

		// Mental Attributes
		ResiliencePts:   mentalAttrs[enum.Resilience].GetPoints(),
		AdaptabilityPts: mentalAttrs[enum.Adaptability].GetPoints(),
		WeightingPts:    mentalAttrs[enum.Weighting].GetPoints(),
		CreativityPts:   mentalAttrs[enum.Creativity].GetPoints(),
		ResilienceExp:   mentalAttrs[enum.Resilience].GetExpPoints(),
		AdaptabilityExp: mentalAttrs[enum.Adaptability].GetExpPoints(),
		WeightingExp:    mentalAttrs[enum.Weighting].GetExpPoints(),
		CreativityExp:   mentalAttrs[enum.Creativity].GetExpPoints(),

		// Physical Skills
		VitalityExp:      physSkills[enum.Vitality].GetExpPoints(),
		EnergyExp:        physSkills[enum.Energy].GetExpPoints(),
		DefenseExp:       physSkills[enum.Defense].GetExpPoints(),
		PushExp:          physSkills[enum.Push].GetExpPoints(),
		GrabExp:          physSkills[enum.Grab].GetExpPoints(),
		CarryCapacityExp: physSkills[enum.CarryCapacity].GetExpPoints(),
		VelocityExp:      physSkills[enum.Velocity].GetExpPoints(),
		AccelerateExp:    physSkills[enum.Accelerate].GetExpPoints(),
		BrakeExp:         physSkills[enum.Brake].GetExpPoints(),
		AttackSpeedExp:   physSkills[enum.AttackSpeed].GetExpPoints(),
		RepelExp:         physSkills[enum.Repel].GetExpPoints(),
		FeintExp:         physSkills[enum.Feint].GetExpPoints(),
		AcrobaticsExp:    physSkills[enum.Acrobatics].GetExpPoints(),
		EvasionExp:       physSkills[enum.Evasion].GetExpPoints(),
		SneakExp:         physSkills[enum.Sneak].GetExpPoints(),
		ReflexExp:        physSkills[enum.Reflex].GetExpPoints(),
		AccuracyExp:      physSkills[enum.Accuracy].GetExpPoints(),
		StealthExp:       physSkills[enum.Stealth].GetExpPoints(),
		VisionExp:        physSkills[enum.Vision].GetExpPoints(),
		HearingExp:       physSkills[enum.Hearing].GetExpPoints(),
		SmellExp:         physSkills[enum.Smell].GetExpPoints(),
		TactExp:          physSkills[enum.Tact].GetExpPoints(),
		TasteExp:         physSkills[enum.Taste].GetExpPoints(),
		HealExp:          physSkills[enum.Heal].GetExpPoints(),
		BreathExp:        physSkills[enum.Breath].GetExpPoints(),
		TenacityExp:      physSkills[enum.Tenacity].GetExpPoints(),

		// Nen Principles
		TenExp:   principles[enum.Ten].GetExpPoints(),
		ZetsuExp: principles[enum.Zetsu].GetExpPoints(),
		RenExp:   principles[enum.Ren].GetExpPoints(),
		GyoExp:   principles[enum.Gyo].GetExpPoints(),
		KouExp:   principles[enum.Kou].GetExpPoints(),
		KenExp:   principles[enum.Ken].GetExpPoints(),
		RyuExp:   principles[enum.Ryu].GetExpPoints(),
		InExp:    principles[enum.In].GetExpPoints(),
		EnExp:    principles[enum.En].GetExpPoints(),

		// Nen Categories
		ReinforcementExp:   categories[enum.Reinforcement].GetExpPoints(),
		TransmutationExp:   categories[enum.Transmutation].GetExpPoints(),
		MaterializationExp: categories[enum.Materialization].GetExpPoints(),
		SpecializationExp:  categories[enum.Specialization].GetExpPoints(),
		ManipulationExp:    categories[enum.Manipulation].GetExpPoints(),
		EmissionExp:        categories[enum.Emission].GetExpPoints(),

		StaminaCurrPts: statusBars[enum.Stamina].GetCurrent(),
		HealthCurrPts:  statusBars[enum.Health].GetCurrent(),

		Proficiencies:      modelProfs,
		JointProficiencies: modelJointProfs,

		CreatedAt: now,
		UpdatedAt: now,
	}
	return charSheetModel
}
