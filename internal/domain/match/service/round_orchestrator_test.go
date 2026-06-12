package service_test

import (
	"errors"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

func TestRoundOrchestrator_NextAction(t *testing.T) {
	orch := service.RoundOrchestrator{}

	t.Run("extracts highest-speed action and appends Turn to Round", func(t *testing.T) {
		r := round.NewRound(enum.Race)
		q := newQueue(makeActionWithSpeed(5), makeActionWithSpeed(10))

		tRn, err := orch.NextAction(r, &q)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tRn == nil {
			t.Fatal("expected non-nil Turn")
		}
		act := tRn.GetAction()
		if act.Speed.Result != 10 {
			t.Errorf("expected speed 10, got %d", act.Speed.Result)
		}
		if r.CurrentTurn() != tRn {
			t.Error("Round should reference the new Turn as CurrentTurn")
		}
	})

	t.Run("closes previous open turn before opening next", func(t *testing.T) {
		r := round.NewRound(enum.Race)
		q := newQueue(makeActionWithSpeed(5), makeActionWithSpeed(10))

		first, _ := orch.NextAction(r, &q)
		if r.HasOpenTurn() == false {
			t.Fatal("first turn should be open")
		}

		second, err := orch.NextAction(r, &q)

		if err != nil {
			t.Fatalf("unexpected error on second call: %v", err)
		}
		if first.GetFinishedAt() == nil {
			t.Error("first turn should be closed after second NextAction")
		}
		if r.CurrentTurn() != second {
			t.Error("Round should reference the second Turn as CurrentTurn")
		}
	})

	t.Run("returns ErrQueueEmpty when queue is empty", func(t *testing.T) {
		r := round.NewRound(enum.Race)
		q := action.NewActionPriorityQueue(nil)

		_, err := orch.NextAction(r, &q)

		if !errors.Is(err, service.ErrQueueEmpty) {
			t.Errorf("expected ErrQueueEmpty, got %v", err)
		}
	})
}

func TestRoundOrchestrator_PullAction(t *testing.T) {
	orch := service.RoundOrchestrator{}

	t.Run("extracts specific action by UUID and appends Turn", func(t *testing.T) {
		r := round.NewRound(enum.Free)
		target := makeActionWithSpeed(7)
		other := makeActionWithSpeed(15)
		q := newQueue(target, other)
		targetID := target.GetID()

		tRn, err := orch.PullAction(r, &q, targetID)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		act := tRn.GetAction()
		if act.GetID() != targetID {
			t.Errorf("expected action %v, got %v", targetID, act.GetID())
		}
	})

	t.Run("returns ErrActionNotFound for unknown UUID", func(t *testing.T) {
		r := round.NewRound(enum.Free)
		q := newQueue(makeActionWithSpeed(5))

		_, err := orch.PullAction(r, &q, uuid.New())

		if !errors.Is(err, service.ErrActionNotFound) {
			t.Errorf("expected ErrActionNotFound, got %v", err)
		}
	})
}

func TestRoundOrchestrator_CloseTurn(t *testing.T) {
	orch := service.RoundOrchestrator{}

	t.Run("sets finishedAt on current turn", func(t *testing.T) {
		r := round.NewRound(enum.Race)
		q := newQueue(makeActionWithSpeed(5))
		orch.NextAction(r, &q) //nolint:errcheck

		at := time.Now()
		closed := orch.CloseTurn(r, at)

		if closed == nil {
			t.Fatal("expected closed turn")
		}
		if closed.GetFinishedAt() == nil {
			t.Error("expected finishedAt to be set")
		}
		if r.HasOpenTurn() {
			t.Error("round should have no open turn after CloseTurn")
		}
	})

	t.Run("returns ErrNoCurrentTurn when round has no turns", func(t *testing.T) {
		r := round.NewRound(enum.Free)
		_, err := orch.CloseTurnErr(r, time.Now())
		if !errors.Is(err, service.ErrNoCurrentTurn) {
			t.Errorf("expected ErrNoCurrentTurn, got %v", err)
		}
	})
}

func TestRoundOrchestrator_AttachReaction(t *testing.T) {
	orch := service.RoundOrchestrator{}

	t.Run("attaches reaction targeting current action", func(t *testing.T) {
		r := round.NewRound(enum.Race)
		q := newQueue(makeActionWithSpeed(5))
		tRn, _ := orch.NextAction(r, &q)
		act := tRn.GetAction()
		actionID := act.GetID()

		reaction := makeReactionTo(actionID)
		err := orch.AttachReaction(r, reaction)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(r.CurrentTurn().GetReactions()) != 1 {
			t.Error("expected 1 reaction on current turn")
		}
	})

	t.Run("returns ErrReactionNotCompatible for wrong target", func(t *testing.T) {
		r := round.NewRound(enum.Race)
		q := newQueue(makeActionWithSpeed(5))
		orch.NextAction(r, &q) //nolint:errcheck

		reaction := makeReactionTo(uuid.New()) // wrong target
		err := orch.AttachReaction(r, reaction)

		if !errors.Is(err, service.ErrReactionNotCompatible) {
			t.Errorf("expected ErrReactionNotCompatible, got %v", err)
		}
	})

	t.Run("returns ErrNoCurrentTurn when round has no turns", func(t *testing.T) {
		r := round.NewRound(enum.Free)
		reaction := makeReactionTo(uuid.New())
		err := orch.AttachReaction(r, reaction)
		if !errors.Is(err, service.ErrNoCurrentTurn) {
			t.Errorf("expected ErrNoCurrentTurn, got %v", err)
		}
	})
}

func TestRoundOrchestrator_CloseRound(t *testing.T) {
	orch := service.RoundOrchestrator{}

	t.Run("sets finishedAt on Round", func(t *testing.T) {
		r := round.NewRound(enum.Free)
		at := time.Now()

		closed := orch.CloseRound(r, at)

		if closed.GetFinishedAt() == nil {
			t.Error("expected finishedAt to be set on Round")
		}
	})
}

// ── helpers ──────────────────────────────────────────────────────────────────

func newQueue(actions ...*action.Action) action.PriorityQueue {
	q := action.NewActionPriorityQueue(nil)
	for _, a := range actions {
		q.Insert(a)
	}
	return q
}

func makeActionWithSpeed(speed int) *action.Action {
	return action.NewAction(
		uuid.New(),
		nil,
		uuid.Nil,
		nil,
		action.ActionSpeed{RollCheck: action.RollCheck{Result: speed}},
		nil, nil, nil, nil, nil, nil, nil,
	)
}

func makeReactionTo(targetID uuid.UUID) *action.Action {
	a := action.NewAction(
		uuid.New(),
		nil,
		targetID,
		nil,
		action.ActionSpeed{},
		nil, nil, nil, nil, nil, nil, nil,
	)
	a.ReactToID = targetID
	return a
}
