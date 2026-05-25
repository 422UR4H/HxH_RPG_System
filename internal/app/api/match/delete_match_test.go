package match_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/match"
	matchUC "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestDeleteMatchHandler(t *testing.T) {
	userUUID := uuid.New()
	matchUUID := uuid.New()

	tests := []struct {
		name       string
		uuidPath   string
		mockFn     func(ctx context.Context, input *matchUC.DeleteMatchInput) error
		wantStatus int
	}{
		{
			name:     "success",
			uuidPath: matchUUID.String(),
			mockFn: func(_ context.Context, input *matchUC.DeleteMatchInput) error {
				if input.MatchUUID != matchUUID {
					t.Errorf("match uuid not forwarded: got %v", input.MatchUUID)
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
			mockFn:     func(_ context.Context, _ *matchUC.DeleteMatchInput) error { return nil },
			wantStatus: http.StatusBadRequest,
		},
		{
			name:     "match_not_found",
			uuidPath: matchUUID.String(),
			mockFn: func(_ context.Context, _ *matchUC.DeleteMatchInput) error {
				return matchUC.ErrMatchNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:     "not_master",
			uuidPath: matchUUID.String(),
			mockFn: func(_ context.Context, _ *matchUC.DeleteMatchInput) error {
				return matchUC.ErrNotMatchMaster
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:     "already_started",
			uuidPath: matchUUID.String(),
			mockFn: func(_ context.Context, _ *matchUC.DeleteMatchInput) error {
				return matchUC.ErrMatchAlreadyStarted
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:     "internal_server_error",
			uuidPath: matchUUID.String(),
			mockFn: func(_ context.Context, _ *matchUC.DeleteMatchInput) error {
				return errors.New("db error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)
			handler := match.DeleteMatchHandler(&mockDeleteMatch{fn: tt.mockFn})

			huma.Register(api, huma.Operation{
				Method: http.MethodDelete,
				Path:   "/matches/{uuid}",
				Errors: []int{
					http.StatusBadRequest, http.StatusNotFound,
					http.StatusForbidden, http.StatusUnprocessableEntity,
					http.StatusInternalServerError,
				},
				DefaultStatus: http.StatusNoContent,
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
			resp := api.DeleteCtx(ctx, "/matches/"+tt.uuidPath)

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}
		})
	}
}
