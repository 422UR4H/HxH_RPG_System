package enrollment_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/enrollment"
	"github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	"github.com/422UR4H/HxH_RPG_System/internal/application/enrollment"
	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestAcceptEnrollmentHandler(t *testing.T) {
	masterUUID := uuid.New()
	enrollmentUUID := uuid.New()

	tests := []struct {
		name       string
		pathUUID   string
		mockFn     func(ctx context.Context, enrollmentUUID, masterUUID uuid.UUID) error
		wantStatus int
	}{
		{
			name:     "success",
			pathUUID: enrollmentUUID.String(),
			mockFn: func(ctx context.Context, eUUID, mUUID uuid.UUID) error {
				return nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:     "invalid uuid in path",
			pathUUID: "not-a-valid-uuid",
			mockFn: func(ctx context.Context, eUUID, mUUID uuid.UUID) error {
				return nil
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:     "enrollment not found",
			pathUUID: enrollmentUUID.String(),
			mockFn: func(ctx context.Context, eUUID, mUUID uuid.UUID) error {
				return enrollment.ErrEnrollmentNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:     "match not found",
			pathUUID: enrollmentUUID.String(),
			mockFn: func(ctx context.Context, eUUID, mUUID uuid.UUID) error {
				return match.ErrMatchNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:     "campaign not found",
			pathUUID: enrollmentUUID.String(),
			mockFn: func(ctx context.Context, eUUID, mUUID uuid.UUID) error {
				return campaign.ErrCampaignNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:     "not match master",
			pathUUID: enrollmentUUID.String(),
			mockFn: func(ctx context.Context, eUUID, mUUID uuid.UUID) error {
				return enrollment.ErrNotMatchMaster
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:     "generic error",
			pathUUID: enrollmentUUID.String(),
			mockFn: func(ctx context.Context, eUUID, mUUID uuid.UUID) error {
				return errors.New("unexpected database error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)

			mock := &mockAcceptEnrollment{fn: tt.mockFn}
			handler := enrollment.AcceptEnrollmentHandler(mock)

			huma.Register(api, huma.Operation{
				Method: http.MethodPost,
				Path:   "/enrollments/{uuid}/accept",
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, masterUUID)
			resp := api.PostCtx(ctx, "/enrollments/"+tt.pathUUID+"/accept")

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}
		})
	}
}
