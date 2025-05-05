package scenario

import (
	"context"
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/scenario"
	scenarioPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/scenario"
	"github.com/google/uuid"
)

type IGetScenario interface {
	GetScenario(uuid uuid.UUID) (*scenario.Scenario, error)
}

type GetScenarioInput struct {
	UUID uuid.UUID
}

type GetScenarioUC struct {
	repo IRepository
}

func NewGetScenarioUC(repo IRepository) *GetScenarioUC {
	return &GetScenarioUC{
		repo: repo,
	}
}

func (uc *GetScenarioUC) GetScenario(uuid uuid.UUID) (*scenario.Scenario, error) {
	scenario, err := uc.repo.GetScenario(context.Background(), uuid)
	if err != nil {
		if errors.Is(err, scenarioPg.ErrScenarioNotFound) {
			return nil, ErrScenarioNotFound
		}
		return nil, err
	}
	return scenario, nil
}
