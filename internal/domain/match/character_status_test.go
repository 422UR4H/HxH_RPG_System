package match_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
)

func TestNewCharacterStatus(t *testing.T) {
	s := match.NewCharacterStatus()

	if s == nil {
		t.Fatal("expected non-nil CharacterStatus")
	}
	if s.ActionBar.Balance != 0 || s.MoveBar.Balance != 0 {
		t.Error("expected both bars to start at zero balance")
	}
	if len(s.ActionBar.Speeds) != 0 || len(s.MoveBar.Speeds) != 0 {
		t.Error("expected both bars to start with no recorded speeds")
	}
	if s.Stance != match.StanceNone {
		t.Errorf("expected StanceNone while posture rules do not exist, got %q", s.Stance)
	}
	if len(s.Ledger.All()) != 0 {
		t.Error("expected an empty ledger")
	}
	if s.Velocity.Speed != 0 {
		t.Errorf("expected zero velocity, got %v", s.Velocity.Speed)
	}
}

func TestResourceBar_RecordAndReset(t *testing.T) {
	s := match.NewCharacterStatus()

	s.ActionBar.RecordSpeed(20)
	s.ActionBar.RecordSpeed(14)
	s.ActionBar.Balance = 9

	if got := len(s.ActionBar.Speeds); got != 2 {
		t.Fatalf("expected 2 recorded speeds, got %d", got)
	}

	s.ActionBar.ResetRound()

	if got := len(s.ActionBar.Speeds); got != 0 {
		t.Errorf("expected the speed history cleared, got %d entries", got)
	}
	if s.ActionBar.Balance != 9 {
		t.Errorf("expected the carry-over balance to survive the round reset, got %d", s.ActionBar.Balance)
	}
}

func TestResourceBar_BarsAreIndependent(t *testing.T) {
	s := match.NewCharacterStatus()

	s.ActionBar.RecordSpeed(20)

	if len(s.MoveBar.Speeds) != 0 {
		t.Error("expected the move bar to be untouched by an action-bar record")
	}
}

func TestCharacterStatus_ExpireModifiers(t *testing.T) {
	s := match.NewCharacterStatus()
	s.Ledger.Add(match.Modifier{Amount: 3, ExpiresAt: match.ScopeEndOfTurn, Source: match.SourceSystem})
	s.Ledger.Add(match.Modifier{Amount: 7, ExpiresAt: match.ScopeEndOfRound, Source: match.SourceMaster})

	s.ExpireModifiers(match.ScopeEndOfTurn)

	if got := s.Ledger.TotalAmount(nil); got != 7 {
		t.Errorf("expected only the round-scoped modifier to survive, got %d", got)
	}
}
