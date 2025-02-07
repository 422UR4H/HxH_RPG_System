package item

import "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"

type WeaponsManager struct {
	weapons map[enum.WeaponName]Weapon
}

func NewWeaponsManager(weapons map[enum.WeaponName]Weapon) *WeaponsManager {
	return &WeaponsManager{weapons: weapons}
}
