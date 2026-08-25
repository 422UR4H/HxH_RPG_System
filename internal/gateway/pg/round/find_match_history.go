package round

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/google/uuid"
)

// FindMatchHistory returns the match's closed turns as the TREE the domain already is —
// Scene -> Round -> Turn -> Action — not a flat list.
//
// The hierarchy is not decoration: the front renders action cards INSIDE the scope of each
// scene, because scenes are the logical blocks the match is organised into. Flattening here
// would push the regrouping onto every consumer.
//
// One query, ordered, assembled in one pass. Reactions come back in the same result set,
// discriminated by react_to_uuid IS NOT NULL, so there is no N+1 over turns.
//
// The master's own actions (Turn.GetMasterActions()) are NOT read here: actions has no column
// for them — they are deliberately never persisted (spec § A edição do mestre). What the
// master's edits displaced lives in overridden_action_values, and that table is not part of
// this read either; the history shows the edited action, which IS the action.
func (r *Repository) FindMatchHistory(
	ctx context.Context, matchUUID uuid.UUID,
) ([]appmatch.HistoryScene, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT s.uuid, s.category, s.brief_initial_description, s.created_at, s.finished_at,
		        ro.uuid, ro.mode, ro.created_at, ro.finished_at,
		        t.uuid, t.created_at, t.finished_at, t.resolution,
		        a.uuid, a.actor_uuid, a.react_to_uuid, a.target_ids, a.type, a.reaction_kind,
		        a.speed, a.skills, a.move, a.attack, a.defense, a.dodge, a.repel, a.feint, a.trigger
		 FROM scenes s
		 JOIN rounds ro ON ro.scene_uuid = s.uuid
		 JOIN turns  t  ON t.round_uuid = ro.uuid
		 JOIN actions a ON a.turn_uuid = t.uuid
		 WHERE s.match_uuid = $1
		 ORDER BY s.created_at, ro.created_at, t.finished_at,
		          (a.react_to_uuid IS NOT NULL), a.created_at`,
		matchUUID,
	)
	if err != nil {
		return nil, fmt.Errorf("FindMatchHistory: %w", err)
	}
	defer rows.Close()

	// Never nil: a match with no closed turns is an empty slice, so it marshals as [] on the
	// wire (Task 12), not null.
	scenes := make([]appmatch.HistoryScene, 0)
	var curScene *appmatch.HistoryScene
	var curRound *appmatch.HistoryRound
	var curTurn *appmatch.HistoryTurn

	for rows.Next() {
		var (
			sceneUUID       uuid.UUID
			category        string
			briefDesc       string
			sceneCreatedAt  time.Time
			sceneFinishedAt *time.Time

			roundUUID       uuid.UUID
			mode            string
			roundCreatedAt  time.Time
			roundFinishedAt *time.Time

			turnUUID       uuid.UUID
			turnCreatedAt  time.Time
			turnFinishedAt time.Time
			resolutionRaw  []byte

			actionUUID   uuid.UUID
			actorUUID    uuid.UUID
			reactToUUID  *uuid.UUID
			targetIDs    []uuid.UUID
			actionType   string // derived at write time; not part of the domain Action, unused here
			reactionKind *string

			speedRaw, skillsRaw, moveRaw, attackRaw []byte
			defenseRaw, dodgeRaw, repelRaw          []byte
			feintRaw, triggerRaw                    []byte
		)

		if err := rows.Scan(
			&sceneUUID, &category, &briefDesc, &sceneCreatedAt, &sceneFinishedAt,
			&roundUUID, &mode, &roundCreatedAt, &roundFinishedAt,
			&turnUUID, &turnCreatedAt, &turnFinishedAt, &resolutionRaw,
			&actionUUID, &actorUUID, &reactToUUID, &targetIDs, &actionType, &reactionKind,
			&speedRaw, &skillsRaw, &moveRaw, &attackRaw, &defenseRaw, &dodgeRaw, &repelRaw,
			&feintRaw, &triggerRaw,
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
		}

		if curRound == nil || curRound.UUID != roundUUID {
			curScene.Rounds = append(curScene.Rounds, appmatch.HistoryRound{
				UUID: roundUUID, Mode: mode,
				CreatedAt: roundCreatedAt, FinishedAt: roundFinishedAt,
				Turns: make([]appmatch.HistoryTurn, 0),
			})
			curRound = &curScene.Rounds[len(curScene.Rounds)-1]
			curTurn = nil
		}

		act, err := decodeActionRow(
			actorUUID, reactToUUID, targetIDs, reactionKind,
			speedRaw, skillsRaw, moveRaw, attackRaw, defenseRaw, dodgeRaw, repelRaw,
			feintRaw, triggerRaw,
		)
		if err != nil {
			return nil, fmt.Errorf("FindMatchHistory decode action %s: %w", actionUUID, err)
		}

		// The ORDER BY puts a turn's own action (react_to_uuid IS NULL) before its reactions,
		// so the first row of a new turn is always the action — no second sort needed here.
		if curTurn == nil || curTurn.UUID != turnUUID {
			curRound.Turns = append(curRound.Turns, appmatch.HistoryTurn{
				UUID: turnUUID, CreatedAt: turnCreatedAt, FinishedAt: turnFinishedAt,
				Action:     *act,
				Resolution: DecodeResolution(resolutionRaw),
			})
			curTurn = &curRound.Turns[len(curRound.Turns)-1]
		} else {
			curTurn.Reactions = append(curTurn.Reactions, *act)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("FindMatchHistory rows: %w", err)
	}
	return scenes, nil
}

// decodeActionRow rebuilds one actions row — the turn's own action or one of its reactions,
// the same table and the same shape — back into an action.Action. It is the read-side mirror
// of insertAction in persist_turn_close.go: one unmarshal per JSONB component, in the same
// order they were written.
func decodeActionRow(
	actorUUID uuid.UUID, reactToUUID *uuid.UUID, targetIDs []uuid.UUID, reactionKind *string,
	speedRaw, skillsRaw, moveRaw, attackRaw, defenseRaw, dodgeRaw, repelRaw []byte,
	feintRaw, triggerRaw []byte,
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

	reactTo := uuid.Nil
	if reactToUUID != nil {
		reactTo = *reactToUUID
	}

	// NewAction always mints a fresh id (it has no parameter for it, and the entity exposes
	// no setter) — the row's own a.uuid is scanned above only to drive the assembly loop, not
	// to round-trip identity. actorID, by contrast, IS a constructor parameter, so it survives.
	act := action.NewAction(
		actorUUID, targetIDs, reactTo, skills, speed,
		feint, move, attack, defense, dodge, trigger, nil,
	)
	act.Repel = repel
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
