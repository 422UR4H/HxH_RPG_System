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
) *Action {
	return &Action{
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
}

func (a *Action) GetID() uuid.UUID {
	return a.id
}

// ReconstructID stamps a persisted identity onto an action NewAction already built, and
// returns the same pointer so it can be chained at the call site.
//
// It deliberately does NOT mirror scene.ReconstructScene / round.ReconstructRound's shape — a
// full reconstructor duplicating NewAction's twelve parameters just to also accept an id would
// be unreadable, and buys no safety NewAction's own parameters don't already provide. Building
// via NewAction and then stamping the id here is fine and honest: NewAction minting a fresh
// uuid.New() is the RIGHT answer for an action being newly created, which has no identity yet
// to preserve, and the WRONG answer for one being read back from storage, which already does.
//
// Skipping this on a read path is not merely cosmetic. actions.uuid is exactly what
// react_to_uuid on this action's own reactions points at, and what action_uuid in
// overridden_action_values keys on — so a row read back with a fabricated id becomes
// uncorrelatable with everything else in the same result set and in that audit table. A turn
// whose reaction's ReactToID doesn't equal its own action's GetID() is not a cosmetic gap; it
// is a tree that lies about its own shape.
func (a *Action) ReconstructID(id uuid.UUID) *Action {
	a.id = id
	return a
}

func (a *Action) GetActorID() uuid.UUID {
	return a.actorID
}
