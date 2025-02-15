package item

import "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"

type WeaponsManagerFactory struct{}

func NewWeaponsManagerFactory() *WeaponsManagerFactory {
	return &WeaponsManagerFactory{}
}

func (w *WeaponsManagerFactory) Build() *WeaponsManager {
	weapons := map[enum.WeaponName]Weapon{
		enum.Dagger:         *NewWeapon([]int{8}, 5, 0, 0.3, 0.4, 3, false),
		enum.ThrowingDagger: *NewWeapon([]int{8}, 2, 0, 0.2, 0.3, 2, false),
		enum.Halberd:        *NewWeapon([]int{12, 10, 6}, 1, 0, 2.2, 7, 20, false),
		enum.Bow:            *NewWeapon([]int{12}, 3, 0, 1.5, 1, 12, false),
		enum.Longbow:        *NewWeapon([]int{12, 12}, 3, 0, 2, 2, 20, false),
		enum.Staff:          *NewWeapon([]int{10, 8}, 0, 0, 1.7, 2.5, 16, false),
		enum.Scimitar:       *NewWeapon([]int{6, 4}, 4, 0, 0.9, 1.2, 12, false),
		enum.Whip:           *NewWeapon([]int{4, 4, 8}, 0, 0, 2.5, 1.3, 12, false),
		enum.Club:           *NewWeapon([]int{8, 8}, 1, 0, 0.6, 2, 12, false),
		enum.Longclub:       *NewWeapon([]int{8, 8, 12}, 1, 0, 1.2, 4, 20, false),
		enum.Sword:          *NewWeapon([]int{10, 4}, 2, 0, 0.8, 1.5, 12, false),
		enum.Longsword:      *NewWeapon([]int{12, 10, 4}, 2, 0, 1.2, 2.5, 20, false),
		enum.Scythe:         *NewWeapon([]int{4, 4, 6}, 2, 0, 1.6, 3, 12, false),
		enum.Longscythe:     *NewWeapon([]int{4, 4, 6, 12}, 2, 0, 2.4, 6, 20, false),
		enum.Katana:         *NewWeapon([]int{4, 12}, 7, 0, 1, 1.3, 14, false),
		enum.Katar:          *NewWeapon([]int{6}, 6, 0, 0.4, 0.8, 3, false),
		enum.Spear:          *NewWeapon([]int{8, 4}, 3, 0, 2, 2.5, 10, false),
		enum.Longspear:      *NewWeapon([]int{12, 8, 4}, 3, 0, 3, 4.5, 16, false),
		enum.Axe:            *NewWeapon([]int{10, 6}, 1, 0, 0.7, 2.5, 12, false),
		enum.Longaxe:        *NewWeapon([]int{10, 6, 12}, 1, 0, 1.2, 4.5, 20, false),
		enum.ThrowingAxe:    *NewWeapon([]int{10}, 1, 0, 0.4, 1.5, 3, false),
		enum.Hammer:         *NewWeapon([]int{12, 6}, 0, 0, 0.6, 2.5, 12, false),
		enum.Warhammer:      *NewWeapon([]int{12, 12, 6}, 0, 0, 1.2, 6, 20, false),
		enum.ThrowingHammer: *NewWeapon([]int{12}, 0, 0, 0.4, 1.5, 3, false),
		enum.Massa:          *NewWeapon([]int{12, 4}, 1, 0, 0.7, 3.5, 12, false),
		enum.Longmass:       *NewWeapon([]int{12, 12, 4}, 1, 0, 1.2, 6, 20, false),
		enum.Mangual:        *NewWeapon([]int{12, 4}, 1, 0, 0.8, 4, 12, false),
		enum.Pickaxe:        *NewWeapon([]int{8, 6}, 2, 0, 0.9, 3.5, 14, false),
		enum.Fist:           *NewWeapon([]int{6, 6, 4}, 0, 0, 0.2, 0.8, 3, false),
		enum.Rapier:         *NewWeapon([]int{4, 4}, 5, 0, 1, 1.2, 10, false),
		enum.Trident:        *NewWeapon([]int{8, 8, 8}, 3, 0, 1.5, 2.5, 20, false),
		enum.Tchaco:         *NewWeapon([]int{10}, 4, 0, 0.7, 2, 10, false),
		enum.Crossbow:       *NewWeapon([]int{12, 12, 12}, 2, 0, 0.9, 4, 16, true),
		enum.Ak47:           *NewWeapon([]int{10, 10, 10}, 1, 0, 0.8, 4.5, 14, true),
		enum.Ar15:           *NewWeapon([]int{10, 10}, 6, 0, 0.9, 3.5, 14, true),
		enum.MachineGun:     *NewWeapon([]int{12, 10}, 3, 0, 6, 3, 14, true),
		enum.Pistol38:       *NewWeapon([]int{12}, 4, 0, 1.3, 0.9, 3, true),
		enum.Rifle:          *NewWeapon([]int{12, 10}, 8, 0, 1.2, 6, 16, true),
		enum.Uzi:            *NewWeapon([]int{12, 8}, 1, 0, 0.4, 3, 6, true),
	}

	return &WeaponsManager{weapons: weapons}
}
