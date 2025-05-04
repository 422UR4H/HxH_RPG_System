package scenario

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/scenario"
)

type IRepository interface {
	CreateScenario(ctx context.Context, scenario *scenario.Scenario) error
	ExistsScenarioWithName(ctx context.Context, name string) (bool, error)
}
