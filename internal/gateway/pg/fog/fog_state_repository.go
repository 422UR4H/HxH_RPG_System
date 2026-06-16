package fog

import (
	"context"

	"github.com/google/uuid"
	fogentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
)

// FogStateRepository is the Postgres-backed fog persistence.
// TODO(10-D persistence): implement Upsert/FindByMatchMap/DeleteByMatch against
// player_fog_states (serialize ExploredCells as JSONB [[a,b],...]). Until then the
// session keeps fog state in memory; start_match seeding works via the in-memory path.
type FogStateRepository struct {
	// db *pgxpool.Pool  // wire when implementing
}

func NewFogStateRepository( /* db *pgxpool.Pool */ ) *FogStateRepository {
	return &FogStateRepository{}
}

func (r *FogStateRepository) Upsert(ctx context.Context, state fogentity.PlayerFogState) error {
	// TODO: INSERT ... ON CONFLICT (match_id, map_id, player_id) DO UPDATE.
	return nil
}

func (r *FogStateRepository) FindByMatchMap(ctx context.Context, matchID, mapID uuid.UUID) ([]fogentity.PlayerFogState, error) {
	// TODO: SELECT ... WHERE match_id=$1 AND map_id=$2.
	return nil, nil
}

func (r *FogStateRepository) DeleteByMatch(ctx context.Context, matchID uuid.UUID) error {
	// TODO: DELETE FROM player_fog_states WHERE match_id=$1.
	return nil
}
