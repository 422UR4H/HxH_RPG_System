package round

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/google/uuid"
)

// FindMatchHistory returns the match's scenes, rounds and closed turns as the TREE the domain
// already is — Scene -> Round -> Turn -> Action — not a flat list.
//
// The hierarchy is not decoration: the front renders action cards INSIDE the scope of each
// scene, because scenes are the logical blocks the match is organised into. Flattening here
// would push the regrouping onto every consumer.
//
// One query, ordered, assembled in one pass. Reactions come back in the same result set,
// discriminated by react_to_uuid IS NOT NULL, so there is no N+1 over turns.
//
// The joins are LEFT joins (B15, spec §4.5): a scene and a round are rows from the moment they
// are born, not from their first closed turn, and a scene the table spent talking or a round
// that closed without a turn happened just the same. So a row may stop at the scene (no round
// yet) or at the round (no turn), and the assembly only builds a level whose UUID is not NULL.
// A turn row with no action row cannot be shown — the action IS the turn — and is skipped; it
// never takes its round with it.
//
// s.uuid and ro.uuid tiebreak s.created_at and ro.created_at for the same reason t.uuid
// tiebreaks t.finished_at below: the assembly groups scenes and rounds by "does the UUID
// still match the one being built" (curScene.UUID != sceneUUID, curRound.UUID != roundUUID),
// and two scenes or two rounds tied on their timestamp could otherwise interleave and corrupt
// that grouping — the identical defect the turn level already paid to fix.
//
// The master's own actions ARE persisted now, in a table of their own (master_actions, spec
// §4.8), and so are the round's regime changes (match_events) — but neither is read here. Both
// come from separate queries the use case stitches into this tree by round and turn UUID
// (GetMatchHistoryUC), so this join does not multiply by them. What the master's EDITS
// displaced lives in overridden_action_values, which is not part of this read either: an edit
// is not a master action, and the history shows the edited action, which IS the action.
func (r *Repository) FindMatchHistory(
	ctx context.Context, matchUUID uuid.UUID,
) ([]appmatch.HistoryScene, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT s.uuid, s.category, s.brief_initial_description, s.created_at, s.finished_at,
		        ro.uuid, ro.mode, ro.created_at, ro.finished_at,
		        t.uuid, t.created_at, t.finished_at, t.resolution,
		        a.uuid, a.actor_uuid, a.react_to_uuid, a.target_ids, a.type, a.reaction_kind,
		        a.speed, a.skills, a.move, a.attack, a.defense, a.dodge, a.repel, a.feint,
		        a.trigger, a.interact, a.system_bias, a.move_views, a.consumed_action_ids
		 FROM scenes s
		 LEFT JOIN rounds  ro ON ro.scene_uuid = s.uuid
		 LEFT JOIN turns   t  ON t.round_uuid = ro.uuid
		 LEFT JOIN actions a  ON a.turn_uuid = t.uuid
		 WHERE s.match_uuid = $1
		 ORDER BY s.created_at, s.uuid, ro.created_at, ro.uuid, t.finished_at, t.uuid,
		          (a.react_to_uuid IS NOT NULL), a.created_at`,
		matchUUID,
	)
	if err != nil {
		return nil, fmt.Errorf("FindMatchHistory: %w", err)
	}
	defer rows.Close()

	// Never nil: a match with no scene is an empty slice, so it marshals as [] on the wire
	// (Task 12), not null.
	scenes := make([]appmatch.HistoryScene, 0)
	var curScene *appmatch.HistoryScene
	var curRound *appmatch.HistoryRound
	var curTurn *appmatch.HistoryTurn

	// turnPoisoned/poisonedTurnUUID track a turn whose action or a reaction failed to decode
	// and was dropped mid-stream (see the decode-failure handling below). Once a turn is
	// poisoned, every remaining row that still carries its turnUUID must be silently absorbed
	// — not re-attempted as if it were a fresh turn — until a row for a DIFFERENT turn arrives.
	var turnPoisoned bool
	var poisonedTurnUUID uuid.UUID

	for rows.Next() {
		// Everything past the scene is a pointer (or a nil-able slice): the LEFT joins hand a
		// NULL to every level that does not exist for this row.
		var (
			sceneUUID       uuid.UUID
			category        string
			briefDesc       string
			sceneCreatedAt  time.Time
			sceneFinishedAt *time.Time

			roundUUID       *uuid.UUID
			mode            *string
			roundCreatedAt  *time.Time
			roundFinishedAt *time.Time

			turnUUID       *uuid.UUID
			turnCreatedAt  *time.Time
			turnFinishedAt *time.Time
			resolutionRaw  []byte

			actionUUID  *uuid.UUID
			actorUUID   *uuid.UUID
			reactToUUID *uuid.UUID
			targetIDs   []uuid.UUID
			// actionType is scanned only for symmetry with insertAction's INSERT column list
			// (persist_turn_close.go) and to keep this query's column list self-documenting
			// against the table — deriveActionType is a write-time classification with no
			// field on the domain Action to land in, so it is read and discarded here.
			actionType   *string
			reactionKind *string

			speedRaw, skillsRaw, moveRaw, attackRaw []byte
			defenseRaw, dodgeRaw, repelRaw          []byte
			feintRaw, triggerRaw, interactRaw       []byte
			systemBias                              *int
			// moveViewsRaw is the turn's action's verdicts, or — on a reaction row — what each
			// player saw of an opened reaction's destination (NULL when it was never opened).
			moveViewsRaw []byte
			consumed     []uuid.UUID
		)

		if err := rows.Scan(
			&sceneUUID, &category, &briefDesc, &sceneCreatedAt, &sceneFinishedAt,
			&roundUUID, &mode, &roundCreatedAt, &roundFinishedAt,
			&turnUUID, &turnCreatedAt, &turnFinishedAt, &resolutionRaw,
			&actionUUID, &actorUUID, &reactToUUID, &targetIDs, &actionType, &reactionKind,
			&speedRaw, &skillsRaw, &moveRaw, &attackRaw, &defenseRaw, &dodgeRaw, &repelRaw,
			&feintRaw, &triggerRaw, &interactRaw, &systemBias, &moveViewsRaw, &consumed,
		); err != nil {
			return nil, fmt.Errorf("FindMatchHistory scan: %w", err)
		}

		if curScene == nil || curScene.UUID != sceneUUID {
			scenes = append(scenes, appmatch.HistoryScene{
				UUID: sceneUUID, Category: category, BriefDesc: briefDesc,
				CreatedAt: sceneCreatedAt, FinishedAt: sceneFinishedAt,
				Rounds: make([]appmatch.HistoryRound, 0),
			})
			curScene = &scenes[len(scenes)-1]
			curRound = nil
			curTurn = nil
			turnPoisoned = false
		}

		// A scene with no round: the scene row is all there is.
		if roundUUID == nil {
			continue
		}

		if curRound == nil || curRound.UUID != *roundUUID {
			curScene.Rounds = append(curScene.Rounds, appmatch.HistoryRound{
				UUID: *roundUUID, Mode: *mode,
				CreatedAt: *roundCreatedAt, FinishedAt: roundFinishedAt,
				Turns: make([]appmatch.HistoryTurn, 0),
			})
			curRound = &curScene.Rounds[len(curScene.Rounds)-1]
			curTurn = nil
			turnPoisoned = false
		}

		// A round with no turn — born, and maybe closed, without one closing inside it.
		if turnUUID == nil {
			continue
		}

		// A turn already dropped for an unreadable row: absorb the rest of its rows without
		// retrying them — see the doc on turnPoisoned above.
		if turnPoisoned && *turnUUID == poisonedTurnUUID {
			continue
		}

		// A turn row with no action row. PersistTurnClose writes the two in one transaction, so
		// only drift produces this; the turn cannot be shown without the action that drove it,
		// and it must not take its round down with it.
		if actionUUID == nil {
			log.Printf(
				"FindMatchHistory: dropping turn %s (scene %s, round %s) — it has no action row",
				*turnUUID, sceneUUID, *roundUUID,
			)
			turnPoisoned, poisonedTurnUUID = true, *turnUUID
			curTurn = nil
			continue
		}

		row := actionRow{
			actionUUID: *actionUUID, actorUUID: derefUUID(actorUUID), reactToUUID: reactToUUID,
			targetIDs: targetIDs, reactionKind: reactionKind,
			speedRaw: speedRaw, skillsRaw: skillsRaw, moveRaw: moveRaw, attackRaw: attackRaw,
			defenseRaw: defenseRaw, dodgeRaw: dodgeRaw, repelRaw: repelRaw,
			feintRaw: feintRaw, triggerRaw: triggerRaw, interactRaw: interactRaw,
			consumed: consumed,
		}
		if systemBias != nil {
			row.systemBias = *systemBias
		}

		switch {
		case curTurn != nil && curTurn.UUID == *turnUUID:
			// A reaction row for the turn currently being assembled.
			react, err := row.decode()
			if err != nil {
				// Contain the damage to this ONE turn, the same trade-off DecodeResolution
				// already makes for a stored collision: a history with one logged hole is
				// worse than a complete one and far better than none at all. A turn already
				// appended but missing a reaction that answered it would silently under-report
				// — so the whole turn, not just this row, comes back out.
				log.Printf(
					"FindMatchHistory: dropping turn %s (scene %s, round %s) — reaction %s failed to decode: %v",
					*turnUUID, sceneUUID, *roundUUID, *actionUUID, err,
				)
				curRound.Turns = curRound.Turns[:len(curRound.Turns)-1]
				curTurn = nil
				turnPoisoned, poisonedTurnUUID = true, *turnUUID
				continue
			}
			curTurn.Reactions = append(curTurn.Reactions, *react)
			if views := decodeMoveViews(moveViewsRaw, *turnUUID, *actionUUID); views != nil {
				if curTurn.ReactionMoveViews == nil {
					curTurn.ReactionMoveViews = map[uuid.UUID]map[uuid.UUID]masteraction.View{}
				}
				curTurn.ReactionMoveViews[react.GetID()] = views
			}

		default:
			// The ORDER BY's t.uuid tiebreaker (ahead of the react-to-uuid boolean) keeps a
			// turn's rows contiguous even when two turns in the same round share a
			// finished_at — which insertAction makes the norm, not a corner case: it writes
			// the SAME timestamp as both created_at and finished_at for a turn's action AND
			// every one of its reactions. Without t.uuid, two turns tied on finished_at
			// interleave, and this branch (reached whenever the running turn changes) treats a
			// REACTION row from the other turn as if it were this row's own action.
			//
			// That is no longer a semantic argument: strike t.uuid from the ORDER BY above and
			// TestFindMatchHistoryKeepsEachTiedTurnsReactionWithItsOwnTurn reports four turns
			// where there are two. It takes BOTH tied turns carrying a reaction to reproduce —
			// with only one, the remaining sort keys still happen to produce a workable order,
			// which is why the earlier tie test passed either way.
			act, err := row.decode()
			if err != nil {
				// See the reaction-row branch above for why this is contained to the turn
				// rather than propagated: one bad row must not take the whole match's history
				// offline, the same trade-off DecodeResolution makes.
				log.Printf(
					"FindMatchHistory: dropping turn %s (scene %s, round %s) — action %s failed to decode: %v",
					*turnUUID, sceneUUID, *roundUUID, *actionUUID, err,
				)
				turnPoisoned, poisonedTurnUUID = true, *turnUUID
				curTurn = nil
				continue
			}
			// What each player saw of the move live. An unreadable value is dropped, not the
			// turn: nil fails closed in the use case (category only), which is what a row from
			// before the column reads as anyway.
			moveViews := decodeMoveViews(moveViewsRaw, *turnUUID, *actionUUID)
			// turns.created_at and turns.finished_at are NOT NULL; the pointers only exist
			// because the LEFT join could have handed NULL for a turn that is not there.
			curRound.Turns = append(curRound.Turns, appmatch.HistoryTurn{
				UUID: *turnUUID, CreatedAt: derefTime(turnCreatedAt), FinishedAt: derefTime(turnFinishedAt),
				Action:       *act,
				Resolution:   DecodeResolution(resolutionRaw),
				MoveViews:    moveViews,
				LandingViews: decodeLandingViews(resolutionRaw),
			})
			curTurn = &curRound.Turns[len(curRound.Turns)-1]
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("FindMatchHistory rows: %w", err)
	}
	return scenes, nil
}

// actionRow is one actions row as FindMatchHistory scanned it, bundled so the two branches
// that decode one (the turn's action, a reaction) do not repeat sixteen arguments each.
type actionRow struct {
	actionUUID, actorUUID                   uuid.UUID
	reactToUUID                             *uuid.UUID
	targetIDs                               []uuid.UUID
	reactionKind                            *string
	speedRaw, skillsRaw, moveRaw, attackRaw []byte
	defenseRaw, dodgeRaw, repelRaw          []byte
	feintRaw, triggerRaw, interactRaw       []byte
	systemBias                              int
	consumed                                []uuid.UUID
}

// decodeMoveViews reads actions.move_views of the turn's action or of a reaction. An unreadable
// value is logged and dropped, not the turn: nil fails closed in the use case, which is what a
// row from before the column reads as anyway.
func decodeMoveViews(raw []byte, turnUUID, actionUUID uuid.UUID) map[uuid.UUID]masteraction.View {
	if len(raw) == 0 {
		return nil
	}
	var views map[uuid.UUID]masteraction.View
	if err := json.Unmarshal(raw, &views); err != nil {
		log.Printf("FindMatchHistory: turn %s — move_views of action %s failed to decode, shown to nobody "+
			"but the master and the owner: %v", turnUUID, actionUUID, err)
		return nil
	}
	return views
}

func (a actionRow) decode() (*action.Action, error) {
	act, err := decodeActionRow(
		a.actionUUID, a.actorUUID, a.reactToUUID, a.targetIDs, a.reactionKind,
		a.speedRaw, a.skillsRaw, a.moveRaw, a.attackRaw, a.defenseRaw, a.dodgeRaw, a.repelRaw,
		a.feintRaw, a.triggerRaw, a.interactRaw, a.systemBias,
	)
	if err != nil {
		return nil, err
	}
	// NULL scans as nil: "nothing consumed" is nil here the way consumedOrNil wrote it.
	act.ConsumedActionIDs = a.consumed
	return act, nil
}

func derefUUID(p *uuid.UUID) uuid.UUID {
	if p == nil {
		return uuid.Nil
	}
	return *p
}

func derefTime(p *time.Time) time.Time {
	if p == nil {
		return time.Time{}
	}
	return *p
}

// decodeActionRow rebuilds one actions row — the turn's own action or one of its reactions,
// the same table and the same shape — back into an action.Action. It is the read-side mirror
// of insertAction in persist_turn_close.go: one unmarshal per JSONB component, in the same
// order they were written.
func decodeActionRow(
	actionUUID, actorUUID uuid.UUID, reactToUUID *uuid.UUID, targetIDs []uuid.UUID,
	reactionKind *string,
	speedRaw, skillsRaw, moveRaw, attackRaw, defenseRaw, dodgeRaw, repelRaw []byte,
	feintRaw, triggerRaw, interactRaw []byte,
	systemBias int,
) (*action.Action, error) {
	var speed action.ActionSpeed
	if len(speedRaw) > 0 {
		if err := json.Unmarshal(speedRaw, &speed); err != nil {
			return nil, fmt.Errorf("unmarshal speed: %w", err)
		}
	}

	var skills []action.Skill
	if len(skillsRaw) > 0 {
		if err := json.Unmarshal(skillsRaw, &skills); err != nil {
			return nil, fmt.Errorf("unmarshal skills: %w", err)
		}
	}

	move, err := unmarshalNullablePtr[action.Move](moveRaw)
	if err != nil {
		return nil, fmt.Errorf("unmarshal move: %w", err)
	}
	attack, err := unmarshalNullablePtr[action.Attack](attackRaw)
	if err != nil {
		return nil, fmt.Errorf("unmarshal attack: %w", err)
	}
	defense, err := unmarshalNullablePtr[action.Defense](defenseRaw)
	if err != nil {
		return nil, fmt.Errorf("unmarshal defense: %w", err)
	}
	dodge, err := unmarshalNullablePtr[action.Dodge](dodgeRaw)
	if err != nil {
		return nil, fmt.Errorf("unmarshal dodge: %w", err)
	}
	repel, err := unmarshalNullablePtr[action.Repel](repelRaw)
	if err != nil {
		return nil, fmt.Errorf("unmarshal repel: %w", err)
	}
	feint, err := unmarshalNullablePtr[action.RollCheck](feintRaw)
	if err != nil {
		return nil, fmt.Errorf("unmarshal feint: %w", err)
	}
	trigger, err := unmarshalNullablePtr[action.Trigger](triggerRaw)
	if err != nil {
		return nil, fmt.Errorf("unmarshal trigger: %w", err)
	}
	interact, err := unmarshalNullablePtr[action.Interact](interactRaw)
	if err != nil {
		return nil, fmt.Errorf("unmarshal interact: %w", err)
	}

	reactTo := uuid.Nil
	if reactToUUID != nil {
		reactTo = *reactToUUID
	}

	// WithReconstructedID overrides the id NewAction would otherwise mint. actions.uuid is
	// exactly what react_to_uuid on this action's own reactions points at, and what
	// action_uuid in overridden_action_values keys on — a fabricated id here would make the
	// tree uncorrelatable with both. See the option's own doc in action.go for why this is a
	// constructor-time option and not a mutating setter.
	act := action.NewAction(
		actorUUID, targetIDs, reactTo, skills, speed,
		feint, move, attack, defense, dodge, trigger, interact,
		action.WithReconstructedID(actionUUID),
	)
	act.Repel = repel
	// SystemBias is set after construction for the same reason Repel is: NewAction already
	// takes twelve positional parameters, and the field's own doc says growing it buys
	// nothing. See action.Action.SystemBias for why a stored 0 and a re-derived 0 are not the
	// same claim — reading it back is what keeps the history's version honest.
	act.SystemBias = systemBias
	if reactionKind != nil {
		act.ReactionKind = action.ReactionKind(*reactionKind)
	}
	return act, nil
}

// unmarshalNullablePtr is the read-side mirror of marshalNullablePtr in persist_turn_close.go:
// nil for a NULL/empty JSONB column, else the unmarshaled value.
func unmarshalNullablePtr[T any](raw []byte) (*T, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	return &v, nil
}
