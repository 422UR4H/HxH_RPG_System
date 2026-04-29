package submission_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/submission"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	domainSubmission "github.com/422UR4H/HxH_RPG_System/internal/domain/submission"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestAcceptSheetSubmissionHandler(t *testing.T) {
	masterUUID := uuid.New()
	sheetUUID := uuid.New()

	tests := []struct {
		name       string
		pathUUID   string
		mockFn     func(ctx context.Context, sheetUUID, masterUUID uuid.UUID) error
		wantStatus int
	}{
		{
			name:     "success",
			pathUUID: sheetUUID.String(),
			mockFn: func(ctx context.Context, sheetUUID, masterUUID uuid.UUID) error {
				return nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:     "invalid_uuid_in_path",
			pathUUID: "not-a-valid-uuid",
			mockFn: func(ctx context.Context, sheetUUID, masterUUID uuid.UUID) error {
				return nil
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:     "campaign_not_found",
			pathUUID: sheetUUID.String(),
			mockFn: func(ctx context.Context, sheetUUID, masterUUID uuid.UUID) error {
				return campaign.ErrCampaignNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:     "submission_not_found",
			pathUUID: sheetUUID.String(),
			mockFn: func(ctx context.Context, sheetUUID, masterUUID uuid.UUID) error {
				return domainSubmission.ErrSubmissionNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:     "not_campaign_master",
			pathUUID: sheetUUID.String(),
			mockFn: func(ctx context.Context, sheetUUID, masterUUID uuid.UUID) error {
				return domainSubmission.ErrNotCampaignMaster
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:     "generic_error",
			pathUUID: sheetUUID.String(),
			mockFn: func(ctx context.Context, sheetUUID, masterUUID uuid.UUID) error {
				return errors.New("unexpected database error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)

			mock := &mockAcceptCharacterSheetSubmission{fn: tt.mockFn}
			handler := submission.AcceptSheetSubmissionHandler(mock)

			huma.Register(api, huma.Operation{
				Method: http.MethodPost,
				Path:   "/submissions/{sheet_uuid}/accept",
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, masterUUID)
			resp := api.PostCtx(ctx, "/submissions/"+tt.pathUUID+"/accept")

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}
		})
	}
}
