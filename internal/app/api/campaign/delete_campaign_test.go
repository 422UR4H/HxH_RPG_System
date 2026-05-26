package campaign_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/campaign"
	campaignUC "github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestDeleteCampaignHandler(t *testing.T) {
	userUUID := uuid.New()
	campaignUUID := uuid.New()

	tests := []struct {
		name       string
		uuidPath   string
		mockFn     func(ctx context.Context, input *campaignUC.DeleteCampaignInput) error
		wantStatus int
	}{
		{
			name:     "success",
			uuidPath: campaignUUID.String(),
			mockFn: func(_ context.Context, input *campaignUC.DeleteCampaignInput) error {
				if input.CampaignUUID != campaignUUID {
					t.Errorf("campaign uuid not forwarded: got %v", input.CampaignUUID)
				}
				if input.MasterUUID != userUUID {
					t.Errorf("master uuid not forwarded: got %v", input.MasterUUID)
				}
				return nil
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "invalid_uuid",
			uuidPath:   "not-a-uuid",
			mockFn:     func(_ context.Context, _ *campaignUC.DeleteCampaignInput) error { return nil },
			wantStatus: http.StatusBadRequest,
		},
		{
			name:     "campaign_not_found",
			uuidPath: campaignUUID.String(),
			mockFn: func(_ context.Context, _ *campaignUC.DeleteCampaignInput) error {
				return campaignUC.ErrCampaignNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:     "not_owner",
			uuidPath: campaignUUID.String(),
			mockFn: func(_ context.Context, _ *campaignUC.DeleteCampaignInput) error {
				return campaignUC.ErrNotCampaignOwner
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:     "has_started_match",
			uuidPath: campaignUUID.String(),
			mockFn: func(_ context.Context, _ *campaignUC.DeleteCampaignInput) error {
				return campaignUC.ErrCampaignHasStartedMatch
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:     "internal_server_error",
			uuidPath: campaignUUID.String(),
			mockFn: func(_ context.Context, _ *campaignUC.DeleteCampaignInput) error {
				return errors.New("db error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)
			handler := campaign.DeleteCampaignHandler(&mockDeleteCampaign{fn: tt.mockFn})

			huma.Register(api, huma.Operation{
				Method: http.MethodDelete,
				Path:   "/campaigns/{uuid}",
				Errors: []int{
					http.StatusBadRequest, http.StatusNotFound,
					http.StatusForbidden, http.StatusUnprocessableEntity,
					http.StatusInternalServerError,
				},
				DefaultStatus: http.StatusNoContent,
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
			resp := api.DeleteCtx(ctx, "/campaigns/"+tt.uuidPath)

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}
		})
	}
}
