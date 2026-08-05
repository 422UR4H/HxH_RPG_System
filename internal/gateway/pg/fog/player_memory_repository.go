package fog

import (
	"context"

	"github.com/google/uuid"
	fogentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
)

// PlayerMemoryRepository is the Postgres-backed memory persistence.
// TODO(persistence): implement Upsert/FindByMatchMap/DeleteByMatch against
// player_memories (serialize Seen as JSONB [{"kind":"wall","id":"<uuid>"}, ...]).
// Until then the session keeps memory in RAM; start_match seeding works in-memory.
type PlayerMemoryRepository struct {
	// db *pgxpool.Pool  // wire when implementing
}

func NewPlayerMemoryRepository( /* db *pgxpool.Pool */ ) *PlayerMemoryRepository {
	return &PlayerMemoryRepository{}
}

func (r *PlayerMemoryRepository) Upsert(ctx context.Context, m fogentity.PlayerMemory) error {
	// TODO: INSERT ... ON CONFLICT (match_id, map_id, player_id) DO UPDATE.
	return nil
}

func (r *PlayerMemoryRepository) FindByMatchMap(ctx context.Context, matchID, mapID uuid.UUID) ([]fogentity.PlayerMemory, error) {
	// TODO: SELECT ... WHERE match_id=$1 AND map_id=$2.
	return nil, nil
}

func (r *PlayerMemoryRepository) DeleteByMatch(ctx context.Context, matchID uuid.UUID) error {
	// TODO: DELETE FROM player_memories WHERE match_id=$1.
	return nil
}
