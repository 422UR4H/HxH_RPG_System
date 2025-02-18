package charactersheet

import (
	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/sheet"
)

type ICreateCharacterSheet interface {
	CreateCharacterSheet() *sheet.CharacterSheet
}

type CreateCharacterSheetUC struct {
	// repo
}

func NewCreateCharacterSheetUC() *CreateCharacterSheetUC {
	return &CreateCharacterSheetUC{
		// repo
	}
}

type CreateCharacterSheetInput struct {
	characterClass cc.CharacterClass
	profile        sheet.CharacterProfile
	set            sheet.TalentByCategorySet
}

func (uc *CreateCharacterSheetUC) CreateCharacterSheet(
	input CreateCharacterSheetInput,
) *sheet.CharacterSheet {
	factory := sheet.NewCharacterSheetFactory()
	// TODO: validate character class
	// validar se todas as proficienciesExps e skillsExps existem em characterClasses global
	// as que não existirem, validar se existem nos alloweds e verificar a quantidade exata
	characterSheet := factory.Build(input.profile, input.set, &input.characterClass)
	return characterSheet
}
