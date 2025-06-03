package sheet

import (
	"context"
	"errors"
	"net/http"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type SubmitCharacterRequestBody struct {
	SheetUUID    uuid.UUID `json:"sheet_uuid" required:"true" doc:"UUID of the character sheet"`
	CampaignUUID uuid.UUID `json:"campaign_uuid" required:"true" doc:"UUID of the campaign"`
}

type SubmitCharacterRequest struct {
	Body SubmitCharacterRequestBody `json:"body"`
}

type SubmitCharacterSheetResponse struct {
	Status int `json:"status"`
}

func SubmitCharacterSheetHandler(
	uc charactersheet.ISubmitCharacterSheet,
) func(context.Context, *SubmitCharacterRequest) (*SubmitCharacterSheetResponse, error) {

	return func(ctx context.Context, req *SubmitCharacterRequest) (*SubmitCharacterSheetResponse, error) {
		userUUID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		err := uc.Submit(ctx, userUUID, req.Body.SheetUUID, req.Body.CampaignUUID)
		if err != nil {
			switch {
			case errors.Is(err, charactersheet.ErrCharacterSheetNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, campaign.ErrCampaignNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, charactersheet.ErrNotCharacterSheetOwner):
				return nil, huma.Error403Forbidden(err.Error())
			case errors.Is(err, charactersheet.ErrMasterCannotSubmitOwnSheet):
				return nil, huma.Error403Forbidden(err.Error())
			case errors.Is(err, charactersheet.ErrCharacterAlreadySubmitted):
				return nil, huma.Error409Conflict(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}
		return &SubmitCharacterSheetResponse{
			Status: http.StatusCreated,
		}, nil
	}
}
