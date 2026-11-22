package submission

import (
	"context"
	"errors"
	"net/http"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/apierr"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	submissionUC "github.com/422UR4H/HxH_RPG_System/internal/application/submission"
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
	uc submissionUC.IRejectCharacterSheetSubmission,
) func(context.Context, *RejectSheetSubmissionRequest) (*RejectSheetSubmissionResponse, error) {

	return func(ctx context.Context, req *RejectSheetSubmissionRequest) (*RejectSheetSubmissionResponse, error) {
		masterUUID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, apierr.NoUser()
		}

		sheetUUID, err := uuid.Parse(req.SheetUUID)
		if err != nil {
			return nil, huma.Error400BadRequest("Ficha inválida.")
		}

		err = uc.Reject(ctx, sheetUUID, masterUUID)
		if err != nil {
			switch {
			case errors.Is(err, campaign.ErrCampaignNotFound):
				return nil, huma.Error404NotFound(apierr.Text(err))
			case errors.Is(err, submissionUC.ErrSubmissionNotFound):
				return nil, huma.Error404NotFound(apierr.Text(err))
			case errors.Is(err, submissionUC.ErrNotCampaignMaster):
				return nil, huma.Error403Forbidden(apierr.Text(err))
			default:
				return nil, apierr.Internal(err)
			}
		}
		return &RejectSheetSubmissionResponse{
			Status: http.StatusNoContent,
		}, nil
	}
}
