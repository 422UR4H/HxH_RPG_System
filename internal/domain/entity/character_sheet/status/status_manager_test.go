package status_test

import (
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/status"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func newTestStatusManager() *status.Manager {
	bars := make(map[enum.StatusName]status.IStatusBar)

	physicals := &mockAbility{bonus: 2.0}
	resistance := &mockDistributableAttribute{value: 1}
	vitality := &mockSkill{level: 1}
	bars[enum.Health] = status.NewHealthPoints(physicals, resistance, vitality)

	energy := &mockSkill{level: 1}
	bars[enum.Stamina] = status.NewStaminaPoints(physicals, resistance, energy)

	return status.NewStatusManager(bars)
}

func TestStatusManager_Get_Found(t *testing.T) {
	mgr := newTestStatusManager()

	tests := []enum.StatusName{enum.Health, enum.Stamina}
	for _, name := range tests {
		t.Run(string(name), func(t *testing.T) {
			bar, err := mgr.Get(name)
			if err != nil {
				t.Fatalf("Get(%s) unexpected error: %v", name, err)
			}
			if bar == nil {
				t.Fatalf("Get(%s) returned nil", name)
			}
		})
	}
}

func TestStatusManager_Get_NotFound(t *testing.T) {
	mgr := newTestStatusManager()

	_, err := mgr.Get(enum.Aura)
	if err == nil {
		t.Fatal("expected error for Aura (not added)")
	}
	if !errors.Is(err, status.ErrStatusNotFound) {
		t.Errorf("expected ErrStatusNotFound, got %v", err)
	}
}

func TestStatusManager_GetMaxOf(t *testing.T) {
	mgr := newTestStatusManager()

	maxHP, err := mgr.GetMaxOf(enum.Health)
	if err != nil {
		t.Fatalf("GetMaxOf(Health) error: %v", err)
	}
	if maxHP <= 0 {
		t.Errorf("HP max should be positive, got %d", maxHP)
	}
}

func TestStatusManager_GetCurrentOf(t *testing.T) {
	mgr := newTestStatusManager()

	curr, err := mgr.GetCurrentOf(enum.Health)
	if err != nil {
		t.Fatalf("GetCurrentOf(Health) error: %v", err)
	}
	if curr <= 0 {
		t.Errorf("HP current should be positive initially, got %d", curr)
	}
}

func TestStatusManager_SetCurrent(t *testing.T) {
	mgr := newTestStatusManager()

	maxHP, _ := mgr.GetMaxOf(enum.Health)
	err := mgr.SetCurrent(enum.Health, maxHP-1)
	if err != nil {
		t.Fatalf("SetCurrent(Health, %d) error: %v", maxHP-1, err)
	}

	curr, _ := mgr.GetCurrentOf(enum.Health)
	if curr != maxHP-1 {
		t.Errorf("current after SetCurrent: got %d, want %d", curr, maxHP-1)
	}
}

func TestStatusManager_SetCurrent_InvalidValue(t *testing.T) {
	mgr := newTestStatusManager()

	maxHP, _ := mgr.GetMaxOf(enum.Health)
	err := mgr.SetCurrent(enum.Health, maxHP+1)
	if err == nil {
		t.Fatal("expected error for value > max")
	}
}

func TestStatusManager_SetCurrent_NotFound(t *testing.T) {
	mgr := newTestStatusManager()

	err := mgr.SetCurrent(enum.Aura, 0)
	if err == nil {
		t.Fatal("expected error for non-existent status")
	}
}

func TestStatusManager_Upgrade(t *testing.T) {
	mgr := newTestStatusManager()

	err := mgr.Upgrade()
	if err != nil {
		t.Fatalf("Upgrade() error: %v", err)
	}
}

func TestStatusManager_GetAllMaximuns(t *testing.T) {
	mgr := newTestStatusManager()

	maxs := mgr.GetAllMaximuns()
	if len(maxs) != 2 {
		t.Errorf("expected 2 status entries, got %d", len(maxs))
	}
}

func TestStatusManager_GetAllStatus(t *testing.T) {
	mgr := newTestStatusManager()

	all := mgr.GetAllStatus()
	if len(all) != 2 {
		t.Errorf("expected 2 status bars, got %d", len(all))
	}
}
