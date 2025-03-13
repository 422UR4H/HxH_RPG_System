package sheet

import (
	"context"
	"net/http"

	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
)

type ListCharacterClassesBody struct {
	CharacterClasses []CharacterClassResponse
}

type ListCharacterClassesResponse struct {
	Body   ListCharacterClassesBody
	Status int
}

func ListClassesHandler(
	uc charactersheet.IListCharacterClasses,
) func(context.Context, *struct{}) (*ListCharacterClassesResponse, error) {

	return func(_ context.Context, _ *struct{}) (*ListCharacterClassesResponse, error) {
		charClasses := uc.ListCharacterClasses()
		var response []CharacterClassResponse

		for _, charClass := range charClasses {
			response = append(response, NewCharacterClassResponse(charClass))
		}
		return &ListCharacterClassesResponse{
			Body: ListCharacterClassesBody{
				CharacterClasses: response,
			},
			Status: http.StatusOK,
		}, nil
	}
}
