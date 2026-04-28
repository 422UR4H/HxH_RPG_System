package proficiency_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/proficiency"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func newTestJointProficiency(name string, weapons ...enum.WeaponName) *proficiency.JointProficiency {
	table := experience.NewExpTable(1.0)
	exp := *experience.NewExperience(table)
	return proficiency.NewJointProficiency(exp, name, weapons)
}

func TestJointProficiency_Init(t *testing.T) {
	jp := newTestJointProficiency("dual-wield", enum.Sword, enum.Dagger)
	mockPhys := &mockCascadeUpgrade{}
	mockAbility := &mockCascadeUpgrade{}

	t.Run("successful init", func(t *testing.T) {
		if err := jp.Init(mockPhys, mockAbility); err != nil {
			t.Fatalf("Init() error: %v", err)
		}
	})

	t.Run("double init fails", func(t *testing.T) {
		if err := jp.Init(mockPhys, mockAbility); err == nil {
			t.Error("double Init() should return error")
		}
	})
}

func TestJointProficiency_InitNilArgs(t *testing.T) {
	tests := []struct {
		name    string
		phys    experience.ICascadeUpgrade
		ability experience.ICascadeUpgrade
	}{
		{"nil phys", nil, &mockCascadeUpgrade{}},
		{"nil ability", &mockCascadeUpgrade{}, nil},
		{"both nil", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jp := newTestJointProficiency("test", enum.Sword)
			if err := jp.Init(tt.phys, tt.ability); err == nil {
				t.Error("Init with nil args should return error")
			}
		})
	}
}

func TestJointProficiency_ContainsWeapon(t *testing.T) {
	jp := newTestJointProficiency("blades", enum.Sword, enum.Dagger)

	if !jp.ContainsWeapon(enum.Sword) {
		t.Error("should contain Sword")
	}
	if jp.ContainsWeapon(enum.Bow) {
		t.Error("should not contain Bow")
	}
}

func TestJointProficiency_AddWeapon(t *testing.T) {
	jp := newTestJointProficiency("blades", enum.Sword)
	jp.AddWeapon(enum.Dagger)

	if !jp.ContainsWeapon(enum.Dagger) {
		t.Error("should contain Dagger after AddWeapon")
	}
	if len(jp.GetWeapons()) != 2 {
		t.Errorf("weapons count = %d, want 2", len(jp.GetWeapons()))
	}
}

func TestJointProficiency_BuffManagement(t *testing.T) {
	jp := newTestJointProficiency("blades", enum.Sword)

	if jp.GetBuff() != 0 {
		t.Errorf("initial buff = %d, want 0", jp.GetBuff())
	}

	jp.SetBuff(enum.Sword, 5)
	if jp.GetBuff() != 5 {
		t.Errorf("buff after set = %d, want 5", jp.GetBuff())
	}

	jp.DeleteBuff(enum.Sword)
	if jp.GetBuff() != 0 {
		t.Errorf("buff after delete = %d, want 0", jp.GetBuff())
	}
}

func TestJointProficiency_CascadeUpgradeTrigger(t *testing.T) {
	jp := newTestJointProficiency("blades", enum.Sword, enum.Dagger)
	mockPhys := &mockCascadeUpgrade{}
	mockAbility := &mockCascadeUpgrade{}
	jp.Init(mockPhys, mockAbility)

	values := experience.NewUpgradeCascade(100)
	jp.CascadeUpgradeTrigger(values)

	if jp.GetExpPoints() != 100 {
		t.Errorf("exp after cascade = %d, want 100", jp.GetExpPoints())
	}
	cascade, ok := values.Proficiency["blades"]
	if !ok {
		t.Fatal("joint proficiency cascade not found")
	}
	if cascade.Lvl != jp.GetLevel() {
		t.Errorf("cascade Lvl = %d, want %d", cascade.Lvl, jp.GetLevel())
	}
}
