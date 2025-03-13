package sheet

import (
	"context"
	"net/http"

	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
)

type GetCharacterClassRequest struct {
	Name string `path:"name" required:"true" doc:"Character class name"`
}

type GetCharacterClassBody struct {
	CharacterClass CharacterClassResponse
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
			return nil, err
		}
		var response = NewCharacterClassResponse(charClass)

		return &GetCharacterClassResponse{
			Body: GetCharacterClassBody{
				CharacterClass: response,
			},
			Status: http.StatusOK,
		}, nil
	}
}
