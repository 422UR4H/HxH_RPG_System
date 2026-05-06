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
	domainAuth "github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestGetMatchHandler(t *testing.T) {
	userUUID := uuid.New()
	matchUUID := uuid.New()
	now := time.Now()

	tests := []struct {
		name       string
		mockFn     func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*matchEntity.Match, error)
		wantStatus int
	}{
		{
			name: "success",
			mockFn: func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*matchEntity.Match, error) {
				return &matchEntity.Match{
					UUID:                    id,
					CampaignUUID:            uuid.New(),
					MasterUUID:              uid,
					Title:                   "My Match",
					BriefInitialDescription: "Brief",
					Description:             "Full",
					IsPublic:                true,
					GameScheduledAt:         now,
					StoryStartAt:            now,
					CreatedAt:               now,
					UpdatedAt:               now,
				}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "not_found",
			mockFn: func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*matchEntity.Match, error) {
				return nil, domainMatch.ErrMatchNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "forbidden_insufficient_permissions",
			mockFn: func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*matchEntity.Match, error) {
				return nil, domainAuth.ErrInsufficientPermissions
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "internal_server_error",
			mockFn: func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*matchEntity.Match, error) {
				return nil, errors.New("unexpected error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)

			mock := &mockGetMatch{fn: tt.mockFn}
			handler := match.GetMatchHandler(mock)

			huma.Register(api, huma.Operation{
				Method: http.MethodGet,
				Path:   "/matches/{uuid}",
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
			resp := api.GetCtx(ctx, "/matches/"+matchUUID.String())

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
				if matchData["title"] != "My Match" {
					t.Errorf("got title %v, want 'My Match'", matchData["title"])
				}
				if matchData["master_uuid"] != userUUID.String() {
					t.Errorf("got master_uuid %v, want %v", matchData["master_uuid"], userUUID.String())
				}
			}
		})
	}
}
