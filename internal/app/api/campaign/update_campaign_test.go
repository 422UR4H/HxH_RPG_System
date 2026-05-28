package campaign_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/campaign"
	"github.com/422UR4H/HxH_RPG_System/internal/domain"
	campaignUC "github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	campaignEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestUpdateCampaignHandler(t *testing.T) {
	userUUID := uuid.New()
	campaignUUID := uuid.New()
	now := time.Now()

	baseResp := func(name string) *campaignEntity.Campaign {
		return &campaignEntity.Campaign{
			UUID:                    campaignUUID,
			MasterUUID:              userUUID,
			Name:                    name,
			BriefInitialDescription: "brief",
			Description:             "full",
			IsPublic:                true,
			CallLink:                "https://discord.gg/abc",
			StoryStartAt:            now,
			UpdatedAt:               now,
		}
	}

	tests := []struct {
		name       string
		body       map[string]any
		mockFn     func(ctx context.Context, input *campaignUC.UpdateCampaignInput) (*campaignEntity.Campaign, error)
		wantStatus int
	}{
		{
			name: "success_full_patch",
			body: map[string]any{
				"name":                      "New Name",
				"brief_initial_description": "new brief",
				"description":               "new desc",
				"is_public":                 false,
				"call_link":                 "https://meet.new",
				"story_start_at":            "2026-07-20",
				"story_current_at":          "2026-07-20T10:00:00Z",
			},
			mockFn: func(_ context.Context, input *campaignUC.UpdateCampaignInput) (*campaignEntity.Campaign, error) {
				if input.Name == nil || *input.Name != "New Name" {
					t.Errorf("name not forwarded: %+v", input.Name)
				}
				return baseResp("New Name"), nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "success_partial_always_editable",
			body: map[string]any{"call_link": "https://new-link.com"},
			mockFn: func(_ context.Context, input *campaignUC.UpdateCampaignInput) (*campaignEntity.Campaign, error) {
				if input.Name != nil {
					t.Errorf("name should be nil, got %v", *input.Name)
				}
				return baseResp("Original"), nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "success_empty_body_noop",
			body: map[string]any{},
			mockFn: func(_ context.Context, _ *campaignUC.UpdateCampaignInput) (*campaignEntity.Campaign, error) {
				return baseResp("Original"), nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "invalid_story_start_at",
			body: map[string]any{"story_start_at": "not-a-date"},
			mockFn: func(_ context.Context, _ *campaignUC.UpdateCampaignInput) (*campaignEntity.Campaign, error) {
				t.Fatal("UC should not be called when date parsing fails")
				return nil, nil
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "invalid_story_current_at",
			body: map[string]any{"story_current_at": "not-a-date"},
			mockFn: func(_ context.Context, _ *campaignUC.UpdateCampaignInput) (*campaignEntity.Campaign, error) {
				t.Fatal("UC should not be called when date parsing fails")
				return nil, nil
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "not_found",
			body: map[string]any{"name": "x"},
			mockFn: func(_ context.Context, _ *campaignUC.UpdateCampaignInput) (*campaignEntity.Campaign, error) {
				return nil, campaignUC.ErrCampaignNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "not_owner",
			body: map[string]any{"name": "x"},
			mockFn: func(_ context.Context, _ *campaignUC.UpdateCampaignInput) (*campaignEntity.Campaign, error) {
				return nil, campaignUC.ErrNotCampaignOwner
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "already_ended",
			body: map[string]any{"name": "x"},
			mockFn: func(_ context.Context, _ *campaignUC.UpdateCampaignInput) (*campaignEntity.Campaign, error) {
				return nil, campaignUC.ErrCampaignAlreadyEnded
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "locked_after_match_start",
			body: map[string]any{"name": "x"},
			mockFn: func(_ context.Context, _ *campaignUC.UpdateCampaignInput) (*campaignEntity.Campaign, error) {
				return nil, campaignUC.ErrLockedAfterMatchStart
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "cannot_regress_story_current_at",
			body: map[string]any{"story_current_at": "2024-01-01T00:00:00Z"},
			mockFn: func(_ context.Context, _ *campaignUC.UpdateCampaignInput) (*campaignEntity.Campaign, error) {
				return nil, campaignUC.ErrCannotRegressStoryCurrentAt
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "validation_error",
			body: map[string]any{"name": "x"},
			mockFn: func(_ context.Context, _ *campaignUC.UpdateCampaignInput) (*campaignEntity.Campaign, error) {
				return nil, domain.ErrValidation
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "internal_error",
			body: map[string]any{"name": "x"},
			mockFn: func(_ context.Context, _ *campaignUC.UpdateCampaignInput) (*campaignEntity.Campaign, error) {
				return nil, errors.New("db down")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)
			mock := &mockUpdateCampaign{fn: tt.mockFn}
			handler := campaign.UpdateCampaignHandler(mock)

			huma.Register(api, huma.Operation{
				Method: http.MethodPatch,
				Path:   "/campaigns/{uuid}",
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
			resp := api.PatchCtx(ctx, "/campaigns/"+campaignUUID.String(), tt.body)

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}
			if tt.wantStatus == http.StatusOK {
				var result map[string]any
				if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
					t.Fatalf("unmarshal failed: %v", err)
				}
				c, ok := result["campaign"].(map[string]any)
				if !ok {
					t.Fatal("response missing 'campaign' field")
				}
				if c["master_uuid"] != userUUID.String() {
					t.Errorf("master_uuid = %v, want %v", c["master_uuid"], userUUID.String())
				}
			}
		})
	}
}
