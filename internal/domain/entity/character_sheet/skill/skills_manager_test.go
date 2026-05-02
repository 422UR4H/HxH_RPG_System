package skill_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/skill"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func newTestSkillsManager() *skill.Manager {
	table := experience.NewExpTable(1.0)
	exp := *experience.NewExperience(table)
	mockAbilityExp := &mockCascadeUpgrade{}
	return skill.NewSkillsManager(exp, mockAbilityExp)
}

func initManagerWithSkills(t testing.TB, m *skill.Manager, names ...enum.SkillName) {
	skills := make(map[enum.SkillName]skill.ISkill)
	for _, name := range names {
		skills[name] = newTestCommonSkill(name, 3)
	}
	if err := m.Init(skills); err != nil {
		t.Fatal(err)
	}
}

func TestSkillsManager_Init(t *testing.T) {
	m := newTestSkillsManager()
	skills := map[enum.SkillName]skill.ISkill{
		enum.Vitality: newTestCommonSkill(enum.Vitality, 3),
	}

	if err := m.Init(skills); err != nil {
		t.Fatalf("Init() error: %v", err)
	}
	if err := m.Init(skills); err == nil {
		t.Error("double Init() should return error")
	}
}

func TestSkillsManager_Get(t *testing.T) {
	m := newTestSkillsManager()
	initManagerWithSkills(t, m, enum.Vitality, enum.Energy)

	t.Run("existing skill", func(t *testing.T) {
		sk, err := m.Get(enum.Vitality)
		if err != nil {
			t.Fatalf("Get(Vitality) error: %v", err)
		}
		if sk == nil {
			t.Fatal("Get(Vitality) returned nil")
		}
	})

	t.Run("non-existing skill", func(t *testing.T) {
		_, err := m.Get(enum.Stealth)
		if err == nil {
			t.Error("Get(Stealth) should return error for non-existing skill")
		}
	})
}

func TestSkillsManager_GetValueForTestOf(t *testing.T) {
	m := newTestSkillsManager()
	initManagerWithSkills(t, m, enum.Vitality)

	val, err := m.GetValueForTestOf(enum.Vitality)
	if err != nil {
		t.Fatalf("GetValueForTestOf error: %v", err)
	}
	if val != 3 {
		t.Errorf("value for test = %d, want 3 (attrPower=3, level=0)", val)
	}
}

func TestSkillsManager_BuffManagement(t *testing.T) {
	m := newTestSkillsManager()
	initManagerWithSkills(t, m, enum.Vitality)

	m.SetBuff(enum.Vitality, 5)

	val, err := m.GetValueForTestOf(enum.Vitality)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if val != 8 {
		t.Errorf("buffed value = %d, want 8", val)
	}

	m.DeleteBuff(enum.Vitality)

	val, _ = m.GetValueForTestOf(enum.Vitality)
	if val != 3 {
		t.Errorf("after delete buff = %d, want 3", val)
	}
}

func TestSkillsManager_AddJointSkill(t *testing.T) {
	m := newTestSkillsManager()
	initManagerWithSkills(t, m, enum.Vision, enum.Stealth)

	js := newTestJointSkill("hunt", 3, enum.Vision, enum.Stealth)

	t.Run("uninitialized joint skill rejected", func(t *testing.T) {
		if err := m.AddJointSkill(js); err == nil {
			t.Error("should reject uninitialized joint skill")
		}
	})

	mockAbilityExp := &mockCascadeUpgrade{}
	if err := js.Init(mockAbilityExp); err != nil {
		t.Fatal(err)
	}

	t.Run("initialized joint skill accepted", func(t *testing.T) {
		if err := m.AddJointSkill(js); err != nil {
			t.Fatalf("AddJointSkill error: %v", err)
		}
	})

	t.Run("duplicate joint skill rejected", func(t *testing.T) {
		js2 := newTestJointSkill("hunt", 3, enum.Vision)
		if err := js2.Init(mockAbilityExp); err != nil {
			t.Fatal(err)
		}
		if err := m.AddJointSkill(js2); err == nil {
			t.Error("should reject duplicate joint skill name")
		}
	})

	t.Run("joint skill found via Get", func(t *testing.T) {
		sk, err := m.Get(enum.Vision)
		if err != nil {
			t.Fatalf("Get(Vision) error: %v", err)
		}
		if sk.GetValueForTest() != js.GetValueForTest() {
			t.Error("Get should return the joint skill for contained skill names")
		}
	})
}

func TestSkillsManager_IncreaseExp(t *testing.T) {
	m := newTestSkillsManager()
	initManagerWithSkills(t, m, enum.Defense)

	values := experience.NewUpgradeCascade(50)
	if err := m.IncreaseExp(values, enum.Defense); err != nil {
		t.Fatalf("IncreaseExp error: %v", err)
	}

	exp, _ := m.GetExpPointsOf(enum.Defense)
	if exp != 50 {
		t.Errorf("exp after increase = %d, want 50", exp)
	}

	if err := m.IncreaseExp(values, enum.Stealth); err == nil {
		t.Error("IncreaseExp for non-existing skill should return error")
	}
}

func TestSkillsManager_BatchGetters(t *testing.T) {
	m := newTestSkillsManager()
	initManagerWithSkills(t, m, enum.Vitality, enum.Energy, enum.Defense)

	levels := m.GetSkillsLevel()
	if len(levels) != 3 {
		t.Errorf("skills count = %d, want 3", len(levels))
	}
	for name, lvl := range levels {
		if lvl != 0 {
			t.Errorf("skill %s initial level = %d, want 0", name, lvl)
		}
	}
}
