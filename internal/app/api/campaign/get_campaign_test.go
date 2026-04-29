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
	domainAuth "github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	domainCampaign "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	campaignEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestGetCampaignHandler(t *testing.T) {
	userUUID := uuid.New()
	campaignUUID := uuid.New()
	now := time.Now()

	tests := []struct {
		name       string
		mockFn     func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*campaignEntity.Campaign, error)
		wantStatus int
	}{
		{
			name: "success",
			mockFn: func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*campaignEntity.Campaign, error) {
				return &campaignEntity.Campaign{
					UUID:                    id,
					MasterUUID:              uid,
					Name:                    "My Campaign",
					BriefInitialDescription: "Brief",
					Description:             "Full",
					IsPublic:                true,
					CallLink:                "https://meet.example.com",
					StoryStartAt:            now,
					CreatedAt:               now,
					UpdatedAt:               now,
				}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "not_found",
			mockFn: func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*campaignEntity.Campaign, error) {
				return nil, domainCampaign.ErrCampaignNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "forbidden_insufficient_permissions",
			mockFn: func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*campaignEntity.Campaign, error) {
				return nil, domainAuth.ErrInsufficientPermissions
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "internal_server_error",
			mockFn: func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*campaignEntity.Campaign, error) {
				return nil, errors.New("unexpected error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)

			mock := &mockGetCampaign{fn: tt.mockFn}
			handler := campaign.GetCampaignHandler(mock)

			huma.Register(api, huma.Operation{
				Method: http.MethodGet,
				Path:   "/campaigns/{uuid}",
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
			resp := api.GetCtx(ctx, "/campaigns/"+campaignUUID.String())

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				var result map[string]any
				if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				campaignData, ok := result["campaign"].(map[string]any)
				if !ok {
					t.Fatal("response missing 'campaign' field")
				}
				if campaignData["name"] != "My Campaign" {
					t.Errorf("got name %v, want 'My Campaign'", campaignData["name"])
				}
			}
		})
	}
}
