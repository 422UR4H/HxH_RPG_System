package matchmapuc

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// MatchInfo holds the minimal match data needed by matchmap use cases.
//
// CampaignUUID was added for B16 (spec §4.3): inheriting a board across matches is only
// allowed within the same campaign, so AttachMatchMapUC needs it for both the destination
// and the source match.
type MatchInfo struct {
	MasterUUID   uuid.UUID
	GameStartAt  *time.Time
	CampaignUUID uuid.UUID
}

type IMatchRepository interface {
	GetMatchInfo(ctx context.Context, matchUUID uuid.UUID) (*MatchInfo, error)
}
