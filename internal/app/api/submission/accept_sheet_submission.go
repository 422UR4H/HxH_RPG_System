package submission

import (
	"context"
	"errors"
	"net/http"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	domainCampaign "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	domainSubmission "github.com/422UR4H/HxH_RPG_System/internal/domain/submission"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type AcceptSheetSubmissionRequest struct {
	SheetUUID string `path:"sheet_uuid" required:"true" doc:"submitted character sheet UUID"`
}

type AcceptSheetSubmissionResponse struct {
	Status int `json:"status"`
}

func AcceptSheetSubmissionHandler(
	uc domainSubmission.IAcceptCharacterSheetSubmission,
) func(context.Context, *AcceptSheetSubmissionRequest) (*AcceptSheetSubmissionResponse, error) {

	return func(ctx context.Context, req *AcceptSheetSubmissionRequest) (*AcceptSheetSubmissionResponse, error) {
		masterUUID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		sheetUUID, err := uuid.Parse(req.SheetUUID)
		if err != nil {
			return nil, huma.Error400BadRequest("invalid sheet UUID")
		}

		err = uc.Accept(ctx, sheetUUID, masterUUID)
		if err != nil {
			switch {
			case errors.Is(err, domainCampaign.ErrCampaignNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, domainSubmission.ErrSubmissionNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, domainSubmission.ErrNotCampaignMaster):
				return nil, huma.Error403Forbidden(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}
		return &AcceptSheetSubmissionResponse{
			Status: http.StatusOK,
		}, nil
	}
}
