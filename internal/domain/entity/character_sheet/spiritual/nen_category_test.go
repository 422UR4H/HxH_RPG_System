package spiritual_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/spiritual"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type mockHatsu struct {
	percentOf    float64
	valueForTest int
	level        int
}

func (m *mockHatsu) GetPercentOf(_ enum.CategoryName) float64    { return m.percentOf }
func (m *mockHatsu) GetValueForTest() int                        { return m.valueForTest }
func (m *mockHatsu) GetLevel() int                               { return m.level }
func (m *mockHatsu) CascadeUpgrade(_ *experience.UpgradeCascade) {}
func (m *mockHatsu) GetNextLvlAggregateExp() int                 { return 0 }
func (m *mockHatsu) GetNextLvlBaseExp() int                      { return 0 }
func (m *mockHatsu) GetCurrentExp() int                          { return 0 }
func (m *mockHatsu) GetExpPoints() int                           { return 0 }

func TestNenCategory_GetValueForTest(t *testing.T) {
	tests := []struct {
		name         string
		percentOf    float64
		hatsuTestVal int
		want         int
	}{
		{"100% own category", 100.0, 10, 10},
		{"80% adjacent", 80.0, 10, 8},
		{"0% specialization not current", 0.0, 10, 0},
		{"60% two steps away", 60.0, 10, 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := experience.NewDefaultExpTable()
			exp := *experience.NewExperience(table)
			mock := &mockHatsu{percentOf: tt.percentOf, valueForTest: tt.hatsuTestVal}
			nc := spiritual.NewNenCategory(exp, enum.Reinforcement, mock)

			// valueForTest = (catLevel(0) + hatsuTestVal) * percent / 100
			got := nc.GetValueForTest()
			if got != tt.want {
				t.Errorf("GetValueForTest() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestNenCategory_GetPercent(t *testing.T) {
	table := experience.NewDefaultExpTable()
	exp := *experience.NewExperience(table)
	mock := &mockHatsu{percentOf: 80.0}
	nc := spiritual.NewNenCategory(exp, enum.Transmutation, mock)

	if pct := nc.GetPercent(); pct != 80.0 {
		t.Errorf("GetPercent() = %f, want 80.0", pct)
	}
}

func TestNenCategory_CascadeUpgradeTrigger(t *testing.T) {
	table := experience.NewDefaultExpTable()
	exp := *experience.NewExperience(table)
	mock := &mockHatsu{percentOf: 100.0, valueForTest: 5}
	nc := spiritual.NewNenCategory(exp, enum.Reinforcement, mock)

	values := experience.NewUpgradeCascade(50)
	nc.CascadeUpgradeTrigger(values)

	if nc.GetExpPoints() != 50 {
		t.Errorf("exp after cascade = %d, want 50", nc.GetExpPoints())
	}
}

func TestNenCategory_Clone(t *testing.T) {
	table := experience.NewDefaultExpTable()
	exp := *experience.NewExperience(table)
	mock := &mockHatsu{percentOf: 100.0}
	nc := spiritual.NewNenCategory(exp, enum.Reinforcement, mock)

	cloned := nc.Clone(enum.Emission)
	if cloned.GetName() != enum.Emission {
		t.Errorf("cloned name = %s, want %s", cloned.GetName(), enum.Emission)
	}
}
