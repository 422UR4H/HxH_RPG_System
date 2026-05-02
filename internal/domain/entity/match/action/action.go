package action

import (
	"time"

	"github.com/google/uuid"
)

// TODO: REFACTOR TO IMPLEMENT I_ACTION INTERFACE
type Action struct {
	id        uuid.UUID
	actorID   uuid.UUID
	TargetID  []uuid.UUID
	ReactToID uuid.UUID // maybe change to *Action

	Speed  ActionSpeed
	Skills []Skill

	Trigger *Trigger
	Feint   *RollCheck
	Move    *Move
	Attack  *Attack
	Defense *Defense
	Dodge   *Dodge

	openedAt    *time.Time  //nolint:unused // WIP: match system under development
	confirmedAt *time.Time  //nolint:unused // WIP: match system under development
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
	}
}

func (a *Action) GetID() uuid.UUID {
	return a.id
}

func (a *Action) GetActorID() uuid.UUID {
	return a.actorID
}
