package enrollment

import (
	"context"
	"errors"
	"net/http"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	domainCampaign "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	domainEnrollment "github.com/422UR4H/HxH_RPG_System/internal/domain/enrollment"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
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
	uc domainEnrollment.IAcceptEnrollment,
) func(context.Context, *AcceptEnrollmentRequest) (*AcceptEnrollmentResponse, error) {

	return func(ctx context.Context, req *AcceptEnrollmentRequest) (*AcceptEnrollmentResponse, error) {
		masterUUID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		enrollmentUUID, err := uuid.Parse(req.EnrollmentUUID)
		if err != nil {
			return nil, huma.Error400BadRequest("invalid enrollment UUID")
		}

		err = uc.Accept(ctx, enrollmentUUID, masterUUID)
		if err != nil {
			switch {
			case errors.Is(err, domainEnrollment.ErrEnrollmentNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, domainMatch.ErrMatchNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, domainCampaign.ErrCampaignNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, domainEnrollment.ErrNotMatchMaster):
				return nil, huma.Error403Forbidden(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}
		return &AcceptEnrollmentResponse{
			Status: http.StatusOK,
		}, nil
	}
}
