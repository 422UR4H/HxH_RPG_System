package fog

import (
	"context"

	"github.com/google/uuid"
	fogentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
)

// IPlayerMemoryRepository persists per-player static-feature memory (walls today,
// decorations in phase 11). Implementation is a TODO (see gateway/pg/fog).
type IPlayerMemoryRepository interface {
	Upsert(ctx context.Context, memory fogentity.PlayerMemory) error
	FindByMatchMap(ctx context.Context, matchID, mapID uuid.UUID) ([]fogentity.PlayerMemory, error)
	DeleteByMatch(ctx context.Context, matchID uuid.UUID) error
}
