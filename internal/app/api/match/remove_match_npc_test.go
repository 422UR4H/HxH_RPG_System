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

func TestRemoveMatchNPCHandler(t *testing.T) {
	userUUID := uuid.New()
	matchUUID := uuid.New()
	sheetUUID := uuid.New()

	tests := []struct {
		name       string
		mockFn     func(ctx context.Context, input *matchUC.RemoveMatchNPCInput) error
		wantStatus int
	}{
		{
			name: "success",
			mockFn: func(_ context.Context, input *matchUC.RemoveMatchNPCInput) error {
				if input.RequesterUUID != userUUID {
					t.Errorf("requester uuid not forwarded: got %v", input.RequesterUUID)
				}
				if input.MatchUUID != matchUUID {
					t.Errorf("match uuid not forwarded: got %v", input.MatchUUID)
				}
				if input.SheetUUID != sheetUUID {
					t.Errorf("sheet uuid not forwarded: got %v", input.SheetUUID)
				}
				return nil
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name: "match_not_found",
			mockFn: func(_ context.Context, _ *matchUC.RemoveMatchNPCInput) error {
				return matchUC.ErrMatchNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "npc_not_in_match",
			mockFn: func(_ context.Context, _ *matchUC.RemoveMatchNPCInput) error {
				return matchUC.ErrNPCNotInMatch
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "not_master",
			mockFn: func(_ context.Context, _ *matchUC.RemoveMatchNPCInput) error {
				return matchUC.ErrNotMatchMaster
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "internal_server_error",
			mockFn: func(_ context.Context, _ *matchUC.RemoveMatchNPCInput) error {
				return errors.New("db error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)
			handler := match.RemoveMatchNPCHandler(&mockRemoveMatchNPC{fn: tt.mockFn})

			huma.Register(api, huma.Operation{
				Method: http.MethodDelete,
				Path:   "/matches/{uuid}/npcs/{sheet_uuid}",
				Errors: []int{
					http.StatusBadRequest, http.StatusForbidden,
					http.StatusNotFound, http.StatusUnprocessableEntity,
					http.StatusInternalServerError,
				},
				DefaultStatus: http.StatusNoContent,
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
			resp := api.DeleteCtx(ctx, "/matches/"+matchUUID.String()+"/npcs/"+sheetUUID.String())

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}
		})
	}
}
