package battle

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match/action"
	"github.com/google/uuid"
)

type Blow struct {
	actorID       uuid.UUID     //nolint:unused // WIP: match system under development
	targetID      uuid.UUID     //nolint:unused // WIP: match system under development
	attack        action.Attack //nolint:unused // WIP: match system under development
	attackSkills  *action.Skill //nolint:unused // WIP: match system under development
	defense       action.Defense //nolint:unused // WIP: match system under development
	defenseSkills *action.Skill  //nolint:unused // WIP: match system under development
}
