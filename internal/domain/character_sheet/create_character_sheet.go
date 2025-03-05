package charactersheet

import (
	"fmt"

	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/sheet"
)

type ICreateCharacterSheet interface {
	CreateCharacterSheet() *sheet.CharacterSheet
}

type CreateCharacterSheetUC struct {
	// add repo
	characterClasses map[enum.CharacterClassName]cc.CharacterClass
	factory          *sheet.CharacterSheetFactory
}

func NewCreateCharacterSheetUC(
	charClasses map[enum.CharacterClassName]cc.CharacterClass,
	factory *sheet.CharacterSheetFactory,
) *CreateCharacterSheetUC {
	return &CreateCharacterSheetUC{
		// add repo
		characterClasses: charClasses,
		factory:          factory,
	}
}

type DistributionInput struct {
}

type CreateCharacterSheetInput struct {
	Profile           sheet.CharacterProfile
	CharacterClass    enum.CharacterClassName
	CategorySet       sheet.TalentByCategorySet
	SkillsExps        map[enum.SkillName]int
	ProficienciesExps map[enum.WeaponName]int
}

func (uc *CreateCharacterSheetUC) CreateCharacterSheet(
	input CreateCharacterSheetInput,
) (*sheet.CharacterSheet, error) {
	for _, charClass := range uc.characterClasses {
		if charClass.GetName().String() == input.Profile.NickName {
			return nil, fmt.Errorf("nickname is not allowed")
		}
	}
	// TODO: validate input.Profile.NickName in repo

	charClass := uc.characterClasses[input.CharacterClass]
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
	characterSheet, err := uc.factory.Build(input.Profile, &input.CategorySet, &charClass)
	// save to repo
	return characterSheet, err
}
