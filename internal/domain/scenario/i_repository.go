package scenario

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/scenario"
	"github.com/google/uuid"
)

type IRepository interface {
	CreateScenario(ctx context.Context, scenario *scenario.Scenario) error
	ExistsScenario(ctx context.Context, uuid uuid.UUID) (bool, error)
	ExistsScenarioWithName(ctx context.Context, name string) (bool, error)
}
