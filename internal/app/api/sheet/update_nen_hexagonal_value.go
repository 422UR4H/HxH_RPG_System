package sheet

import (
	"context"
	"errors"
	"net/http"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	cs "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/spiritual"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

// TODO: evaluate adding campaignUUID to get of campaign sync.Map
type UpdateNenHexagonValueRequest struct {
	CharSheetUUID string `path:"character_sheet_uuid" required:"true" doc:"UUID of the character sheet"`
	Method        string `path:"method" enum:"increase,decrease" required:"true" doc:"Method to update (increase or decrease) the nen hexagon value"`
}

type UpdateNenHexagonValueResponseBody struct {
	PercentList     map[enum.CategoryName]float64 `json:"percent_list"`
	CategoryName    enum.CategoryName             `json:"category_name"`
	CurrentHexValue int                           `json:"current_hex_value"`
}

type UpdateNenHexagonValueResponse struct {
	Body   UpdateNenHexagonValueResponseBody `json:"body"`
	Status int                               `json:"status"`
}

func UpdateNenHexagonValueHandler(
	hexValUC cs.IUpdateNenHexagonValue,
	charSheetUC cs.IGetCharacterSheet,
) func(context.Context, *UpdateNenHexagonValueRequest) (*UpdateNenHexagonValueResponse, error) {

	return func(ctx context.Context, req *UpdateNenHexagonValueRequest) (*UpdateNenHexagonValueResponse, error) {
		playerUUID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		charSheetId, err := uuid.Parse(req.CharSheetUUID)
		if err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}

		characterSheet, err := charSheetUC.GetCharacterSheet(ctx, charSheetId, playerUUID)
		if err != nil {
			if errors.Is(err, cs.ErrCharacterSheetNotFound) {
				return nil, huma.Error404NotFound(err.Error())
			}
			return nil, huma.Error500InternalServerError(err.Error())
		}

		if characterSheet.GetCurrHexValue() == nil {
			return nil, huma.Error422UnprocessableEntity(cs.ErrNenHexNotInitialized.Error())
		}

		result, err := hexValUC.UpdateNenHexagonValue(ctx, characterSheet, req.Method)
		if err != nil {
			if errors.Is(err, spiritual.ErrNenHexNotInitialized) {
				return nil, huma.Error400BadRequest(err.Error())
			}
			return nil, huma.Error500InternalServerError(err.Error())
		}

		return &UpdateNenHexagonValueResponse{
			Body: UpdateNenHexagonValueResponseBody{
				PercentList:     result.PercentList,
				CategoryName:    result.CategoryName,
				CurrentHexValue: result.CurrentHexVal,
			},
			Status: http.StatusOK,
		}, nil
	}
}
