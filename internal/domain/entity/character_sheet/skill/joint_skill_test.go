package skill_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/skill"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func newTestJointSkill(name string, attrPower int, skills ...enum.SkillName) *skill.JointSkill {
	table := experience.NewExpTable(1.0)
	exp := *experience.NewExperience(table)
	mockAttr := &mockGameAttribute{power: attrPower}
	commonSkills := make(map[enum.SkillName]skill.ISkill)
	for _, sn := range skills {
		commonSkills[sn] = newTestCommonSkill(sn, attrPower)
	}
	return skill.NewJointSkill(exp, name, mockAttr, commonSkills)
}

func TestJointSkill_InitAndIsInitialized(t *testing.T) {
	js := newTestJointSkill("hunt", 3, enum.Vision, enum.Stealth)

	if js.IsInitialized() {
		t.Error("should not be initialized before Init()")
	}

	mockAbilityExp := &mockCascadeUpgrade{}
	if err := js.Init(mockAbilityExp); err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	if !js.IsInitialized() {
		t.Error("should be initialized after Init()")
	}

	if err := js.Init(mockAbilityExp); err == nil {
		t.Error("double Init() should return error")
	}
}

func TestJointSkill_InitWithNil(t *testing.T) {
	js := newTestJointSkill("hunt", 3, enum.Vision)

	if err := js.Init(nil); err == nil {
		t.Error("Init(nil) should return error")
	}
}

func TestJointSkill_GetValueForTest(t *testing.T) {
	js := newTestJointSkill("roguery", 5, enum.Stealth, enum.Sneak)

	if got := js.GetValueForTest(); got != 5 {
		t.Errorf("GetValueForTest() = %d, want 5", got)
	}

	js.SetBuff(3)
	if got := js.GetValueForTest(); got != 8 {
		t.Errorf("GetValueForTest() with buff = %d, want 8", got)
	}
}

func TestJointSkill_Contains(t *testing.T) {
	js := newTestJointSkill("athletics", 3, enum.Velocity, enum.Acrobatics)

	if !js.Contains(enum.Velocity) {
		t.Error("should contain Velocity")
	}
	if js.Contains(enum.Stealth) {
		t.Error("should not contain Stealth")
	}
}

func TestJointSkill_Properties(t *testing.T) {
	js := newTestJointSkill("hunt", 3, enum.Vision)

	if js.GetName() != "hunt" {
		t.Errorf("GetName() = %s, want hunt", js.GetName())
	}
	if js.GetBuff() != 0 {
		t.Errorf("initial buff = %d, want 0", js.GetBuff())
	}
	js.SetBuff(5)
	if js.GetBuff() != 5 {
		t.Errorf("buff after set = %d, want 5", js.GetBuff())
	}
}

func TestJointSkill_CascadeUpgradeTrigger(t *testing.T) {
	js := newTestJointSkill("hack", 2, enum.Focus, enum.Accuracy)
	mockAbilityExp := &mockCascadeUpgrade{}
	if err := js.Init(mockAbilityExp); err != nil {
		t.Fatalf("Init() error: %v", err)
	}
	values := experience.NewUpgradeCascade(100)
	js.CascadeUpgradeTrigger(values)

	if js.GetExpPoints() != 100 {
		t.Errorf("exp after cascade = %d, want 100", js.GetExpPoints())
	}
	if values.GetExp() != 200 {
		t.Errorf("cascade exp should be multiplied by common skills count: got %d, want 200", values.GetExp())
	}
}
