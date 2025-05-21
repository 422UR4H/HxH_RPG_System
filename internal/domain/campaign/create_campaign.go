package campaign

import (
	"context"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/scenario"
	"github.com/google/uuid"
)

type ICreateCampaign interface {
	CreateCampaign(
		ctx context.Context, input *CreateCampaignInput,
	) (*campaign.Campaign, error)
}

type CreateCampaignInput struct {
	UserUUID         uuid.UUID
	ScenarioUUID     *uuid.UUID
	Name             string
	BriefDescription string
	Description      string
	StoryStartAt     time.Time
	StoryCurrentAt   *time.Time
}

type CreateCampaignUC struct {
	campaignRepo IRepository
	scenarioRepo scenario.IRepository
}

func NewCreateCampaignUC(
	campaignRepo IRepository,
	scenarioRepo scenario.IRepository,
) *CreateCampaignUC {
	return &CreateCampaignUC{
		campaignRepo: campaignRepo,
		scenarioRepo: scenarioRepo,
	}
}

func (uc *CreateCampaignUC) CreateCampaign(
	ctx context.Context, input *CreateCampaignInput,
) (*campaign.Campaign, error) {
	if len(input.Name) < 5 {
		return nil, ErrMinNameLength
	}

	if len(input.Name) > 32 {
		return nil, ErrMaxNameLength
	}

	if input.StoryStartAt.IsZero() {
		return nil, ErrInvalidStartDate
	}

	if len(input.BriefDescription) > 64 {
		return nil, ErrMaxBriefDescLength
	}

	// Currently campaigns do not belong to scenarios, but this will change soon
	if input.ScenarioUUID != nil {
		exists, err := uc.scenarioRepo.ExistsScenario(
			ctx, *input.ScenarioUUID,
		)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, scenario.ErrScenarioNotFound
		}
	}

	newCampaign, err := campaign.NewCampaign(
		input.UserUUID,
		input.ScenarioUUID,
		input.Name,
		input.BriefDescription,
		input.Description,
		input.StoryStartAt,
		input.StoryCurrentAt,
	)
	if err != nil {
		return nil, err
	}

	err = uc.campaignRepo.CreateCampaign(ctx, newCampaign)
	if err != nil {
		return nil, err
	}
	return newCampaign, nil
}
