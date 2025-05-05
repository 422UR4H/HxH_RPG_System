package scenario

import (
	"context"

	scenarioEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/scenario"
	"github.com/google/uuid"
)

type IListScenarios interface {
	ListScenarios(userUUID uuid.UUID) ([]*scenarioEntity.Summary, error)
}

type ListScenariosUC struct {
	repo IRepository
}

func NewListScenariosUC(repo IRepository) *ListScenariosUC {
	return &ListScenariosUC{
		repo: repo,
	}
}

func (uc *ListScenariosUC) ListScenarios(userUUID uuid.UUID) ([]*scenarioEntity.Summary, error) {
	scenarios, err := uc.repo.ListScenariosByUserUUID(context.Background(), userUUID)
	if err != nil {
		return nil, err
	}
	return scenarios, nil
}
