package round

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/fog"
	pgmasteraction "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/masteraction"
	pgmatchboard "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/matchboard"
	pgsheet "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/sheet"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// PersistTurnClose atomically writes scene (idempotent), round (idempotent — with its
// finished_at when this close also ended it, and then the round born in its place, NextRound),
// turn, action, the action's reactions, the turn's settled resolution (nullable), the values
// the master's edits displaced, the bars of every sheet the close damaged, the master actions
// applied while the turn was open, and the board (with every player's fog memory) as the close
// left it, within a single database transaction.
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
	// A successor is born only when this close also ended the round: written next to a round
	// that is still open, the scene would hold two.
	if d.NextRound != nil && (rnd == nil || rnd.GetFinishedAt() == nil) {
		return fmt.Errorf("PersistTurnClose: a next round needs the closed round's finished_at")
	}
	if err := validateRoundEnds(d.UnwrittenRoundEnds); err != nil {
		return fmt.Errorf("PersistTurnClose: %w", err)
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

	// Round ends an earlier command could not write go first, closed: the round below may be the
	// one born after them, and it must never be born open next to a predecessor still open on disk.
	if err := writeRoundEnds(ctx, tx, matchUUID, d.UnwrittenRoundEnds); err != nil {
		return fmt.Errorf("PersistTurnClose %w", err)
	}

	// Scene and round — idempotent upserts (on conflict only the round's mode and a missing
	// finished_at are refreshed, see ensureSceneAndRound), and the SAME two inserts
	// EnsureSceneAndRound runs on its own: a master action may already have written them before
	// this round's first turn closed (spec §4.8). Inside this transaction, like everything else.
	if err := ensureSceneAndRound(ctx, tx, matchUUID, sc, rnd); err != nil {
		return fmt.Errorf("PersistTurnClose %w", err)
	}
	// This close also ended the round (rnd carries its finished_at, which the upsert above just
	// wrote — the round-close SQL is that COALESCE): the round born in its place is a row from
	// this same transaction (owner decision, 2026-10-02). The round's end, the successor's birth
	// and the turn are one master command, so they are durable together or not at all.
	if d.NextRound != nil {
		if err := ensureSceneAndRound(ctx, tx, matchUUID, sc, d.NextRound); err != nil {
			return fmt.Errorf("PersistTurnClose next round: %w", err)
		}
	}

	// Insert turn — turn entity has no createdAt field. AGENTS.md's own known-issues entry
	// names the approximation this is supposed to be: finishedAt stands in for created_at,
	// not time.Now(). Persistence always happens strictly AFTER the turn closes, so a
	// time.Now() here would make every turn's created_at land strictly AFTER its finished_at —
	// invisible while nothing read created_at, and nonsense the moment something sorts by it
	// (HistoryTurnResponse.CreatedAt, since Task 12).
	finishedAt := t.GetFinishedAt()
	resolutionJSON, err := encodeResolution(d.Resolution, d.LandingViews)
	if err != nil {
		return fmt.Errorf("PersistTurnClose marshal resolution: %w", err)
	}
	_, err = tx.Exec(ctx,
		`INSERT INTO turns (uuid, round_uuid, created_at, finished_at, resolution)
		 VALUES ($1, $2, $3, $4, $5)`,
		t.GetID(), rnd.GetID(), *finishedAt, finishedAt, resolutionJSON,
	)
	if err != nil {
		return fmt.Errorf("PersistTurnClose insert turn: %w", err)
	}

	// The action carries what each player saw of its move live (move_views); a reaction carries
	// what each player saw of its destination when it was opened (nil if never opened); an
	// escape's landing is recorded in the resolution above.
	if err := insertAction(ctx, tx, act, t.GetID(), *finishedAt, d.MoveViews); err != nil {
		return fmt.Errorf("PersistTurnClose insert action: %w", err)
	}

	// Reactions after the action, never before: react_to_uuid references it.
	reactions := t.GetReactions()
	for i := range reactions {
		if err := insertAction(ctx, tx, &reactions[i], t.GetID(), *finishedAt, d.ReactionMoveViews[reactions[i].GetID()]); err != nil {
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

	// The HP the close applied, in this same transaction (owner decision, 2026-10-02): one master
	// command, one transaction — a failure anywhere in it (the master actions, the board) takes
	// the sheets back with the turn. The sheet gateway's own UpdateStatusBars, run on this
	// transaction — the SQL lives in one place.
	if len(d.StatusBars) > 0 {
		sheets := pgsheet.NewRepository(tx)
		for _, sb := range d.StatusBars {
			if err := sheets.UpdateStatusBars(ctx, sb.CharacterID.String(), sb.Health, sb.Stamina, sb.Aura); err != nil {
				return fmt.Errorf("PersistTurnClose status bars of %s: %w", sb.CharacterID, err)
			}
		}
	}

	// What the master did while this turn was open becomes durable with the turn or not at all
	// (owner decision, 2026-10-01): a turn lost to a restart takes its master actions with it,
	// and a master action that fails here takes the turn back with it. The master-action
	// gateway's own Insert, run on this transaction — the SQL lives in one place.
	if len(d.MasterActions) > 0 {
		mas := pgmasteraction.NewRepository(tx)
		for i := range d.MasterActions {
			if err := mas.Insert(ctx, d.MasterActions[i]); err != nil {
				return fmt.Errorf("PersistTurnClose master action %d: %w", i, err)
			}
		}
	}

	// The board the close left — the turn's move, its escapes, the master's changes inside it —
	// and every player's fog memory, in this same transaction: a turn on disk always has its
	// board on disk, and a turn that is not written leaves the board as the last close did.
	// Unlike SaveMatchBoardUC, one memory failing fails the whole close: inside a transaction
	// there is no "keep the others" — the first error aborts it anyway. Same pattern as
	// pgmatchboard.Copy: both gateways run on the tx, their SQL stays where it lives.
	if d.Board != nil {
		if err := pgmatchboard.NewRepository(tx).Save(ctx, d.Board); err != nil {
			return fmt.Errorf("PersistTurnClose board: %w", err)
		}
		mems := fog.NewPlayerMemoryRepository(tx)
		for i := range d.Memories {
			if err := mems.Upsert(ctx, d.Memories[i]); err != nil {
				return fmt.Errorf("PersistTurnClose player memory %s: %w", d.Memories[i].PlayerID, err)
			}
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
//
// moveViews is move_views: nil writes SQL NULL ("not recorded"), an empty map '{}' ("recorded,
// nobody but the master and the owner saw it") — the history reads both the same way.
func insertAction(
	ctx context.Context,
	tx pgx.Tx,
	act *action.Action,
	turnID uuid.UUID,
	createdAt time.Time,
	moveViews map[uuid.UUID]masteraction.View,
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

	interactJSON, err := marshalNullablePtr(act.Interact)
	if err != nil {
		return fmt.Errorf("marshal interact: %w", err)
	}
	defaultDefenseJSON, err := marshalNullablePtr(act.DefaultDefense)
	if err != nil {
		return fmt.Errorf("marshal default defense: %w", err)
	}

	var moveViewsJSON []byte
	if moveViews != nil {
		if moveViewsJSON, err = json.Marshal(moveViews); err != nil {
			return fmt.Errorf("marshal move views: %w", err)
		}
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
		  interact, system_bias, reaction_kind, created_at, move_views, consumed_action_ids,
		  default_defense)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)`,
		act.GetID(), turnID, act.GetActorID(), reactToUUID,
		targetIDs, deriveActionType(act),
		speedJSON, skillsJSON, moveJSON, attackJSON,
		defenseJSON, dodgeJSON, repelJSON, feintJSON, triggerJSON,
		interactJSON, act.SystemBias,
		reactionKind, createdAt, moveViewsJSON, consumedOrNil(act.ConsumedActionIDs),
		defaultDefenseJSON,
	)
	return err
}

// consumedOrNil is consumed_action_ids: NULL when the reaction consumed nothing (or the row is a
// plain action), never an empty array — one way to say "nothing", the way the history reads it.
func consumedOrNil(ids []uuid.UUID) []uuid.UUID {
	if len(ids) == 0 {
		return nil
	}
	return ids
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
	case act.Interact != nil:
		// Ahead of the skills row on purpose: a lockpick carries BOTH an Interact and a
		// Skills entry, and what it is is an interaction — the skill is how it is attempted.
		return "interact"
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
