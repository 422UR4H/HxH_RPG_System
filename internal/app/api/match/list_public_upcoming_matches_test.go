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
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestListPublicUpcomingMatchesHandler(t *testing.T) {
	userUUID := uuid.New()
	now := time.Now()

	tests := []struct {
		name       string
		mockFn     func(ctx context.Context, uid uuid.UUID) ([]*matchEntity.Summary, error)
		wantStatus int
		wantCount  int
	}{
		{
			name: "success_with_matches",
			mockFn: func(ctx context.Context, uid uuid.UUID) ([]*matchEntity.Summary, error) {
				return []*matchEntity.Summary{
					{
						UUID:                    uuid.New(),
						CampaignUUID:            uuid.New(),
						Title:                   "Public Match",
						BriefInitialDescription: "Upcoming",
						IsPublic:                true,
						GameStartAt:             now.Add(24 * time.Hour),
						StoryStartAt:            now,
						CreatedAt:               now,
						UpdatedAt:               now,
					},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantCount:  1,
		},
		{
			name: "internal_server_error",
			mockFn: func(ctx context.Context, uid uuid.UUID) ([]*matchEntity.Summary, error) {
				return nil, errors.New("db connection failed")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)

			mock := &mockListPublicUpcomingMatches{fn: tt.mockFn}
			handler := match.ListPublicUpcomingMatchesHandler(mock)

			huma.Register(api, huma.Operation{
				Method: http.MethodGet,
				Path:   "/public/matches",
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
			resp := api.GetCtx(ctx, "/public/matches")

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				var result map[string]any
				if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				matches, ok := result["matches"].([]any)
				if !ok {
					t.Fatal("response missing 'matches' field")
				}
				if len(matches) != tt.wantCount {
					t.Errorf("got %d matches, want %d", len(matches), tt.wantCount)
				}
			}
		})
	}
}
