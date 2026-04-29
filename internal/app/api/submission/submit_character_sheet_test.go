package submission_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/submission"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	domainSubmission "github.com/422UR4H/HxH_RPG_System/internal/domain/submission"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestSubmitCharacterSheetHandler(t *testing.T) {
	userUUID := uuid.New()

	tests := []struct {
		name       string
		body       map[string]any
		mockFn     func(ctx context.Context, userUUID, sheetUUID, campaignUUID uuid.UUID) error
		wantStatus int
	}{
		{
			name: "success",
			body: map[string]any{
				"sheet_uuid":    uuid.New().String(),
				"campaign_uuid": uuid.New().String(),
			},
			mockFn: func(ctx context.Context, userUUID, sheetUUID, campaignUUID uuid.UUID) error {
				return nil
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "sheet_not_found",
			body: map[string]any{
				"sheet_uuid":    uuid.New().String(),
				"campaign_uuid": uuid.New().String(),
			},
			mockFn: func(ctx context.Context, userUUID, sheetUUID, campaignUUID uuid.UUID) error {
				return charactersheet.ErrCharacterSheetNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "campaign_not_found",
			body: map[string]any{
				"sheet_uuid":    uuid.New().String(),
				"campaign_uuid": uuid.New().String(),
			},
			mockFn: func(ctx context.Context, userUUID, sheetUUID, campaignUUID uuid.UUID) error {
				return campaign.ErrCampaignNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "not_sheet_owner",
			body: map[string]any{
				"sheet_uuid":    uuid.New().String(),
				"campaign_uuid": uuid.New().String(),
			},
			mockFn: func(ctx context.Context, userUUID, sheetUUID, campaignUUID uuid.UUID) error {
				return charactersheet.ErrNotCharacterSheetOwner
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "master_cannot_submit_own_sheet",
			body: map[string]any{
				"sheet_uuid":    uuid.New().String(),
				"campaign_uuid": uuid.New().String(),
			},
			mockFn: func(ctx context.Context, userUUID, sheetUUID, campaignUUID uuid.UUID) error {
				return domainSubmission.ErrMasterCannotSubmitOwnSheet
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "already_submitted",
			body: map[string]any{
				"sheet_uuid":    uuid.New().String(),
				"campaign_uuid": uuid.New().String(),
			},
			mockFn: func(ctx context.Context, userUUID, sheetUUID, campaignUUID uuid.UUID) error {
				return domainSubmission.ErrCharacterAlreadySubmitted
			},
			wantStatus: http.StatusConflict,
		},
		{
			name: "generic_error",
			body: map[string]any{
				"sheet_uuid":    uuid.New().String(),
				"campaign_uuid": uuid.New().String(),
			},
			mockFn: func(ctx context.Context, userUUID, sheetUUID, campaignUUID uuid.UUID) error {
				return errors.New("unexpected database error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)

			mock := &mockSubmitCharacterSheet{fn: tt.mockFn}
			handler := submission.SubmitCharacterSheetHandler(mock)

			huma.Register(api, huma.Operation{
				Method:        http.MethodPost,
				Path:          "/submissions/charactersheets/submit",
				DefaultStatus: http.StatusCreated,
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
			resp := api.PostCtx(ctx, "/submissions/charactersheets/submit", tt.body)

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}
		})
	}
}
