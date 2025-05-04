package scenario

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/scenario"
	"github.com/google/uuid"
)

type ICreateScenario interface {
	CreateScenario(input *CreateScenarioInput) (*scenario.Scenario, error)
}

type CreateScenarioInput struct {
	UserUUID         uuid.UUID
	Name             string
	BriefDescription string
	Description      string
}

type CreateScenarioUC struct {
	repo IRepository
}

func NewCreateScenarioUC(repo IRepository) *CreateScenarioUC {
	return &CreateScenarioUC{
		repo: repo,
	}
}

func (uc *CreateScenarioUC) CreateScenario(
	input *CreateScenarioInput,
) (*scenario.Scenario, error) {

	exists, err := uc.repo.ExistsScenarioWithName(context.Background(), input.Name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrScenarioNameAlreadyExists
	}

	newScenario, err := scenario.NewScenario(
		input.UserUUID,
		input.Name,
		input.BriefDescription,
		input.Description,
	)
	if err != nil {
		return nil, err
	}

	err = uc.repo.CreateScenario(context.Background(), newScenario)
	if err != nil {
		return nil, err
	}
	return newScenario, nil
}
