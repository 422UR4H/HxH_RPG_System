package match_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/google/uuid"
)

func TestScope_AppliesTo(t *testing.T) {
	a, b := uuid.New(), uuid.New()

	t.Run("anyone applies to everybody, including a roll with no named target", func(t *testing.T) {
		s := match.ScopeAnyone()
		if !s.AppliesTo(nil) || !s.AppliesTo(&a) {
			t.Fatal("ScopeAnyone must apply to every roll")
		}
	})

	t.Run("only X applies to X and to nobody else", func(t *testing.T) {
		s := match.ScopeOnly(a)
		if !s.AppliesTo(&a) {
			t.Error("ScopeOnly must apply to its own target")
		}
		if s.AppliesTo(&b) {
			t.Error("ScopeOnly must not apply to a different target")
		}
		// A targeted modifier never applies to an untargeted roll.
		if s.AppliesTo(nil) {
			t.Error("ScopeOnly must not apply to a roll with no target")
		}
	})

	t.Run("all but X is the closed dodge's reserve — everyone except the duel opponent", func(t *testing.T) {
		s := match.ScopeAllBut(a)
		if s.AppliesTo(&a) {
			t.Error("ScopeAllBut must not apply to the excluded target")
		}
		if !s.AppliesTo(&b) {
			t.Error("ScopeAllBut must apply to a third party")
		}
		// No named target means nobody in particular, which is not the excluded one.
		if !s.AppliesTo(nil) {
			t.Error("ScopeAllBut must apply to an untargeted roll")
		}
	})
}

func TestScope_KindAndIDRoundTripThroughScopeFrom(t *testing.T) {
	id := uuid.New()

	t.Run("anyone", func(t *testing.T) {
		s := match.ScopeAnyone()
		got := match.ScopeFrom(s.Kind(), s.ID())
		if got.Kind() != s.Kind() {
			t.Fatalf("expected kind %q, got %q", s.Kind(), got.Kind())
		}
		if !got.AppliesTo(nil) {
			t.Error("round-tripped ScopeAnyone must still apply to everyone")
		}
	})

	t.Run("only", func(t *testing.T) {
		s := match.ScopeOnly(id)
		got := match.ScopeFrom(s.Kind(), s.ID())
		if got.Kind() != s.Kind() || got.ID() != s.ID() {
			t.Fatalf("expected kind %q id %s, got kind %q id %s", s.Kind(), s.ID(), got.Kind(), got.ID())
		}
		if !got.AppliesTo(&id) {
			t.Error("round-tripped ScopeOnly must still apply to its own target")
		}
	})

	t.Run("all but", func(t *testing.T) {
		s := match.ScopeAllBut(id)
		got := match.ScopeFrom(s.Kind(), s.ID())
		if got.Kind() != s.Kind() || got.ID() != s.ID() {
			t.Fatalf("expected kind %q id %s, got kind %q id %s", s.Kind(), s.ID(), got.Kind(), got.ID())
		}
		if got.AppliesTo(&id) {
			t.Error("round-tripped ScopeAllBut must still exclude its own target")
		}
	})

	t.Run("an unknown kind reads as anyone, the safe answer for a lost scope", func(t *testing.T) {
		got := match.ScopeFrom("garbage", id)
		if got.Kind() != match.ScopeAnyone().Kind() {
			t.Fatalf("expected unknown kind to read as anyone, got %q", got.Kind())
		}
		if !got.AppliesTo(nil) {
			t.Error("an unknown kind must apply to everyone, like ScopeAnyone")
		}
	})
}

func TestModifierLedger_TotalsAreScopedByDimension(t *testing.T) {
	attacker := uuid.New()
	l := match.NewModifierLedger()
	// The duel reserve: actionSpeed, only against the attacker.
	l.Add(match.Modifier{
		Amount: 7, Applies: match.DimActionSpeed, Source: match.SourceSystem,
		Against: match.ScopeOnly(attacker), ExpiresAt: match.LifetimeNextTurn,
	})
	// The closed dodge reserve: dodge, against everyone but the attacker.
	l.Add(match.Modifier{
		Amount: 4, Applies: match.DimDodge, Source: match.SourceSystem,
		Against: match.ScopeAllBut(attacker), ExpiresAt: match.LifetimeNextTurn,
	})

	if got := l.TotalAmount(match.DimActionSpeed, &attacker); got != 7 {
		t.Errorf("actionSpeed against the attacker = %d, want 7", got)
	}
	if got := l.TotalAmount(match.DimDodge, &attacker); got != 0 {
		t.Errorf("dodge against the attacker = %d, want 0 — the reserve is for third parties", got)
	}
	third := uuid.New()
	if got := l.TotalAmount(match.DimDodge, &third); got != 4 {
		t.Errorf("dodge against a third party = %d, want 4", got)
	}
	if got := l.TotalAmount(match.DimActionSpeed, &third); got != 0 {
		t.Errorf("actionSpeed against a third party = %d, want 0 — the duel bonus is targeted", got)
	}
}

func TestModifierLedger_AdvanceTurn(t *testing.T) {
	l := match.NewModifierLedger()
	l.Add(match.Modifier{Amount: 1, Applies: match.DimActionSpeed, ExpiresAt: match.LifetimeEndOfTurn, Reason: "this turn"})
	l.Add(match.Modifier{Amount: 2, Applies: match.DimActionSpeed, ExpiresAt: match.LifetimeNextTurn, Reason: "next turn"})
	l.Add(match.Modifier{Amount: 4, Applies: match.DimActionSpeed, ExpiresAt: match.LifetimeEndOfRound, Reason: "the round"})

	l.AdvanceTurn()

	// end_of_turn died with its own turn; next_turn was demoted and is now live for one turn.
	if got := l.TotalAmount(match.DimActionSpeed, nil); got != 6 {
		t.Fatalf("after one turn = %d, want 6 (2 demoted + 4 round-scoped)", got)
	}
	l.AdvanceTurn()
	if got := l.TotalAmount(match.DimActionSpeed, nil); got != 4 {
		t.Fatalf("after two turns = %d, want 4 — the demoted bonus lasted exactly one turn", got)
	}
}

func TestModifierLedger_Totals(t *testing.T) {
	enemy := uuid.New()
	other := uuid.New()

	l := match.NewModifierLedger()
	l.Add(match.Modifier{
		Amount: -2, Applies: match.DimActionSpeed, Source: match.SourceSystem, ExpiresAt: match.LifetimeEndOfRound,
		Reason: "off balance after a parry",
	})
	l.Add(match.Modifier{
		Amount: 5, Applies: match.DimActionSpeed, Against: match.ScopeOnly(enemy), Source: match.SourceSystem,
		ExpiresAt: match.LifetimeEndOfTurn, Reason: "read the opponent",
	})
	l.Add(match.Modifier{
		Bias: -1, Applies: match.DimActionSpeed, Source: match.SourceSystem, ExpiresAt: match.LifetimeEndOfTurn,
		Reason: "swapped a declared action",
	})
	l.Add(match.Modifier{
		Bias: -1, Applies: match.DimActionSpeed, Source: match.SourceSystem, ExpiresAt: match.LifetimeEndOfTurn,
		Reason: "converted the action into a reaction",
	})

	t.Run("amounts against the named target include the targeted bonus", func(t *testing.T) {
		if got := l.TotalAmount(match.DimActionSpeed, &enemy); got != 3 { // -2 general + 5 targeted
			t.Errorf("expected 3, got %d", got)
		}
	})

	t.Run("amounts against another target exclude it", func(t *testing.T) {
		if got := l.TotalAmount(match.DimActionSpeed, &other); got != -2 {
			t.Errorf("expected -2, got %d", got)
		}
	})

	t.Run("bias accumulates independently of amount", func(t *testing.T) {
		if got := l.TotalBias(match.DimActionSpeed, &enemy); got != -2 {
			t.Errorf("expected -2, got %d", got)
		}
	})
}

func TestModifierLedger_Expire(t *testing.T) {
	l := match.NewModifierLedger()
	l.Add(match.Modifier{Amount: 1, Applies: match.DimActionSpeed, ExpiresAt: match.LifetimeEndOfTurn, Reason: "turn scoped"})
	l.Add(match.Modifier{Amount: 10, Applies: match.DimActionSpeed, ExpiresAt: match.LifetimeEndOfRound, Reason: "round scoped"})

	l.Expire(match.LifetimeEndOfTurn)

	if got := len(l.All()); got != 1 {
		t.Fatalf("expected 1 modifier left, got %d", got)
	}
	if got := l.TotalAmount(match.DimActionSpeed, nil); got != 10 {
		t.Errorf("expected only the round-scoped modifier to survive, got total %d", got)
	}
}

func TestModifierLedger_AllIsACopy(t *testing.T) {
	l := match.NewModifierLedger()
	l.Add(match.Modifier{Amount: 1, Applies: match.DimActionSpeed, ExpiresAt: match.LifetimeEndOfTurn})

	got := l.All()
	got[0].Amount = 999

	if l.TotalAmount(match.DimActionSpeed, nil) != 1 {
		t.Error("expected All() to hand back a copy, not the ledger's own slice")
	}
}
