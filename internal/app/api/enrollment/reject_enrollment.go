package enrollment

import (
	"context"
	"errors"
	"net/http"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	enrollmentUC "github.com/422UR4H/HxH_RPG_System/internal/application/enrollment"
	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type RejectEnrollmentRequest struct {
	EnrollmentUUID string `path:"uuid" required:"true" doc:"enrollment UUID to reject"`
}

type RejectEnrollmentResponse struct {
	Status int `json:"status"`
}

func RejectEnrollmentHandler(
	uc enrollmentUC.IRejectEnrollment,
) func(context.Context, *RejectEnrollmentRequest) (*RejectEnrollmentResponse, error) {

	return func(ctx context.Context, req *RejectEnrollmentRequest) (*RejectEnrollmentResponse, error) {
		masterUUID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		enrollmentUUID, err := uuid.Parse(req.EnrollmentUUID)
		if err != nil {
			return nil, huma.Error400BadRequest("invalid enrollment UUID")
		}

		err = uc.Reject(ctx, enrollmentUUID, masterUUID)
		if err != nil {
			switch {
			case errors.Is(err, enrollmentUC.ErrEnrollmentNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, match.ErrMatchNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, campaign.ErrCampaignNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, enrollmentUC.ErrNotMatchMaster):
				return nil, huma.Error403Forbidden(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}
		return &RejectEnrollmentResponse{
			Status: http.StatusOK,
		}, nil
	}
}
