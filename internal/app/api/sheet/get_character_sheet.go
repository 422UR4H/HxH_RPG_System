package sheet

import (
	"context"
	"errors"
	"net/http"

	cs "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

// TODO: evaluate adding campaignUUID to get of campaign sync.Map
type GetCharacterSheetRequest struct {
	UUID string `path:"uuid" required:"true" doc:"UUID of the character sheet"`
}

type GetCharacterSheetResponseBody struct {
	CharacterSheet CharacterSheetResponse `json:"character_sheet"`
}

type GetCharacterSheetResponse struct {
	Body   GetCharacterSheetResponseBody `json:"body"`
	Status int                           `json:"status"`
}

func GetCharacterSheetHandler(
	uc cs.IGetCharacterSheet,
) func(context.Context, *GetCharacterSheetRequest) (*GetCharacterSheetResponse, error) {

	return func(ctx context.Context, req *GetCharacterSheetRequest) (*GetCharacterSheetResponse, error) {

		uuid, err := uuid.Parse(req.UUID)
		if err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}

		characterSheet, err := uc.GetCharacterSheet(ctx, uuid)
		if err != nil {
			if errors.Is(err, cs.ErrCharacterSheetNotFound) {
				return nil, huma.Error404NotFound(err.Error())
			}
			return nil, huma.Error500InternalServerError(err.Error())
		}
		response := NewCharacterSheetResponse(characterSheet)

		return &GetCharacterSheetResponse{
			Body: GetCharacterSheetResponseBody{
				CharacterSheet: *response,
			},
			Status: http.StatusOK,
		}, nil
	}
}
