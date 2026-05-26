package campaign

import (
	"context"
	"errors"
	"net/http"

	apiAuth "github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	campaignUC "github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type DeleteCampaignRequest struct {
	UUID string `path:"uuid" required:"true"`
}

type DeleteCampaignResponse struct {
	Status int
}

func DeleteCampaignHandler(
	uc campaignUC.IDeleteCampaign,
) func(context.Context, *DeleteCampaignRequest) (*DeleteCampaignResponse, error) {
	return func(ctx context.Context, req *DeleteCampaignRequest) (*DeleteCampaignResponse, error) {
		userUUID, ok := ctx.Value(apiAuth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		campaignUUID, err := uuid.Parse(req.UUID)
		if err != nil {
			return nil, huma.Error400BadRequest("invalid uuid")
		}

		err = uc.Delete(ctx, &campaignUC.DeleteCampaignInput{
			CampaignUUID: campaignUUID,
			MasterUUID:   userUUID,
		})
		if err != nil {
			switch {
			case errors.Is(err, campaignUC.ErrCampaignNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, campaignUC.ErrNotCampaignOwner):
				return nil, huma.Error403Forbidden(err.Error())
			case errors.Is(err, campaignUC.ErrCampaignHasStartedMatch):
				return nil, huma.Error422UnprocessableEntity(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}

		return &DeleteCampaignResponse{Status: http.StatusNoContent}, nil
	}
}
