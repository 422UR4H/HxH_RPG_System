package scenario

import (
	"context"
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/scenario"
	scenarioPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/scenario"
	"github.com/google/uuid"
)

type IGetScenario interface {
	GetScenario(
		ctx context.Context, uuid uuid.UUID, userUUID uuid.UUID,
	) (*scenario.Scenario, error)
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

func (uc *GetScenarioUC) GetScenario(
	ctx context.Context, uuid uuid.UUID, userUUID uuid.UUID,
) (*scenario.Scenario, error) {
	scenario, err := uc.repo.GetScenario(ctx, uuid)
	if err != nil {
		if errors.Is(err, scenarioPg.ErrScenarioNotFound) {
			return nil, ErrScenarioNotFound
		}
		return nil, err
	}

	if scenario.UserUUID != userUUID {
		return nil, auth.ErrInsufficientPermissions
	}
	return scenario, nil
}
