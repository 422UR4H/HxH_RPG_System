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
// and a way to re-stamp identity on one of those later would corrupt react_to_uuid linkage and
// scheduler keys out from under them.
//
// Option is SEALED — its only method is apply, and apply is unexported — specifically so that
// holding an Option value outside this package is not enough to invoke it. Per the language
// spec, an unexported method name is only "the same" method across two types if both are
// declared in the same package, so no type defined outside this package can implement Option,
// and no caller outside this package can write `opt.apply(x)` even on an Option it legitimately
// holds (returned by WithReconstructedID) — that expression does not compile there. NewAction
// is the only call site inside this package that invokes apply, and it does so once, before
// returning. That is what makes the guarantee true rather than aspirational: after NewAction
// returns, nothing outside this package can change an Action's id, full stop — not through
// Option, and not through any other exported name, because there is no other exported name
// that touches the id field.
type Option interface {
	apply(*Action)
}

// reconstructedID is the concrete Option WithReconstructedID returns. It exists only to carry
// apply's unexported implementation — see Option's own doc for why that seals the type.
type reconstructedID uuid.UUID

func (id reconstructedID) apply(a *Action) {
	a.id = uuid.UUID(id)
}

// WithReconstructedID overrides the id NewAction would otherwise mint, for the one caller that
// legitimately needs to: a gateway rebuilding an Action from a persisted row. NewAction minting
// a fresh uuid.New() is correct for an action being newly created, which has no identity yet to
// preserve, and wrong for one being read back from storage, which already does — actions.uuid
// is exactly what react_to_uuid on this action's own reactions points at, and what action_uuid
// in overridden_action_values keys on, so a row read back with a fabricated id becomes
// uncorrelatable with everything else in the same result set and in that audit table.
//
// This is a sealed Option and not a thirteenth positional parameter, and not an exported
// setter either: NewAction already takes twelve parameters and growing it buys nothing (see
// ReactionKind's own doc for that same call), and a setter — or an Option whose apply method
// were exported — would reopen exactly the live-object mutation window this shape exists to
// close.
func WithReconstructedID(id uuid.UUID) Option {
	return reconstructedID(id)
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
		opt.apply(a)
	}
	return a
}

func (a *Action) GetID() uuid.UUID {
	return a.id
}

func (a *Action) GetActorID() uuid.UUID {
	return a.actorID
}
