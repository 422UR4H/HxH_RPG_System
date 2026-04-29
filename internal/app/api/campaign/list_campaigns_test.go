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

func TestListCampaignsHandler(t *testing.T) {
	userUUID := uuid.New()
	now := time.Now()

	tests := []struct {
		name       string
		mockFn     func(ctx context.Context, uid uuid.UUID) ([]*campaignEntity.Summary, error)
		wantStatus int
		wantCount  int
	}{
		{
			name: "success_with_campaigns",
			mockFn: func(ctx context.Context, uid uuid.UUID) ([]*campaignEntity.Summary, error) {
				return []*campaignEntity.Summary{
					{
						UUID:                    uuid.New(),
						Name:                    "Campaign 1",
						BriefInitialDescription: "Brief 1",
						IsPublic:                true,
						CallLink:                "https://meet.example.com/1",
						StoryStartAt:            now,
						CreatedAt:               now,
						UpdatedAt:               now,
					},
					{
						UUID:                    uuid.New(),
						Name:                    "Campaign 2",
						BriefInitialDescription: "Brief 2",
						IsPublic:                false,
						CallLink:                "https://meet.example.com/2",
						StoryStartAt:            now,
						CreatedAt:               now,
						UpdatedAt:               now,
					},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantCount:  2,
		},
		{
			name: "success_empty_list",
			mockFn: func(ctx context.Context, uid uuid.UUID) ([]*campaignEntity.Summary, error) {
				return []*campaignEntity.Summary{}, nil
			},
			wantStatus: http.StatusOK,
			wantCount:  0,
		},
		{
			name: "internal_server_error",
			mockFn: func(ctx context.Context, uid uuid.UUID) ([]*campaignEntity.Summary, error) {
				return nil, errors.New("db connection failed")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)

			mock := &mockListCampaigns{fn: tt.mockFn}
			handler := campaign.ListCampaignsHandler(mock)

			huma.Register(api, huma.Operation{
				Method: http.MethodGet,
				Path:   "/campaigns",
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
			resp := api.GetCtx(ctx, "/campaigns")

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
			}
		})
	}
}
