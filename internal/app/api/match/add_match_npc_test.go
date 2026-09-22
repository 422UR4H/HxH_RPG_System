package match_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/match"
	matchUC "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestAddMatchNPCHandler(t *testing.T) {
	userUUID := uuid.New()
	matchUUID := uuid.New()
	sheetUUID := uuid.New()
	participantUUID := uuid.New()
	joinedAt := time.Now()

	tests := []struct {
		name       string
		mockFn     func(ctx context.Context, input *matchUC.AddMatchNPCInput) (*matchEntity.Participant, error)
		wantStatus int
	}{
		{
			name: "success",
			mockFn: func(_ context.Context, input *matchUC.AddMatchNPCInput) (*matchEntity.Participant, error) {
				if input.RequesterUUID != userUUID {
					t.Errorf("requester uuid not forwarded: got %v", input.RequesterUUID)
				}
				if input.MatchUUID != matchUUID {
					t.Errorf("match uuid not forwarded: got %v", input.MatchUUID)
				}
				if input.SheetUUID != sheetUUID {
					t.Errorf("sheet uuid not forwarded: got %v", input.SheetUUID)
				}
				return &matchEntity.Participant{
					UUID:      participantUUID,
					MatchUUID: matchUUID,
					Sheet:     csEntity.Summary{UUID: sheetUUID},
					JoinedAt:  joinedAt,
				}, nil
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "match_not_found",
			mockFn: func(_ context.Context, _ *matchUC.AddMatchNPCInput) (*matchEntity.Participant, error) {
				return nil, matchUC.ErrMatchNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "sheet_not_found",
			mockFn: func(_ context.Context, _ *matchUC.AddMatchNPCInput) (*matchEntity.Participant, error) {
				return nil, matchUC.ErrCharacterSheetNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "not_master",
			mockFn: func(_ context.Context, _ *matchUC.AddMatchNPCInput) (*matchEntity.Participant, error) {
				return nil, matchUC.ErrNotMatchMaster
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "sheet_not_npc",
			mockFn: func(_ context.Context, _ *matchUC.AddMatchNPCInput) (*matchEntity.Participant, error) {
				return nil, matchUC.ErrSheetNotNPC
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "sheet_not_owned_by_master",
			mockFn: func(_ context.Context, _ *matchUC.AddMatchNPCInput) (*matchEntity.Participant, error) {
				return nil, matchUC.ErrSheetNotOwnedByMaster
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "match_already_finished",
			mockFn: func(_ context.Context, _ *matchUC.AddMatchNPCInput) (*matchEntity.Participant, error) {
				return nil, matchUC.ErrMatchAlreadyFinished
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "npc_already_in_match",
			mockFn: func(_ context.Context, _ *matchUC.AddMatchNPCInput) (*matchEntity.Participant, error) {
				return nil, matchUC.ErrNPCAlreadyInMatch
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "internal_server_error",
			mockFn: func(_ context.Context, _ *matchUC.AddMatchNPCInput) (*matchEntity.Participant, error) {
				return nil, errors.New("db error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)
			handler := match.AddMatchNPCHandler(&mockAddMatchNPC{fn: tt.mockFn})

			huma.Register(api, huma.Operation{
				Method: http.MethodPost,
				Path:   "/matches/{uuid}/npcs",
				Errors: []int{
					http.StatusBadRequest, http.StatusForbidden,
					http.StatusNotFound, http.StatusUnprocessableEntity,
					http.StatusInternalServerError,
				},
				DefaultStatus: http.StatusCreated,
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
			body := map[string]any{"characterSheetUuid": sheetUUID.String()}
			resp := api.PostCtx(ctx, "/matches/"+matchUUID.String()+"/npcs", body)

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}

			if tt.wantStatus == http.StatusCreated {
				var result map[string]any
				if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				participant, ok := result["participant"].(map[string]any)
				if !ok {
					t.Fatal("response missing 'participant' field")
				}
				if participant["uuid"] != participantUUID.String() {
					t.Errorf("got uuid %v, want %v", participant["uuid"], participantUUID.String())
				}
				if participant["matchUuid"] != matchUUID.String() {
					t.Errorf("got matchUuid %v, want %v", participant["matchUuid"], matchUUID.String())
				}
				if participant["characterSheetUuid"] != sheetUUID.String() {
					t.Errorf("got characterSheetUuid %v, want %v", participant["characterSheetUuid"], sheetUUID.String())
				}
				if participant["joinedAt"] != joinedAt.Format(time.RFC3339) {
					t.Errorf("got joinedAt %v, want %v", participant["joinedAt"], joinedAt.Format(time.RFC3339))
				}
			}
		})
	}
}
