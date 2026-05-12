package testutil

import (
	"context"

	scenarioEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/scenario"
	"github.com/google/uuid"
)

type MockScenarioRepo struct {
	CreateScenarioFn          func(ctx context.Context, scenario *scenarioEntity.Scenario) error
	GetScenarioFn             func(ctx context.Context, uuid uuid.UUID) (*scenarioEntity.Scenario, error)
	ExistsScenarioFn          func(ctx context.Context, uuid uuid.UUID) (bool, error)
	ExistsScenarioWithNameFn  func(ctx context.Context, name string) (bool, error)
	ListScenariosByUserUUIDFn func(ctx context.Context, userUUID uuid.UUID) ([]*scenarioEntity.Summary, error)
}

func (m *MockScenarioRepo) CreateScenario(ctx context.Context, scenario *scenarioEntity.Scenario) error {
	if m.CreateScenarioFn != nil {
		return m.CreateScenarioFn(ctx, scenario)
	}
	return nil
}

func (m *MockScenarioRepo) GetScenario(ctx context.Context, id uuid.UUID) (*scenarioEntity.Scenario, error) {
	if m.GetScenarioFn != nil {
		return m.GetScenarioFn(ctx, id)
	}
	return nil, nil
}

func (m *MockScenarioRepo) ExistsScenario(ctx context.Context, id uuid.UUID) (bool, error) {
	if m.ExistsScenarioFn != nil {
		return m.ExistsScenarioFn(ctx, id)
	}
	return false, nil
}

func (m *MockScenarioRepo) ExistsScenarioWithName(ctx context.Context, name string) (bool, error) {
	if m.ExistsScenarioWithNameFn != nil {
		return m.ExistsScenarioWithNameFn(ctx, name)
	}
	return false, nil
}

func (m *MockScenarioRepo) ListScenariosByUserUUID(ctx context.Context, userUUID uuid.UUID) ([]*scenarioEntity.Summary, error) {
	if m.ListScenariosByUserUUIDFn != nil {
		return m.ListScenariosByUserUUIDFn(ctx, userUUID)
	}
	return nil, nil
}
