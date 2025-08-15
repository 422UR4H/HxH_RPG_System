package sheet

import (
	"context"
	"net/http"

	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	characterclass "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
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
		var response []CharacterClassResponse
		charClasses := uc.ListCharacterClasses()
		classSheets := uc.ListClassSheets()
		classes := make(map[enum.CharacterClassName]characterclass.CharacterClass)

		for _, charClass := range charClasses {
			classes[charClass.GetName()] = charClass
		}
		for _, classSheet := range classSheets {
			charClass := classes[classSheet.GetClass()]
			response = append(response, NewCharacterClassResponse(classSheet, charClass))
		}
		return &ListCharacterClassesResponse{
			Body: ListCharacterClassesBody{
				CharacterClasses: response,
			},
			Status: http.StatusOK,
		}, nil
	}
}
