// internal/gateway/pg/masteraction/repository.go
package pgmasteraction

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	pgfs "github.com/422UR4H/HxH_RPG_System/pkg"
)

// Repository persists master actions: the master's own actions, in a table of their own
// (spec §4.8, B14) — separate from actions because the actor is the master (a USER), not
// a character sheet, and the action can happen OUTSIDE any turn.
//
// q is the pool for a master action applied with no turn open, and PersistTurnClose's own
// transaction for one applied inside a turn — which is written with that turn's close, or not
// at all (owner decision, 2026-10-01). Same Insert either way: the SQL lives here only.
type Repository struct {
	q pgfs.IQuerier
}

func NewRepository(q pgfs.IQuerier) *Repository {
	return &Repository{q: q}
}

func (r *Repository) Insert(ctx context.Context, rec masteraction.Record) error {
	content := rec.Content
	if content == nil {
		content = json.RawMessage("{}")
	}

	// uuid.UUID implements encoding.TextMarshaler/TextUnmarshaler, so json handles
	// map[uuid.UUID]View directly as a {"<uuid>": "<view>"} object.
	views := []byte("{}")
	if len(rec.Views) > 0 {
		var err error
		views, err = json.Marshal(rec.Views)
		if err != nil {
			return fmt.Errorf("marshal master action views: %w", err)
		}
	}

	const q = `
		INSERT INTO master_actions
			(uuid, match_uuid, scene_uuid, round_uuid, turn_uuid, master_uuid, kind, content, views, happened_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.q.Exec(ctx, q,
		rec.UUID, rec.MatchUUID, rec.SceneUUID, rec.RoundUUID, rec.TurnUUID, rec.MasterUUID,
		string(rec.Kind), []byte(content), views, rec.HappenedAt,
	)
	if err != nil {
		return fmt.Errorf("insert master action: %w", err)
	}
	return nil
}

func (r *Repository) ListByMatch(ctx context.Context, matchUUID uuid.UUID) ([]masteraction.Record, error) {
	const q = `
		SELECT uuid, match_uuid, scene_uuid, round_uuid, turn_uuid, master_uuid, kind, content, views, happened_at
		FROM master_actions
		WHERE match_uuid = $1
		ORDER BY happened_at, uuid
	`
	rows, err := r.q.Query(ctx, q, matchUUID)
	if err != nil {
		return nil, fmt.Errorf("list master actions: %w", err)
	}
	defer rows.Close()

	records := []masteraction.Record{}
	for rows.Next() {
		var rec masteraction.Record
		var kind string
		var content, views []byte

		if err := rows.Scan(
			&rec.UUID, &rec.MatchUUID, &rec.SceneUUID, &rec.RoundUUID, &rec.TurnUUID, &rec.MasterUUID,
			&kind, &content, &views, &rec.HappenedAt,
		); err != nil {
			return nil, fmt.Errorf("scan master action: %w", err)
		}

		rec.Kind = masteraction.Kind(kind)
		rec.Content = json.RawMessage(content)
		if len(views) > 0 {
			if err := json.Unmarshal(views, &rec.Views); err != nil {
				return nil, fmt.Errorf("unmarshal master action views: %w", err)
			}
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate master actions: %w", err)
	}
	return records, nil
}
