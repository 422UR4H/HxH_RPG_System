package service_test

import (
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

func TestResolve_ListsTheConditionsInForceWhileTheTurnIsOpen(t *testing.T) {
	build := func() (*turn.Turn, *action.Action, *action.Action) {
		a := action.NewAction(uuid.New(), nil, uuid.Nil, nil, action.ActionSpeed{},
			nil, nil, &action.Attack{}, nil, nil, nil, nil)
		a.Attack.Hit.Context.Condition = &action.RollCondition{Bias: -1, Modifier: -2, Description: "escuridao"}
		tn := turn.NewTurn(*a)
		r := reactionWith(action.ReactDodge, []int{5, 5}, nil, nil)
		r.Dodge.Context.Condition = &action.RollCondition{Bias: 1}
		r.DefaultDefense = &action.RollCheck{
			Context: action.RollContext{Condition: &action.RollCondition{Modifier: 2}},
		}
		tn.AddReaction(r)
		return tn, a, r
	}

	t.Run("open: every check with a condition, action first, then each reaction", func(t *testing.T) {
		tn, a, r := build()
		res := service.TurnResolver{}.Resolve(resolveWith(tn, noopTargetReader{}))
		want := []service.CheckCondition{
			{ActionID: a.GetID(), Field: "hit", Condition: action.RollCondition{Bias: -1, Modifier: -2, Description: "escuridao"}},
			{ActionID: r.GetID(), Field: "dodge", Condition: action.RollCondition{Bias: 1}},
			{ActionID: r.GetID(), Field: "defense", Condition: action.RollCondition{Modifier: 2}},
		}
		if len(res.Conditions) != len(want) {
			t.Fatalf("Conditions = %+v, want %+v", res.Conditions, want)
		}
		for i := range want {
			if res.Conditions[i] != want[i] {
				t.Errorf("Conditions[%d] = %+v, want %+v", i, res.Conditions[i], want[i])
			}
		}
	})

	t.Run("settled: none — the conditions live in the actions table, not in the record", func(t *testing.T) {
		tn, _, _ := build()
		tn.Close(time.Now())
		res := service.TurnResolver{}.Resolve(resolveWith(tn, noopTargetReader{}))
		if res.Conditions != nil {
			t.Fatalf("Conditions = %+v on a settled turn, want nil", res.Conditions)
		}
	})
}
