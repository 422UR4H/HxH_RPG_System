// internal/gateway/pg/matchevent/repository.go
package pgmatchevent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchevent"
	pgfs "github.com/422UR4H/HxH_RPG_System/pkg"
)

// Repository persists match events: what happens inside a round that is not a turn and
// is nobody's action (spec §4.5, B15). Master actions have a table of their own
// (gateway/pg/masteraction) and are never recorded here.
type Repository struct {
	q pgfs.IQuerier
}

func NewRepository(q pgfs.IQuerier) *Repository {
	return &Repository{q: q}
}

func (r *Repository) Insert(ctx context.Context, e matchevent.Event) error {
	payload := e.Payload
	if payload == nil {
		payload = json.RawMessage("{}")
	}

	const q = `
		INSERT INTO match_events (uuid, match_uuid, scene_uuid, round_uuid, kind, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.q.Exec(ctx, q,
		e.UUID, e.MatchUUID, e.SceneUUID, e.RoundUUID, string(e.Kind), []byte(payload), e.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert match event: %w", err)
	}
	return nil
}

func (r *Repository) ListByMatch(ctx context.Context, matchUUID uuid.UUID) ([]matchevent.Event, error) {
	const q = `
		SELECT uuid, match_uuid, scene_uuid, round_uuid, kind, payload, created_at
		FROM match_events
		WHERE match_uuid = $1
		ORDER BY created_at, uuid
	`
	rows, err := r.q.Query(ctx, q, matchUUID)
	if err != nil {
		return nil, fmt.Errorf("list match events: %w", err)
	}
	defer rows.Close()

	events := []matchevent.Event{}
	for rows.Next() {
		var e matchevent.Event
		var kind string
		var payload []byte
		if err := rows.Scan(&e.UUID, &e.MatchUUID, &e.SceneUUID, &e.RoundUUID, &kind, &payload, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan match event: %w", err)
		}
		e.Kind = matchevent.Kind(kind)
		e.Payload = json.RawMessage(payload)
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate match events: %w", err)
	}
	return events, nil
}
