package action_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/google/uuid"
)

// stackedMove é um movimento puro empilhado sobre outro (D3): StackedOn aponta o anterior.
func stackedMove(speed int, on *action.Action) *action.Action {
	return action.NewAction(uuid.New(), nil, uuid.Nil, nil, action.ActionSpeed{},
		nil, &action.Move{Category: enum.Dash, FinalSpeed: speed, StackedOn: on.GetID()},
		nil, nil, nil, nil, nil)
}

func TestAction_IsBlockedIn(t *testing.T) {
	head := moveOnly(10)
	tail := stackedMove(20, head)

	t.Run("a stacked move is blocked while its predecessor is pending", func(t *testing.T) {
		if !tail.IsBlockedIn([]*action.Action{head, tail}) {
			t.Error("the predecessor is still pending: the stacked move must be blocked")
		}
	})
	t.Run("it is free once its predecessor left the pending list", func(t *testing.T) {
		if tail.IsBlockedIn([]*action.Action{tail}) {
			t.Error("the predecessor is gone (opened, consumed): the stacked move is a normal candidate")
		}
	})
	t.Run("an unstacked action is never blocked", func(t *testing.T) {
		if head.IsBlockedIn([]*action.Action{head, tail}) {
			t.Error("the head of a chain is never blocked")
		}
		if attackOnly(5).IsBlockedIn([]*action.Action{head}) {
			t.Error("a non-move action never stacks")
		}
	})
}

// D4 no Free: um movimento empilhado nunca sai antes do anterior, mesmo mais rápido.
func TestPriorityQueue_ExtractMax_SkipsStackedMoves(t *testing.T) {
	pq := action.NewActionPriorityQueue(nil)
	head := moveOnly(10)
	tail := stackedMove(30, head)
	other := attackOnly(15)
	pq.Insert(head)
	pq.Insert(tail)
	pq.Insert(other)

	if got := pq.Peek(); got != other {
		t.Fatalf("Peek: want the attack (15) — the 30 is stacked on a pending move")
	}
	for i, want := range []*action.Action{other, head, tail} {
		if got := pq.ExtractMax(); got != want {
			t.Fatalf("extract #%d: wrong action (OrderSpeed %d)", i, got.OrderSpeed())
		}
	}
}
