package action

import (
	"time"

	"github.com/google/uuid"
)

type Action struct {
	id        uuid.UUID
	actorID   uuid.UUID
	TargetID  []uuid.UUID
	ReactToID uuid.UUID // maybe change to *Action

	Speed  ActionSpeed
	Skills []Skill

	Trigger  *Trigger
	Feint    *RollCheck
	Move     *Move
	Attack   *Attack
	Defense  *Defense
	Dodge    *Dodge
	Repel    *Repel
	Interact *Interact

	// ReactionKind is set only on a reaction, and it is what decides the cost — never the
	// shape. It is deliberately a field rather than a constructor parameter: NewAction already
	// takes twelve positional arguments and is called from dozens of sites, and growing it
	// buys nothing. The discriminator for "this is a reaction" is still ReactToID.
	ReactionKind ReactionKind

	// SystemBias is the engine-imposed advantage/disadvantage this action was last derived
	// under — 0 for a plain action, -1 for a reaction that displaced a queued action (see
	// MatchSession.AttachReaction's "swapping what you were going to do costs Disadvantage").
	// It is NOT the master's RollCondition and NOT the character's ModifierLedger; it is a
	// third, engine-owned origin, and unlike those two it is never set by an edit — only ever
	// by MatchSession.deriveSpeeds, which stores here exactly the value it was just asked to
	// apply. It has to be stored, not just passed through: deriveSpeeds runs again every time
	// the master edits an unrelated condition on this same action (ApplyMasterAction
	// re-derives to let a speed/moveSpeed edit read through), and a literal 0 there would
	// silently erase whatever disadvantage this action was actually charged under.
	SystemBias int

	openedAt    *time.Time //nolint:unused // WIP: match system under development
	confirmedAt *time.Time //nolint:unused // WIP: match system under development
}

// Option customizes an Action at construction time only — applied once, inside NewAction,
// before the pointer it built is ever handed to a caller. That is deliberate and load-bearing,
// not incidental: live pointers to an in-flight action really do escape into structures keyed
// on GetID() (MatchSession.PendingActions()'s priority queue, Turn.ActionRef()'s open turn),
// and an exported method that could re-stamp identity on one of those later would corrupt
// react_to_uuid linkage and scheduler keys out from under them. Routing identity through a
// constructor-only option closes that window by construction: after NewAction returns, nothing
// exported in this package can change an Action's id, because there is no such method — only
// this option, which NewAction stops accepting suggestions from the moment it returns.
type Option func(*Action)

// WithReconstructedID overrides the id NewAction would otherwise mint, for the one caller that
// legitimately needs to: a gateway rebuilding an Action from a persisted row. NewAction minting
// a fresh uuid.New() is correct for an action being newly created, which has no identity yet to
// preserve, and wrong for one being read back from storage, which already does — actions.uuid
// is exactly what react_to_uuid on this action's own reactions points at, and what action_uuid
// in overridden_action_values keys on, so a row read back with a fabricated id becomes
// uncorrelatable with everything else in the same result set and in that audit table.
//
// This is an Option and not a thirteenth positional parameter, and not an exported setter
// either: NewAction already takes twelve parameters and growing it buys nothing (see
// ReactionKind's own doc for that same call), and a setter would reopen exactly the live-object
// mutation window this shape exists to close.
func WithReconstructedID(id uuid.UUID) Option {
	return func(a *Action) { a.id = id }
}

func NewAction(
	actorID uuid.UUID,
	targetID []uuid.UUID,
	reactToID uuid.UUID,
	skills []Skill,
	actionSpeed ActionSpeed,
	feint *RollCheck,
	move *Move,
	attack *Attack,
	defense *Defense,
	dodge *Dodge,
	trigger *Trigger,
	interact *Interact,
	opts ...Option,
) *Action {
	a := &Action{
		id:        uuid.New(),
		actorID:   actorID,
		TargetID:  targetID,
		ReactToID: reactToID,
		Skills:    skills,
		Speed:     actionSpeed,
		Feint:     feint,
		Move:      move,
		Attack:    attack,
		Defense:   defense,
		Dodge:     dodge,
		Trigger:   trigger,
		Interact:  interact,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

func (a *Action) GetID() uuid.UUID {
	return a.id
}

func (a *Action) GetActorID() uuid.UUID {
	return a.actorID
}
