package action

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type Skill struct {
	SkillName  enum.SkillName
	Roll       *RollCondition // strategy set dices based on campaign\match rules
	Difficulty *int           // difficulty class (DC -> CD in pt-br)
}
