package campaign

import (
	"context"
	"errors"

	apiAuth "github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	domainAuth "github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	domainCampaign "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type GetCampaignRequest struct {
	UUID uuid.UUID `path:"uuid" required:"true" doc:"UUID of the campaign to retrieve"`
}

type GetCampaignResponseBody struct {
	Campaign any `json:"campaign"`
}

type GetCampaignResponse struct {
	Body GetCampaignResponseBody `json:"body"`
}

func GetCampaignHandler(
	uc domainCampaign.IGetCampaign,
) func(context.Context, *GetCampaignRequest) (*GetCampaignResponse, error) {

	return func(ctx context.Context, req *GetCampaignRequest) (*GetCampaignResponse, error) {
		userUUID, ok := ctx.Value(apiAuth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		campaign, err := uc.GetCampaign(ctx, req.UUID, userUUID)
		if err != nil {
			switch {
			case errors.Is(err, domainCampaign.ErrCampaignNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, domainAuth.ErrInsufficientPermissions):
				return nil, huma.Error403Forbidden(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}

		var response any
		if campaign.MasterUUID == userUUID {
			response = ToMasterResponse(campaign)
		} else {
			response = ToPlayerResponse(campaign)
		}
		return &GetCampaignResponse{
			Body: GetCampaignResponseBody{
				Campaign: response,
			},
		}, nil
	}
}
