package proficiency_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/proficiency"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func TestProficiencyManager_AddCommon(t *testing.T) {
	m := proficiency.NewManager()
	p := newTestProficiency(enum.Sword)

	if err := m.AddCommon(enum.Sword, p); err != nil {
		t.Fatalf("AddCommon error: %v", err)
	}
	if err := m.AddCommon(enum.Sword, p); err == nil {
		t.Error("duplicate AddCommon should return error")
	}
}

func TestProficiencyManager_Get(t *testing.T) {
	m := proficiency.NewManager()
	if err := m.AddCommon(enum.Sword, newTestProficiency(enum.Sword)); err != nil {
		t.Fatal(err)
	}

	t.Run("existing weapon", func(t *testing.T) {
		prof, err := m.Get(enum.Sword)
		if err != nil {
			t.Fatalf("Get error: %v", err)
		}
		if prof == nil {
			t.Fatal("Get returned nil")
		}
	})

	t.Run("non-existing weapon", func(t *testing.T) {
		_, err := m.Get(enum.Bow)
		if err == nil {
			t.Error("Get for non-existing weapon should return error")
		}
	})
}

func TestProficiencyManager_GetFindsJointFirst(t *testing.T) {
	m := proficiency.NewManager()
	if err := m.AddCommon(enum.Sword, newTestProficiency(enum.Sword)); err != nil {
		t.Fatal(err)
	}

	jp := newTestJointProficiency("blades", enum.Sword, enum.Dagger)
	mockPhys := &mockCascadeUpgrade{}
	mockAbility := &mockCascadeUpgrade{}
	if err := m.AddJoint(jp, mockPhys, mockAbility); err != nil {
		t.Fatal(err)
	}

	// Get(Sword) should return joint proficiency since it contains Sword
	prof, err := m.Get(enum.Sword)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if prof == nil {
		t.Fatal("Get returned nil")
	}
	// Verify it's the joint one by giving exp only to the joint
	values := experience.NewUpgradeCascade(77)
	jp.CascadeUpgradeTrigger(values)

	// Now Get(Sword) should still return the joint (with exp=77)
	prof2, _ := m.Get(enum.Sword)
	if prof2.GetExpPoints() != 77 {
		t.Errorf("expected joint proficiency (exp=77), got exp=%d", prof2.GetExpPoints())
	}
}

func TestProficiencyManager_AddJoint(t *testing.T) {
	m := proficiency.NewManager()
	jp := newTestJointProficiency("blades", enum.Sword)
	mockPhys := &mockCascadeUpgrade{}
	mockAbility := &mockCascadeUpgrade{}

	if err := m.AddJoint(jp, mockPhys, mockAbility); err != nil {
		t.Fatalf("AddJoint error: %v", err)
	}

	jp2 := newTestJointProficiency("blades", enum.Dagger)
	if err := m.AddJoint(jp2, mockPhys, mockAbility); err == nil {
		t.Error("duplicate AddJoint should return error")
	}
}

func TestProficiencyManager_IncreaseExp(t *testing.T) {
	m := proficiency.NewManager()
	if err := m.AddCommon(enum.Sword, newTestProficiency(enum.Sword)); err != nil {
		t.Fatal(err)
	}

	values := experience.NewUpgradeCascade(50)
	if err := m.IncreaseExp(values, enum.Sword); err != nil {
		t.Fatalf("IncreaseExp error: %v", err)
	}

	exp, _ := m.GetExpPointsOf(enum.Sword)
	if exp != 50 {
		t.Errorf("exp after increase = %d, want 50", exp)
	}
}

func TestProficiencyManager_BuffManagement(t *testing.T) {
	m := proficiency.NewManager()
	if err := m.AddCommon(enum.Sword, newTestProficiency(enum.Sword)); err != nil {
		t.Fatal(err)
	}

	m.SetBuff(enum.Sword, 3)

	val, err := m.GetValueForTestOf(enum.Sword)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	// level=0 + buff=3 = 3
	if val != 3 {
		t.Errorf("buffed value = %d, want 3", val)
	}

	m.DeleteBuff(enum.Sword)
	val, _ = m.GetValueForTestOf(enum.Sword)
	if val != 0 {
		t.Errorf("after delete buff = %d, want 0", val)
	}
}
