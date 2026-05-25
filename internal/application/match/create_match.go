package match

import (
	"context"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	pgCampaign "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	"github.com/google/uuid"
)

type ICreateMatch interface {
	CreateMatch(ctx context.Context, input *CreateMatchInput) (*match.Match, error)
}

type CreateMatchInput struct {
	MasterUUID              uuid.UUID
	CampaignUUID            uuid.UUID
	Title                   string
	BriefInitialDescription string
	Description             string
	IsPublic                bool
	GameScheduledAt         time.Time
	StoryStartAt            time.Time
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

func (uc *CreateMatchUC) CreateMatch(
	ctx context.Context, input *CreateMatchInput,
) (*match.Match, error) {
	if len(input.Title) < 5 {
		return nil, ErrMinTitleLength
	}
	if len(input.Title) > 32 {
		return nil, ErrMaxTitleLength
	}

	if len(input.BriefInitialDescription) > 255 {
		return nil, ErrMaxBriefDescLength
	}

	if input.GameScheduledAt.Before(time.Now()) {
		return nil, ErrMinOfGameScheduledAt
	}
	if input.GameScheduledAt.After(time.Now().AddDate(1, 0, 0)) {
		return nil, ErrMaxOfGameScheduledAt
	}

	c, err := uc.campaignRepo.GetCampaignStoryDates(ctx, input.CampaignUUID)
	if err == pgCampaign.ErrCampaignNotFound {
		return nil, campaign.ErrCampaignNotFound
	}
	if err != nil {
		return nil, err
	}

	if c.MasterUUID != input.MasterUUID {
		return nil, campaign.ErrNotCampaignOwner
	}

	if input.StoryStartAt.Before(c.StoryStartAt) {
		return nil, ErrMinOfStoryStartAt
	}
	if c.StoryEndAt != nil && input.StoryStartAt.After(*c.StoryEndAt) {
		return nil, ErrMaxOfStoryStartAt
	}

	newMatch, err := match.NewMatch(
		input.MasterUUID,
		input.CampaignUUID,
		input.Title,
		input.BriefInitialDescription,
		input.Description,
		input.IsPublic,
		input.GameScheduledAt,
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
