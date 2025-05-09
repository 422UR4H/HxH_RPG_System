package match

import (
	"context"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	pgCampaign "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	"github.com/google/uuid"
)

type ICreateMatch interface {
	CreateMatch(input *CreateMatchInput) (*match.Match, error)
}

type CreateMatchInput struct {
	MasterUUID       uuid.UUID
	CampaignUUID     uuid.UUID
	Title            string
	BriefDescription string
	Description      string
	StoryStartAt     time.Time
}

type CreateMatchUC struct {
	matchRepo    IRepository
	campaignRepo campaign.IRepository
}

func NewCreateMatchUC(
	matchRepo IRepository,
	campaignRepo campaign.IRepository,
) *CreateMatchUC {
	return &CreateMatchUC{
		matchRepo:    matchRepo,
		campaignRepo: campaignRepo,
	}
}

func (uc *CreateMatchUC) CreateMatch(input *CreateMatchInput) (*match.Match, error) {
	if len(input.Title) < 5 {
		return nil, ErrMinTitleLength
	}

	campaign, err := uc.campaignRepo.GetCampaign(context.Background(), input.CampaignUUID)
	if err == pgCampaign.ErrCampaignNotFound {
		return nil, ErrCampaignNotFound
	}
	if err != nil {
		return nil, err
	}

	if input.StoryStartAt.Before(campaign.StoryStartAt) {
		return nil, ErrMinOfStartDate
	}
	if campaign.StoryEndAt != nil && input.StoryStartAt.After(*campaign.StoryEndAt) {
		return nil, ErrMaxOfStartDate
	}

	newMatch, err := match.NewMatch(
		input.MasterUUID,
		input.CampaignUUID,
		input.Title,
		input.BriefDescription,
		input.Description,
		input.StoryStartAt,
	)
	if err != nil {
		return nil, err
	}

	err = uc.matchRepo.CreateMatch(context.Background(), newMatch)
	if err != nil {
		return nil, err
	}
	return newMatch, nil
}
