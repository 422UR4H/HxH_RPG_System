package round_test

import (
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/google/uuid"
)

func TestRound_AppendTurn(t *testing.T) {
	r := round.NewRound(enum.Free)
	a := makeAction()
	tRn := turn.NewTurn(a)

	r.AppendTurn(tRn)

	if r.CurrentTurn() != tRn {
		t.Error("CurrentTurn should return the appended turn")
	}
}

func TestRound_HasOpenTurn(t *testing.T) {
	t.Run("false when no turns", func(t *testing.T) {
		r := round.NewRound(enum.Free)
		if r.HasOpenTurn() {
			t.Error("expected false when Round has no turns")
		}
	})

	t.Run("true when current turn is open", func(t *testing.T) {
		r := round.NewRound(enum.Race)
		r.AppendTurn(turn.NewTurn(makeAction()))
		if !r.HasOpenTurn() {
			t.Error("expected true when Turn has no finishedAt")
		}
	})

	t.Run("false when current turn is closed", func(t *testing.T) {
		r := round.NewRound(enum.Race)
		tRn := turn.NewTurn(makeAction())
		r.AppendTurn(tRn)
		tRn.Close(time.Now())
		if r.HasOpenTurn() {
			t.Error("expected false when Turn is closed")
		}
	})
}

func TestRound_Close(t *testing.T) {
	r := round.NewRound(enum.Free)
	at := time.Now()
	r.Close(at)
	if r.GetFinishedAt() == nil {
		t.Error("expected finishedAt to be set after Close")
	}
}

func makeAction() action.Action {
	return action.Action{ReactToID: uuid.Nil}
}

func TestRound_GetID(t *testing.T) {
	r := round.NewRound(enum.Free)
	if r.GetID() == (uuid.UUID{}) {
		t.Error("expected non-zero ID from NewRound")
	}
}

func TestRound_GetCreatedAt(t *testing.T) {
	before := time.Now()
	r := round.NewRound(enum.Free)
	after := time.Now()
	if r.GetCreatedAt().Before(before) || r.GetCreatedAt().After(after) {
		t.Errorf("createdAt %v not in [%v, %v]", r.GetCreatedAt(), before, after)
	}
}

func TestReconstructRound(t *testing.T) {
	id := uuid.New()
	now := time.Now()
	r := round.ReconstructRound(id, enum.Race, now)
	if r.GetID() != id {
		t.Errorf("expected ID %v, got %v", id, r.GetID())
	}
	if r.GetMode() != enum.Race {
		t.Errorf("expected mode Race, got %v", r.GetMode())
	}
	if !r.GetCreatedAt().Equal(now) {
		t.Errorf("expected createdAt %v, got %v", now, r.GetCreatedAt())
	}
	if r.GetFinishedAt() != nil {
		t.Error("expected nil finishedAt on reconstructed round")
	}
}

func TestRound_Prices(t *testing.T) {
	t.Run("a fresh round has no price on either bar", func(t *testing.T) {
		r := round.NewRound(enum.Race)
		if _, frozen := r.Price(action.BarAction); frozen {
			t.Error("the action bar must start unfrozen")
		}
		if _, frozen := r.Price(action.BarMove); frozen {
			t.Error("the move bar must start unfrozen")
		}
	})

	t.Run("freezing one bar leaves the other alone", func(t *testing.T) {
		r := round.NewRound(enum.Race)
		r.FreezePrice(action.BarAction, 11)

		got, frozen := r.Price(action.BarAction)
		if !frozen || got != 11 {
			t.Errorf("action price = (%d, %v), want (11, true)", got, frozen)
		}
		if _, frozen := r.Price(action.BarMove); frozen {
			t.Error("the move bar must still be unfrozen — the bars freeze independently")
		}
	})

	t.Run("the price never moves once frozen", func(t *testing.T) {
		r := round.NewRound(enum.Race)
		r.FreezePrice(action.BarAction, 11)
		r.FreezePrice(action.BarAction, 4)

		got, _ := r.Price(action.BarAction)
		if got != 11 {
			t.Errorf("price = %d, want 11: a later, slower action must not re-price the round", got)
		}
	})
}

func TestRound_HasOpenedAction(t *testing.T) {
	r := round.NewRound(enum.Race)
	a := action.NewAction(uuid.New(), nil, uuid.Nil, nil, action.ActionSpeed{},
		nil, nil, nil, nil, nil, nil, nil)

	if r.HasOpenedAction(a.GetID()) {
		t.Error("nothing has opened yet")
	}
	r.AppendTurn(turn.NewTurn(*a))
	if !r.HasOpenedAction(a.GetID()) {
		t.Error("the action opened, so the round must say so — this is what the dependency edge reads")
	}
}

func TestRound_SetMode(t *testing.T) {
	r := round.NewRound(enum.Free)
	r.SetMode(enum.Race)
	if r.GetMode() != enum.Race {
		t.Errorf("mode = %q, want Race", r.GetMode())
	}
	r.SetMode(enum.Race)
	if r.GetMode() != enum.Race {
		t.Error("SetMode must be idempotent, unlike ToggleMode")
	}
}
