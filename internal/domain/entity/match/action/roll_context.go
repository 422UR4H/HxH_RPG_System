package action

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/die"
)

type RollContext struct {
	Dice      []die.Die
	Condition *RollCondition
}

func (rc *RollContext) GetResultSum(d die.Die) int {
	sum := 0
	for _, die := range rc.Dice {
		sum += die.GetResult()
	}
	return sum
}
