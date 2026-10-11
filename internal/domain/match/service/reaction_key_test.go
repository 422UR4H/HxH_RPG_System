package service

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
)

// The key the master reads beside a pending reaction: its speed on the bars it charges. A dodge
// charges none and rolls no speed — no key, never a misleading zero.
func TestReactionKey(t *testing.T) {
	dodge := action.Action{ReactionKind: action.ReactDodge}
	if got := reactionKey(dodge); got != nil {
		t.Errorf("dodge key = %d, want none", *got)
	}
	repel := action.Action{ReactionKind: action.ReactRepel, Speed: action.ActionSpeed{RollCheck: action.RollCheck{Result: 14}}}
	if got := reactionKey(repel); got == nil || *got != 14 {
		t.Errorf("repel key = %v, want 14", got)
	}
}
