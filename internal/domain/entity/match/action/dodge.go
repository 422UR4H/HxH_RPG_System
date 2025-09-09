package action

import "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"

type Dodge struct {
	Category enum.DodgeCategory
	Roll     *RollCondition
}
