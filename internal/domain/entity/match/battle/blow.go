package battle

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match/action"
	"github.com/google/uuid"
)

type Blow struct {
	actorID       uuid.UUID
	targetID      uuid.UUID
	attack        action.Attack
	attackSkills  *action.Skill
	defense       action.Defense
	defenseSkills *action.Skill
}
