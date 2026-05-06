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
	campaignEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestListPublicUpcomingCampaignsHandler(t *testing.T) {
	userUUID := uuid.New()
	now := time.Now()
	nextScheduled := now.Add(24 * time.Hour)

	tests := []struct {
		name              string
		mockFn            func(ctx context.Context, uid uuid.UUID) ([]*campaignEntity.PublicSummary, error)
		wantStatus        int
		wantCount         int
		wantNextScheduled *bool
	}{
		{
			name: "success_with_upcoming_match",
			mockFn: func(ctx context.Context, uid uuid.UUID) ([]*campaignEntity.PublicSummary, error) {
				return []*campaignEntity.PublicSummary{
					{
						Summary: campaignEntity.Summary{
							UUID:                    uuid.New(),
							Name:                    "Public Campaign",
							BriefInitialDescription: "Brief",
							IsPublic:                true,
							CallLink:                "https://meet.example.com/1",
							StoryStartAt:            now,
							CreatedAt:               now,
							UpdatedAt:               now,
						},
						NextGameScheduledAt: &nextScheduled,
					},
				}, nil
			},
			wantStatus:        http.StatusOK,
			wantCount:         1,
			wantNextScheduled: boolPtr(true),
		},
		{
			name: "success_without_future_match",
			mockFn: func(ctx context.Context, uid uuid.UUID) ([]*campaignEntity.PublicSummary, error) {
				return []*campaignEntity.PublicSummary{
					{
						Summary: campaignEntity.Summary{
							UUID:                    uuid.New(),
							Name:                    "No Schedule Campaign",
							BriefInitialDescription: "Brief",
							IsPublic:                true,
							CallLink:                "https://meet.example.com/2",
							StoryStartAt:            now,
							CreatedAt:               now,
							UpdatedAt:               now,
						},
						NextGameScheduledAt: nil,
					},
				}, nil
			},
			wantStatus:        http.StatusOK,
			wantCount:         1,
			wantNextScheduled: boolPtr(false),
		},
		{
			name: "success_empty_list",
			mockFn: func(ctx context.Context, uid uuid.UUID) ([]*campaignEntity.PublicSummary, error) {
				return []*campaignEntity.PublicSummary{}, nil
			},
			wantStatus: http.StatusOK,
			wantCount:  0,
		},
		{
			name: "internal_server_error",
			mockFn: func(ctx context.Context, uid uuid.UUID) ([]*campaignEntity.PublicSummary, error) {
				return nil, errors.New("db connection failed")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)

			mock := &mockListPublicUpcomingCampaigns{fn: tt.mockFn}
			handler := campaign.ListPublicUpcomingCampaignsHandler(mock)

			huma.Register(api, huma.Operation{
				Method: http.MethodGet,
				Path:   "/public/campaigns",
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
			resp := api.GetCtx(ctx, "/public/campaigns")

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				var result map[string]any
				if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				campaigns, ok := result["campaigns"].([]any)
				if !ok {
					t.Fatal("response missing 'campaigns' field")
				}
				if len(campaigns) != tt.wantCount {
					t.Errorf("got %d campaigns, want %d", len(campaigns), tt.wantCount)
				}
				if tt.wantNextScheduled != nil && len(campaigns) > 0 {
					first := campaigns[0].(map[string]any)
					_, present := first["next_game_scheduled_at"]
					if *tt.wantNextScheduled && !present {
						t.Error("expected next_game_scheduled_at to be present, got absent")
					}
					if !*tt.wantNextScheduled && present {
						t.Errorf("expected next_game_scheduled_at to be absent, got %v", first["next_game_scheduled_at"])
					}
				}
			}
		})
	}
}

func boolPtr(b bool) *bool { return &b }
