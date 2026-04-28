package spiritual_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/spiritual"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func TestNenHexagon_NewWithNilCategory(t *testing.T) {
	tests := []struct {
		name     string
		hexValue int
		wantCat  enum.CategoryName
	}{
		{"hex 0 → Reinforcement", 0, enum.Reinforcement},
		{"hex 49 → Reinforcement", 49, enum.Reinforcement},
		{"hex 100 → Transmutation", 100, enum.Transmutation},
		{"hex 200 → Materialization", 200, enum.Materialization},
		{"hex 300 → Specialization", 300, enum.Specialization},
		{"hex 400 → Manipulation", 400, enum.Manipulation},
		{"hex 500 → Emission", 500, enum.Emission},
		{"hex 599 → Reinforcement (wraps)", 599, enum.Reinforcement},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nh := spiritual.NewNenHexagon(tt.hexValue, nil)
			if got := nh.GetCategoryName(); got != tt.wantCat {
				t.Errorf("category = %s, want %s", got, tt.wantCat)
			}
		})
	}
}

func TestNenHexagon_NewWithExplicitCategory(t *testing.T) {
	cat := enum.Transmutation
	nh := spiritual.NewNenHexagon(0, &cat)
	if got := nh.GetCategoryName(); got != enum.Transmutation {
		t.Errorf("category = %s, want %s", got, enum.Transmutation)
	}
}

func TestNenHexagon_ModuloWrapping(t *testing.T) {
	nh := spiritual.NewNenHexagon(700, nil)
	if got := nh.GetCurrHexValue(); got != 100 {
		t.Errorf("hex value = %d, want 100 (700 mod 600)", got)
	}
}

func TestNenHexagon_IncreaseCurrHexValue(t *testing.T) {
	nh := spiritual.NewNenHexagon(0, nil)

	result := nh.IncreaseCurrHexValue()
	if result.CurrentHexVal != 1 {
		t.Errorf("hex after increase = %d, want 1", result.CurrentHexVal)
	}

	// Test wrap around at 599
	nh2 := spiritual.NewNenHexagon(599, nil)
	result2 := nh2.IncreaseCurrHexValue()
	if result2.CurrentHexVal != 0 {
		t.Errorf("hex after wrap = %d, want 0", result2.CurrentHexVal)
	}
}

func TestNenHexagon_DecreaseCurrHexValue(t *testing.T) {
	nh := spiritual.NewNenHexagon(1, nil)
	result := nh.DecreaseCurrHexValue()
	if result.CurrentHexVal != 0 {
		t.Errorf("hex after decrease = %d, want 0", result.CurrentHexVal)
	}

	// Test wrap around at 0
	nh2 := spiritual.NewNenHexagon(0, nil)
	result2 := nh2.DecreaseCurrHexValue()
	if result2.CurrentHexVal != 599 {
		t.Errorf("hex after wrap = %d, want 599", result2.CurrentHexVal)
	}
}

func TestNenHexagon_GetPercentOf(t *testing.T) {
	// At position 0 (Reinforcement center)
	nh := spiritual.NewNenHexagon(0, nil)

	t.Run("own category = 100%", func(t *testing.T) {
		pct := nh.GetPercentOf(enum.Reinforcement)
		if pct != 100.0 {
			t.Errorf("Reinforcement%% = %f, want 100.0", pct)
		}
	})

	t.Run("adjacent category = 80%", func(t *testing.T) {
		pct := nh.GetPercentOf(enum.Transmutation)
		if pct != 80.0 {
			t.Errorf("Transmutation%% = %f, want 80.0", pct)
		}
	})

	t.Run("opposite category = 40%", func(t *testing.T) {
		// Specialization returns 0 if not the current category
		pct := nh.GetPercentOf(enum.Specialization)
		if pct != 0.0 {
			t.Errorf("Specialization%% (not current) = %f, want 0.0", pct)
		}
	})

	t.Run("emission = 80% (symmetric)", func(t *testing.T) {
		pct := nh.GetPercentOf(enum.Emission)
		if pct != 80.0 {
			t.Errorf("Emission%% = %f, want 80.0", pct)
		}
	})
}

func TestNenHexagon_SpecializationPercentOnlyWhenCurrent(t *testing.T) {
	// At Reinforcement center: Specialization returns 0
	nh1 := spiritual.NewNenHexagon(0, nil)
	if pct := nh1.GetPercentOf(enum.Specialization); pct != 0.0 {
		t.Errorf("Specialization when not current = %f, want 0.0", pct)
	}

	// At Specialization center: Specialization returns 100
	nh2 := spiritual.NewNenHexagon(300, nil)
	if pct := nh2.GetPercentOf(enum.Specialization); pct != 100.0 {
		t.Errorf("Specialization when current = %f, want 100.0", pct)
	}
}

func TestNenHexagon_ResetCategory(t *testing.T) {
	cat := enum.Transmutation
	nh := spiritual.NewNenHexagon(120, &cat) // 20 off center of Transmutation(100)

	resetVal := nh.ResetCategory()
	if resetVal != 100 {
		t.Errorf("reset value = %d, want 100 (Transmutation center)", resetVal)
	}
}

func TestNenHexagon_GetCategoryPercents(t *testing.T) {
	nh := spiritual.NewNenHexagon(0, nil)
	percents := nh.GetCategoryPercents()

	if len(percents) != 6 {
		t.Errorf("percents count = %d, want 6", len(percents))
	}
	// Reinforcement should be 100% at position 0
	if percents[enum.Reinforcement] != 100.0 {
		t.Errorf("Reinforcement%% = %f, want 100.0", percents[enum.Reinforcement])
	}
}
