package service_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/item"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
)

func catalogue() *item.WeaponsManager { return item.NewWeaponsManagerFactory().Build() }

func TestWeaponDice(t *testing.T) {
	cat := catalogue()

	t.Run("a named weapon yields its own dice", func(t *testing.T) {
		sword := enum.Sword
		got, err := service.WeaponDice(&sword, cat)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 2 || got[0] != enum.D10 || got[1] != enum.D4 {
			t.Errorf("Sword dice = %v, want [D10 D4]", got)
		}
	})

	t.Run("no weapon means bare hands, which is a real catalogue entry", func(t *testing.T) {
		got, err := service.WeaponDice(nil, cat)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 3 {
			t.Errorf("Fist dice = %v, want 3 dice", got)
		}
	})

	t.Run("an unknown weapon is an error, not a silent zero", func(t *testing.T) {
		bogus := enum.WeaponName("Excalibur")
		if _, err := service.WeaponDice(&bogus, cat); err == nil {
			t.Error("expected an error for a weapon outside the catalogue")
		}
	})
}

func TestRawDamage(t *testing.T) {
	cat := catalogue()

	t.Run("raw damage is the dice plus the weapon's flat bonus", func(t *testing.T) {
		sword := enum.Sword // D10 + D4, flat damage 2
		got, err := service.RawDamage([]int{7, 3}, &sword, cat)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != 12 { // 7 + 3 + 2
			t.Errorf("RawDamage() = %d, want 12", got)
		}
	})

	t.Run("bare hands add nothing flat", func(t *testing.T) {
		got, err := service.RawDamage([]int{4, 5, 1}, nil, cat) // Fist, flat damage 0
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != 10 {
			t.Errorf("RawDamage() = %d, want 10", got)
		}
	})
}

func TestApplicableDefense(t *testing.T) {
	cat := catalogue()
	sword, dagger := enum.Sword, enum.Dagger

	t.Run("a failed defense subtracts nothing", func(t *testing.T) {
		got := service.ApplicableDefense(service.DefenseInput{
			AttackWeapon: &sword, DefenseWeapon: &dagger, Defended: false, Catalogue: cat,
		})
		if got.Amount != 0 || got.BlocksEntirely {
			t.Errorf("got %+v, want a no-op", got)
		}
	})

	t.Run("weapon against weapon lets nothing through", func(t *testing.T) {
		got := service.ApplicableDefense(service.DefenseInput{
			AttackWeapon: &sword, DefenseWeapon: &dagger, Defended: true, Catalogue: cat,
		})
		if !got.BlocksEntirely {
			t.Error("expected an armed defense against an armed attack to block entirely")
		}
	})

	t.Run("an unarmed attack passes damage through the defense", func(t *testing.T) {
		got := service.ApplicableDefense(service.DefenseInput{
			AttackWeapon: nil, DefenseWeapon: &dagger, Defended: true, Catalogue: cat,
		})
		if got.BlocksEntirely {
			t.Error("an unarmed attack must not be blocked entirely")
		}
		// Every weapon in the catalogue currently carries defense 0, so this is 0 today.
		// The assertion is on the shape, not on the number.
		if got.Amount != 0 {
			t.Errorf("Amount = %d, want the dagger's defense bonus (0 in today's catalogue)", got.Amount)
		}
	})

	t.Run("an armed attack against a bare-handed defense is not reduced", func(t *testing.T) {
		got := service.ApplicableDefense(service.DefenseInput{
			AttackWeapon: &sword, DefenseWeapon: nil, Defended: true, Catalogue: cat,
		})
		if got.BlocksEntirely || got.Amount != 0 {
			t.Errorf("got %+v, want no reduction while damage types do not exist", got)
		}
	})
}

func TestEffectiveDamage(t *testing.T) {
	tests := []struct {
		name string
		raw  int
		def  service.DefenseOutcome
		want int
	}{
		{"undefended damage passes whole", 12, service.DefenseOutcome{}, 12},
		{"a blocking defense zeroes it", 12, service.DefenseOutcome{BlocksEntirely: true}, 0},
		{"a reducing defense subtracts", 12, service.DefenseOutcome{Amount: 5}, 7},
		{"the reduction never goes below zero", 3, service.DefenseOutcome{Amount: 9}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := service.EffectiveDamage(tt.raw, tt.def); got != tt.want {
				t.Errorf("EffectiveDamage() = %d, want %d", got, tt.want)
			}
		})
	}
}
