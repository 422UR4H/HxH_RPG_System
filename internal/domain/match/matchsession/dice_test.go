package matchsession_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
)

// ─── A resolução tira os dados que lhe faltam, uma vez ──────────────────────
//
// O resolvedor é puro e nunca rola: ele informa os dados que faltaram para ler cada teste
// como o viés pede (TurnResolution.DiceShort), e a sessão os tira uma vez e resolve de novo.

func TestResolveTurnDrawsMissingDiceOnce(t *testing.T) {
	f := newDefaultFixture(t)
	rolls := f.source.rolls

	f.edit(t, f.actionID(), action.FieldHit, action.RollCondition{Bias: 2})
	if got := f.source.rolls - rolls; got != 2 {
		t.Fatalf("a +2 on the hit drew %d dice, want 2 — one extra per level, the set already fell", got)
	}
	res := f.session.ResolveTurn(f.turn)
	if len(res.DiceShort) != 0 {
		t.Fatalf("DiceShort = %+v after the top-up, want none", res.DiceShort)
	}
	if got := len(res.CharacterResults[0].Hit.Rolled); got != 4 {
		t.Fatalf("the hit read %d dice, want 4 (the 2D10 set plus two extras)", got)
	}
	after := f.source.rolls
	f.session.ResolveTurn(f.turn)
	if f.source.rolls != after {
		t.Fatalf("resolving again rolled %d dice, want 0", f.source.rolls-after)
	}
}

func TestBiasDownThenUpDrawsNothingNew(t *testing.T) {
	f := newDefaultFixture(t)
	f.edit(t, f.actionID(), action.FieldHit, action.RollCondition{Bias: 2})
	rolls := f.source.rolls

	for _, bias := range []int{-1, 0, 1, -2, 2} {
		f.edit(t, f.actionID(), action.FieldHit, action.RollCondition{Bias: bias})
	}
	if f.source.rolls != rolls {
		t.Fatalf("moving the bias inside what already fell drew %d dice, want 0", f.source.rolls-rolls)
	}
	f.edit(t, f.actionID(), action.FieldHit, action.RollCondition{Bias: 3})
	if got := f.source.rolls - rolls; got != 1 {
		t.Fatalf("a third level drew %d dice, want 1 — only the missing extra", got)
	}
}

func TestPassiveDefaultDefenseBiasDrawsBaseSet(t *testing.T) {
	f := newDefaultFixture(t)

	t.Run("a defense the resolution reads draws the set plus one extra", func(t *testing.T) {
		// A esquiva de C não vence o acerto, e a defesa padrão é lida.
		sc := f.seatedOf(t, f.c)
		f.edit(t, sc.GetID(), action.FieldDodge, action.RollCondition{Modifier: -100})
		rolls := f.source.rolls
		f.edit(t, sc.GetID(), action.FieldDefense, action.RollCondition{Bias: -1})
		if got := f.source.rolls - rolls; got != 3 {
			t.Fatalf("rolled %d dice for the defense, want 3 (the set plus one extra)", got)
		}
		if cr := f.resultOf(t, f.c); cr.Defense.Total != 11 {
			t.Fatalf("Defense = %d, want 11 — disadvantage keeps the worse of 9+9 and the base", cr.Defense.Total)
		}
	})

	t.Run("a defense the resolution does not read draws nothing yet", func(t *testing.T) {
		// A esquiva de B passa a vencer o acerto, e a defesa padrão não é lida.
		sb := f.seatedOf(t, f.b)
		f.edit(t, sb.GetID(), action.FieldDodge, action.RollCondition{Modifier: 100})
		rolls := f.source.rolls
		f.edit(t, sb.GetID(), action.FieldDefense, action.RollCondition{Bias: -1})
		if got := f.source.rolls - rolls; got != 0 {
			t.Fatalf("rolled %d dice for a defense nobody reads, want 0", got)
		}
		// When the dodge stops clearing the hit, the defense is read and its dice fall then.
		f.edit(t, sb.GetID(), action.FieldDodge, action.RollCondition{Modifier: -100})
		if got := f.source.rolls - rolls; got != 3 {
			t.Fatalf("rolled %d dice once the defense is read, want 3", got)
		}
	})
}

func TestLedgerBiasOnPassiveDodgeDrawsOnResolve(t *testing.T) {
	f := newDefaultFixture(t)
	st, err := f.session.GetCharacterStatus(f.b)
	if err != nil {
		t.Fatalf("GetCharacterStatus: %v", err)
	}
	// Um viés que só o resolvedor vê (o ledger de quem esquiva): a edição do mestre nunca o
	// soube, e mesmo assim os dados caem na leitura.
	st.Ledger.Add(match.Modifier{
		Bias: 1, Applies: match.DimDodge, Source: match.SourceSystem,
		Against: match.ScopeAnyone(), ExpiresAt: match.LifetimeEndOfTurn,
	})
	rolls := f.source.rolls
	cr := f.resultOf(t, f.b)
	if got := f.source.rolls - rolls; got != 3 {
		t.Fatalf("a ledger bias on the passive dodge drew %d dice, want 3", got)
	}
	if len(cr.Dodge.Rolled) != 3 || cr.Dodge.Total != 18 {
		t.Fatalf("Dodge = %+v, want 3 dice read and advantage keeping 9+9", cr.Dodge)
	}
	after := f.source.rolls
	f.resultOf(t, f.b)
	if f.source.rolls != after {
		t.Fatalf("resolving again rolled %d dice, want 0", f.source.rolls-after)
	}
}

func TestPeekTurnResolutionNeverDraws(t *testing.T) {
	f := newDefaultFixture(t)
	st, err := f.session.GetCharacterStatus(f.b)
	if err != nil {
		t.Fatalf("GetCharacterStatus: %v", err)
	}
	st.Ledger.Add(match.Modifier{
		Bias: 1, Applies: match.DimDodge, Source: match.SourceSystem,
		Against: match.ScopeAnyone(), ExpiresAt: match.LifetimeEndOfTurn,
	})
	rolls := f.source.rolls
	res := f.session.PeekTurnResolution(f.turn)
	if f.source.rolls != rolls {
		t.Fatalf("a read drew %d dice, want 0 — a read never writes", f.source.rolls-rolls)
	}
	if len(res.DiceShort) != 1 || res.DiceShort[0].Need != 3 {
		t.Fatalf("DiceShort = %+v, want the passive dodge short of 3", res.DiceShort)
	}
	f.session.ResolveTurn(f.turn)
	if got := f.source.rolls - rolls; got != 3 {
		t.Fatalf("ResolveTurn drew %d dice, want 3", got)
	}
}
