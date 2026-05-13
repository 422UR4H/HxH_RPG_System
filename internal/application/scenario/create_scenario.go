package scenario

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/scenario"
	"github.com/google/uuid"
)

type ICreateScenario interface {
	CreateScenario(
		ctx context.Context, input *CreateScenarioInput,
	) (*scenario.Scenario, error)
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
	ctx context.Context, input *CreateScenarioInput,
) (*scenario.Scenario, error) {
	if len(input.Name) < 5 {
		return nil, ErrMinNameLength
	}

	if len(input.Name) > 32 {
		return nil, ErrMaxNameLength
	}

	if len(input.BriefDescription) > 64 {
		return nil, ErrMaxBriefDescLength
	}

	exists, err := uc.repo.ExistsScenarioWithName(ctx, input.Name)
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

	err = uc.repo.CreateScenario(ctx, newScenario)
	if err != nil {
		return nil, err
	}
	return newScenario, nil
}
