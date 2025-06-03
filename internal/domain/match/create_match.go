package match

import (
	"context"
	"time"

	domainCampaign "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	pgCampaign "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	"github.com/google/uuid"
)

type ICreateMatch interface {
	CreateMatch(ctx context.Context, input *CreateMatchInput) (*match.Match, error)
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
	campaignRepo domainCampaign.IRepository
}

func NewCreateMatchUC(
	matchRepo IRepository,
	campaignRepo domainCampaign.IRepository,
) *CreateMatchUC {
	return &CreateMatchUC{
		matchRepo:    matchRepo,
		campaignRepo: campaignRepo,
	}
}

func (uc *CreateMatchUC) CreateMatch(
	ctx context.Context, input *CreateMatchInput,
) (*match.Match, error) {
	if len(input.Title) < 5 {
		return nil, ErrMinTitleLength
	}

	if len(input.Title) > 32 {
		return nil, ErrMaxTitleLength
	}

	if len(input.BriefDescription) > 64 {
		return nil, ErrMaxBriefDescLength
	}

	campaign, err := uc.campaignRepo.GetCampaignStoryDates(ctx, input.CampaignUUID)
	if err == pgCampaign.ErrCampaignNotFound {
		return nil, domainCampaign.ErrCampaignNotFound
	}
	if err != nil {
		return nil, err
	}

	if campaign.UserUUID != input.MasterUUID {
		return nil, ErrNotCampaignOwner
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

	err = uc.matchRepo.CreateMatch(ctx, newMatch)
	if err != nil {
		return nil, err
	}
	return newMatch, nil
}
