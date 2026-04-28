package skill_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/skill"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type mockGameAttribute struct {
	power int
	level int
}

func (m *mockGameAttribute) GetPower() int                               { return m.power }
func (m *mockGameAttribute) GetAbilityBonus() float64                    { return 0 }
func (m *mockGameAttribute) GetLevel() int                               { return m.level }
func (m *mockGameAttribute) CascadeUpgrade(_ *experience.UpgradeCascade) {}
func (m *mockGameAttribute) GetNextLvlAggregateExp() int                 { return 0 }
func (m *mockGameAttribute) GetNextLvlBaseExp() int                      { return 0 }
func (m *mockGameAttribute) GetCurrentExp() int                          { return 0 }
func (m *mockGameAttribute) GetExpPoints() int                           { return 0 }

type mockCascadeUpgrade struct {
	level int
}

func (m *mockCascadeUpgrade) CascadeUpgrade(_ *experience.UpgradeCascade) {}
func (m *mockCascadeUpgrade) GetLevel() int                               { return m.level }

func newTestCommonSkill(name enum.SkillName, attrPower int) *skill.CommonSkill {
	table := experience.NewExpTable(1.0)
	exp := *experience.NewExperience(table)
	mockAttr := &mockGameAttribute{power: attrPower}
	mockAbilityExp := &mockCascadeUpgrade{}
	return skill.NewCommonSkill(name, exp, mockAttr, mockAbilityExp)
}

func TestCommonSkill_GetValueForTest(t *testing.T) {
	tests := []struct {
		name      string
		attrPower int
		want      int
	}{
		{"zero power zero level", 0, 0},
		{"with attribute power", 5, 5},
		{"high power", 20, 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := newTestCommonSkill(enum.Vitality, tt.attrPower)
			got := cs.GetValueForTest()
			if got != tt.want {
				t.Errorf("GetValueForTest() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCommonSkill_InitialState(t *testing.T) {
	cs := newTestCommonSkill(enum.Vitality, 3)

	if cs.GetLevel() != 0 {
		t.Errorf("initial level = %d, want 0", cs.GetLevel())
	}
	if cs.GetExpPoints() != 0 {
		t.Errorf("initial exp = %d, want 0", cs.GetExpPoints())
	}
	if cs.GetName() != enum.Vitality {
		t.Errorf("name = %s, want %s", cs.GetName(), enum.Vitality)
	}
}

func TestCommonSkill_CascadeUpgradeTrigger(t *testing.T) {
	cs := newTestCommonSkill(enum.Defense, 2)
	values := experience.NewUpgradeCascade(50)

	cs.CascadeUpgradeTrigger(values)

	if cs.GetExpPoints() != 50 {
		t.Errorf("exp after cascade = %d, want 50", cs.GetExpPoints())
	}
	cascade, ok := values.Skills[enum.Defense.String()]
	if !ok {
		t.Fatal("skill cascade not found in UpgradeCascade.Skills")
	}
	if cascade.Exp != cs.GetCurrentExp() {
		t.Errorf("cascade Exp = %d, want %d", cascade.Exp, cs.GetCurrentExp())
	}
}

func TestCommonSkill_Clone(t *testing.T) {
	cs := newTestCommonSkill(enum.Vitality, 5)
	cloned := cs.Clone(enum.Energy)

	if cloned.GetName() != enum.Energy {
		t.Errorf("cloned name = %s, want %s", cloned.GetName(), enum.Energy)
	}
	if cloned.GetExpPoints() != 0 {
		t.Errorf("cloned should start with 0 exp, got %d", cloned.GetExpPoints())
	}
}
