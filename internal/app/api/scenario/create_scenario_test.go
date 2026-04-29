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
	domainScenario "github.com/422UR4H/HxH_RPG_System/internal/domain/scenario"
	scenarioEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/scenario"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestCreateScenarioHandler(t *testing.T) {
	userUUID := uuid.New()
	now := time.Now()

	tests := []struct {
		name       string
		body       map[string]any
		mockFn     func(ctx context.Context, input *domainScenario.CreateScenarioInput) (*scenarioEntity.Scenario, error)
		wantStatus int
		useAuth    bool
	}{
		{
			name: "success",
			body: map[string]any{"name": "Test Scenario", "brief_description": "A test", "description": "Full desc"},
			mockFn: func(ctx context.Context, input *domainScenario.CreateScenarioInput) (*scenarioEntity.Scenario, error) {
				return &scenarioEntity.Scenario{
					UUID:             uuid.New(),
					UserUUID:         input.UserUUID,
					Name:             input.Name,
					BriefDescription: input.BriefDescription,
					Description:      input.Description,
					CreatedAt:        now,
					UpdatedAt:        now,
				}, nil
			},
			wantStatus: http.StatusCreated,
			useAuth:    true,
		},
		{
			name: "conflict_name_already_exists",
			body: map[string]any{"name": "Existing Scenario", "brief_description": "desc", "description": "full"},
			mockFn: func(ctx context.Context, input *domainScenario.CreateScenarioInput) (*scenarioEntity.Scenario, error) {
				return nil, domainScenario.ErrScenarioNameAlreadyExists
			},
			wantStatus: http.StatusConflict,
			useAuth:    true,
		},
		{
			name: "unprocessable_entity_validation_error",
			body: map[string]any{"name": "ab", "brief_description": "desc", "description": "full"},
			mockFn: func(ctx context.Context, input *domainScenario.CreateScenarioInput) (*scenarioEntity.Scenario, error) {
				return nil, domainScenario.ErrMinNameLength
			},
			wantStatus: http.StatusUnprocessableEntity,
			useAuth:    true,
		},
		{
			name: "internal_server_error",
			body: map[string]any{"name": "Valid Name", "brief_description": "desc", "description": "full"},
			mockFn: func(ctx context.Context, input *domainScenario.CreateScenarioInput) (*scenarioEntity.Scenario, error) {
				return nil, errors.New("unexpected db error")
			},
			wantStatus: http.StatusInternalServerError,
			useAuth:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)

			mock := &mockCreateScenario{fn: tt.mockFn}
			handler := scenario.CreateScenarioHandler(mock)

			huma.Register(api, huma.Operation{
				Method:        http.MethodPost,
				Path:          "/scenarios",
				DefaultStatus: http.StatusCreated,
			}, handler)

			ctx := context.Background()
			if tt.useAuth {
				ctx = context.WithValue(ctx, auth.UserIDKey, userUUID)
			}

			resp := api.PostCtx(ctx, "/scenarios", tt.body)

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}

			if tt.wantStatus == http.StatusCreated {
				var result map[string]any
				if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				scenarioData, ok := result["scenario"].(map[string]any)
				if !ok {
					t.Fatal("response missing 'scenario' field")
				}
				if scenarioData["name"] != "Test Scenario" {
					t.Errorf("got name %v, want 'Test Scenario'", scenarioData["name"])
				}
			}
		})
	}
}
