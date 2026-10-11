// internal/gateway/pg/cancelledaction/repository.go
package pgcancelledaction

import (
	"context"
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/cancelledaction"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/actioncodec"
	pgfs "github.com/422UR4H/HxH_RPG_System/pkg"
	"github.com/google/uuid"
)

// Repository persists cancelled actions (spec 2026-10-10 §8.1 P6): actions taken back out of
// the queue before they opened, in a table of their own because they never had a turn.
//
// q is the pool for a cancellation with no turn open, and PersistTurnClose's own transaction for
// one made inside a turn — written with that turn's close, or not at all (P7), exactly like
// pgmasteraction. Same Insert either way: the SQL lives here only.
type Repository struct {
	q pgfs.IQuerier
}

func NewRepository(q pgfs.IQuerier) *Repository {
	return &Repository{q: q}
}

func (r *Repository) Insert(ctx context.Context, rec cancelledaction.Record) error {
	// The action in the same storage pieces actions keeps in its columns — actioncodec, the
	// one mapping — so the history rebuilds both the same way.
	raw, err := actioncodec.Marshal(&rec.Action)
	if err != nil {
		return fmt.Errorf("marshal cancelled action: %w", err)
	}

	const q = `
		INSERT INTO cancelled_actions
			(uuid, match_uuid, scene_uuid, round_uuid, turn_uuid, actor_uuid, cancelled_by, action, cancelled_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err = r.q.Exec(ctx, q,
		rec.UUID, rec.MatchUUID, rec.SceneUUID, rec.RoundUUID, rec.TurnUUID, rec.ActorUUID,
		rec.CancelledBy, raw, rec.CancelledAt,
	)
	if err != nil {
		return fmt.Errorf("insert cancelled action: %w", err)
	}
	return nil
}

// ListByMatch reads the match's cancelled actions in the order they happened, each action
// rebuilt by actioncodec — the same decoding the history runs over actions. Never nil.
func (r *Repository) ListByMatch(ctx context.Context, matchUUID uuid.UUID) ([]cancelledaction.Record, error) {
	const q = `
		SELECT uuid, match_uuid, scene_uuid, round_uuid, turn_uuid, actor_uuid, cancelled_by, action, cancelled_at
		FROM cancelled_actions
		WHERE match_uuid = $1
		ORDER BY cancelled_at, uuid
	`
	rows, err := r.q.Query(ctx, q, matchUUID)
	if err != nil {
		return nil, fmt.Errorf("list cancelled actions: %w", err)
	}
	defer rows.Close()

	records := []cancelledaction.Record{}
	for rows.Next() {
		var rec cancelledaction.Record
		var raw []byte
		if err := rows.Scan(
			&rec.UUID, &rec.MatchUUID, &rec.SceneUUID, &rec.RoundUUID, &rec.TurnUUID, &rec.ActorUUID,
			&rec.CancelledBy, &raw, &rec.CancelledAt,
		); err != nil {
			return nil, fmt.Errorf("scan cancelled action: %w", err)
		}
		act, err := actioncodec.Unmarshal(rec.UUID, rec.ActorUUID, raw)
		if err != nil {
			return nil, fmt.Errorf("decode cancelled action %s: %w", rec.UUID, err)
		}
		rec.Action = *act
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cancelled actions: %w", err)
	}
	return records, nil
}
