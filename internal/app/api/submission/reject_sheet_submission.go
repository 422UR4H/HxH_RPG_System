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

type RejectSheetSubmissionRequest struct {
	SheetUUID string `path:"sheet_uuid" required:"true" doc:"submitted character sheet UUID to reject"`
}

type RejectSheetSubmissionResponse struct {
	Status int `json:"status"`
}

func RejectSheetSubmissionHandler(
	uc domainSubmission.IRejectCharacterSheetSubmission,
) func(context.Context, *RejectSheetSubmissionRequest) (*RejectSheetSubmissionResponse, error) {

	return func(ctx context.Context, req *RejectSheetSubmissionRequest) (*RejectSheetSubmissionResponse, error) {
		masterUUID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		sheetUUID, err := uuid.Parse(req.SheetUUID)
		if err != nil {
			return nil, huma.Error400BadRequest("invalid sheet UUID")
		}

		err = uc.Reject(ctx, sheetUUID, masterUUID)
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
		return &RejectSheetSubmissionResponse{
			Status: http.StatusNoContent,
		}, nil
	}
}
