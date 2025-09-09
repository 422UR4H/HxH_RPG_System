package action

import "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"

type Defense struct {
	Weapon *enum.WeaponName
	Roll   *RollCondition
}
