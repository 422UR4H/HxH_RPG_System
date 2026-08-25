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

func (a *Action) GetActorID() uuid.UUID {
	return a.actorID
}
