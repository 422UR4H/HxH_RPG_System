package action_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
)

func TestRollAttempts_IsEmpty(t *testing.T) {
	tests := []struct {
		name string
		a    action.RollAttempts
		want bool
	}{
		{"zero value is empty", action.RollAttempts{}, true},
		{"only primary is not empty", action.RollAttempts{Primary: []int{3, 7}}, false},
		{"only secondary is not empty", action.RollAttempts{Secondary: []int{3, 7}}, false},
		{"both sets is not empty", action.RollAttempts{Primary: []int{1}, Secondary: []int{2}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.IsEmpty(); got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRollCheck_CarriesAttempts(t *testing.T) {
	rc := action.RollCheck{SkillName: "Accuracy"}
	if !rc.Attempts.IsEmpty() {
		t.Fatal("a fresh RollCheck must carry no dice")
	}
	rc.Attempts = action.RollAttempts{Primary: []int{10, 10}}
	if rc.Attempts.IsEmpty() {
		t.Error("expected the RollCheck to hold the dice it was given")
	}
}
