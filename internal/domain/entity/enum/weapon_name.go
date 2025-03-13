package enum

import "fmt"

type WeaponName int

const (
	Dagger WeaponName = iota
	ThrowingDagger
	Halberd // long
	Bow
	Longbow
	Staff
	Scimitar
	Rapier
	Whip
	Club
	Longclub
	Sword
	Longsword
	Scythe
	Longscythe
	Katana
	Katar
	Spear
	Longspear
	Axe
	Longaxe
	ThrowingAxe
	Hammer
	Warhammer // long
	ThrowingHammer
	Massa
	Mangual
	Longmass
	Pickaxe
	Fist
	Trident
	Tchaco

	Crossbow
	Ak47
	Ar15
	MachineGun
	Pistol38
	Rifle
	Uzi
	Bomb
)

func (sn WeaponName) String() string {
	switch sn {
	case Dagger:
		return "Dagger"
	case ThrowingDagger:
		return "Throwing Dagger"
	case Halberd:
		return "Halberd"
	case Bow:
		return "Bow"
	case Longbow:
		return "Longbow"
	case Staff:
		return "Staff"
	case Scimitar:
		return "Scimitar"
	case Whip:
		return "Whip"
	case Club:
		return "Club"
	case Longclub:
		return "Longclub"
	case Sword:
		return "Sword"
	case Longsword:
		return "Longsword"
	case Scythe:
		return "Scythe"
	case Longscythe:
		return "Longscythe"
	case Katana:
		return "Katana"
	case Katar:
		return "Katar"
	case Spear:
		return "Spear"
	case Longspear:
		return "Longspear"
	case Axe:
		return "Axe"
	case Longaxe:
		return "Longaxe"
	case ThrowingAxe:
		return "Throwing Axe"
	case Hammer:
		return "Hammer"
	case Warhammer:
		return "Warhammer"
	case ThrowingHammer:
		return "Throwing Hammer"
	case Massa:
		return "Massa"
	case Mangual:
		return "Mangual"
	case Longmass:
		return "Longmass"
	case Pickaxe:
		return "Pickaxe"
	case Fist:
		return "Fist"
	case Rapier:
		return "Rapier"
	case Trident:
		return "Trident"
	case Tchaco:
		return "Tchaco"
	case Crossbow:
		return "Crossbow"
	case Ak47:
		return "Ak47"
	case Ar15:
		return "Ar15"
	case MachineGun:
		return "Machine Gun"
	case Pistol38:
		return "Pistol .38"
	case Rifle:
		return "Rifle"
	case Uzi:
		return "Uzi"
	case Bomb:
		return "Bomb"
	default:
		return "Unknown"
	}
}

func WeaponNameFrom(s string) (WeaponName, error) {
	switch s {
	case "Dagger":
		return Dagger, nil
	case "Throwing Dagger":
		return ThrowingDagger, nil
	case "Halberd":
		return Halberd, nil
	case "Bow":
		return Bow, nil
	case "Longbow":
		return Longbow, nil
	case "Staff":
		return Staff, nil
	case "Scimitar":
		return Scimitar, nil
	case "Rapier":
		return Rapier, nil
	case "Whip":
		return Whip, nil
	case "Club":
		return Club, nil
	case "Longclub":
		return Longclub, nil
	case "Sword":
		return Sword, nil
	case "Longsword":
		return Longsword, nil
	case "Scythe":
		return Scythe, nil
	case "Longscythe":
		return Longscythe, nil
	case "Katana":
		return Katana, nil
	case "Katar":
		return Katar, nil
	case "Spear":
		return Spear, nil
	case "Longspear":
		return Longspear, nil
	case "Axe":
		return Axe, nil
	case "Longaxe":
		return Longaxe, nil
	case "Throwing Axe":
		return ThrowingAxe, nil
	case "Hammer":
		return Hammer, nil
	case "Warhammer":
		return Warhammer, nil
	case "Throwing Hammer":
		return ThrowingHammer, nil
	case "Massa":
		return Massa, nil
	case "Mangual":
		return Mangual, nil
	case "Longmass":
		return Longmass, nil
	case "Pickaxe":
		return Pickaxe, nil
	case "Fist":
		return Fist, nil
	case "Trident":
		return Trident, nil
	case "Tchaco":
		return Tchaco, nil
	case "Crossbow":
		return Crossbow, nil
	case "Ak47":
		return Ak47, nil
	case "Ar15":
		return Ar15, nil
	case "Machine Gun":
		return MachineGun, nil
	case "Pistol .38":
		return Pistol38, nil
	case "Rifle":
		return Rifle, nil
	case "Uzi":
		return Uzi, nil
	case "Bomb":
		return Bomb, nil
	default:
		return 0, fmt.Errorf("invalid weapon name: %s", s)
	}
}
