package charactersheet

import "testing"

func TestNormalizeStatus(t *testing.T) {
	tests := []struct {
		name          string
		curr          int
		oldMax        int
		newMax        int
		minVal        int
		wantCurr      int
		wantCorrected bool
	}{
		{"no correction - curr within new max", 70, 100, 90, 0, 70, false},
		{"no correction - curr equals new max", 90, 100, 90, 0, 90, false},
		{"no correction - newMax is zero", 80, 100, 0, 0, 80, false},
		{"no correction - curr is zero", 0, 100, 90, 0, 0, false},
		{"proportional correction", 90, 100, 80, 0, 72, true},
		{"fully healed correction", 100, 100, 90, 0, 90, true},
		{"oldMax zero fallback returns newMax", 100, 0, 90, 0, 90, true},
		{"result clamped to newMax", 100, 100, 50, 0, 50, true},
		{"minVal clamp applied", 81, 1000, 80, 10, 10, true},
		{"proportional with rounding up", 91, 100, 80, 0, 73, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCurr, gotCorrected := normalizeStatus(tt.curr, tt.oldMax, tt.newMax, tt.minVal)
			if gotCurr != tt.wantCurr {
				t.Errorf("curr = %d, want %d", gotCurr, tt.wantCurr)
			}
			if gotCorrected != tt.wantCorrected {
				t.Errorf("corrected = %v, want %v", gotCorrected, tt.wantCorrected)
			}
		})
	}
}
