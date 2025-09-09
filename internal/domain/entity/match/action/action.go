package action

import "github.com/google/uuid"

type Action struct {
	actorID     uuid.UUID
	TargetID    []uuid.UUID // maybe change to *CharacterSheet
	ReactToID   []uuid.UUID // maybe change to *Action
	Skills      []Skill
	ActionSpeed RollContext
	Feint       *RollContext // TODO: change to obj in v0.1
	Move        *Move
	Attack      *Attack
	Defense     *Defense
	Dodge       *Dodge
}

func NewAction(
	actorID uuid.UUID,
	targetID []uuid.UUID,
	reactToID []uuid.UUID,
	skills []Skill,
	actionSpeed RollContext,
	feint *RollContext,
	move *Move,
	attack *Attack,
	defense *Defense,
	dodge *Dodge,
) *Action {
	return &Action{
		actorID:     actorID,
		TargetID:    targetID,
		ReactToID:   reactToID,
		Skills:      skills,
		ActionSpeed: actionSpeed,
		Feint:       feint,
		Move:        move,
		Attack:      attack,
		Defense:     defense,
		Dodge:       dodge,
	}
}

func (a *Action) GetActorId() uuid.UUID {
	return a.actorID
}
