package sheet

import (
	"context"
	"errors"
	"net/http"

	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/danielgtaylor/huma/v2"
)

type GetCharacterClassRequest struct {
	Name string `path:"name" required:"true" doc:"Character class name"`
}

type GetCharacterClassBody struct {
	CharacterClass CharacterClassResponse `json:"characterClass"`
}

type GetCharacterClassResponse struct {
	Body   GetCharacterClassBody
	Status int
}

func GetClassHandler(
	uc charactersheet.IGetCharacterClass,
) func(context.Context, *GetCharacterClassRequest) (*GetCharacterClassResponse, error) {

	return func(_ context.Context, req *GetCharacterClassRequest) (*GetCharacterClassResponse, error) {
		charClass, err := uc.GetCharacterClass(req.Name)
		if err != nil {
			// nome que não é classe e classe não registrada: ambos "não existe"
			switch {
			case errors.Is(err, enum.ErrInvalidNameOf),
				errors.Is(err, charactersheet.ErrCharacterClassNotFound):
				return nil, huma.Error404NotFound(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}
		classSheet, err := uc.GetClassSheet(req.Name)
		if err != nil {
			return nil, huma.Error500InternalServerError(err.Error())
		}
		var response = NewCharacterClassResponse(classSheet, charClass)

		return &GetCharacterClassResponse{
			Body: GetCharacterClassBody{
				CharacterClass: response,
			},
			Status: http.StatusOK,
		}, nil
	}
}
