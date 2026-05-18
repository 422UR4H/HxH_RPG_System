package sheet

import (
	"context"
	"errors"
	"net/http"

	apiAuth "github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	domainAuth "github.com/422UR4H/HxH_RPG_System/internal/application/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	cs "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type UpdateCharacterSheetRequest struct {
	UUID string                           `path:"uuid" required:"true"`
	Body CreateCharacterSheetRequestBody
}

type UpdateCharacterSheetResponseBody struct {
	CharacterSheet CharacterSheetResponse `json:"character_sheet"`
}

type UpdateCharacterSheetResponse struct {
	Body   UpdateCharacterSheetResponseBody `json:"body"`
	Status int                              `json:"status"`
}

func UpdateCharacterSheetHandler(
	uc cs.IUpdateCharacterSheet,
	getUC cs.IGetCharacterSheet,
) func(context.Context, *UpdateCharacterSheetRequest) (*UpdateCharacterSheetResponse, error) {
	return func(ctx context.Context, req *UpdateCharacterSheetRequest) (*UpdateCharacterSheetResponse, error) {
		userUUID, ok := ctx.Value(apiAuth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		sheetUUID, err := uuid.Parse(req.UUID)
		if err != nil {
			return nil, huma.Error400BadRequest("invalid uuid")
		}

		input, err := castRequest(&req.Body)
		if err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}

		if err := uc.UpdateCharacterSheet(ctx, sheetUUID, userUUID, input); err != nil {
			switch {
			case errors.Is(err, cs.ErrCharacterSheetNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, cs.ErrCharacterClassNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, domainAuth.ErrInsufficientPermissions):
				return nil, huma.Error403Forbidden(err.Error())
			case errors.Is(err, cs.ErrCharacterSheetNotFreeToManage):
				return nil, huma.Error422UnprocessableEntity(err.Error())
			case errors.Is(err, campaign.ErrCampaignNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, domain.ErrValidation):
				return nil, huma.Error422UnprocessableEntity(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}

		charSheet, err := getUC.GetCharacterSheet(ctx, sheetUUID, userUUID)
		if err != nil {
			return nil, huma.Error500InternalServerError(err.Error())
		}

		return &UpdateCharacterSheetResponse{
			Body:   UpdateCharacterSheetResponseBody{CharacterSheet: *NewCharacterSheetResponse(charSheet)},
			Status: http.StatusOK,
		}, nil
	}
}
