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
	"github.com/422UR4H/HxH_RPG_System/internal/domain"
	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestUpdateMatchHandler(t *testing.T) {
	userUUID := uuid.New()
	matchUUID := uuid.New()
	now := time.Now()

	baseResp := func(title string) *matchEntity.Match {
		return &matchEntity.Match{
			UUID:                    matchUUID,
			MasterUUID:              userUUID,
			CampaignUUID:            uuid.New(),
			Title:                   title,
			BriefInitialDescription: "brief",
			Description:             "full",
			IsPublic:                true,
			GameScheduledAt:         now,
			StoryStartAt:            now,
			CreatedAt:               now,
			UpdatedAt:               now,
		}
	}

	tests := []struct {
		name       string
		body       map[string]any
		mockFn     func(ctx context.Context, input *match.UpdateMatchInput) (*matchEntity.Match, error)
		wantStatus int
	}{
		{
			name: "success_full_patch",
			body: map[string]any{
				"title":                     "Patched Title",
				"brief_initial_description": "Patched brief",
				"description":               "Patched desc",
				"is_public":                 false,
				"game_scheduled_at":         "2026-07-20T19:30:00Z",
				"story_start_at":            "2026-07-20",
			},
			mockFn: func(_ context.Context, input *match.UpdateMatchInput) (*matchEntity.Match, error) {
				if input.Title == nil || *input.Title != "Patched Title" {
					t.Errorf("title not forwarded: %+v", input.Title)
				}
				return baseResp("Patched Title"), nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "success_partial_patch_title_only",
			body: map[string]any{"title": "Only Title"},
			mockFn: func(_ context.Context, input *match.UpdateMatchInput) (*matchEntity.Match, error) {
				if input.BriefInitialDescription != nil {
					t.Errorf("brief should be nil, got %+v", *input.BriefInitialDescription)
				}
				return baseResp("Only Title"), nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "success_empty_body_is_noop",
			body: map[string]any{},
			mockFn: func(_ context.Context, _ *match.UpdateMatchInput) (*matchEntity.Match, error) {
				return baseResp("Original"), nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "invalid_game_scheduled_at",
			body: map[string]any{"game_scheduled_at": "not-a-date"},
			mockFn: func(_ context.Context, _ *match.UpdateMatchInput) (*matchEntity.Match, error) {
				t.Fatal("UC should not be called when date parsing fails")
				return nil, nil
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "invalid_story_start_at",
			body: map[string]any{"story_start_at": "not-a-date"},
			mockFn: func(_ context.Context, _ *match.UpdateMatchInput) (*matchEntity.Match, error) {
				t.Fatal("UC should not be called when date parsing fails")
				return nil, nil
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "match_not_found",
			body: map[string]any{"title": "anything"},
			mockFn: func(_ context.Context, _ *match.UpdateMatchInput) (*matchEntity.Match, error) {
				return nil, match.ErrMatchNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "not_master",
			body: map[string]any{"title": "anything"},
			mockFn: func(_ context.Context, _ *match.UpdateMatchInput) (*matchEntity.Match, error) {
				return nil, match.ErrNotMatchMaster
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "already_started",
			body: map[string]any{"title": "anything"},
			mockFn: func(_ context.Context, _ *match.UpdateMatchInput) (*matchEntity.Match, error) {
				return nil, match.ErrMatchAlreadyStarted
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "already_finished",
			body: map[string]any{"title": "anything"},
			mockFn: func(_ context.Context, _ *match.UpdateMatchInput) (*matchEntity.Match, error) {
				return nil, match.ErrMatchAlreadyFinished
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "validation_error",
			body: map[string]any{"title": "anything"},
			mockFn: func(_ context.Context, _ *match.UpdateMatchInput) (*matchEntity.Match, error) {
				return nil, domain.ErrValidation
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "internal_server_error",
			body: map[string]any{"title": "anything"},
			mockFn: func(_ context.Context, _ *match.UpdateMatchInput) (*matchEntity.Match, error) {
				return nil, errors.New("db down")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)
			mock := &mockUpdateMatch{fn: tt.mockFn}
			handler := match.UpdateMatchHandler(mock)

			huma.Register(api, huma.Operation{
				Method: http.MethodPatch,
				Path:   "/matches/{uuid}",
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
			resp := api.PatchCtx(ctx, "/matches/"+matchUUID.String(), tt.body)

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				var result map[string]any
				if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				matchData, ok := result["match"].(map[string]any)
				if !ok {
					t.Fatal("response missing 'match' field")
				}
				if matchData["master_uuid"] != userUUID.String() {
					t.Errorf("master_uuid = %v, want %v", matchData["master_uuid"], userUUID.String())
				}
			}
		})
	}
}
