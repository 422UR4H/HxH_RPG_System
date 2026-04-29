package scenario_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/scenario"
	domainAuth "github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	domainScenario "github.com/422UR4H/HxH_RPG_System/internal/domain/scenario"
	scenarioEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/scenario"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestGetScenarioHandler(t *testing.T) {
	userUUID := uuid.New()
	scenarioUUID := uuid.New()
	now := time.Now()

	tests := []struct {
		name       string
		mockFn     func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*scenarioEntity.Scenario, error)
		wantStatus int
	}{
		{
			name: "success",
			mockFn: func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*scenarioEntity.Scenario, error) {
				return &scenarioEntity.Scenario{
					UUID:             id,
					UserUUID:         uid,
					Name:             "My Scenario",
					BriefDescription: "Brief",
					Description:      "Full",
					CreatedAt:        now,
					UpdatedAt:        now,
				}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "not_found",
			mockFn: func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*scenarioEntity.Scenario, error) {
				return nil, domainScenario.ErrScenarioNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "forbidden_insufficient_permissions",
			mockFn: func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*scenarioEntity.Scenario, error) {
				return nil, domainAuth.ErrInsufficientPermissions
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "internal_server_error",
			mockFn: func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*scenarioEntity.Scenario, error) {
				return nil, errors.New("unexpected error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)

			mock := &mockGetScenario{fn: tt.mockFn}
			handler := scenario.GetScenarioHandler(mock)

			huma.Register(api, huma.Operation{
				Method: http.MethodGet,
				Path:   "/scenarios/{uuid}",
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
			resp := api.GetCtx(ctx, "/scenarios/"+scenarioUUID.String())

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				var result map[string]any
				if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				scenarioData, ok := result["scenario"].(map[string]any)
				if !ok {
					t.Fatal("response missing 'scenario' field")
				}
				if scenarioData["name"] != "My Scenario" {
					t.Errorf("got name %v, want 'My Scenario'", scenarioData["name"])
				}
			}
		})
	}
}
