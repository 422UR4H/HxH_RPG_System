package action

import "github.com/google/uuid"

type Action struct {
	id        uuid.UUID
	actorID   uuid.UUID
	TargetID  []uuid.UUID
	ReactToID uuid.UUID // maybe change to *Action
	Skills    []Skill
	Speed     RollCheck
	Feint     *RollCheck
	Move      *Move
	Attack    *Attack
	Defense   *Defense
	Dodge     *Dodge
}

func NewAction(
	actorID uuid.UUID,
	targetID []uuid.UUID,
	reactToID uuid.UUID,
	skills []Skill,
	actionSpeed RollCheck,
	feint *RollCheck,
	move *Move,
	attack *Attack,
	defense *Defense,
	dodge *Dodge,
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
	}
}

func (a *Action) GetID() uuid.UUID {
	return a.id
}

func (a *Action) GetActorID() uuid.UUID {
	return a.actorID
}
