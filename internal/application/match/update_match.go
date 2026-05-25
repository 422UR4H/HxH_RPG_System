package match

import (
	"context"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	pgCampaign "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	pgMatch "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

type IUpdateMatch interface {
	Update(ctx context.Context, input *UpdateMatchInput) (*match.Match, error)
}

type UpdateMatchInput struct {
	MatchUUID  uuid.UUID
	MasterUUID uuid.UUID

	Title                   *string
	BriefInitialDescription *string
	Description             *string
	IsPublic                *bool
	GameScheduledAt         *time.Time
	StoryStartAt            *time.Time
}

type UpdateMatchUC struct {
	matchRepo    IRepository
	campaignRepo campaign.IRepository
}

func NewUpdateMatchUC(
	matchRepo IRepository,
	campaignRepo campaign.IRepository,
) *UpdateMatchUC {
	return &UpdateMatchUC{matchRepo: matchRepo, campaignRepo: campaignRepo}
}

func (uc *UpdateMatchUC) Update(
	ctx context.Context, input *UpdateMatchInput,
) (*match.Match, error) {
	m, err := uc.matchRepo.GetMatch(ctx, input.MatchUUID)
	if err != nil {
		if err == pgMatch.ErrMatchNotFound {
			return nil, ErrMatchNotFound
		}
		return nil, err
	}
	if m.MasterUUID != input.MasterUUID {
		return nil, ErrNotMatchMaster
	}
	if m.GameStartAt != nil {
		return nil, ErrMatchAlreadyStarted
	}
	if m.StoryEndAt != nil {
		return nil, ErrMatchAlreadyFinished
	}

	if input.Title == nil && input.BriefInitialDescription == nil &&
		input.Description == nil && input.IsPublic == nil &&
		input.GameScheduledAt == nil && input.StoryStartAt == nil {
		return m, nil
	}

	if input.Title != nil {
		if len(*input.Title) < 5 {
			return nil, ErrMinTitleLength
		}
		if len(*input.Title) > 32 {
			return nil, ErrMaxTitleLength
		}
	}
	if input.BriefInitialDescription != nil && len(*input.BriefInitialDescription) > 64 {
		return nil, ErrMaxBriefDescLength
	}
	if input.GameScheduledAt != nil {
		now := time.Now()
		if input.GameScheduledAt.Before(now) {
			return nil, ErrMinOfGameScheduledAt
		}
		if input.GameScheduledAt.After(now.AddDate(1, 0, 0)) {
			return nil, ErrMaxOfGameScheduledAt
		}
	}
	if input.StoryStartAt != nil {
		c, err := uc.campaignRepo.GetCampaignStoryDates(ctx, m.CampaignUUID)
		if err == pgCampaign.ErrCampaignNotFound {
			return nil, campaign.ErrCampaignNotFound
		}
		if err != nil {
			return nil, err
		}
		if input.StoryStartAt.Before(c.StoryStartAt) {
			return nil, ErrMinOfStoryStartAt
		}
		if c.StoryEndAt != nil && input.StoryStartAt.After(*c.StoryEndAt) {
			return nil, ErrMaxOfStoryStartAt
		}
	}

	if input.Title != nil {
		m.Title = *input.Title
	}
	if input.BriefInitialDescription != nil {
		m.BriefInitialDescription = *input.BriefInitialDescription
	}
	if input.Description != nil {
		m.Description = *input.Description
	}
	if input.IsPublic != nil {
		m.IsPublic = *input.IsPublic
	}
	if input.GameScheduledAt != nil {
		m.GameScheduledAt = *input.GameScheduledAt
	}
	if input.StoryStartAt != nil {
		m.StoryStartAt = *input.StoryStartAt
	}
	m.UpdatedAt = time.Now()

	if err := uc.matchRepo.UpdateMatch(ctx, m); err != nil {
		if err == pgMatch.ErrMatchNotFound {
			return nil, ErrMatchAlreadyStarted
		}
		return nil, err
	}
	return m, nil
}
