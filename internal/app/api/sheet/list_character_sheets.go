package sheet

import (
	"context"
	"net/http"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type ListCharacterSheetsBody struct {
	CharacterSheets []CharacterPlayerSummaryResponse `json:"character_sheets"`
}

type ListCharacterSheetsResponse struct {
	Body   ListCharacterSheetsBody `json:"body"`
	Status int                     `json:"status"`
}

func ListCharacterSheetsHandler(
	uc charactersheet.IListCharacterSheets,
) func(context.Context, *struct{}) (*ListCharacterSheetsResponse, error) {

	return func(ctx context.Context, _ *struct{}) (*ListCharacterSheetsResponse, error) {
		playerUUID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		sheets, err := uc.ListCharacterSheets(ctx, playerUUID)
		if err != nil {
			return nil, huma.Error500InternalServerError(err.Error())
		}

		responses := make([]CharacterPlayerSummaryResponse, len(sheets))
		for i, sheet := range sheets {
			responses[i] = ToSummaryPlayerResponse(&sheet)
		}

		return &ListCharacterSheetsResponse{
			Body: ListCharacterSheetsBody{
				CharacterSheets: responses,
			},
			Status: http.StatusOK,
		}, nil
	}
}
