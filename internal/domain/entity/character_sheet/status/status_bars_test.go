package status_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/status"
)

// --- Mock implementations ---

type mockAbility struct {
	bonus float64
	level int
}

func (m *mockAbility) GetBonus() float64                           { return m.bonus }
func (m *mockAbility) GetLevel() int                               { return m.level }
func (m *mockAbility) CascadeUpgrade(_ *experience.UpgradeCascade) {}
func (m *mockAbility) GetNextLvlAggregateExp() int                 { return 0 }
func (m *mockAbility) GetNextLvlBaseExp() int                      { return 0 }
func (m *mockAbility) GetCurrentExp() int                          { return 0 }
func (m *mockAbility) GetExpPoints() int                           { return 0 }
func (m *mockAbility) GetExpReference() experience.ICascadeUpgrade { return nil }

type mockDistributableAttribute struct {
	value int
	level int
}

func (m *mockDistributableAttribute) GetValue() int                              { return m.value }
func (m *mockDistributableAttribute) GetPoints() int                             { return 0 }
func (m *mockDistributableAttribute) GetPower() int                              { return m.value }
func (m *mockDistributableAttribute) GetAbilityBonus() float64                   { return 0 }
func (m *mockDistributableAttribute) GetLevel() int                              { return m.level }
func (m *mockDistributableAttribute) CascadeUpgrade(_ *experience.UpgradeCascade) {}
func (m *mockDistributableAttribute) GetNextLvlAggregateExp() int                { return 0 }
func (m *mockDistributableAttribute) GetNextLvlBaseExp() int                     { return 0 }
func (m *mockDistributableAttribute) GetCurrentExp() int                         { return 0 }
func (m *mockDistributableAttribute) GetExpPoints() int                          { return 0 }

type mockGameAttribute struct {
	power int
	level int
}

func (m *mockGameAttribute) GetPower() int                              { return m.power }
func (m *mockGameAttribute) GetAbilityBonus() float64                   { return 0 }
func (m *mockGameAttribute) GetLevel() int                              { return m.level }
func (m *mockGameAttribute) CascadeUpgrade(_ *experience.UpgradeCascade) {}
func (m *mockGameAttribute) GetNextLvlAggregateExp() int                { return 0 }
func (m *mockGameAttribute) GetNextLvlBaseExp() int                     { return 0 }
func (m *mockGameAttribute) GetCurrentExp() int                         { return 0 }
func (m *mockGameAttribute) GetExpPoints() int                          { return 0 }

type mockSkill struct {
	valueForTest int
	level        int
}

func (m *mockSkill) GetValueForTest() int                               { return m.valueForTest }
func (m *mockSkill) GetLevel() int                                      { return m.level }
func (m *mockSkill) CascadeUpgradeTrigger(_ *experience.UpgradeCascade) {}
func (m *mockSkill) GetNextLvlAggregateExp() int                        { return 0 }
func (m *mockSkill) GetNextLvlBaseExp() int                             { return 0 }
func (m *mockSkill) GetCurrentExp() int                                 { return 0 }
func (m *mockSkill) GetExpPoints() int                                  { return 0 }

// --- HealthPoints Tests ---

func TestHealthPoints_Formula(t *testing.T) {
	tests := []struct {
		name          string
		abilityBonus  float64
		vitalityLevel int
		resistanceVal int
		wantMax       int
	}{
		{
			"zero values",
			0.0, 0, 0,
			20,
		},
		{
			"with vitality and resistance",
			2.0, 3, 2,
			20 + int(float64(3+2)*2.0),
		},
		{
			"high values",
			5.0, 10, 5,
			20 + int(float64(10+5)*5.0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			physicals := &mockAbility{bonus: tt.abilityBonus}
			resistance := &mockDistributableAttribute{value: tt.resistanceVal}
			vitality := &mockSkill{level: tt.vitalityLevel}

			hp := status.NewHealthPoints(physicals, resistance, vitality)

			if hp.GetMax() != tt.wantMax {
				t.Errorf("HP max: got %d, want %d", hp.GetMax(), tt.wantMax)
			}
			if hp.GetCurrent() != tt.wantMax {
				t.Errorf("HP current should equal max initially: got %d, want %d",
					hp.GetCurrent(), tt.wantMax)
			}
		})
	}
}

// --- StaminaPoints Tests ---

func TestStaminaPoints_Formula(t *testing.T) {
	tests := []struct {
		name          string
		abilityBonus  float64
		energyLevel   int
		resistanceVal int
		wantMax       int
	}{
		{
			"zero values",
			0.0, 0, 0,
			0,
		},
		{
			"with energy and resistance",
			2.0, 3, 2,
			10 * int(float64(3+2)*2.0),
		},
		{
			"high values",
			5.0, 10, 5,
			10 * int(float64(10+5)*5.0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			physicals := &mockAbility{bonus: tt.abilityBonus}
			resistance := &mockDistributableAttribute{value: tt.resistanceVal}
			energy := &mockSkill{level: tt.energyLevel}

			sp := status.NewStaminaPoints(physicals, resistance, energy)

			if sp.GetMax() != tt.wantMax {
				t.Errorf("SP max: got %d, want %d", sp.GetMax(), tt.wantMax)
			}
			if sp.GetCurrent() != tt.wantMax {
				t.Errorf("SP current should equal max initially: got %d, want %d",
					sp.GetCurrent(), tt.wantMax)
			}
		})
	}
}

// --- AuraPoints Tests ---

func TestAuraPoints_Formula(t *testing.T) {
	tests := []struct {
		name             string
		abilityBonus     float64
		mopLevel         int
		conscienceNenLvl int
		wantMax          int
	}{
		{
			"zero values",
			0.0, 0, 0,
			0,
		},
		{
			"with mop and conscience",
			5.0, 3, 2,
			int(10 * float64(3+2) * float64(int(5.0))),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spirituals := &mockAbility{bonus: tt.abilityBonus}
			conscienceNen := &mockGameAttribute{level: tt.conscienceNenLvl}
			mop := &mockSkill{level: tt.mopLevel}

			ap, err := status.NewAuraPoints(spirituals, conscienceNen, mop)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ap.GetMax() != tt.wantMax {
				t.Errorf("AP max: got %d, want %d", ap.GetMax(), tt.wantMax)
			}
		})
	}
}

func TestAuraPoints_NilSpiritual_ReturnsError(t *testing.T) {
	conscienceNen := &mockGameAttribute{level: 1}
	mop := &mockSkill{level: 1}

	_, err := status.NewAuraPoints(nil, conscienceNen, mop)
	if err == nil {
		t.Fatal("expected error for nil spiritual ability")
	}
}

func TestHealthPoints_UpgradeKeepsFullHP(t *testing.T) {
	physicals := &mockAbility{bonus: 2.0}
	resistance := &mockDistributableAttribute{value: 1}
	vitality := &mockSkill{level: 1}

	hp := status.NewHealthPoints(physicals, resistance, vitality)
	initialMax := hp.GetMax()

	if hp.GetCurrent() != initialMax {
		t.Fatalf("current should be max initially")
	}

	hp.Upgrade()

	if hp.GetCurrent() != hp.GetMax() {
		t.Errorf("after Upgrade with full HP, current should still equal max: got %d, want %d",
			hp.GetCurrent(), hp.GetMax())
	}
}

// --- HealthPoints negative minimum ---

func TestHealthPoints_NegativeMax(t *testing.T) {
	// ⌊10 + nível de Vitality + valor de Resistance + bônus de Physicals⌋
	physicals := &mockAbility{bonus: 4.5}
	resistance := &mockDistributableAttribute{value: 3}
	vitality := &mockSkill{level: 2}

	hp := status.NewHealthPoints(physicals, resistance, vitality)

	if got := hp.NegativeMax(); got != 19 {
		t.Errorf("NegativeMax: got %d, want 19", got)
	}
	if got := hp.GetMin(); got != -19 {
		t.Errorf("GetMin: got %d, want -19", got)
	}
}

func TestHealthPoints_NegativeMaxFollowsUpgrade(t *testing.T) {
	physicals := &mockAbility{bonus: 1}
	resistance := &mockDistributableAttribute{value: 1}
	vitality := &mockSkill{level: 1}
	hp := status.NewHealthPoints(physicals, resistance, vitality)
	if hp.GetMin() != -13 {
		t.Fatalf("initial min: got %d, want -13", hp.GetMin())
	}

	vitality.level = 5
	hp.Upgrade()
	if hp.GetMin() != -17 {
		t.Errorf("min after Upgrade: got %d, want -17", hp.GetMin())
	}
}

func TestHealthPoints_BelowMin(t *testing.T) {
	newHP := func() status.IStatusBar {
		return status.NewHealthPoints(
			&mockAbility{bonus: 1},
			&mockDistributableAttribute{value: 1},
			&mockSkill{level: 1},
		)
	}

	t.Run("health decreases past min", func(t *testing.T) {
		hp := newHP() // min -13
		if got := hp.DecreaseAt(100); got != hp.GetCurrent() || got >= hp.GetMin() {
			t.Errorf("curr %d should sit below min %d", got, hp.GetMin())
		}
		if got := hp.GetCurrent(); got != 20+2-100 {
			t.Errorf("curr: got %d, want %d", got, 20+2-100)
		}
	})

	t.Run("health set below min", func(t *testing.T) {
		hp := newHP()
		if err := hp.SetCurrent(-40); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hp.GetCurrent() != -40 {
			t.Errorf("curr: got %d, want -40", hp.GetCurrent())
		}
		if err := hp.SetCurrent(hp.GetMax() + 1); err == nil {
			t.Error("setting above max should fail")
		}
	})

	t.Run("stamina still floors at zero", func(t *testing.T) {
		sp := status.NewStaminaPoints(
			&mockAbility{bonus: 1},
			&mockDistributableAttribute{value: 1},
			&mockSkill{level: 1},
		)
		sp.DecreaseAt(1_000_000)
		if sp.GetCurrent() != 0 {
			t.Errorf("stamina curr: got %d, want 0", sp.GetCurrent())
		}
		if err := sp.SetCurrent(-1); err == nil {
			t.Error("stamina below min should fail")
		}
	})
}
