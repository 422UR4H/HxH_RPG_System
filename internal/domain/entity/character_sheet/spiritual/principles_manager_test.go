package spiritual_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/spiritual"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func newTestPrinciplesManager() *spiritual.Manager {
	table := experience.NewDefaultExpTable()
	flame := &mockGameAttribute{level: 1}
	conscience := &mockGameAttribute{power: 2}

	// Build principles
	principles := make(map[enum.PrincipleName]*spiritual.NenPrinciple)
	for _, name := range enum.AllNenPrincipleNames() {
		if name == enum.Hatsu {
			continue
		}
		exp := *experience.NewExperience(table)
		p := spiritual.NewNenPrinciple(name, exp, flame, conscience)
		principles[name] = p
	}

	// Build hatsu + hexagon
	hexagon := spiritual.NewNenHexagon(0, nil) // Reinforcement
	hatsuExp := *experience.NewExperience(table)
	percents := hexagon.GetCategoryPercents()
	hatsu := spiritual.NewHatsu(hatsuExp, flame, conscience, nil, percents)

	categories := make(map[enum.CategoryName]spiritual.NenCategory)
	for _, name := range enum.AllNenCategoryNames() {
		exp := *experience.NewExperience(table)
		categories[name] = *spiritual.NewNenCategory(exp, name, hatsu)
	}
	hatsu.Init(categories)

	return spiritual.NewPrinciplesManager(principles, hexagon, hatsu)
}

func TestPrinciplesManager_Get(t *testing.T) {
	m := newTestPrinciplesManager()

	t.Run("existing principle", func(t *testing.T) {
		p, err := m.Get(enum.Ten)
		if err != nil {
			t.Fatalf("Get(Ten) error: %v", err)
		}
		if p == nil {
			t.Fatal("Get(Ten) returned nil")
		}
	})

	t.Run("hatsu returns hatsu object", func(t *testing.T) {
		p, err := m.Get(enum.Hatsu)
		if err != nil {
			t.Fatalf("Get(Hatsu) error: %v", err)
		}
		if p == nil {
			t.Fatal("Get(Hatsu) returned nil")
		}
	})
}

func TestPrinciplesManager_IncreaseExpByPrinciple(t *testing.T) {
	m := newTestPrinciplesManager()
	values := experience.NewUpgradeCascade(50)

	if err := m.IncreaseExpByPrinciple(enum.Ten, values); err != nil {
		t.Fatalf("IncreaseExpByPrinciple error: %v", err)
	}

	// Verify exp was actually stored (pointer fix)
	expMap := m.GetExpPointsOfPrinciples()
	if expMap[enum.Ten] != 50 {
		t.Errorf("Ten exp points = %d, want 50", expMap[enum.Ten])
	}

	// Verify the cascade was triggered by checking the Principles map
	cascade, ok := values.Principles[enum.Hatsu]
	if !ok {
		t.Fatal("cascade not propagated to Principles[Hatsu]")
	}
	if cascade.Lvl < 0 {
		t.Errorf("cascade Lvl = %d, want >= 0", cascade.Lvl)
	}
}

func TestPrinciplesManager_IncreaseExpByPrinciple_NotFound(t *testing.T) {
	m := newTestPrinciplesManager()
	values := experience.NewUpgradeCascade(50)

	err := m.IncreaseExpByPrinciple("NonExistent", values)
	if err == nil {
		t.Error("expected error for non-existent principle")
	}
}

func TestPrinciplesManager_InitNenHexagon(t *testing.T) {
	table := experience.NewDefaultExpTable()
	flame := &mockGameAttribute{level: 1}
	conscience := &mockGameAttribute{power: 2}

	principles := make(map[enum.PrincipleName]*spiritual.NenPrinciple)
	for _, name := range enum.AllNenPrincipleNames() {
		if name == enum.Hatsu {
			continue
		}
		exp := *experience.NewExperience(table)
		p := spiritual.NewNenPrinciple(name, exp, flame, conscience)
		principles[name] = p
	}

	hatsuExp := *experience.NewExperience(table)
	hatsu := spiritual.NewHatsu(hatsuExp, flame, conscience, nil, nil)

	// Create manager WITHOUT a hexagon
	m := spiritual.NewPrinciplesManager(principles, nil, hatsu)

	t.Run("init hexagon succeeds when nil", func(t *testing.T) {
		hexagon := spiritual.NewNenHexagon(0, nil)
		err := m.InitNenHexagon(hexagon)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
	})

	t.Run("init hexagon fails when already set", func(t *testing.T) {
		hexagon2 := spiritual.NewNenHexagon(100, nil)
		err := m.InitNenHexagon(hexagon2)
		if err == nil {
			t.Fatal("expected ErrNenHexAlreadyInitialized, got nil")
		}
	})
}

func TestPrinciplesManager_HexagonOperations(t *testing.T) {
	m := newTestPrinciplesManager()

	t.Run("initial hex value", func(t *testing.T) {
		val, err := m.GetCurrHexValue()
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if val != 0 {
			t.Errorf("initial hex = %d, want 0", val)
		}
	})

	t.Run("increase hex", func(t *testing.T) {
		result, err := m.IncreaseCurrHexValue()
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if result.CurrentHexVal != 1 {
			t.Errorf("hex after increase = %d, want 1", result.CurrentHexVal)
		}
	})

	t.Run("decrease hex", func(t *testing.T) {
		result, err := m.DecreaseCurrHexValue()
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if result.CurrentHexVal != 0 {
			t.Errorf("hex after decrease = %d, want 0", result.CurrentHexVal)
		}
	})

	t.Run("category name", func(t *testing.T) {
		cat, err := m.GetNenCategoryName()
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if cat != enum.Reinforcement {
			t.Errorf("category = %s, want %s", cat, enum.Reinforcement)
		}
	})
}

func TestPrinciplesManager_BatchGetters(t *testing.T) {
	m := newTestPrinciplesManager()
	levels := m.GetLevelOfPrinciples()

	// Should have 10 principles (all except Hatsu)
	if len(levels) != 10 {
		t.Errorf("principles count = %d, want 10", len(levels))
	}
}
