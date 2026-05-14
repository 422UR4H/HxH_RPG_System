package round

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	roundentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	sceneentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
	turnentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/google/uuid"
)

// PersistTurnClose atomically writes scene (idempotent), round (idempotent),
// turn, and action within a single database transaction.
func (r *Repository) PersistTurnClose(
	ctx context.Context,
	sc *sceneentity.Scene,
	rnd *roundentity.Round,
	t *turnentity.Turn,
	act *action.Action,
	matchUUID uuid.UUID,
) error {
	if t.GetFinishedAt() == nil {
		return fmt.Errorf("PersistTurnClose: turn must be closed before persisting")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("PersistTurnClose begin tx: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
		_ = tx.Rollback(ctx) // no-op after Commit
	}()

	// Insert scene — idempotent via ON CONFLICT DO NOTHING
	_, err = tx.Exec(ctx,
		`INSERT INTO scenes (uuid, match_uuid, category, brief_initial_description, created_at)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (uuid) DO NOTHING`,
		sc.GetID(), matchUUID, string(sc.GetCategory()), sc.BriefInitialDescription, sc.GetCreatedAt(),
	)
	if err != nil {
		return fmt.Errorf("PersistTurnClose insert scene: %w", err)
	}

	// Insert round — idempotent via ON CONFLICT DO NOTHING
	_, err = tx.Exec(ctx,
		`INSERT INTO rounds (uuid, scene_uuid, mode, created_at)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (uuid) DO NOTHING`,
		rnd.GetID(), sc.GetID(), string(rnd.GetMode()), rnd.GetCreatedAt(),
	)
	if err != nil {
		return fmt.Errorf("PersistTurnClose insert round: %w", err)
	}

	// Insert turn — turn entity has no createdAt field; use time.Now() for created_at
	now := time.Now()
	finishedAt := t.GetFinishedAt()
	_, err = tx.Exec(ctx,
		`INSERT INTO turns (uuid, round_uuid, created_at, finished_at)
		 VALUES ($1, $2, $3, $4)`,
		t.GetID(), rnd.GetID(), now, finishedAt,
	)
	if err != nil {
		return fmt.Errorf("PersistTurnClose insert turn: %w", err)
	}

	// Build nullable JSON columns for action sub-types
	speedJSON, err := json.Marshal(act.Speed)
	if err != nil {
		return fmt.Errorf("PersistTurnClose marshal speed: %w", err)
	}

	skillsJSON, err := marshalNullableSlice(act.Skills)
	if err != nil {
		return fmt.Errorf("PersistTurnClose marshal skills: %w", err)
	}

	moveJSON, err := marshalNullablePtr(act.Move)
	if err != nil {
		return fmt.Errorf("PersistTurnClose marshal move: %w", err)
	}

	attackJSON, err := marshalNullablePtr(act.Attack)
	if err != nil {
		return fmt.Errorf("PersistTurnClose marshal attack: %w", err)
	}

	defenseJSON, err := marshalNullablePtr(act.Defense)
	if err != nil {
		return fmt.Errorf("PersistTurnClose marshal defense: %w", err)
	}

	dodgeJSON, err := marshalNullablePtr(act.Dodge)
	if err != nil {
		return fmt.Errorf("PersistTurnClose marshal dodge: %w", err)
	}

	feintJSON, err := marshalNullablePtr(act.Feint)
	if err != nil {
		return fmt.Errorf("PersistTurnClose marshal feint: %w", err)
	}

	triggerJSON, err := marshalNullablePtr(act.Trigger)
	if err != nil {
		return fmt.Errorf("PersistTurnClose marshal trigger: %w", err)
	}

	// react_to_uuid: nil SQL when ReactToID is zero UUID
	var reactToUUID *uuid.UUID
	if act.ReactToID != uuid.Nil {
		v := act.ReactToID
		reactToUUID = &v
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO actions
		 (uuid, turn_uuid, actor_uuid, react_to_uuid, target_ids, type,
		  speed, skills, move, attack, defense, dodge, feint, trigger, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		act.GetID(), t.GetID(), act.GetActorID(), reactToUUID,
		act.TargetID, deriveActionType(act),
		speedJSON, skillsJSON, moveJSON, attackJSON,
		defenseJSON, dodgeJSON, feintJSON, triggerJSON,
		finishedAt,
	)
	if err != nil {
		return fmt.Errorf("PersistTurnClose insert action: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("PersistTurnClose commit: %w", err)
	}
	return nil
}

// deriveActionType returns a string action type based on which payload field is set.
func deriveActionType(act *action.Action) string {
	switch {
	case act.Attack != nil:
		return "attack"
	case act.Move != nil:
		return "move"
	case act.Defense != nil:
		return "defense"
	case act.Dodge != nil:
		return "dodge"
	case act.Feint != nil:
		return "feint"
	case len(act.Skills) > 0:
		return "skill"
	default:
		return "unspecified"
	}
}

// marshalNullablePtr returns nil (SQL NULL) for a nil pointer, else JSON bytes.
func marshalNullablePtr[T any](v *T) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}

// marshalNullableSlice returns nil (SQL NULL) for a nil or empty slice, else JSON bytes.
func marshalNullableSlice[T any](v []T) ([]byte, error) {
	if len(v) == 0 {
		return nil, nil
	}
	return json.Marshal(v)
}
