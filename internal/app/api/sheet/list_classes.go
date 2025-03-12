package sheet

import (
	"context"

	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
)

type ListCharacterClassesBody struct {
	CharacterClasses []CharacterClassResponse
}

type ListClassesResponse struct {
	Body   ListCharacterClassesBody
	Status int
}

func ListClassesHandler(
	uc charactersheet.IListCharacterClasses,
) func(context.Context, *struct{}) (*ListClassesResponse, error) {

	return func(_ context.Context, _ *struct{}) (*ListClassesResponse, error) {
		charClasses := uc.ListCharacterClasses()
		var response []CharacterClassResponse

		for _, charClass := range charClasses {
			response = append(response, NewCharacterClassResponse(charClass))
		}
		return &ListClassesResponse{
			Body: ListCharacterClassesBody{
				CharacterClasses: response,
			},
			Status: 200,
		}, nil
	}
}
