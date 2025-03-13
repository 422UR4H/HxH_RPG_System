package charactersheet

import (
	"fmt"
	"sync"

	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/sheet"
)

type ICreateCharacterSheet interface {
	CreateCharacterSheet(input *CreateCharacterSheetInput) (*sheet.CharacterSheet, error)
}

type CreateCharacterSheetUC struct {
	// add repo
	characterClasses *sync.Map
	factory          *sheet.CharacterSheetFactory
}

func NewCreateCharacterSheetUC(
	charClasses *sync.Map,
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
	input *CreateCharacterSheetInput,
) (*sheet.CharacterSheet, error) {
	if err := uc.validateNickName(input.Profile.NickName); err != nil {
		return nil, err
	}

	class, exists := uc.characterClasses.Load(input.CharacterClass)
	if !exists {
		return nil, fmt.Errorf(
			"character class %s not found",
			input.CharacterClass.String(),
		)
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
	characterSheet, err := uc.factory.Build(input.Profile, &input.CategorySet, &charClass)
	// save to repo
	return characterSheet, err
}

// TODO: validate input.Profile.NickName in repo
func (uc *CreateCharacterSheetUC) validateNickName(nick string) error {
	var allowedNickName = true
	uc.characterClasses.Range(func(_, value any) bool {
		charClass := value.(cc.CharacterClass)
		if charClass.GetName().String() == nick {
			allowedNickName = false
			return false
		}
		return true
	})
	if !allowedNickName {
		return fmt.Errorf("nickname is not allowed")
	}
	return nil
}
