package service_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
)

func TestClimbLadder(t *testing.T) {
	const step = 10

	tests := []struct {
		name     string
		margin   int
		wantRung service.LadderRung
		wantDiff int
	}{
		{"a full step over is a great success", 10, service.RungGreatSuccess, 10},
		{"well over is a great success", 27, service.RungGreatSuccess, 27},
		{"one under the step is a plain success", 9, service.RungSuccess, 0},
		{"landing exactly on the CD is a success", 0, service.RungSuccess, 0},
		{"one under the CD is a near miss", -1, service.RungNearMiss, 1},
		{"a full step under is still a near miss", -10, service.RungNearMiss, 10},
		{"more than a step under is a failure", -11, service.RungFailure, 0},
		{"far under is a failure", -40, service.RungFailure, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.ClimbLadder(tt.margin, step)
			if got.Rung != tt.wantRung {
				t.Errorf("Rung = %q, want %q", got.Rung, tt.wantRung)
			}
			if got.Difference != tt.wantDiff {
				t.Errorf("Difference = %d, want %d", got.Difference, tt.wantDiff)
			}
			if got.Margin != tt.margin {
				t.Errorf("Margin = %d, want %d", got.Margin, tt.margin)
			}
		})
	}
}

func TestClimbLadder_StepIsAMatchRule(t *testing.T) {
	// The step size is configurable; the shape of the ladder is not. With a step of 5 the
	// same margin lands on a different rung.
	if got := service.ClimbLadder(6, 5); got.Rung != service.RungGreatSuccess {
		t.Errorf("with step 5, margin 6 should be a great success, got %q", got.Rung)
	}
	if got := service.ClimbLadder(6, 10); got.Rung != service.RungSuccess {
		t.Errorf("with step 10, margin 6 should be a plain success, got %q", got.Rung)
	}
}

func TestClimbLadder_NonPositiveStepDegradesToPassFail(t *testing.T) {
	// Defensive: a zero step must not collapse every margin onto one rung silently.
	if got := service.ClimbLadder(3, 0); got.Rung != service.RungGreatSuccess {
		t.Errorf("with step 0, any success is a great success, got %q", got.Rung)
	}
	if got := service.ClimbLadder(-3, 0); got.Rung != service.RungFailure {
		t.Errorf("with step 0, any miss is a failure, got %q", got.Rung)
	}
}
