package fog

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	fogentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
	pgfs "github.com/422UR4H/HxH_RPG_System/pkg"
)

// PlayerMemoryRepository is the Postgres-backed memory persistence, against
// player_memories (match_id, map_id, player_id, seen_features JSONB, updated_at).
// Seen is stored as [{"kind":"wall","id":"<id>"}, ...] — the same shape spelled out in
// spec §4.3.
type PlayerMemoryRepository struct {
	q pgfs.IQuerier
}

func NewPlayerMemoryRepository(q pgfs.IQuerier) *PlayerMemoryRepository {
	return &PlayerMemoryRepository{q: q}
}

// seenFeatureDTO is the wire shape of one entry of PlayerMemory.Seen. A plain string map
// key (fogentity.FeatureRef) cannot be a JSON object key, so this is the on-disk shape;
// the domain entity never sees it.
type seenFeatureDTO struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

func marshalSeen(seen map[fogentity.FeatureRef]struct{}) ([]byte, error) {
	refs := make([]seenFeatureDTO, 0, len(seen))
	for ref := range seen {
		refs = append(refs, seenFeatureDTO{Kind: string(ref.Kind), ID: ref.ID})
	}
	b, err := json.Marshal(refs)
	if err != nil {
		return nil, fmt.Errorf("marshal seen features: %w", err)
	}
	return b, nil
}

func unmarshalSeen(raw []byte) (map[fogentity.FeatureRef]struct{}, error) {
	var refs []seenFeatureDTO
	if err := json.Unmarshal(raw, &refs); err != nil {
		return nil, fmt.Errorf("unmarshal seen features: %w", err)
	}
	seen := make(map[fogentity.FeatureRef]struct{}, len(refs))
	for _, ref := range refs {
		seen[fogentity.FeatureRef{Kind: fogentity.FeatureKind(ref.Kind), ID: ref.ID}] = struct{}{}
	}
	return seen, nil
}

func (r *PlayerMemoryRepository) Upsert(ctx context.Context, m fogentity.PlayerMemory) error {
	seen, err := marshalSeen(m.Seen)
	if err != nil {
		return err
	}

	// Timestamp generated in Go, not SQL now() (gateway-conventions.instructions.md).
	updatedAt := time.Now().UTC()

	const q = `
		INSERT INTO player_memories (match_id, map_id, player_id, seen_features, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (match_id, map_id, player_id) DO UPDATE SET
			seen_features = EXCLUDED.seen_features,
			updated_at = EXCLUDED.updated_at
	`
	_, err = r.q.Exec(ctx, q, m.MatchID, m.MapID, m.PlayerID, seen, updatedAt)
	if err != nil {
		return fmt.Errorf("upsert player memory: %w", err)
	}
	return nil
}

func (r *PlayerMemoryRepository) FindByMatchMap(ctx context.Context, matchID, mapID uuid.UUID) ([]fogentity.PlayerMemory, error) {
	const q = `
		SELECT player_id, seen_features, updated_at
		FROM player_memories
		WHERE match_id = $1 AND map_id = $2
	`
	rows, err := r.q.Query(ctx, q, matchID, mapID)
	if err != nil {
		return nil, fmt.Errorf("find player memories: %w", err)
	}
	defer rows.Close()

	memories := []fogentity.PlayerMemory{}
	for rows.Next() {
		var m fogentity.PlayerMemory
		var seenRaw []byte
		if err := rows.Scan(&m.PlayerID, &seenRaw, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan player memory: %w", err)
		}
		m.MatchID = matchID
		m.MapID = mapID
		seen, err := unmarshalSeen(seenRaw)
		if err != nil {
			return nil, err
		}
		m.Seen = seen
		memories = append(memories, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate player memories: %w", err)
	}
	return memories, nil
}

func (r *PlayerMemoryRepository) DeleteByMatch(ctx context.Context, matchID uuid.UUID) error {
	const q = `DELETE FROM player_memories WHERE match_id = $1`
	if _, err := r.q.Exec(ctx, q, matchID); err != nil {
		return fmt.Errorf("delete player memories by match: %w", err)
	}
	return nil
}

// CopyMatch copies every player_memories row of srcMatchID ON mapID to dstMatchID (spec
// §4.3, B16): same map_id/player_id/seen_features/updated_at, a fresh id, and dstMatchID in
// place of srcMatchID. dstMatchID's own rows are cleared first — same "replace, not merge"
// semantics as matchboard's Copy — so a destination match that already had memories (e.g.
// from a board on a different map) never collides with the UNIQUE(match_id, map_id,
// player_id) constraint.
//
// mapID restricts the SELECT to the source board's own map (fix round 1, review finding):
// srcMatchID is a PAST match, and nothing prevents it from also holding stray/orphan
// player_memories rows for some OTHER map it was once attached to (e.g. before an earlier
// re-attach deleted its board but not, at the time, its memories on the old map — or any
// future code path that writes memories without going through persistBoard). Without this
// filter those rows would ride along into dstMatchID despite belonging to a map dstMatchID
// was never even shown.
//
// r.q may be a transaction (pgx.Tx satisfies pgfs.IQuerier) so this can run inside the same
// atomic unit as matchboard.Repository.Copy's own board-row copy — see NewPlayerMemoryRepository
// call in pgmatchboard.Repository.Copy.
func (r *PlayerMemoryRepository) CopyMatch(ctx context.Context, srcMatchID, dstMatchID, mapID uuid.UUID) error {
	const del = `DELETE FROM player_memories WHERE match_id = $1`
	if _, err := r.q.Exec(ctx, del, dstMatchID); err != nil {
		return fmt.Errorf("copy match player memories: delete dst: %w", err)
	}

	const ins = `
		INSERT INTO player_memories (id, match_id, map_id, player_id, seen_features, updated_at)
		SELECT gen_random_uuid(), $1, map_id, player_id, seen_features, updated_at
		FROM player_memories WHERE match_id = $2 AND map_id = $3
	`
	if _, err := r.q.Exec(ctx, ins, dstMatchID, srcMatchID, mapID); err != nil {
		return fmt.Errorf("copy match player memories: insert: %w", err)
	}
	return nil
}
