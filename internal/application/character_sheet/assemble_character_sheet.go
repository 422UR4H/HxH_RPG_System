package charactersheet

import (
	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/google/uuid"
)

// SheetOwner agrupa os donos de uma ficha montada. Exatamente um de PlayerUUID e
// MasterUUID deve vir preenchido (regra de sheet.NewCharacterSheet).
type SheetOwner struct {
	PlayerUUID   *uuid.UUID
	MasterUUID   *uuid.UUID
	CampaignUUID *uuid.UUID
}

// validateDistribution confere a distribuição do jogador contra a classe.
func validateDistribution(charClass cc.CharacterClass, input *CreateCharacterSheetInput) error {
	if err := charClass.ValidateSkills(input.SkillsExps); err != nil {
		return err
	}
	return charClass.ValidateProficiencies(input.ProficienciesExps)
}

// AssembleCharacterSheet monta a ficha a partir da classe e do que o jogador
// distribuiu (perícias, proficiências, pontos de atributo). É o único caminho
// de montagem: criação e edição passam por aqui, e o teste de paridade com o
// front (internal/app/api/sheet/sheet_parity_test.go) também — o motor de
// prévia do front (src/features/sheet/engine) repete exatamente este cálculo.
//
// charClass é recebido por valor: ApplySkills/ApplyProficiencies são
// copy-on-write, então a classe guardada no cache nunca é alterada.
func AssembleCharacterSheet(
	factory *sheet.CharacterSheetFactory,
	charClass cc.CharacterClass,
	owner SheetOwner,
	input *CreateCharacterSheetInput,
) (*sheet.CharacterSheet, error) {
	if err := validateDistribution(charClass, input); err != nil {
		return nil, err
	}
	charClass.ApplySkills(input.SkillsExps)
	charClass.ApplyProficiencies(input.ProficienciesExps)

	charSheet, err := factory.Build(
		owner.PlayerUUID,
		owner.MasterUUID,
		owner.CampaignUUID,
		input.Profile,
		nil,
		nil,
		&charClass,
	)
	if err != nil {
		return nil, err
	}

	if len(input.AttributePoints) > 0 {
		if err := charSheet.ApplyInitialAttributePoints(input.AttributePoints); err != nil {
			return nil, err
		}
	}
	return charSheet, nil
}
