package die_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/die"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func TestNewDie(t *testing.T) {
	tests := []struct {
		name          string
		sides         enum.DieSides
		expectedSides int
	}{
		{"D4", enum.D4, 4},
		{"D6", enum.D6, 6},
		{"D8", enum.D8, 8},
		{"D10", enum.D10, 10},
		{"D12", enum.D12, 12},
		{"D20", enum.D20, 20},
		{"D100", enum.D100, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := die.NewDie(tt.sides)
			if d.GetSides() != tt.expectedSides {
				t.Errorf("GetSides() = %d, want %d", d.GetSides(), tt.expectedSides)
			}
			if d.GetResult() != 0 {
				t.Errorf("GetResult() before Roll() = %d, want 0", d.GetResult())
			}
		})
	}
}

func TestDie_Roll(t *testing.T) {
	tests := []struct {
		name  string
		sides enum.DieSides
		max   int
	}{
		{"D4 produces 1-4", enum.D4, 4},
		{"D6 produces 1-6", enum.D6, 6},
		{"D8 produces 1-8", enum.D8, 8},
		{"D10 produces 1-10", enum.D10, 10},
		{"D12 produces 1-12", enum.D12, 12},
		{"D20 produces 1-20", enum.D20, 20},
		{"D100 produces 1-100", enum.D100, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := die.NewDie(tt.sides)
			for i := 0; i < 100; i++ {
				result := d.Roll()
				if result < 1 || result > tt.max {
					t.Fatalf("Roll() = %d, want 1-%d", result, tt.max)
				}
				if d.GetResult() != result {
					t.Fatalf("GetResult() = %d, want %d (last Roll)", d.GetResult(), result)
				}
			}
		})
	}
}

func TestDie_Roll_UpdatesResult(t *testing.T) {
	d := die.NewDie(enum.D20)
	first := d.Roll()
	if first < 1 || first > 20 {
		t.Fatalf("first Roll() = %d, out of range", first)
	}
	if d.GetResult() != first {
		t.Fatalf("GetResult() after first Roll() = %d, want %d", d.GetResult(), first)
	}

	second := d.Roll()
	if second < 1 || second > 20 {
		t.Fatalf("second Roll() = %d, out of range", second)
	}
	if d.GetResult() != second {
		t.Fatalf("GetResult() after second Roll() = %d, want %d", d.GetResult(), second)
	}
}
