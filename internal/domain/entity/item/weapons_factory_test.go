package item_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/item"
)

func TestWeaponsManagerFactory_Build(t *testing.T) {
	factory := item.NewWeaponsManagerFactory()
	wm := factory.Build()

	allWeapons := enum.GetAllWeaponNames()
	all := wm.GetAll()

	if len(all) != len(allWeapons) {
		t.Fatalf("Build() produced %d weapons, want %d", len(all), len(allWeapons))
	}

	for _, name := range allWeapons {
		if _, ok := all[name]; !ok {
			t.Errorf("Build() missing weapon: %s", name)
		}
	}
}

func TestWeaponsManagerFactory_Build_WeaponProperties(t *testing.T) {
	factory := item.NewWeaponsManagerFactory()
	wm := factory.Build()

	tests := []struct {
		name         string
		weapon       enum.WeaponName
		damage       int
		isFireWeapon bool
	}{
		{"Dagger is melee with damage 5", enum.Dagger, 5, false},
		{"Katana is melee with damage 7", enum.Katana, 7, false},
		{"Pistol38 is fire weapon with damage 4", enum.Pistol38, 4, true},
		{"Crossbow is fire weapon with damage 2", enum.Crossbow, 2, true},
		{"Bomb is melee (thrown) with damage 0", enum.Bomb, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, err := wm.Get(tt.weapon)
			if err != nil {
				t.Fatalf("Get(%s) error = %v", tt.weapon, err)
			}
			if w.GetDamage() != tt.damage {
				t.Errorf("GetDamage() = %d, want %d", w.GetDamage(), tt.damage)
			}
			if w.IsFireWeapon() != tt.isFireWeapon {
				t.Errorf("IsFireWeapon() = %v, want %v", w.IsFireWeapon(), tt.isFireWeapon)
			}
		})
	}
}
