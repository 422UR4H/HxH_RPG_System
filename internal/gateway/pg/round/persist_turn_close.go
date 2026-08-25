package round

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// PersistTurnClose atomically writes scene (idempotent), round (idempotent),
// turn, action, the action's reactions, and the turn's settled resolution (nullable) within
// a single database transaction.
//
// A turn is an action AND its reactions — that is the vocabulary the whole engine is built
// on — so writing only the action would persist half a turn. The reactions go in after the
// action they answer, because actions.react_to_uuid points back at it inside this same
// transaction.
func (r *Repository) PersistTurnClose(ctx context.Context, d appmatch.TurnCloseData) error {
	sc, rnd, t, act, matchUUID := d.Scene, d.Round, d.Turn, d.Action, d.MatchUUID
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
	resolutionJSON, err := encodeResolution(d.Resolution)
	if err != nil {
		return fmt.Errorf("PersistTurnClose marshal resolution: %w", err)
	}
	_, err = tx.Exec(ctx,
		`INSERT INTO turns (uuid, round_uuid, created_at, finished_at, resolution)
		 VALUES ($1, $2, $3, $4, $5)`,
		t.GetID(), rnd.GetID(), now, finishedAt, resolutionJSON,
	)
	if err != nil {
		return fmt.Errorf("PersistTurnClose insert turn: %w", err)
	}

	if err := insertAction(ctx, tx, act, t.GetID(), *finishedAt); err != nil {
		return fmt.Errorf("PersistTurnClose insert action: %w", err)
	}

	// Reactions after the action, never before: react_to_uuid references it.
	reactions := t.GetReactions()
	for i := range reactions {
		if err := insertAction(ctx, tx, &reactions[i], t.GetID(), *finishedAt); err != nil {
			return fmt.Errorf("PersistTurnClose insert reaction %d: %w", i, err)
		}
	}

	// The overrides go in the SAME transaction as the actions they point at: the FK requires
	// the action row to exist, and a capture that outlived its action would be unreadable.
	for _, ov := range d.Overrides {
		original, err := marshalNullableAny(ov.Original)
		if err != nil {
			return fmt.Errorf("PersistTurnClose marshal override %s: %w", ov.Field, err)
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO overridden_action_values
			 (action_uuid, field, origin, master_uuid, overridden_at, original_value)
			 VALUES ($1,$2,$3,$4,$5,$6)
			 ON CONFLICT (action_uuid, field) DO NOTHING`,
			ov.ActionID, ov.Field, string(ov.Origin), ov.MasterUUID, ov.At, original,
		)
		if err != nil {
			return fmt.Errorf("PersistTurnClose insert override: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("PersistTurnClose commit: %w", err)
	}
	return nil
}

// insertAction writes one row of the actions table, for the turn's action or for one of its
// reactions. The two are the same shape and the same table; what tells them apart in the row
// is react_to_uuid being set and reaction_kind carrying the declared kind.
func insertAction(
	ctx context.Context,
	tx pgx.Tx,
	act *action.Action,
	turnID uuid.UUID,
	createdAt time.Time,
) error {
	speedJSON, err := json.Marshal(act.Speed)
	if err != nil {
		return fmt.Errorf("marshal speed: %w", err)
	}

	skillsJSON, err := marshalNullableSlice(act.Skills)
	if err != nil {
		return fmt.Errorf("marshal skills: %w", err)
	}

	moveJSON, err := marshalNullablePtr(act.Move)
	if err != nil {
		return fmt.Errorf("marshal move: %w", err)
	}

	attackJSON, err := marshalNullablePtr(act.Attack)
	if err != nil {
		return fmt.Errorf("marshal attack: %w", err)
	}

	defenseJSON, err := marshalNullablePtr(act.Defense)
	if err != nil {
		return fmt.Errorf("marshal defense: %w", err)
	}

	dodgeJSON, err := marshalNullablePtr(act.Dodge)
	if err != nil {
		return fmt.Errorf("marshal dodge: %w", err)
	}

	repelJSON, err := marshalNullablePtr(act.Repel)
	if err != nil {
		return fmt.Errorf("marshal repel: %w", err)
	}

	feintJSON, err := marshalNullablePtr(act.Feint)
	if err != nil {
		return fmt.Errorf("marshal feint: %w", err)
	}

	triggerJSON, err := marshalNullablePtr(act.Trigger)
	if err != nil {
		return fmt.Errorf("marshal trigger: %w", err)
	}

	// react_to_uuid: nil SQL when ReactToID is zero UUID
	var reactToUUID *uuid.UUID
	if act.ReactToID != uuid.Nil {
		v := act.ReactToID
		reactToUUID = &v
	}

	// reaction_kind: nil SQL on an action, the declared kind on a reaction
	var reactionKind *string
	if act.ReactionKind != "" {
		v := string(act.ReactionKind)
		reactionKind = &v
	}

	// target_ids: ensure it's never nil (use empty slice if nil)
	targetIDs := act.TargetID
	if targetIDs == nil {
		targetIDs = []uuid.UUID{}
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO actions
		 (uuid, turn_uuid, actor_uuid, react_to_uuid, target_ids, type,
		  speed, skills, move, attack, defense, dodge, repel, feint, trigger,
		  reaction_kind, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		act.GetID(), turnID, act.GetActorID(), reactToUUID,
		targetIDs, deriveActionType(act),
		speedJSON, skillsJSON, moveJSON, attackJSON,
		defenseJSON, dodgeJSON, repelJSON, feintJSON, triggerJSON,
		reactionKind, createdAt,
	)
	return err
}

// deriveActionType returns a string action type based on which payload field is set.
//
// A reaction is never classified this way: its kind is declared, not inferred, and inferring
// it from the shape is exactly what ReactionKind exists to stop — the three escapes carry the
// same fields. The row says "reaction" and reaction_kind says which one.
func deriveActionType(act *action.Action) string {
	if act.ReactToID != uuid.Nil {
		return "reaction"
	}
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

// marshalNullableAny returns nil (SQL NULL) for a nil value. NULL says "there was no value"
// — which is the honest answer for a RollCondition the player never sent — where 'null'::jsonb
// would claim there was one and it was null.
func marshalNullableAny(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}
