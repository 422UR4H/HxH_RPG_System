package matchmapuc

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// MatchInfo holds the minimal match data needed by matchmap use cases.
type MatchInfo struct {
	MasterUUID  uuid.UUID
	GameStartAt *time.Time
}

type IMatchRepository interface {
	GetMatchInfo(ctx context.Context, matchUUID uuid.UUID) (*MatchInfo, error)
}
