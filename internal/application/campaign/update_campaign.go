package campaign

import (
	"context"
	"errors"
	"time"

	campaignEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	campaignPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	"github.com/google/uuid"
)

type IUpdateCampaign interface {
	Update(ctx context.Context, input *UpdateCampaignInput) (*campaignEntity.Campaign, error)
}

// CampaignUpdateContext is returned by GetCampaignForUpdate: all editable fields
// plus validation flags, fetched in a single query.
type CampaignUpdateContext struct {
	MasterUUID              uuid.UUID
	Name                    string
	BriefInitialDescription string
	Description             string
	IsPublic                bool
	CallLink                string
	StoryStartAt            time.Time
	StoryCurrentAt          *time.Time
	StoryEndAt              *time.Time
	HasStartedMatch         bool
}

type UpdateCampaignInput struct {
	CampaignUUID uuid.UUID
	MasterUUID   uuid.UUID
	// Always editable
	BriefInitialDescription *string
	Description             *string
	IsPublic                *bool
	CallLink                *string
	StoryCurrentAt          *time.Time
	// Free mode only (locked after any match starts)
	Name         *string
	StoryStartAt *time.Time
}

type UpdateCampaignUC struct {
	repo IRepository
}

func NewUpdateCampaignUC(repo IRepository) *UpdateCampaignUC {
	return &UpdateCampaignUC{repo: repo}
}

func (uc *UpdateCampaignUC) Update(
	ctx context.Context, input *UpdateCampaignInput,
) (*campaignEntity.Campaign, error) {
	ctxData, err := uc.repo.GetCampaignForUpdate(ctx, input.CampaignUUID)
	if err != nil {
		if errors.Is(err, campaignPg.ErrCampaignNotFound) {
			return nil, ErrCampaignNotFound
		}
		return nil, err
	}
	if ctxData.MasterUUID != input.MasterUUID {
		return nil, ErrNotCampaignOwner
	}
	if ctxData.StoryEndAt != nil {
		return nil, ErrCampaignAlreadyEnded
	}
	if ctxData.HasStartedMatch && (input.Name != nil || input.StoryStartAt != nil) {
		return nil, ErrLockedAfterMatchStart
	}
	if input.StoryCurrentAt != nil && ctxData.StoryCurrentAt != nil &&
		input.StoryCurrentAt.Before(*ctxData.StoryCurrentAt) {
		return nil, ErrCannotRegressStoryCurrentAt
	}

	c := buildFromContext(input.CampaignUUID, ctxData)

	if input.Name == nil && input.BriefInitialDescription == nil &&
		input.Description == nil && input.IsPublic == nil &&
		input.CallLink == nil && input.StoryCurrentAt == nil &&
		input.StoryStartAt == nil {
		return c, nil
	}

	if input.Name != nil {
		if len(*input.Name) < 5 {
			return nil, ErrMinNameLength
		}
		if len(*input.Name) > 32 {
			return nil, ErrMaxNameLength
		}
		c.Name = *input.Name
	}
	if input.BriefInitialDescription != nil {
		if len(*input.BriefInitialDescription) > 255 {
			return nil, ErrMaxBriefDescLength
		}
		c.BriefInitialDescription = *input.BriefInitialDescription
	}
	if input.Description != nil {
		c.Description = *input.Description
	}
	if input.IsPublic != nil {
		c.IsPublic = *input.IsPublic
	}
	if input.CallLink != nil {
		if len(*input.CallLink) > 255 {
			return nil, ErrMaxCallLinkLength
		}
		c.CallLink = *input.CallLink
	}
	if input.StoryCurrentAt != nil {
		c.StoryCurrentAt = input.StoryCurrentAt
	}
	if input.StoryStartAt != nil {
		c.StoryStartAt = *input.StoryStartAt
	}
	c.UpdatedAt = time.Now()

	if err := uc.repo.UpdateCampaign(ctx, c); err != nil {
		if errors.Is(err, campaignPg.ErrCampaignNotFound) {
			return nil, ErrCampaignNotFound
		}
		return nil, err
	}
	return c, nil
}

func buildFromContext(campaignUUID uuid.UUID, d *CampaignUpdateContext) *campaignEntity.Campaign {
	return &campaignEntity.Campaign{
		UUID:                    campaignUUID,
		MasterUUID:              d.MasterUUID,
		Name:                    d.Name,
		BriefInitialDescription: d.BriefInitialDescription,
		Description:             d.Description,
		IsPublic:                d.IsPublic,
		CallLink:                d.CallLink,
		StoryStartAt:            d.StoryStartAt,
		StoryCurrentAt:          d.StoryCurrentAt,
	}
}
