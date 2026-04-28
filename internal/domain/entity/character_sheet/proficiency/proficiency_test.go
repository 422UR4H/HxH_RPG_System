package proficiency_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/proficiency"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type mockCascadeUpgrade struct{ level int }

func (m *mockCascadeUpgrade) CascadeUpgrade(_ *experience.UpgradeCascade) {}
func (m *mockCascadeUpgrade) GetLevel() int                               { return m.level }

func newTestProficiency(weapon enum.WeaponName) *proficiency.Proficiency {
	table := experience.NewExpTable(1.0)
	exp := *experience.NewExperience(table)
	mockPhysSkills := &mockCascadeUpgrade{}
	return proficiency.NewProficiency(weapon, exp, mockPhysSkills)
}

func TestProficiency_InitialState(t *testing.T) {
	p := newTestProficiency(enum.Sword)

	if p.GetLevel() != 0 {
		t.Errorf("initial level = %d, want 0", p.GetLevel())
	}
	if p.GetExpPoints() != 0 {
		t.Errorf("initial exp = %d, want 0", p.GetExpPoints())
	}
	if p.GetWeapon() != enum.Sword {
		t.Errorf("weapon = %s, want %s", p.GetWeapon(), enum.Sword)
	}
}

func TestProficiency_GetValueForTest(t *testing.T) {
	p := newTestProficiency(enum.Dagger)
	if got := p.GetValueForTest(); got != 0 {
		t.Errorf("initial GetValueForTest() = %d, want 0", got)
	}
}

func TestProficiency_CascadeUpgradeTrigger(t *testing.T) {
	p := newTestProficiency(enum.Sword)
	values := experience.NewUpgradeCascade(50)

	p.CascadeUpgradeTrigger(values)

	if p.GetExpPoints() != 50 {
		t.Errorf("exp after cascade = %d, want 50", p.GetExpPoints())
	}
	cascade, ok := values.Proficiency[enum.Sword.String()]
	if !ok {
		t.Fatal("proficiency cascade not found")
	}
	if cascade.Lvl != p.GetLevel() {
		t.Errorf("cascade Lvl = %d, want %d", cascade.Lvl, p.GetLevel())
	}
}
