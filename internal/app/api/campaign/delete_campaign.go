package campaign

import (
	"context"
	"errors"
	"net/http"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/apierr"
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
			return nil, apierr.NoUser()
		}

		campaignUUID, err := uuid.Parse(req.UUID)
		if err != nil {
			return nil, huma.Error400BadRequest("Campanha inválida.")
		}

		err = uc.Delete(ctx, &campaignUC.DeleteCampaignInput{
			CampaignUUID: campaignUUID,
			MasterUUID:   userUUID,
		})
		if err != nil {
			switch {
			case errors.Is(err, campaignUC.ErrCampaignNotFound):
				return nil, huma.Error404NotFound(apierr.Text(err))
			case errors.Is(err, campaignUC.ErrNotCampaignOwner):
				return nil, huma.Error403Forbidden(apierr.Text(err))
			case errors.Is(err, campaignUC.ErrCampaignHasStartedMatch):
				return nil, huma.Error422UnprocessableEntity(apierr.Text(err))
			default:
				return nil, apierr.Internal(err)
			}
		}

		return &DeleteCampaignResponse{Status: http.StatusNoContent}, nil
	}
}
