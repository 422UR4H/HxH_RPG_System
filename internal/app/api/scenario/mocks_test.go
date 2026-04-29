package scenario_test

import (
	"context"

	domainScenario "github.com/422UR4H/HxH_RPG_System/internal/domain/scenario"
	scenarioEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/scenario"
	"github.com/google/uuid"
)

type mockCreateScenario struct {
	fn func(ctx context.Context, input *domainScenario.CreateScenarioInput) (*scenarioEntity.Scenario, error)
}

func (m *mockCreateScenario) CreateScenario(ctx context.Context, input *domainScenario.CreateScenarioInput) (*scenarioEntity.Scenario, error) {
	return m.fn(ctx, input)
}

type mockGetScenario struct {
	fn func(ctx context.Context, uuid uuid.UUID, userUUID uuid.UUID) (*scenarioEntity.Scenario, error)
}

func (m *mockGetScenario) GetScenario(ctx context.Context, uuid uuid.UUID, userUUID uuid.UUID) (*scenarioEntity.Scenario, error) {
	return m.fn(ctx, uuid, userUUID)
}

type mockListScenarios struct {
	fn func(ctx context.Context, userUUID uuid.UUID) ([]*scenarioEntity.Summary, error)
}

func (m *mockListScenarios) ListScenarios(ctx context.Context, userUUID uuid.UUID) ([]*scenarioEntity.Summary, error) {
	return m.fn(ctx, userUUID)
}
