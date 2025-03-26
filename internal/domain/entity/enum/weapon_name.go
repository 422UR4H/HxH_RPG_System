package enum

import "fmt"

type WeaponName string

const (
	Dagger         WeaponName = "Dagger"
	ThrowingDagger WeaponName = "ThrowingDagger"
	Halberd        WeaponName = "Halberd" // long
	Bow            WeaponName = "Bow"
	Longbow        WeaponName = "Longbow"
	Staff          WeaponName = "Staff"
	Scimitar       WeaponName = "Scimitar"
	Rapier         WeaponName = "Rapier"
	Whip           WeaponName = "Whip"
	Club           WeaponName = "Club"
	Longclub       WeaponName = "Longclub"
	Sword          WeaponName = "Sword"
	Longsword      WeaponName = "Longsword"
	Scythe         WeaponName = "Scythe"
	Longscythe     WeaponName = "Longscythe"
	Katana         WeaponName = "Katana"
	Katar          WeaponName = "Katar"
	Spear          WeaponName = "Spear"
	Longspear      WeaponName = "Longspear"
	Axe            WeaponName = "Axe"
	Longaxe        WeaponName = "Longaxe"
	ThrowingAxe    WeaponName = "ThrowingAxe"
	Hammer         WeaponName = "Hammer"
	Warhammer      WeaponName = "Warhammer" // long
	ThrowingHammer WeaponName = "ThrowingHammer"
	Massa          WeaponName = "Massa"
	Mangual        WeaponName = "Mangual"
	Longmass       WeaponName = "Longmass"
	Pickaxe        WeaponName = "Pickaxe"
	Fist           WeaponName = "Fist"
	Trident        WeaponName = "Trident"
	Tchaco         WeaponName = "Tchaco"

	Crossbow   WeaponName = "Crossbow"
	Ak47       WeaponName = "Ak47"
	Ar15       WeaponName = "Ar15"
	MachineGun WeaponName = "MachineGun"
	Pistol38   WeaponName = "Pistol38"
	Rifle      WeaponName = "Rifle"
	Uzi        WeaponName = "Uzi"
	Bomb       WeaponName = "Bomb"
)

func (sn WeaponName) String() string {
	return string(sn)
}

func GetAllWeaponNames() []WeaponName {
	return []WeaponName{
		Dagger,
		ThrowingDagger,
		Halberd,
		Bow,
		Longbow,
		Staff,
		Scimitar,
		Rapier,
		Whip,
		Club,
		Longclub,
		Sword,
		Longsword,
		Scythe,
		Longscythe,
		Katana,
		Katar,
		Spear,
		Longspear,
		Axe,
		Longaxe,
		ThrowingAxe,
		Hammer,
		Warhammer,
		ThrowingHammer,
		Massa,
		Mangual,
		Longmass,
		Pickaxe,
		Fist,
		Trident,
		Tchaco,
		Crossbow,
		Ak47,
		Ar15,
		MachineGun,
		Pistol38,
		Rifle,
		Uzi,
		Bomb,
	}
}

func WeaponNameFrom(s string) (WeaponName, error) {
	for _, name := range GetAllWeaponNames() {
		if s == name.String() {
			return name, nil
		}
	}
	return "", fmt.Errorf("%w%s: %s", ErrInvalidNameOf, "weapon", s)
}
