package enrollment

import (
	"context"
	"errors"
	"net/http"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/apierr"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	enrollmentUC "github.com/422UR4H/HxH_RPG_System/internal/application/enrollment"
	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type AcceptEnrollmentRequest struct {
	EnrollmentUUID string `path:"uuid" required:"true" doc:"enrollment UUID"`
}

type AcceptEnrollmentResponse struct {
	Status int `json:"status"`
}

func AcceptEnrollmentHandler(
	uc enrollmentUC.IAcceptEnrollment,
) func(context.Context, *AcceptEnrollmentRequest) (*AcceptEnrollmentResponse, error) {

	return func(ctx context.Context, req *AcceptEnrollmentRequest) (*AcceptEnrollmentResponse, error) {
		masterUUID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, apierr.NoUser()
		}

		enrollmentUUID, err := uuid.Parse(req.EnrollmentUUID)
		if err != nil {
			return nil, huma.Error400BadRequest("Inscrição inválida.")
		}

		err = uc.Accept(ctx, enrollmentUUID, masterUUID)
		if err != nil {
			switch {
			case errors.Is(err, enrollmentUC.ErrEnrollmentNotFound):
				return nil, huma.Error404NotFound(apierr.Text(err))
			case errors.Is(err, match.ErrMatchNotFound):
				return nil, huma.Error404NotFound(apierr.Text(err))
			case errors.Is(err, campaign.ErrCampaignNotFound):
				return nil, huma.Error404NotFound(apierr.Text(err))
			case errors.Is(err, enrollmentUC.ErrNotMatchMaster):
				return nil, huma.Error403Forbidden(apierr.Text(err))
			default:
				return nil, apierr.Internal(err)
			}
		}
		return &AcceptEnrollmentResponse{
			Status: http.StatusOK,
		}, nil
	}
}
