package item_test

import (
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/item"
)

func newTestWeaponsManager() *item.WeaponsManager {
	weapons := map[enum.WeaponName]item.Weapon{
		enum.Sword:  *item.NewWeapon([]int{10, 4}, 2, 0, 0.8, 1.5, 12, false),
		enum.Katana: *item.NewWeapon([]int{4, 12}, 7, 0, 1.0, 1.3, 14, false),
	}
	return item.NewWeaponsManager(weapons)
}

func TestWeaponsManager_Get(t *testing.T) {
	wm := newTestWeaponsManager()

	t.Run("existing weapon", func(t *testing.T) {
		w, err := wm.Get(enum.Sword)
		if err != nil {
			t.Fatalf("Get(Sword) error = %v", err)
		}
		if w.GetDamage() != 2 {
			t.Errorf("Sword damage = %d, want 2", w.GetDamage())
		}
	})

	t.Run("non-existing weapon", func(t *testing.T) {
		_, err := wm.Get(enum.Halberd)
		if !errors.Is(err, item.ErrWeaponNotFound) {
			t.Errorf("Get(Halberd) error = %v, want ErrWeaponNotFound", err)
		}
	})
}

func TestWeaponsManager_Add(t *testing.T) {
	wm := item.NewWeaponsManager(make(map[enum.WeaponName]item.Weapon))
	dagger := *item.NewWeapon([]int{8}, 5, 0, 0.3, 0.4, 3, false)

	wm.Add(enum.Dagger, dagger)

	w, err := wm.Get(enum.Dagger)
	if err != nil {
		t.Fatalf("Get after Add error = %v", err)
	}
	if w.GetDamage() != 5 {
		t.Errorf("damage = %d, want 5", w.GetDamage())
	}
}

func TestWeaponsManager_Delete(t *testing.T) {
	wm := newTestWeaponsManager()
	wm.Delete(enum.Sword)

	_, err := wm.Get(enum.Sword)
	if !errors.Is(err, item.ErrWeaponNotFound) {
		t.Errorf("Get after Delete error = %v, want ErrWeaponNotFound", err)
	}
}

func TestWeaponsManager_GetAll(t *testing.T) {
	wm := newTestWeaponsManager()
	all := wm.GetAll()

	if len(all) != 2 {
		t.Fatalf("GetAll() len = %d, want 2", len(all))
	}
	if _, ok := all[enum.Sword]; !ok {
		t.Error("GetAll() missing Sword")
	}
	if _, ok := all[enum.Katana]; !ok {
		t.Error("GetAll() missing Katana")
	}
}

func TestWeaponsManager_Delegates(t *testing.T) {
	wm := newTestWeaponsManager()

	t.Run("GetDamage", func(t *testing.T) {
		dmg, err := wm.GetDamage(enum.Katana)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if dmg != 7 {
			t.Errorf("GetDamage(Katana) = %d, want 7", dmg)
		}
	})

	t.Run("GetDefense", func(t *testing.T) {
		def, err := wm.GetDefense(enum.Sword)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if def != 0 {
			t.Errorf("GetDefense(Sword) = %d, want 0", def)
		}
	})

	t.Run("GetWeight", func(t *testing.T) {
		w, err := wm.GetWeight(enum.Sword)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if w != 1.5 {
			t.Errorf("GetWeight(Sword) = %f, want 1.5", w)
		}
	})

	t.Run("GetHeight", func(t *testing.T) {
		h, err := wm.GetHeight(enum.Katana)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if h != 1.0 {
			t.Errorf("GetHeight(Katana) = %f, want 1.0", h)
		}
	})

	t.Run("GetVolume", func(t *testing.T) {
		v, err := wm.GetVolume(enum.Katana)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if v != 14 {
			t.Errorf("GetVolume(Katana) = %d, want 14", v)
		}
	})

	t.Run("IsFireWeapon", func(t *testing.T) {
		fire, err := wm.IsFireWeapon(enum.Sword)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if fire {
			t.Error("IsFireWeapon(Sword) = true, want false")
		}
	})

	t.Run("GetDice", func(t *testing.T) {
		dice, err := wm.GetDice(enum.Sword)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if len(dice) != 2 || dice[0] != 10 || dice[1] != 4 {
			t.Errorf("GetDice(Sword) = %v, want [10, 4]", dice)
		}
	})

	t.Run("GetPenality", func(t *testing.T) {
		p, err := wm.GetPenality(enum.Sword)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if p != 1.5 {
			t.Errorf("GetPenality(Sword) = %f, want 1.5", p)
		}
	})

	t.Run("GetStaminaCost", func(t *testing.T) {
		sc, err := wm.GetStaminaCost(enum.Sword)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if sc != 1.5 {
			t.Errorf("GetStaminaCost(Sword) = %f, want 1.5", sc)
		}
	})

	t.Run("delegate error on missing weapon", func(t *testing.T) {
		_, err := wm.GetDamage(enum.Halberd)
		if !errors.Is(err, item.ErrWeaponNotFound) {
			t.Errorf("GetDamage(Halberd) error = %v, want ErrWeaponNotFound", err)
		}
	})
}
