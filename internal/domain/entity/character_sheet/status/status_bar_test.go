package status_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/status"
)

func TestStatusBar_InitialState(t *testing.T) {
	bar := status.NewStatusBar()

	if bar.GetMin() != 0 {
		t.Errorf("initial min: got %d, want 0", bar.GetMin())
	}
	if bar.GetCurrent() != 0 {
		t.Errorf("initial current: got %d, want 0", bar.GetCurrent())
	}
	if bar.GetMax() != 0 {
		t.Errorf("initial max: got %d, want 0", bar.GetMax())
	}
}

func TestStatusBar_IncreaseAt(t *testing.T) {
	tests := []struct {
		name     string
		increase int
		wantCurr int
	}{
		{"increase within max", 5, 0},
		{"increase exceeds max stays at max", 100, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bar := status.NewStatusBar()
			got := bar.IncreaseAt(tt.increase)

			if got != tt.wantCurr {
				t.Errorf("IncreaseAt(%d) = %d, want %d (max=0)", tt.increase, got, tt.wantCurr)
			}
		})
	}
}

func TestStatusBar_DecreaseAt(t *testing.T) {
	tests := []struct {
		name     string
		decrease int
		wantCurr int
	}{
		{"decrease within min", 5, 0},
		{"decrease below min stays at min", 100, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bar := status.NewStatusBar()
			got := bar.DecreaseAt(tt.decrease)

			if got != tt.wantCurr {
				t.Errorf("DecreaseAt(%d) = %d, want %d (min=0)", tt.decrease, got, tt.wantCurr)
			}
		})
	}
}

func TestStatusBar_SetCurrent_Valid(t *testing.T) {
	bar := status.NewStatusBar()

	err := bar.SetCurrent(0)
	if err != nil {
		t.Fatalf("SetCurrent(0) unexpected error: %v", err)
	}
	if bar.GetCurrent() != 0 {
		t.Errorf("current after SetCurrent(0): got %d, want 0", bar.GetCurrent())
	}
}

func TestStatusBar_SetCurrent_Invalid(t *testing.T) {
	bar := status.NewStatusBar()

	tests := []struct {
		name  string
		value int
	}{
		{"above max", 1},
		{"below min", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := bar.SetCurrent(tt.value)
			if err == nil {
				t.Errorf("SetCurrent(%d) expected error, got nil", tt.value)
			}
		})
	}
}

// TestReconstructBar: a bar rebuilt from its three numbers reads exactly as them, and is its own
// object — what a turn's close hands the sheet gateway after r.mu is released, while the live
// bar goes on belonging to the session.
func TestReconstructBar(t *testing.T) {
	live := status.ReconstructBar(0, 7, 10)
	if live.GetMin() != 0 || live.GetCurrent() != 7 || live.GetMax() != 10 {
		t.Fatalf("ReconstructBar(0, 7, 10) = %d/%d/%d, want 0/7/10", live.GetMin(), live.GetCurrent(), live.GetMax())
	}

	snap := status.ReconstructBar(live.GetMin(), live.GetCurrent(), live.GetMax())
	live.DecreaseAt(5)
	if snap.GetCurrent() != 7 {
		t.Fatalf("the copy moved with the live bar: current = %d, want 7", snap.GetCurrent())
	}
}
