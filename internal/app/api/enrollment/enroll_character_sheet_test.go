package enrollment_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/enrollment"
	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	domainEnrollment "github.com/422UR4H/HxH_RPG_System/internal/domain/enrollment"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestEnrollCharacterHandler(t *testing.T) {
	playerUUID := uuid.New()

	tests := []struct {
		name       string
		body       map[string]any
		mockFn     func(ctx context.Context, matchUUID, sheetUUID, playerUUID uuid.UUID) error
		wantStatus int
	}{
		{
			name: "success",
			body: map[string]any{
				"sheet_uuid": uuid.New().String(),
				"match_uuid": uuid.New().String(),
			},
			mockFn: func(ctx context.Context, matchUUID, sheetUUID, playerUUID uuid.UUID) error {
				return nil
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "match_not_found",
			body: map[string]any{
				"sheet_uuid": uuid.New().String(),
				"match_uuid": uuid.New().String(),
			},
			mockFn: func(ctx context.Context, matchUUID, sheetUUID, playerUUID uuid.UUID) error {
				return domainMatch.ErrMatchNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "sheet_not_found",
			body: map[string]any{
				"sheet_uuid": uuid.New().String(),
				"match_uuid": uuid.New().String(),
			},
			mockFn: func(ctx context.Context, matchUUID, sheetUUID, playerUUID uuid.UUID) error {
				return charactersheet.ErrCharacterSheetNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "not_sheet_owner",
			body: map[string]any{
				"sheet_uuid": uuid.New().String(),
				"match_uuid": uuid.New().String(),
			},
			mockFn: func(ctx context.Context, matchUUID, sheetUUID, playerUUID uuid.UUID) error {
				return charactersheet.ErrNotCharacterSheetOwner
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "not_in_campaign",
			body: map[string]any{
				"sheet_uuid": uuid.New().String(),
				"match_uuid": uuid.New().String(),
			},
			mockFn: func(ctx context.Context, matchUUID, sheetUUID, playerUUID uuid.UUID) error {
				return domainEnrollment.ErrCharacterNotInCampaign
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "already_enrolled",
			body: map[string]any{
				"sheet_uuid": uuid.New().String(),
				"match_uuid": uuid.New().String(),
			},
			mockFn: func(ctx context.Context, matchUUID, sheetUUID, playerUUID uuid.UUID) error {
				return domainEnrollment.ErrCharacterAlreadyEnrolled
			},
			wantStatus: http.StatusConflict,
		},
		{
			name: "generic_error",
			body: map[string]any{
				"sheet_uuid": uuid.New().String(),
				"match_uuid": uuid.New().String(),
			},
			mockFn: func(ctx context.Context, matchUUID, sheetUUID, playerUUID uuid.UUID) error {
				return errors.New("unexpected database error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)

			mock := &mockEnrollCharacterInMatch{fn: tt.mockFn}
			handler := enrollment.EnrollCharacterHandler(mock)

			huma.Register(api, huma.Operation{
				Method:        http.MethodPost,
				Path:          "/enrollments/charactersheets/enroll",
				DefaultStatus: http.StatusCreated,
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, playerUUID)
			resp := api.PostCtx(ctx, "/enrollments/charactersheets/enroll", tt.body)

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}
		})
	}
}
