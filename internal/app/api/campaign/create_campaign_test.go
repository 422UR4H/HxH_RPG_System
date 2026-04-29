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
	domainCampaign "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	campaignEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/scenario"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestCreateCampaignHandler(t *testing.T) {
	userUUID := uuid.New()
	now := time.Now()

	tests := []struct {
		name       string
		body       map[string]any
		mockFn     func(ctx context.Context, input *domainCampaign.CreateCampaignInput) (*campaignEntity.Campaign, error)
		wantStatus int
	}{
		{
			name: "success",
			body: map[string]any{
				"name":                      "Test Campaign",
				"brief_initial_description": "A brief description",
				"description":               "Full description of the campaign",
				"is_public":                 true,
				"call_link":                 "https://meet.example.com",
				"story_start_at":            "2026-01-15",
			},
			mockFn: func(ctx context.Context, input *domainCampaign.CreateCampaignInput) (*campaignEntity.Campaign, error) {
				return &campaignEntity.Campaign{
					UUID:                    uuid.New(),
					MasterUUID:              input.MasterUUID,
					Name:                    input.Name,
					BriefInitialDescription: input.BriefInitialDescription,
					Description:             input.Description,
					IsPublic:                input.IsPublic,
					CallLink:                input.CallLink,
					StoryStartAt:            input.StoryStartAt,
					StoryCurrentAt:          input.StoryCurrentAt,
					CreatedAt:               now,
					UpdatedAt:               now,
				}, nil
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "forbidden_max_campaigns_limit",
			body: map[string]any{
				"name":                      "Another Campaign",
				"brief_initial_description": "desc",
				"description":               "full",
				"is_public":                 false,
				"call_link":                 "https://meet.example.com",
				"story_start_at":            "2026-01-15",
			},
			mockFn: func(ctx context.Context, input *domainCampaign.CreateCampaignInput) (*campaignEntity.Campaign, error) {
				return nil, domainCampaign.ErrMaxCampaignsLimit
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "not_found_scenario",
			body: map[string]any{
				"name":                      "Campaign X",
				"brief_initial_description": "desc",
				"description":               "full",
				"is_public":                 false,
				"call_link":                 "https://meet.example.com",
				"story_start_at":            "2026-01-15",
			},
			mockFn: func(ctx context.Context, input *domainCampaign.CreateCampaignInput) (*campaignEntity.Campaign, error) {
				return nil, scenario.ErrScenarioNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "unprocessable_entity_validation_error",
			body: map[string]any{
				"name":                      "Bad",
				"brief_initial_description": "desc",
				"description":               "full",
				"is_public":                 false,
				"call_link":                 "https://meet.example.com",
				"story_start_at":            "2026-01-15",
			},
			mockFn: func(ctx context.Context, input *domainCampaign.CreateCampaignInput) (*campaignEntity.Campaign, error) {
				return nil, domain.NewValidationError(errors.New("name too short"))
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "internal_server_error",
			body: map[string]any{
				"name":                      "Valid Campaign",
				"brief_initial_description": "desc",
				"description":               "full",
				"is_public":                 false,
				"call_link":                 "https://meet.example.com",
				"story_start_at":            "2026-01-15",
			},
			mockFn: func(ctx context.Context, input *domainCampaign.CreateCampaignInput) (*campaignEntity.Campaign, error) {
				return nil, errors.New("unexpected db error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)

			mock := &mockCreateCampaign{fn: tt.mockFn}
			handler := campaign.CreateCampaignHandler(mock)

			huma.Register(api, huma.Operation{
				Method:        http.MethodPost,
				Path:          "/campaigns",
				DefaultStatus: http.StatusCreated,
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
			resp := api.PostCtx(ctx, "/campaigns", tt.body)

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}

			if tt.wantStatus == http.StatusCreated {
				var result map[string]any
				if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if result["name"] != "Test Campaign" {
					t.Errorf("got name %v, want 'Test Campaign'", result["name"])
				}
			}
		})
	}
}
