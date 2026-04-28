package spiritual_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/spiritual"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func newTestHatsu() *spiritual.Hatsu {
	table := experience.NewDefaultExpTable()
	exp := *experience.NewExperience(table)
	flame := &mockGameAttribute{level: 1}
	conscience := &mockGameAttribute{power: 2}

	percents := map[enum.CategoryName]float64{
		enum.Reinforcement:   100.0,
		enum.Transmutation:   80.0,
		enum.Materialization: 60.0,
		enum.Specialization:  0.0,
		enum.Manipulation:    60.0,
		enum.Emission:        80.0,
	}
	return spiritual.NewHatsu(exp, flame, conscience, nil, percents)
}

func TestHatsu_Init(t *testing.T) {
	h := newTestHatsu()
	table := experience.NewDefaultExpTable()

	categories := make(map[enum.CategoryName]spiritual.NenCategory)
	for _, name := range enum.AllNenCategoryNames() {
		exp := *experience.NewExperience(table)
		categories[name] = *spiritual.NewNenCategory(exp, name, h)
	}

	if err := h.Init(categories); err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	// Double init should fail
	if err := h.Init(categories); err == nil {
		t.Error("double Init() should return error")
	}
}

func TestHatsu_SetCategoryPercents(t *testing.T) {
	h := newTestHatsu()

	t.Run("valid 6 categories", func(t *testing.T) {
		percents := map[enum.CategoryName]float64{
			enum.Reinforcement:   90.0,
			enum.Transmutation:   70.0,
			enum.Materialization: 50.0,
			enum.Specialization:  0.0,
			enum.Manipulation:    50.0,
			enum.Emission:        70.0,
		}
		if err := h.SetCategoryPercents(percents); err != nil {
			t.Fatalf("SetCategoryPercents error: %v", err)
		}
	})

	t.Run("invalid count", func(t *testing.T) {
		percents := map[enum.CategoryName]float64{
			enum.Reinforcement: 100.0,
		}
		if err := h.SetCategoryPercents(percents); err == nil {
			t.Error("should reject non-6 category percents")
		}
	})
}

func TestHatsu_GetValueForTest(t *testing.T) {
	h := newTestHatsu()
	// valueForTest = hatsuLevel(0) + consciencePower(2) + flameLevel(1) = 3
	if got := h.GetValueForTest(); got != 3 {
		t.Errorf("GetValueForTest() = %d, want 3", got)
	}
}

func TestHatsu_GetPercentOf(t *testing.T) {
	h := newTestHatsu()
	if pct := h.GetPercentOf(enum.Reinforcement); pct != 100.0 {
		t.Errorf("Reinforcement%% = %f, want 100.0", pct)
	}
	if pct := h.GetPercentOf(enum.Transmutation); pct != 80.0 {
		t.Errorf("Transmutation%% = %f, want 80.0", pct)
	}
}
