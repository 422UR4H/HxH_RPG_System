// Package actioncodec is the one storage mapping of an action.Action: the JSON pieces the
// actions table keeps one column each (insertAction writes them, FindMatchHistory reads them
// back), and the same pieces as ONE JSON document, for a table that keeps the whole action in
// a single JSONB column (cancelled_actions.action). Both shapes go through Columns, so the two
// tables can never disagree on how a component is stored.
//
// It lives apart from the round gateway because both round (actions) and pgcancelledaction
// (cancelled_actions) need it, and round already imports pgcancelledaction to write a turn's
// cancellations in its own transaction.
package actioncodec

import (
	"encoding/json"
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/google/uuid"
)

// Columns is one action's JSON pieces, one per JSONB column of actions. nil is SQL NULL —
// "the action has no such component" — except Speed, which every action has.
type Columns struct {
	Speed, Skills, Move, Attack, Defense, Dodge, Repel []byte
	Feint, Trigger, Interact, DefaultDefense           []byte
}

// EncodeColumns marshals each component of act the way the actions table stores it.
func EncodeColumns(act *action.Action) (Columns, error) {
	var c Columns
	var err error
	if c.Speed, err = json.Marshal(act.Speed); err != nil {
		return Columns{}, fmt.Errorf("marshal speed: %w", err)
	}
	if c.Skills, err = marshalNullableSlice(act.Skills); err != nil {
		return Columns{}, fmt.Errorf("marshal skills: %w", err)
	}
	if c.Move, err = marshalNullablePtr(act.Move); err != nil {
		return Columns{}, fmt.Errorf("marshal move: %w", err)
	}
	if c.Attack, err = marshalNullablePtr(act.Attack); err != nil {
		return Columns{}, fmt.Errorf("marshal attack: %w", err)
	}
	if c.Defense, err = marshalNullablePtr(act.Defense); err != nil {
		return Columns{}, fmt.Errorf("marshal defense: %w", err)
	}
	if c.Dodge, err = marshalNullablePtr(act.Dodge); err != nil {
		return Columns{}, fmt.Errorf("marshal dodge: %w", err)
	}
	if c.Repel, err = marshalNullablePtr(act.Repel); err != nil {
		return Columns{}, fmt.Errorf("marshal repel: %w", err)
	}
	if c.Feint, err = marshalNullablePtr(act.Feint); err != nil {
		return Columns{}, fmt.Errorf("marshal feint: %w", err)
	}
	if c.Trigger, err = marshalNullablePtr(act.Trigger); err != nil {
		return Columns{}, fmt.Errorf("marshal trigger: %w", err)
	}
	if c.Interact, err = marshalNullablePtr(act.Interact); err != nil {
		return Columns{}, fmt.Errorf("marshal interact: %w", err)
	}
	if c.DefaultDefense, err = marshalNullablePtr(act.DefaultDefense); err != nil {
		return Columns{}, fmt.Errorf("marshal default defense: %w", err)
	}
	return c, nil
}

// Row is everything one stored action is rebuilt from: its identity, the scalar columns and
// the JSON pieces.
type Row struct {
	ID, ActorID       uuid.UUID
	ReactToID         *uuid.UUID
	TargetIDs         []uuid.UUID
	ReactionKind      *string
	SystemBias        int
	ConsumedActionIDs []uuid.UUID
	Columns
}

// DecodeRow rebuilds an action.Action from its stored pieces — the read-side mirror of
// EncodeColumns: one unmarshal per JSON piece, in the same order they were written.
func DecodeRow(r Row) (*action.Action, error) {
	var speed action.ActionSpeed
	if len(r.Speed) > 0 {
		if err := json.Unmarshal(r.Speed, &speed); err != nil {
			return nil, fmt.Errorf("unmarshal speed: %w", err)
		}
	}

	var skills []action.Skill
	if len(r.Skills) > 0 {
		if err := json.Unmarshal(r.Skills, &skills); err != nil {
			return nil, fmt.Errorf("unmarshal skills: %w", err)
		}
	}

	move, err := unmarshalNullablePtr[action.Move](r.Move)
	if err != nil {
		return nil, fmt.Errorf("unmarshal move: %w", err)
	}
	attack, err := unmarshalNullablePtr[action.Attack](r.Attack)
	if err != nil {
		return nil, fmt.Errorf("unmarshal attack: %w", err)
	}
	defense, err := unmarshalNullablePtr[action.Defense](r.Defense)
	if err != nil {
		return nil, fmt.Errorf("unmarshal defense: %w", err)
	}
	dodge, err := unmarshalNullablePtr[action.Dodge](r.Dodge)
	if err != nil {
		return nil, fmt.Errorf("unmarshal dodge: %w", err)
	}
	repel, err := unmarshalNullablePtr[action.Repel](r.Repel)
	if err != nil {
		return nil, fmt.Errorf("unmarshal repel: %w", err)
	}
	feint, err := unmarshalNullablePtr[action.RollCheck](r.Feint)
	if err != nil {
		return nil, fmt.Errorf("unmarshal feint: %w", err)
	}
	trigger, err := unmarshalNullablePtr[action.Trigger](r.Trigger)
	if err != nil {
		return nil, fmt.Errorf("unmarshal trigger: %w", err)
	}
	interact, err := unmarshalNullablePtr[action.Interact](r.Interact)
	if err != nil {
		return nil, fmt.Errorf("unmarshal interact: %w", err)
	}
	// The history's read of actions does not select default_defense (it is off every wire), so
	// there it is always nil; cancelled_actions keeps it, and it comes back whole.
	defaultDefense, err := unmarshalNullablePtr[action.RollCheck](r.DefaultDefense)
	if err != nil {
		return nil, fmt.Errorf("unmarshal default defense: %w", err)
	}

	reactTo := uuid.Nil
	if r.ReactToID != nil {
		reactTo = *r.ReactToID
	}

	// WithReconstructedID overrides the id NewAction would otherwise mint. actions.uuid is
	// exactly what react_to_uuid on this action's own reactions points at, and what
	// action_uuid in overridden_action_values keys on — a fabricated id here would make the
	// tree uncorrelatable with both. See the option's own doc in action.go for why this is a
	// constructor-time option and not a mutating setter.
	act := action.NewAction(
		r.ActorID, r.TargetIDs, reactTo, skills, speed,
		feint, move, attack, defense, dodge, trigger, interact,
		action.WithReconstructedID(r.ID),
	)
	act.Repel = repel
	act.DefaultDefense = defaultDefense
	// SystemBias is set after construction for the same reason Repel is: NewAction already
	// takes twelve positional parameters, and the field's own doc says growing it buys
	// nothing. See action.Action.SystemBias for why a stored 0 and a re-derived 0 are not the
	// same claim — reading it back is what keeps the history's version honest.
	act.SystemBias = r.SystemBias
	if r.ReactionKind != nil {
		act.ReactionKind = action.ReactionKind(*r.ReactionKind)
	}
	// NULL scans as nil: "nothing consumed" is nil here the way it was written.
	act.ConsumedActionIDs = r.ConsumedActionIDs
	return act, nil
}

// document is the whole action as ONE JSON object (cancelled_actions.action). Its keys are the
// actions table's column names, and each piece is exactly that column's value: the migration
// that creates cancelled_actions documents them. The action's own id and actor are columns of
// the row that holds it, not keys here. An absent piece is an absent key.
type document struct {
	Speed             json.RawMessage `json:"speed"`
	Skills            json.RawMessage `json:"skills,omitempty"`
	Move              json.RawMessage `json:"move,omitempty"`
	Attack            json.RawMessage `json:"attack,omitempty"`
	Defense           json.RawMessage `json:"defense,omitempty"`
	Dodge             json.RawMessage `json:"dodge,omitempty"`
	Repel             json.RawMessage `json:"repel,omitempty"`
	Feint             json.RawMessage `json:"feint,omitempty"`
	Trigger           json.RawMessage `json:"trigger,omitempty"`
	Interact          json.RawMessage `json:"interact,omitempty"`
	DefaultDefense    json.RawMessage `json:"default_defense,omitempty"`
	TargetIDs         []uuid.UUID     `json:"target_ids"`
	ReactToUUID       *uuid.UUID      `json:"react_to_uuid,omitempty"`
	ReactionKind      *string         `json:"reaction_kind,omitempty"`
	SystemBias        int             `json:"system_bias"`
	ConsumedActionIDs []uuid.UUID     `json:"consumed_action_ids,omitempty"`
}

// Marshal is act as one JSON document with the same pieces EncodeColumns writes to actions.
func Marshal(act *action.Action) ([]byte, error) {
	c, err := EncodeColumns(act)
	if err != nil {
		return nil, err
	}
	doc := document{
		Speed: c.Speed, Skills: c.Skills, Move: c.Move, Attack: c.Attack, Defense: c.Defense,
		Dodge: c.Dodge, Repel: c.Repel, Feint: c.Feint, Trigger: c.Trigger, Interact: c.Interact,
		DefaultDefense: c.DefaultDefense,
		// Never null, as in actions.target_ids.
		TargetIDs:         act.TargetID,
		SystemBias:        act.SystemBias,
		ConsumedActionIDs: act.ConsumedActionIDs,
	}
	if doc.TargetIDs == nil {
		doc.TargetIDs = []uuid.UUID{}
	}
	if act.ReactToID != uuid.Nil {
		v := act.ReactToID
		doc.ReactToUUID = &v
	}
	if act.ReactionKind != "" {
		v := string(act.ReactionKind)
		doc.ReactionKind = &v
	}
	return json.Marshal(doc)
}

// Unmarshal rebuilds the action Marshal wrote; id and actorID come from the row holding it.
func Unmarshal(id, actorID uuid.UUID, raw []byte) (*action.Action, error) {
	var doc document
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("unmarshal action document: %w", err)
	}
	return DecodeRow(Row{
		ID: id, ActorID: actorID, ReactToID: doc.ReactToUUID, TargetIDs: doc.TargetIDs,
		ReactionKind: doc.ReactionKind, SystemBias: doc.SystemBias,
		ConsumedActionIDs: doc.ConsumedActionIDs,
		Columns: Columns{
			Speed: doc.Speed, Skills: doc.Skills, Move: doc.Move, Attack: doc.Attack,
			Defense: doc.Defense, Dodge: doc.Dodge, Repel: doc.Repel, Feint: doc.Feint,
			Trigger: doc.Trigger, Interact: doc.Interact, DefaultDefense: doc.DefaultDefense,
		},
	})
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

// unmarshalNullablePtr is the read-side mirror of marshalNullablePtr: nil for a NULL/empty
// JSONB column, else the unmarshaled value.
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
