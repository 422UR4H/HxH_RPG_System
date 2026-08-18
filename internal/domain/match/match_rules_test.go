package match_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
)

func TestNewDefaultMatchRules(t *testing.T) {
	r := match.NewDefaultMatchRules()

	if r.DiceSet != match.DiceSet2D10 {
		t.Errorf("expected default dice set 2d10, got %q", r.DiceSet)
	}
	if r.LadderStep != 10 {
		t.Errorf("expected ladder step 10, got %d", r.LadderStep)
	}
	if r.ReactionTimer != nil {
		t.Error("expected reaction timer off by default")
	}
	if !r.DefaultReactions {
		t.Error("expected default reactions on")
	}
	if r.FogMode != nil {
		t.Error("expected nil FogMode by default (inherits from the map)")
	}
}

func TestDiceSet(t *testing.T) {
	tests := []struct {
		name    string
		set     match.DiceSet
		dice    []enum.DieSides
		passive int
		maxFace int
	}{
		{"2d10", match.DiceSet2D10, []enum.DieSides{enum.D10, enum.D10}, 11, 10},
		{"d20", match.DiceSetD20, []enum.DieSides{enum.D20}, 10, 20},
		{"unknown falls back to 2d10", match.DiceSet("nonsense"), []enum.DieSides{enum.D10, enum.D10}, 11, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.set.Dice()
			if len(got) != len(tt.dice) {
				t.Fatalf("expected %d dice, got %d", len(tt.dice), len(got))
			}
			for i := range got {
				if got[i] != tt.dice[i] {
					t.Errorf("die %d: expected %v, got %v", i, tt.dice[i], got[i])
				}
			}
			if tt.set.PassiveValue() != tt.passive {
				t.Errorf("expected passive %d, got %d", tt.passive, tt.set.PassiveValue())
			}
			if tt.set.MaxFace() != tt.maxFace {
				t.Errorf("expected max face %d, got %d", tt.maxFace, tt.set.MaxFace())
			}
		})
	}
}

func TestMatchRules_PassiveValueFollowsDiceSet(t *testing.T) {
	r := match.NewDefaultMatchRules()
	if r.PassiveValue() != 11 {
		t.Errorf("expected 11 for 2d10, got %d", r.PassiveValue())
	}

	// Swapping the dice set must move the passive value with it — that is the whole
	// reason PassiveValue is derived instead of stored.
	r.DiceSet = match.DiceSetD20
	if r.PassiveValue() != 10 {
		t.Errorf("expected 10 for d20, got %d", r.PassiveValue())
	}
}

func TestMatchRules_ResolveFogMode(t *testing.T) {
	live := fog.FogModeLive

	t.Run("match rule wins when set", func(t *testing.T) {
		r := match.NewDefaultMatchRules()
		r.FogMode = &live
		if got := r.ResolveFogMode(fog.FogModeExplored); got != fog.FogModeLive {
			t.Errorf("expected live, got %q", got)
		}
	})

	t.Run("inherits the map when unset", func(t *testing.T) {
		r := match.NewDefaultMatchRules()
		if got := r.ResolveFogMode(fog.FogModeLive); got != fog.FogModeLive {
			t.Errorf("expected live from the map, got %q", got)
		}
	})

	t.Run("falls back to explored when neither is valid", func(t *testing.T) {
		r := match.NewDefaultMatchRules()
		if got := r.ResolveFogMode(fog.FogMode("")); got != fog.FogModeExplored {
			t.Errorf("expected explored, got %q", got)
		}
	})
}
