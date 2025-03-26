package item

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type WeaponsManager struct {
	weapons map[enum.WeaponName]Weapon
}

func NewWeaponsManager(weapons map[enum.WeaponName]Weapon) *WeaponsManager {
	return &WeaponsManager{weapons: weapons}
}

func (wm *WeaponsManager) Add(name enum.WeaponName, weapon Weapon) {
	wm.weapons[name] = weapon
}

func (wm *WeaponsManager) Delete(name enum.WeaponName) {
	delete(wm.weapons, name)
}

func (wm *WeaponsManager) GetAll() map[enum.WeaponName]Weapon {
	return wm.weapons
}
func (wm *WeaponsManager) Get(name enum.WeaponName) (Weapon, error) {
	weapon, ok := wm.weapons[name]
	if !ok {
		return Weapon{}, ErrWeaponNotFound
	}
	return weapon, nil
}

func (wm *WeaponsManager) GetDamage(name enum.WeaponName) (int, error) {
	weapon, err := wm.Get(name)
	if err != nil {
		return 0, err
	}
	return weapon.GetDamage(), nil
}

func (wm *WeaponsManager) GetDefense(name enum.WeaponName) (int, error) {
	weapon, err := wm.Get(name)
	if err != nil {
		return 0, err
	}
	return weapon.GetDefense(), nil
}

func (wm *WeaponsManager) GetWeight(name enum.WeaponName) (float64, error) {
	weapon, err := wm.Get(name)
	if err != nil {
		return 0, err
	}
	return weapon.GetWeight(), nil
}

func (wm *WeaponsManager) GetHeight(name enum.WeaponName) (float64, error) {
	weapon, err := wm.Get(name)
	if err != nil {
		return 0, err
	}
	return weapon.GetHeight(), nil
}

func (wm *WeaponsManager) GetVolume(name enum.WeaponName) (int, error) {
	weapon, err := wm.Get(name)
	if err != nil {
		return 0, err
	}
	return weapon.GetVolume(), nil
}

func (wm *WeaponsManager) IsFireWeapon(name enum.WeaponName) (bool, error) {
	weapon, err := wm.Get(name)
	if err != nil {
		return false, err
	}
	return weapon.IsFireWeapon(), nil
}

func (wm *WeaponsManager) GetDice(name enum.WeaponName) ([]int, error) {
	weapon, err := wm.Get(name)
	if err != nil {
		return nil, err
	}
	return weapon.GetDice(), nil
}

func (wm *WeaponsManager) GetPenality(name enum.WeaponName) (float64, error) {
	weapon, err := wm.Get(name)
	if err != nil {
		return 0, err
	}
	return weapon.GetPenality(), nil
}

func (wm *WeaponsManager) GetStaminaCost(name enum.WeaponName) (float64, error) {
	weapon, err := wm.Get(name)
	if err != nil {
		return 0, err
	}
	return weapon.GetStaminaCost(), nil
}
