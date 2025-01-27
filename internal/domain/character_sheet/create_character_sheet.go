package charactersheet

import "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/sheet"

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
	profile sheet.CharacterProfile
	set     sheet.TalentByCategorySet
}

func (uc *CreateCharacterSheetUC) CreateCharacterSheet(
	input CreateCharacterSheetInput,
) *sheet.CharacterSheet {
	factory := sheet.NewCharacterSheetFactory()
	characterSheet := factory.Build(input.profile, input.set)
	return characterSheet
}
