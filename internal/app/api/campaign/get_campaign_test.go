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
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestGetCampaignHandler(t *testing.T) {
	userUUID := uuid.New()
	campaignUUID := uuid.New()
	now := time.Now()

	tests := []struct {
		name            string
		mockFn          func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*campaignEntity.Campaign, error)
		enrollmentMock  *mockListPlayerEnrollments
		wantStatus      int
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
			enrollmentMock: &mockListPlayerEnrollments{},
			wantStatus:     http.StatusOK,
		},
		{
			name: "not_found",
			mockFn: func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*campaignEntity.Campaign, error) {
				return nil, domainCampaign.ErrCampaignNotFound
			},
			enrollmentMock: &mockListPlayerEnrollments{},
			wantStatus:     http.StatusNotFound,
		},
		{
			name: "forbidden_insufficient_permissions",
			mockFn: func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*campaignEntity.Campaign, error) {
				return nil, domainAuth.ErrInsufficientPermissions
			},
			enrollmentMock: &mockListPlayerEnrollments{},
			wantStatus:     http.StatusForbidden,
		},
		{
			name: "internal_server_error",
			mockFn: func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*campaignEntity.Campaign, error) {
				return nil, errors.New("unexpected error")
			},
			enrollmentMock: &mockListPlayerEnrollments{},
			wantStatus:     http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)

			mock := &mockGetCampaign{fn: tt.mockFn}
			handler := campaign.GetCampaignHandler(mock, tt.enrollmentMock)

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

func TestGetCampaignHandler_PlayerEnrollmentStatus(t *testing.T) {
	masterUUID := uuid.New()
	playerUUID := uuid.New()
	campaignUUID := uuid.New()
	matchUUID := uuid.New()
	now := time.Now()

	_, api := humatest.New(t)

	mockGet := &mockGetCampaign{
		fn: func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*campaignEntity.Campaign, error) {
			return &campaignEntity.Campaign{
				UUID:                    id,
				MasterUUID:              masterUUID,
				Name:                    "Player Campaign",
				BriefInitialDescription: "Brief",
				Description:             "Full",
				IsPublic:                true,
				CallLink:                "https://meet.example.com",
				StoryStartAt:            now,
				CreatedAt:               now,
				UpdatedAt:               now,
				Matches: []matchEntity.Summary{
					{
						UUID:                    matchUUID,
						CampaignUUID:            id,
						Title:                   "Match 1",
						BriefInitialDescription: "Intro",
						IsPublic:                true,
						GameScheduledAt:         now,
						StoryStartAt:            now,
						CreatedAt:               now,
						UpdatedAt:               now,
					},
				},
			}, nil
		},
	}

	enrollmentMock := &mockListPlayerEnrollments{
		statuses: map[uuid.UUID]string{matchUUID: "pending"},
	}

	handler := campaign.GetCampaignHandler(mockGet, enrollmentMock)

	huma.Register(api, huma.Operation{
		Method: http.MethodGet,
		Path:   "/campaigns/{uuid}",
	}, handler)

	ctx := context.WithValue(context.Background(), auth.UserIDKey, playerUUID)
	resp := api.GetCtx(ctx, "/campaigns/"+campaignUUID.String())

	if resp.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d. Body: %s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var result map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	campaignData, ok := result["campaign"].(map[string]any)
	if !ok {
		t.Fatal("response missing 'campaign' field")
	}
	matches, ok := campaignData["matches"].([]any)
	if !ok {
		t.Fatal("response missing 'matches' field")
	}
	if len(matches) == 0 {
		t.Fatal("expected at least one match in response")
	}
	firstMatch, ok := matches[0].(map[string]any)
	if !ok {
		t.Fatal("expected first match to be an object")
	}
	status, ok := firstMatch["my_enrollment_status"]
	if !ok {
		t.Fatal("expected 'my_enrollment_status' field in match")
	}
	if status != "pending" {
		t.Errorf("got my_enrollment_status %v, want 'pending'", status)
	}
}
