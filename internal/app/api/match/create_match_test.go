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
	"github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestCreateMatchHandler(t *testing.T) {
	userUUID := uuid.New()
	now := time.Now()

	tests := []struct {
		name       string
		body       map[string]any
		mockFn     func(ctx context.Context, input *domainMatch.CreateMatchInput) (*matchEntity.Match, error)
		wantStatus int
	}{
		{
			name: "success",
			body: map[string]any{
				"campaign_uuid":            uuid.New().String(),
				"title":                    "Test Match",
				"brief_initial_description": "brief",
				"description":              "full",
				"is_public":                true,
				"game_scheduled_at":        "2026-06-15T19:30:00Z",
				"story_start_at":           "2026-06-15",
			},
			mockFn: func(ctx context.Context, input *domainMatch.CreateMatchInput) (*matchEntity.Match, error) {
				return &matchEntity.Match{
					UUID:                    uuid.New(),
					CampaignUUID:            input.CampaignUUID,
					MasterUUID:              input.MasterUUID,
					Title:                   input.Title,
					BriefInitialDescription: input.BriefInitialDescription,
					Description:             input.Description,
					IsPublic:                input.IsPublic,
					GameScheduledAt:         input.GameScheduledAt,
					StoryStartAt:            input.StoryStartAt,
					CreatedAt:               now,
					UpdatedAt:               now,
				}, nil
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "campaign_not_found",
			body: map[string]any{
				"campaign_uuid":            uuid.New().String(),
				"title":                    "Test Match",
				"brief_initial_description": "brief",
				"description":              "full",
				"is_public":                true,
				"game_scheduled_at":        "2026-06-15T19:30:00Z",
				"story_start_at":           "2026-06-15",
			},
			mockFn: func(ctx context.Context, input *domainMatch.CreateMatchInput) (*matchEntity.Match, error) {
				return nil, campaign.ErrCampaignNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "not_campaign_owner",
			body: map[string]any{
				"campaign_uuid":            uuid.New().String(),
				"title":                    "Test Match",
				"brief_initial_description": "brief",
				"description":              "full",
				"is_public":                true,
				"game_scheduled_at":        "2026-06-15T19:30:00Z",
				"story_start_at":           "2026-06-15",
			},
			mockFn: func(ctx context.Context, input *domainMatch.CreateMatchInput) (*matchEntity.Match, error) {
				return nil, campaign.ErrNotCampaignOwner
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "validation_error",
			body: map[string]any{
				"campaign_uuid":            uuid.New().String(),
				"title":                    "Test Match",
				"brief_initial_description": "brief",
				"description":              "full",
				"is_public":                true,
				"game_scheduled_at":        "2026-06-15T19:30:00Z",
				"story_start_at":           "2026-06-15",
			},
			mockFn: func(ctx context.Context, input *domainMatch.CreateMatchInput) (*matchEntity.Match, error) {
				return nil, domain.ErrValidation
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "internal_server_error",
			body: map[string]any{
				"campaign_uuid":            uuid.New().String(),
				"title":                    "Test Match",
				"brief_initial_description": "brief",
				"description":              "full",
				"is_public":                true,
				"game_scheduled_at":        "2026-06-15T19:30:00Z",
				"story_start_at":           "2026-06-15",
			},
			mockFn: func(ctx context.Context, input *domainMatch.CreateMatchInput) (*matchEntity.Match, error) {
				return nil, errors.New("unexpected db error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)

			mock := &mockCreateMatch{fn: tt.mockFn}
			handler := match.CreateMatchHandler(mock)

			huma.Register(api, huma.Operation{
				Method:        http.MethodPost,
				Path:          "/matches",
				DefaultStatus: http.StatusCreated,
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
			resp := api.PostCtx(ctx, "/matches", tt.body)

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}

			if tt.wantStatus == http.StatusCreated {
				var result map[string]any
				if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				matchData, ok := result["match"].(map[string]any)
				if !ok {
					t.Fatal("response missing 'match' field")
				}
				if matchData["title"] != "Test Match" {
					t.Errorf("got title %v, want 'Test Match'", matchData["title"])
				}
				if matchData["master_uuid"] != userUUID.String() {
					t.Errorf("got master_uuid %v, want %v", matchData["master_uuid"], userUUID.String())
				}
			}
		})
	}
}
