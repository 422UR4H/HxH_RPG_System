package action_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/google/uuid"
)

func attackOnly(speed int) *action.Action {
	return action.NewAction(uuid.New(), nil, uuid.Nil, nil,
		action.ActionSpeed{RollCheck: action.RollCheck{Result: speed}},
		nil, nil, &action.Attack{}, nil, nil, nil, nil)
}

func moveOnly(speed int) *action.Action {
	return action.NewAction(uuid.New(), nil, uuid.Nil, nil, action.ActionSpeed{},
		nil, &action.Move{Category: enum.Dash, FinalSpeed: speed}, nil, nil, nil, nil, nil)
}

func combined(actionSpeed, moveSpeed int) *action.Action {
	return action.NewAction(uuid.New(), nil, uuid.Nil, nil,
		action.ActionSpeed{RollCheck: action.RollCheck{Result: actionSpeed}},
		nil, &action.Move{Category: enum.Dash, FinalSpeed: moveSpeed},
		&action.Attack{}, nil, nil, nil, nil)
}

func TestAction_Bars(t *testing.T) {
	tests := []struct {
		name string
		a    *action.Action
		want []action.Bar
	}{
		{"an attack pays from the action bar", attackOnly(10), []action.Bar{action.BarAction}},
		{"a movement pays from the move bar", moveOnly(10), []action.Bar{action.BarMove}},
		{
			"an investida is ONE action that pays from both",
			combined(25, 5),
			[]action.Bar{action.BarAction, action.BarMove},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.Bars()
			if len(got) != len(tt.want) {
				t.Fatalf("Bars() = %v, want %v", got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("Bars()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}

	t.Run("an action with nothing in it still pays from the action bar", func(t *testing.T) {
		a := action.NewAction(uuid.New(), nil, uuid.Nil, nil, action.ActionSpeed{},
			nil, nil, nil, nil, nil, nil, nil)
		if bars := a.Bars(); len(bars) != 1 || bars[0] != action.BarAction {
			t.Errorf("Bars() = %v, want just the action bar — never an empty set, or it would be unschedulable", bars)
		}
	})
}

func TestAction_SpeedOn(t *testing.T) {
	t.Run("each bar reads its own speed", func(t *testing.T) {
		a := combined(25, 5)
		if got := a.SpeedOn(action.BarAction); got != 25 {
			t.Errorf("SpeedOn(action) = %d, want 25", got)
		}
		if got := a.SpeedOn(action.BarMove); got != 5 {
			t.Errorf("SpeedOn(move) = %d, want 5", got)
		}
	})

	t.Run("a bar the action does not charge reads zero", func(t *testing.T) {
		if got := attackOnly(10).SpeedOn(action.BarMove); got != 0 {
			t.Errorf("SpeedOn(move) = %d, want 0 — this action does not move", got)
		}
	})
}
