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
	scenarioEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/scenario"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestListScenariosHandler(t *testing.T) {
	userUUID := uuid.New()
	now := time.Now()

	tests := []struct {
		name       string
		mockFn     func(ctx context.Context, uid uuid.UUID) ([]*scenarioEntity.Summary, error)
		wantStatus int
		wantCount  int
	}{
		{
			name: "success_with_scenarios",
			mockFn: func(ctx context.Context, uid uuid.UUID) ([]*scenarioEntity.Summary, error) {
				return []*scenarioEntity.Summary{
					{UUID: uuid.New(), Name: "Scenario 1", BriefDescription: "Brief 1", CreatedAt: now, UpdatedAt: now},
					{UUID: uuid.New(), Name: "Scenario 2", BriefDescription: "Brief 2", CreatedAt: now, UpdatedAt: now},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantCount:  2,
		},
		{
			name: "success_empty_list",
			mockFn: func(ctx context.Context, uid uuid.UUID) ([]*scenarioEntity.Summary, error) {
				return []*scenarioEntity.Summary{}, nil
			},
			wantStatus: http.StatusOK,
			wantCount:  0,
		},
		{
			name: "internal_server_error",
			mockFn: func(ctx context.Context, uid uuid.UUID) ([]*scenarioEntity.Summary, error) {
				return nil, errors.New("db connection failed")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)

			mock := &mockListScenarios{fn: tt.mockFn}
			handler := scenario.ListScenariosHandler(mock)

			huma.Register(api, huma.Operation{
				Method: http.MethodGet,
				Path:   "/scenarios",
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
			resp := api.GetCtx(ctx, "/scenarios")

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				var result map[string]any
				if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				scenarios, ok := result["scenarios"].([]any)
				if !ok {
					t.Fatal("response missing 'scenarios' field")
				}
				if len(scenarios) != tt.wantCount {
					t.Errorf("got %d scenarios, want %d", len(scenarios), tt.wantCount)
				}
			}
		})
	}
}
