package campaign

import (
	"time"

	"github.com/google/uuid"
)

// CampaignUpdateContext holds all editable campaign fields plus validation flags,
// returned by GetCampaignForUpdate in a single query with an EXISTS subquery.
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
