package matchsession_test

import (
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

// deathFixture é um ataque de A em B já aberto, com o dano efetivo que o fechamento vai aplicar
// em B (lido da resolução, antes de fechar).
type deathFixture struct {
	session *matchsession.MatchSession
	turn    *turn.Turn
	b       uuid.UUID
	playerB uuid.UUID
	damage  int
}

func newDeathFixture(t *testing.T) *deathFixture {
	t.Helper()
	s, tn, b := openAttackOn(t, 9, func(uuid.UUID) []statuseffect.Effect { return nil })
	f := &deathFixture{session: s, turn: tn, b: b, playerB: lastPlayerB}
	for _, cr := range s.ResolveTurn(tn).CharacterResults {
		if cr.TargetID == b {
			f.damage = cr.EffectiveDamage
		}
	}
	if f.damage <= 0 {
		t.Fatalf("the attack on B deals %d, the death tests need a blow that lands", f.damage)
	}
	return f
}

// setHP põe o HP atual de B (pode estar abaixo do mínimo).
func (f *deathFixture) setHP(t *testing.T, hp int) {
	t.Helper()
	if err := f.healthBar(t).SetCurrent(hp); err != nil {
		t.Fatalf("SetCurrent(%d): %v", hp, err)
	}
}

func (f *deathFixture) healthBar(t *testing.T) interface {
	GetMin() int
	GetCurrent() int
	SetCurrent(int) error
} {
	t.Helper()
	sheet, err := f.session.GetCharSheet(f.b)
	if err != nil {
		t.Fatalf("GetCharSheet: %v", err)
	}
	return sheet.GetAllStatusBar()[enum.Health]
}

func (f *deathFixture) close(t *testing.T) *matchsession.TurnTransition {
	t.Helper()
	tr, err := f.session.CloseOpenTurn()
	if err != nil {
		t.Fatalf("CloseOpenTurn: %v", err)
	}
	return tr
}

func TestDeathAtTheClose(t *testing.T) {
	t.Run("close kills past the negative max", func(t *testing.T) {
		f := newDeathFixture(t)
		min := f.healthBar(t).GetMin()
		// Cai exatamente um abaixo do mínimo.
		f.setHP(t, min-1+f.damage)

		tr := f.close(t)
		if len(tr.Deaths) != 1 {
			t.Fatalf("Deaths = %d write(s), want 1", len(tr.Deaths))
		}
		w := tr.Deaths[0]
		if w.Delete || w.Effect.Kind != statuseffect.KindDead || w.Effect.CharacterID != f.b {
			t.Fatalf("death write = %+v, want a dead applied on B", w)
		}
		if w.Event == nil || w.Event.Change != statuseffect.ChangeApplied {
			t.Fatalf("death event = %+v, want ChangeApplied", w.Event)
		}
		if !f.session.IsDead(f.b) {
			t.Fatal("IsDead(B) = false after the close")
		}
		kinds := kindsOfEffects(f.session.StatusEffectsOf(f.b))
		if len(kinds) != 1 || kinds[0] != statuseffect.KindDead {
			t.Fatalf("B's effects = %v, want only dead (it does not bring prone)", kinds)
		}
	})

	t.Run("close below zero but within max does not kill", func(t *testing.T) {
		f := newDeathFixture(t)
		min := f.healthBar(t).GetMin()
		// Cai exatamente no mínimo: ainda vivo (a morte é um abaixo dele).
		f.setHP(t, min+f.damage)

		tr := f.close(t)
		if len(tr.Deaths) != 0 {
			t.Fatalf("Deaths = %+v, want none at the bar's minimum", tr.Deaths)
		}
		if f.session.IsDead(f.b) {
			t.Fatal("IsDead(B) = true at the bar's minimum")
		}
		if got := f.healthBar(t).GetCurrent(); got != min || got >= 0 {
			t.Fatalf("HP = %d, want the negative minimum %d", got, min)
		}
	})

	t.Run("dead keeps the exact hp", func(t *testing.T) {
		f := newDeathFixture(t)
		min := f.healthBar(t).GetMin()
		f.setHP(t, min)
		want := min - f.damage

		tr := f.close(t)
		if len(tr.Deaths) != 1 {
			t.Fatalf("Deaths = %d write(s), want 1", len(tr.Deaths))
		}
		if got := f.healthBar(t).GetCurrent(); got != want {
			t.Fatalf("HP = %d, want %d — the blow is not clamped at the minimum", got, want)
		}
		if len(tr.Damaged) != 1 || tr.Damaged[0].NewHP != want {
			t.Fatalf("Damaged = %+v, want B at %d", tr.Damaged, want)
		}
	})

	t.Run("dead character cannot act or react", func(t *testing.T) {
		f := newDeathFixture(t)
		queued := mustEnqueue(t, f.session, f.playerB, moveBy(f.b))
		f.setHP(t, f.healthBar(t).GetMin())

		tr := f.close(t)
		if len(tr.Deaths) != 1 {
			t.Fatalf("Deaths = %d write(s), want 1", len(tr.Deaths))
		}
		dropped := matchsession.DroppedActions(tr.DeathDrops)
		if len(dropped) != 1 || dropped[0].GetID() != queued.GetID() {
			t.Fatalf("DeathDrops = %v, want B's queued move %v", idsOf(dropped), queued.GetID())
		}
		if f.session.IsQueued(queued.GetID()) {
			t.Fatal("the dead character's move is still queued")
		}
		if err := f.session.EnqueueAction(f.playerB, moveBy(f.b)); !errors.Is(err, matchsession.ErrBlockedByStatus) {
			t.Fatalf("EnqueueAction by the dead = %v, want ErrBlockedByStatus", err)
		}

		// Um ataque novo no morto senta um nothing para ele.
		act := f.turn.GetAction()
		attacker := act.GetActorID()
		owner := f.session.GetCharToPlayer()[attacker.String()]
		mustEnqueue(t, f.session, owner, attackBy(attacker, f.b))
		next := mustOpen(t, f.session)
		seated := next.SeatedFor(f.b)
		if seated == nil || seated.ReactionKind != action.ReactNothing {
			t.Fatalf("seated for the dead = %+v, want a nothing", seated)
		}
		// Morrer de novo não acrescenta outro Morto.
		tr2, err := f.session.CloseOpenTurn()
		if err != nil {
			t.Fatalf("CloseOpenTurn: %v", err)
		}
		if len(tr2.Deaths) != 0 {
			t.Fatalf("Deaths on an already dead = %+v, want none", tr2.Deaths)
		}
	})
}

func kindsOfEffects(es []statuseffect.Effect) []statuseffect.Kind {
	out := make([]statuseffect.Kind, 0, len(es))
	for _, e := range es {
		out = append(out, e.Kind)
	}
	return out
}

// healToMax põe o HP do personagem no máximo da barra.
func healToMax(t *testing.T, s *matchsession.MatchSession, charID uuid.UUID) {
	t.Helper()
	sheet, err := s.GetCharSheet(charID)
	if err != nil {
		t.Fatalf("GetCharSheet: %v", err)
	}
	bar := sheet.GetAllStatusBar()[enum.Health]
	if err := bar.SetCurrent(bar.GetMax()); err != nil {
		t.Fatalf("SetCurrent: %v", err)
	}
}

func TestDeadMarksList(t *testing.T) {
	t.Run("dead marks list", func(t *testing.T) {
		s := matchsession.NewMatchSession(uuid.New(), nil, nil)
		alive, dead, undone, never := uuid.New(), uuid.New(), uuid.New(), uuid.New()
		s.SetStatusEffects(map[uuid.UUID][]statuseffect.Effect{
			dead:   {{ID: uuid.New(), CharacterID: dead, Kind: statuseffect.KindDead}},
			undone: {{ID: uuid.New(), CharacterID: undone, Kind: statuseffect.KindDead}},
			never:  {{ID: uuid.New(), CharacterID: never, Kind: statuseffect.KindProne}},
		})
		_ = alive
		// A morte de "undone" é desfeita pelo mestre; "late" morre depois do carregamento.
		s.ReplaceStatusEffects(undone, nil)
		late := uuid.New()
		s.ReplaceStatusEffects(late, []statuseffect.Effect{{ID: uuid.New(), CharacterID: late, Kind: statuseffect.KindDead}})

		gotDead, gotUndone := s.DeadMarks()
		if len(gotDead) != 2 || !(contains(gotDead, dead) && contains(gotDead, late)) {
			t.Errorf("dead = %v, want [dead late]", gotDead)
		}
		if len(gotUndone) != 1 || gotUndone[0] != undone {
			t.Errorf("undone = %v, want [undone]", gotUndone)
		}
	})
}

func contains(ids []uuid.UUID, id uuid.UUID) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}
