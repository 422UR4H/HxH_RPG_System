package item

import "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"

type WeaponsManagerFactory struct{}

func NewWeaponsManagerFactory() *WeaponsManagerFactory {
	return &WeaponsManagerFactory{}
}

func (w *WeaponsManagerFactory) Build() *WeaponsManager {
	weapons := map[enum.WeaponName]Weapon{
		enum.Dagger:         *NewWeapon([]int{8}, 5, 0, 0, 0, 3),
		enum.ThrowingDagger: *NewWeapon([]int{8}, 2, 0, 0, 0, 2),
		enum.Halberd:        *NewWeapon([]int{12, 10, 6}, 1, 0, 0, 0, 20),
		enum.Bow:            *NewWeapon([]int{12}, 3, 0, 0, 0, 12),
		enum.Longbow:        *NewWeapon([]int{12, 12}, 3, 0, 0, 0, 20),
		enum.Staff:          *NewWeapon([]int{10, 8}, 0, 0, 0, 0, 16),
		enum.Scimitar:       *NewWeapon([]int{6, 4}, 4, 0, 0, 0, 12),
		enum.Whip:           *NewWeapon([]int{4, 4, 8}, 0, 0, 0, 0, 12),
		enum.Club:           *NewWeapon([]int{8, 8}, 1, 0, 0, 0, 12),
		enum.Longclub:       *NewWeapon([]int{8, 8, 12}, 1, 0, 0, 0, 20),
		enum.Sword:          *NewWeapon([]int{10, 4}, 2, 0, 0, 0, 12),
		enum.Longsword:      *NewWeapon([]int{12, 10, 4}, 2, 0, 0, 0, 20),
		enum.Scythe:         *NewWeapon([]int{4, 4, 6}, 2, 0, 0, 0, 12),
		enum.Longscythe:     *NewWeapon([]int{4, 4, 6, 12}, 2, 0, 0, 0, 20),
		enum.Katana:         *NewWeapon([]int{4, 12}, 7, 0, 0, 0, 14),
		enum.Katar:          *NewWeapon([]int{6}, 6, 0, 0, 0, 3),
		enum.Spear:          *NewWeapon([]int{8, 4}, 3, 0, 0, 0, 10),
		enum.Longspear:      *NewWeapon([]int{12, 8, 4}, 3, 0, 0, 0, 16),
		enum.Axe:            *NewWeapon([]int{10, 6}, 1, 0, 0, 0, 12),
		enum.Longaxe:        *NewWeapon([]int{10, 6, 12}, 1, 0, 0, 0, 20),
		enum.ThrowingAxe:    *NewWeapon([]int{10}, 1, 0, 0, 0, 3),
		enum.Hammer:         *NewWeapon([]int{12, 6}, 0, 0, 0, 0, 12),
		enum.Warhammer:      *NewWeapon([]int{12, 12, 6}, 0, 0, 0, 0, 20),
		enum.ThrowingHammer: *NewWeapon([]int{12}, 0, 0, 0, 0, 3),
		enum.Massa:          *NewWeapon([]int{12, 4}, 1, 0, 0, 0, 12),
		enum.Mangual:        *NewWeapon([]int{12, 4}, 1, 0, 0, 0, 12),
		enum.Longmass:       *NewWeapon([]int{12, 12, 4}, 1, 0, 0, 0, 20),
		enum.Pickaxe:        *NewWeapon([]int{8, 6}, 2, 0, 0, 0, 14),
		enum.Fist:           *NewWeapon([]int{6, 6, 4}, 0, 0, 0, 0, 3),
		enum.Rapier:         *NewWeapon([]int{4, 4}, 5, 0, 0, 0, 10),
		enum.Trident:        *NewWeapon([]int{8, 8, 8}, 3, 0, 0, 0, 20),
		enum.Tchaco:         *NewWeapon([]int{10}, 4, 0, 0, 0, 10),
		enum.Crossbow:       *NewWeapon([]int{12, 12, 12}, 2, 0, 0, 0, 16),
		enum.Ak47:           *NewWeapon([]int{10, 10, 10}, 1, 0, 0, 0, 14),
		enum.Ar15:           *NewWeapon([]int{10, 10}, 6, 0, 0, 0, 14),
		enum.MachineGun:     *NewWeapon([]int{12, 10}, 3, 0, 0, 0, 14),
		enum.Pistol38:       *NewWeapon([]int{12}, 4, 0, 0, 0, 3),
		enum.Rifle:          *NewWeapon([]int{12, 10}, 8, 0, 0, 0, 16),
		enum.Uzi:            *NewWeapon([]int{12, 8}, 1, 0, 0, 0, 6),
	}

	return &WeaponsManager{weapons: weapons}
}
