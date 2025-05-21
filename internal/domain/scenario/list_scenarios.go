package scenario

import (
	"context"

	scenarioEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/scenario"
	"github.com/google/uuid"
)

type IListScenarios interface {
	ListScenarios(
		ctx context.Context, userUUID uuid.UUID,
	) ([]*scenarioEntity.Summary, error)
}

type ListScenariosUC struct {
	repo IRepository
}

func NewListScenariosUC(repo IRepository) *ListScenariosUC {
	return &ListScenariosUC{
		repo: repo,
	}
}

func (uc *ListScenariosUC) ListScenarios(
	ctx context.Context, userUUID uuid.UUID,
) ([]*scenarioEntity.Summary, error) {
	scenarios, err := uc.repo.ListScenariosByUserUUID(ctx, userUUID)
	if err != nil {
		return nil, err
	}
	return scenarios, nil
}
