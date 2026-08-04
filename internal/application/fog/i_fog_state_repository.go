package fog

import (
	"context"

	"github.com/google/uuid"
	fogentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
)

// IFogStateRepository persists per-player explored-area state.
// Implementation is a Phase 10-D TODO (see gateway/pg/fog).
type IFogStateRepository interface {
	Upsert(ctx context.Context, state fogentity.PlayerFogState) error
	FindByMatchMap(ctx context.Context, matchID, mapID uuid.UUID) ([]fogentity.PlayerFogState, error)
	DeleteByMatch(ctx context.Context, matchID uuid.UUID) error
}
