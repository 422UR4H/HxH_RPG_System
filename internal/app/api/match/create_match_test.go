package match_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/match"
	"github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	matchUC "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
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
		mockFn     func(ctx context.Context, input *matchUC.CreateMatchInput) (*matchEntity.Match, error)
		wantStatus int
	}{
		{
			name: "success",
			body: map[string]any{
				"campaignUuid":            uuid.New().String(),
				"title":                   "Test Match",
				"briefInitialDescription": "brief",
				"description":             "full",
				"isPublic":                true,
				"gameScheduledAt":         "2026-06-15T19:30:00Z",
				"storyStartAt":            "2026-06-15",
			},
			mockFn: func(ctx context.Context, input *matchUC.CreateMatchInput) (*matchEntity.Match, error) {
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
				"campaignUuid":            uuid.New().String(),
				"title":                   "Test Match",
				"briefInitialDescription": "brief",
				"description":             "full",
				"isPublic":                true,
				"gameScheduledAt":         "2026-06-15T19:30:00Z",
				"storyStartAt":            "2026-06-15",
			},
			mockFn: func(ctx context.Context, input *matchUC.CreateMatchInput) (*matchEntity.Match, error) {
				return nil, campaign.ErrCampaignNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "not_campaign_owner",
			body: map[string]any{
				"campaignUuid":            uuid.New().String(),
				"title":                   "Test Match",
				"briefInitialDescription": "brief",
				"description":             "full",
				"isPublic":                true,
				"gameScheduledAt":         "2026-06-15T19:30:00Z",
				"storyStartAt":            "2026-06-15",
			},
			mockFn: func(ctx context.Context, input *matchUC.CreateMatchInput) (*matchEntity.Match, error) {
				return nil, campaign.ErrNotCampaignOwner
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "validation_error",
			body: map[string]any{
				"campaignUuid":            uuid.New().String(),
				"title":                   "Test Match",
				"briefInitialDescription": "brief",
				"description":             "full",
				"isPublic":                true,
				"gameScheduledAt":         "2026-06-15T19:30:00Z",
				"storyStartAt":            "2026-06-15",
			},
			mockFn: func(ctx context.Context, input *matchUC.CreateMatchInput) (*matchEntity.Match, error) {
				return nil, domain.ErrValidation
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "internal_server_error",
			body: map[string]any{
				"campaignUuid":            uuid.New().String(),
				"title":                   "Test Match",
				"briefInitialDescription": "brief",
				"description":             "full",
				"isPublic":                true,
				"gameScheduledAt":         "2026-06-15T19:30:00Z",
				"storyStartAt":            "2026-06-15",
			},
			mockFn: func(ctx context.Context, input *matchUC.CreateMatchInput) (*matchEntity.Match, error) {
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
				if matchData["masterUuid"] != userUUID.String() {
					t.Errorf("got masterUuid %v, want %v", matchData["masterUuid"], userUUID.String())
				}
			}
		})
	}
}

func TestCreateMatchStoryStartTime(t *testing.T) {
	userUUID := uuid.New()
	now := time.Now()
	h30 := 8*time.Hour + 30*time.Minute
	h30s := h30 + 15*time.Second

	tests := []struct {
		name       string
		body       map[string]any
		want       *time.Duration
		wantStatus int
		wantResp   string
	}{
		{"hh:mm", map[string]any{"storyStartTime": "08:30"}, &h30, http.StatusCreated, "08:30:00"},
		{"hh:mm:ss", map[string]any{"storyStartTime": "08:30:15"}, &h30s, http.StatusCreated, "08:30:15"},
		{"empty string is nil", map[string]any{"storyStartTime": ""}, nil, http.StatusCreated, ""},
		{"absent is nil", map[string]any{}, nil, http.StatusCreated, ""},
		{"hour out of range", map[string]any{"storyStartTime": "25:00"}, nil, http.StatusUnprocessableEntity, ""},
		{"wrong format", map[string]any{"storyStartTime": "8h"}, nil, http.StatusUnprocessableEntity, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)
			var got *time.Duration
			called := false
			mock := &mockCreateMatch{fn: func(_ context.Context, in *matchUC.CreateMatchInput) (*matchEntity.Match, error) {
				called = true
				got = in.StoryStartTime
				return &matchEntity.Match{
					UUID: uuid.New(), MasterUUID: in.MasterUUID, CampaignUUID: in.CampaignUUID,
					Title: in.Title, GameScheduledAt: in.GameScheduledAt, StoryStartAt: in.StoryStartAt,
					StoryStartTime: in.StoryStartTime, CreatedAt: now, UpdatedAt: now,
				}, nil
			}}
			huma.Register(api, huma.Operation{
				Method: http.MethodPost, Path: "/matches", DefaultStatus: http.StatusCreated,
			}, match.CreateMatchHandler(mock))

			body := map[string]any{
				"campaignUuid":            uuid.New().String(),
				"title":                   "Test Match",
				"briefInitialDescription": "brief",
				"description":             "full",
				"isPublic":                true,
				"gameScheduledAt":         "2026-06-15T19:30:00Z",
				"storyStartAt":            "2026-06-15",
			}
			for k, v := range tt.body {
				body[k] = v
			}
			ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
			resp := api.PostCtx(ctx, "/matches", body)
			if resp.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}
			if tt.wantStatus != http.StatusCreated {
				if called {
					t.Error("UC should not be called on invalid time")
				}
				if !strings.Contains(resp.Body.String(), "storyStartTime must be HH:MM or HH:MM:SS") {
					t.Errorf("detail missing: %s", resp.Body.String())
				}
				return
			}
			if (got == nil) != (tt.want == nil) || (got != nil && *got != *tt.want) {
				t.Errorf("forwarded = %v, want %v", got, tt.want)
			}
			var result map[string]map[string]any
			_ = json.Unmarshal(resp.Body.Bytes(), &result)
			v, _ := result["match"]["storyStartTime"].(string)
			if v != tt.wantResp {
				t.Errorf("response storyStartTime = %q, want %q", v, tt.wantResp)
			}
		})
	}
}
