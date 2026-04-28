package spiritual_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/spiritual"
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

func newTestNenPrinciple(name enum.PrincipleName, flameLvl, conscPower int) *spiritual.NenPrinciple {
	table := experience.NewDefaultExpTable()
	exp := *experience.NewExperience(table)
	flame := &mockGameAttribute{level: flameLvl}
	conscience := &mockGameAttribute{power: conscPower}
	return spiritual.NewNenPrinciple(name, exp, flame, conscience)
}

func TestNenPrinciple_InitialState(t *testing.T) {
	np := newTestNenPrinciple(enum.Ten, 0, 0)

	if np.GetLevel() != 0 {
		t.Errorf("initial level = %d, want 0", np.GetLevel())
	}
	if np.GetName() != enum.Ten {
		t.Errorf("name = %s, want %s", np.GetName(), enum.Ten)
	}
}

func TestNenPrinciple_GetValueForTest(t *testing.T) {
	tests := []struct {
		name       string
		flameLvl   int
		conscPower int
		want       int
	}{
		{"all zero", 0, 0, 0},
		{"flame only", 5, 0, 5},
		{"conscience only", 0, 3, 3},
		{"both", 5, 3, 8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			np := newTestNenPrinciple(enum.Ren, tt.flameLvl, tt.conscPower)
			// valueForTest = principleLevel(0) + consciencePower + flameLevel
			if got := np.GetValueForTest(); got != tt.want {
				t.Errorf("GetValueForTest() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestNenPrinciple_Clone(t *testing.T) {
	np := newTestNenPrinciple(enum.Ten, 3, 5)
	cloned := np.Clone(enum.Ren)

	if cloned.GetName() != enum.Ren {
		t.Errorf("cloned name = %s, want %s", cloned.GetName(), enum.Ren)
	}
	if cloned.GetExpPoints() != 0 {
		t.Errorf("cloned should have 0 exp, got %d", cloned.GetExpPoints())
	}
}

func TestNenPrinciple_CascadeUpgradeTrigger(t *testing.T) {
	np := newTestNenPrinciple(enum.Gyo, 2, 4)
	values := experience.NewUpgradeCascade(50)

	np.CascadeUpgradeTrigger(values)

	if np.GetExpPoints() != 50 {
		t.Errorf("exp after cascade = %d, want 50", np.GetExpPoints())
	}
	cascade, ok := values.Principles[enum.Hatsu]
	if !ok {
		t.Fatal("principle cascade not found (stored under Hatsu key)")
	}
	if cascade.Lvl != np.GetLevel() {
		t.Errorf("cascade Lvl = %d, want %d", cascade.Lvl, np.GetLevel())
	}
}
