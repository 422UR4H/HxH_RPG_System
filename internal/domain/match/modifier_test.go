package match_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/google/uuid"
)

func TestModifier_AppliesTo(t *testing.T) {
	enemy := uuid.New()
	other := uuid.New()

	t.Run("general modifier applies to anyone", func(t *testing.T) {
		m := match.Modifier{Amount: -3, AgainstID: nil}
		if !m.AppliesTo(&enemy) {
			t.Error("expected a general modifier to apply against a named target")
		}
		if !m.AppliesTo(nil) {
			t.Error("expected a general modifier to apply with no target")
		}
	})

	t.Run("targeted modifier applies only to its target", func(t *testing.T) {
		m := match.Modifier{Amount: 4, AgainstID: &enemy}
		if !m.AppliesTo(&enemy) {
			t.Error("expected the modifier to apply against its own target")
		}
		if m.AppliesTo(&other) {
			t.Error("expected the modifier not to apply against a different target")
		}
		if m.AppliesTo(nil) {
			t.Error("expected a targeted modifier not to apply with no target")
		}
	})
}

func TestModifierLedger_Totals(t *testing.T) {
	enemy := uuid.New()
	other := uuid.New()

	l := match.NewModifierLedger()
	l.Add(match.Modifier{
		Amount: -2, Source: match.SourceSystem, ExpiresAt: match.ScopeEndOfRound,
		Reason: "off balance after a parry",
	})
	l.Add(match.Modifier{
		Amount: 5, AgainstID: &enemy, Source: match.SourceSystem,
		ExpiresAt: match.ScopeEndOfTurn, Reason: "read the opponent",
	})
	l.Add(match.Modifier{
		Bias: -1, Source: match.SourceSystem, ExpiresAt: match.ScopeEndOfTurn,
		Reason: "swapped a declared action",
	})
	l.Add(match.Modifier{
		Bias: -1, Source: match.SourceSystem, ExpiresAt: match.ScopeEndOfTurn,
		Reason: "converted the action into a reaction",
	})

	t.Run("amounts against the named target include the targeted bonus", func(t *testing.T) {
		if got := l.TotalAmount(&enemy); got != 3 { // -2 general + 5 targeted
			t.Errorf("expected 3, got %d", got)
		}
	})

	t.Run("amounts against another target exclude it", func(t *testing.T) {
		if got := l.TotalAmount(&other); got != -2 {
			t.Errorf("expected -2, got %d", got)
		}
	})

	t.Run("bias accumulates independently of amount", func(t *testing.T) {
		if got := l.TotalBias(&enemy); got != -2 {
			t.Errorf("expected -2, got %d", got)
		}
	})
}

func TestModifierLedger_Expire(t *testing.T) {
	l := match.NewModifierLedger()
	l.Add(match.Modifier{Amount: 1, ExpiresAt: match.ScopeEndOfTurn, Reason: "turn scoped"})
	l.Add(match.Modifier{Amount: 10, ExpiresAt: match.ScopeEndOfRound, Reason: "round scoped"})

	l.Expire(match.ScopeEndOfTurn)

	if got := len(l.All()); got != 1 {
		t.Fatalf("expected 1 modifier left, got %d", got)
	}
	if got := l.TotalAmount(nil); got != 10 {
		t.Errorf("expected only the round-scoped modifier to survive, got total %d", got)
	}
}

func TestModifierLedger_AllIsACopy(t *testing.T) {
	l := match.NewModifierLedger()
	l.Add(match.Modifier{Amount: 1, ExpiresAt: match.ScopeEndOfTurn})

	got := l.All()
	got[0].Amount = 999

	if l.TotalAmount(nil) != 1 {
		t.Error("expected All() to hand back a copy, not the ledger's own slice")
	}
}
