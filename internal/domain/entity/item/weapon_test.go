package item_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/item"
)

func TestNewWeapon(t *testing.T) {
	dice := []int{10, 6}
	w := item.NewWeapon(dice, 5, 3, 1.2, 2.5, 12, false)

	if w.GetDamage() != 5 {
		t.Errorf("GetDamage() = %d, want 5", w.GetDamage())
	}
	if w.GetDefense() != 3 {
		t.Errorf("GetDefense() = %d, want 3", w.GetDefense())
	}
	if w.GetHeight() != 1.2 {
		t.Errorf("GetHeight() = %f, want 1.2", w.GetHeight())
	}
	if w.GetWeight() != 2.5 {
		t.Errorf("GetWeight() = %f, want 2.5", w.GetWeight())
	}
	if w.GetVolume() != 12 {
		t.Errorf("GetVolume() = %d, want 12", w.GetVolume())
	}
	if w.IsFireWeapon() {
		t.Error("IsFireWeapon() = true, want false")
	}
}

func TestWeapon_GetDice_ReturnsCopy(t *testing.T) {
	original := []int{10, 6, 4}
	w := item.NewWeapon(original, 0, 0, 0, 0, 0, false)

	dice := w.GetDice()
	if len(dice) != 3 || dice[0] != 10 || dice[1] != 6 || dice[2] != 4 {
		t.Fatalf("GetDice() = %v, want [10, 6, 4]", dice)
	}

	// Mutating the returned slice should not affect internal state
	dice[0] = 999
	fresh := w.GetDice()
	if fresh[0] != 10 {
		t.Errorf("GetDice() returned reference instead of copy: got %d, want 10", fresh[0])
	}
}

func TestWeapon_GetPenality_MeleeWeapon(t *testing.T) {
	tests := []struct {
		name     string
		weight   float64
		expected float64
	}{
		{"light weapon (0.4kg)", 0.4, 0.4},
		{"medium weapon (2.5kg)", 2.5, 2.5},
		{"heavy weapon (6.0kg)", 6.0, 6.0},
		{"zero weight", 0.0, 0.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := item.NewWeapon(nil, 0, 0, 0, tt.weight, 0, false)
			if w.GetPenality() != tt.expected {
				t.Errorf("GetPenality() = %f, want %f", w.GetPenality(), tt.expected)
			}
		})
	}
}

func TestWeapon_GetPenality_FireWeapon(t *testing.T) {
	tests := []struct {
		name     string
		weight   float64
		expected float64
	}{
		{"light fire weapon (0.5kg) => 0", 0.5, 0.0},
		{"exactly 1.0kg => 1", 1.0, 1.0},
		{"heavy fire weapon (4.5kg) => 1", 4.5, 1.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := item.NewWeapon(nil, 0, 0, 0, tt.weight, 0, true)
			if w.GetPenality() != tt.expected {
				t.Errorf("GetPenality() = %f, want %f", w.GetPenality(), tt.expected)
			}
		})
	}
}

func TestWeapon_GetStaminaCost(t *testing.T) {
	tests := []struct {
		name         string
		weight       float64
		isFireWeapon bool
		expected     float64
	}{
		{"melee weapon uses weight", 2.5, false, 2.5},
		{"fire weapon always 1.0", 4.5, true, 1.0},
		{"light melee weapon", 0.4, false, 0.4},
		{"light fire weapon", 0.3, true, 1.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := item.NewWeapon(nil, 0, 0, 0, tt.weight, 0, tt.isFireWeapon)
			if w.GetStaminaCost() != tt.expected {
				t.Errorf("GetStaminaCost() = %f, want %f", w.GetStaminaCost(), tt.expected)
			}
		})
	}
}
