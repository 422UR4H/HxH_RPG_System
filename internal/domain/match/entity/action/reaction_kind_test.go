package action_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/google/uuid"
)

func TestReactionKind_Bars(t *testing.T) {
	tests := []struct {
		kind action.ReactionKind
		want []action.Bar
		why  string
	}{
		{action.ReactRepel, []action.Bar{action.BarAction}, "repel spends the action, never the feet"},
		{action.ReactClosedEscape, []action.Bar{action.BarMove}, "done in the exact instant, without opening the guard — the action comes back"},
		{action.ReactEscape, []action.Bar{action.BarAction, action.BarMove}, "forcing the dodge by displacing costs both"},
		{action.ReactEscapeGuard, []action.Bar{action.BarAction, action.BarMove}, "same price as the standard escape; it only keeps the safety net"},
		{action.ReactClosedDodge, nil, "free — that is what makes it worth the trouble to configure"},
		{action.ReactDodge, nil, "gambling the roll instead of the average costs nothing"},
		{action.ReactNothing, nil, "taking the blow raw costs no bar; it costs HP"},
	}
	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			got := tt.kind.Bars()
			if len(got) != len(tt.want) {
				t.Fatalf("Bars() = %v, want %v — %s", got, tt.want, tt.why)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("Bars() = %v, want %v — %s", got, tt.want, tt.why)
				}
			}
		})
	}
}

func TestAction_BarsAsksTheReactionKindFirst(t *testing.T) {
	t.Run("a free reaction charges nothing even though it carries a dodge", func(t *testing.T) {
		a := action.NewAction(uuid.New(), nil, uuid.New(), nil, action.ActionSpeed{},
			nil, nil, nil, nil, &action.Dodge{}, nil, nil)
		a.ReactionKind = action.ReactClosedDodge
		if got := a.Bars(); len(got) != 0 {
			t.Fatalf("Bars() = %v, want empty — the kind answers, not the shape", got)
		}
	})

	t.Run("the three escapes have the same shape and different prices", func(t *testing.T) {
		build := func(k action.ReactionKind) *action.Action {
			a := action.NewAction(uuid.New(), nil, uuid.New(), nil, action.ActionSpeed{},
				nil, &action.Move{}, nil, nil, &action.Dodge{}, nil, nil)
			a.ReactionKind = k
			return a
		}
		if got := build(action.ReactClosedEscape).Bars(); len(got) != 1 || got[0] != action.BarMove {
			t.Fatalf("closed escape Bars() = %v, want [move]", got)
		}
		if got := build(action.ReactEscape).Bars(); len(got) != 2 {
			t.Fatalf("standard escape Bars() = %v, want [action move]", got)
		}
	})

	t.Run("an action without a reaction kind keeps deriving from its shape", func(t *testing.T) {
		a := action.NewAction(uuid.New(), nil, uuid.Nil, nil, action.ActionSpeed{},
			nil, nil, &action.Attack{}, nil, nil, nil, nil)
		got := a.Bars()
		if len(got) != 1 || got[0] != action.BarAction {
			t.Fatalf("Bars() = %v, want [action] — scheduled actions are unchanged", got)
		}
	})
}

func TestReactionKindFrom(t *testing.T) {
	if _, err := action.ReactionKindFrom("closedEscape"); err != nil {
		t.Errorf("closedEscape must be accepted: %v", err)
	}
	if _, err := action.ReactionKindFrom("parry"); err == nil {
		t.Error("an unknown kind must be refused at the boundary, not defaulted")
	}
	if _, err := action.ReactionKindFrom(""); err == nil {
		t.Error("an empty kind must be refused — the server never infers cost from shape")
	}
}
