package campaign

import (
	"context"
	"time"

	campaignEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
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
