package game

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

func TestNewResolutionUpdatedPayload(t *testing.T) {
	turnID, targetID := uuid.New(), uuid.New()
	margin := 7
	res := &service.TurnResolution{
		IsSettled: false,
		ActionResult: service.RollResult{
			SkillName:  "Accuracy",
			SkillValue: 4,
			DiceRolled: []int{10, 8},
			Total:      22,
			IsCritical: false,
			Margin:     &margin,
		},
		CharacterResults: []service.CharacterResult{{
			TargetID:        targetID,
			Dodged:          false,
			Defended:        true,
			RawDamage:       14,
			EffectiveDamage: 14,
		}},
	}

	p := newResolutionUpdatedPayload(turnID, res)

	t.Run("carries the turn and the action roll", func(t *testing.T) {
		if p.TurnID != turnID {
			t.Errorf("TurnID = %v, want %v", p.TurnID, turnID)
		}
		if p.Action.Total != 22 || len(p.Action.DiceRolled) != 2 {
			t.Errorf("Action = %+v, want total 22 and the two dice", p.Action)
		}
		if p.Action.Margin == nil || *p.Action.Margin != 7 {
			t.Errorf("Margin = %v, want 7", p.Action.Margin)
		}
	})

	t.Run("carries the projected damage per target", func(t *testing.T) {
		if len(p.Targets) != 1 {
			t.Fatalf("Targets = %+v, want one entry", p.Targets)
		}
		if p.Targets[0].ProjectedDamage != 14 || !p.Targets[0].Defended {
			t.Errorf("Targets[0] = %+v", p.Targets[0])
		}
	})

	t.Run("serializes as camelCase", func(t *testing.T) {
		b, err := json.Marshal(p)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		s := string(b)
		for _, key := range []string{`"turnId"`, `"isSettled"`, `"diceRolled"`, `"projectedDamage"`} {
			if !strings.Contains(s, key) {
				t.Errorf("payload is missing %s: %s", key, s)
			}
		}
	})
}

func TestNewResolutionUpdatedPayload_NilResolution(t *testing.T) {
	turnID := uuid.New()
	p := newResolutionUpdatedPayload(turnID, nil)
	if p.TurnID != turnID {
		t.Errorf("TurnID = %v, want %v", p.TurnID, turnID)
	}
	if len(p.Targets) != 0 {
		t.Errorf("Targets = %+v, want empty", p.Targets)
	}
	// An empty slice rather than null, so a client can iterate it unconditionally.
	b, _ := json.Marshal(p)
	if !strings.Contains(string(b), `"targets":[]`) {
		t.Errorf("expected an empty array for targets, got %s", string(b))
	}
}
