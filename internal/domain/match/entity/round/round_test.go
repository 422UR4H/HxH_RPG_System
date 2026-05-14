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
