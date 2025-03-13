package sheet

import (
	"context"
	"net/http"

	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/sheet"
	"github.com/danielgtaylor/huma/v2"
)

type CreateCharacterSheetRequestBody struct {
	Profile           sheet.CharacterProfile `json:"profile"`
	CharacterClass    string                 `json:"character_class"`
	SkillsExps        map[string]int         `json:"skills_exps"`
	ProficienciesExps map[string]int         `json:"proficiencies_exps"`
	Categories        map[string]bool        `json:"categories"`
	InitialHexValue   *int                   `json:"initial_hex_value"`
}

type CreateCharacterSheetRequest struct {
	Body CreateCharacterSheetRequestBody `json:"body"`
}

type CreateCharacterSheetResponseBody struct {
	CharacterSheet CharacterSheetResponse `json:"character_sheet"`
}

type CreateCharacterSheetResponse struct {
	Body   CreateCharacterSheetResponseBody `json:"body"`
	Status int                              `json:"status"`
}

func CreateCharacterSheetHandler(
	uc charactersheet.ICreateCharacterSheet,
) func(context.Context, *CreateCharacterSheetRequest) (*CreateCharacterSheetResponse, error) {

	return func(_ context.Context, req *CreateCharacterSheetRequest) (*CreateCharacterSheetResponse, error) {
		input, err := castRequest(&req.Body)
		if err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}

		characterSheet, err := uc.CreateCharacterSheet(input)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity(err.Error())
		}
		response := NewCharacterSheetResponse(characterSheet)

		return &CreateCharacterSheetResponse{
			Body: CreateCharacterSheetResponseBody{
				CharacterSheet: *response,
			},
			Status: http.StatusOK,
		}, nil
	}
}

func castRequest(
	body *CreateCharacterSheetRequestBody,
) (*charactersheet.CreateCharacterSheetInput, error) {

	skillsExps := make(map[enum.SkillName]int)
	for k, v := range body.SkillsExps {
		skillName, err := enum.SkillNameFrom(k)
		if err != nil {
			return nil, err
		}
		skillsExps[skillName] = v
	}

	proficienciesExps := make(map[enum.WeaponName]int)
	for k, v := range body.ProficienciesExps {
		weaponName, err := enum.WeaponNameFrom(k)
		if err != nil {
			return nil, err
		}
		proficienciesExps[weaponName] = v
	}

	charClassName, err := enum.CharacterClassNameFrom(body.CharacterClass)
	if err != nil {
		return nil, err
	}

	categories := make(map[enum.CategoryName]bool)
	for k, v := range body.Categories {
		categoryName, err := enum.CategoryNameFrom(k)
		if err != nil {
			return nil, err
		}
		categories[categoryName] = v
	}
	talentByCategorySet, err := sheet.NewTalentByCategorySet(
		categories, body.InitialHexValue,
	)
	if err != nil {
		return nil, err
	}

	return &charactersheet.CreateCharacterSheetInput{
		Profile:           body.Profile,
		CharacterClass:    charClassName,
		CategorySet:       *talentByCategorySet,
		SkillsExps:        skillsExps,
		ProficienciesExps: proficienciesExps,
	}, nil
}
