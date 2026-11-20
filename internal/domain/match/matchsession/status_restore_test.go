package matchsession_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

// Devolver o que um status desfeito no turno cancelou (D3): a sessão revalida cada ação contra
// o estado de agora e só devolve o que ainda cabe, com o id, os dados e a velocidade de antes.

func rollsOf(a *action.Action) string {
	out := fmt.Sprintf("speed=%+v", a.Speed)
	if a.Attack != nil {
		out += fmt.Sprintf(" hit=%+v", a.Attack.Hit)
	}
	return out
}

func refusedIDs(rs []matchsession.RestoreRefusal) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(rs))
	for _, r := range rs {
		out = append(out, r.ActionID)
	}
	return out
}

func sameIDs(a, b []uuid.UUID) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestRestoreCancelledByStatus_RestoresOriginalIDAndOrder(t *testing.T) {
	f := newDefaultFixture(t)
	bAttack := mustEnqueue(t, f.session, f.playerB, attackBy(f.b, f.c))
	cAttack := mustEnqueue(t, f.session, f.playerC, attackBy(f.c, f.b))
	bMove := mustEnqueue(t, f.session, f.playerB, moveBy(f.b))
	rolls := map[uuid.UUID]string{bAttack.GetID(): rollsOf(bAttack), bMove.GetID(): rollsOf(bMove)}

	setEffects(f.session, f.b, kindOnly(f.b, statuseffect.KindStunned))
	removed := f.session.DropRefusedByStatus(f.b)
	if seated := f.turn.SeatedFor(f.b); seated == nil || seated.ReactionKind != action.ReactNothing {
		t.Fatalf("stunned B seated = %+v, want a nothing", seated)
	}

	setEffects(f.session, f.b)
	restored, refused := f.session.RestoreCancelledByStatus(removed)
	if len(refused) != 0 {
		t.Fatalf("refused = %+v, want none", refused)
	}
	want := []uuid.UUID{bAttack.GetID(), bMove.GetID()}
	if got := idsOf(restored); !sameIDs(got, want) {
		t.Fatalf("restored = %v, want %v", got, want)
	}
	if got := pendingIDsOf(f.session); !sameIDs(got, []uuid.UUID{cAttack.GetID(), bAttack.GetID(), bMove.GetID()}) {
		t.Fatalf("queue = %v, want C's attack and then B's two, in removal order", got)
	}
	for _, a := range restored {
		if rollsOf(a) != rolls[a.GetID()] {
			t.Errorf("%v came back with %s, want the rolls it had: %s", a.GetID(), rollsOf(a), rolls[a.GetID()])
		}
	}
	// A reação sentada volta a seguir o conjunto de agora.
	if seated := f.turn.SeatedFor(f.b); seated == nil || seated.ReactionKind != action.ReactDefault {
		t.Fatalf("B seated after the restore = %+v, want the default", seated)
	}
}

func TestRestoreCancelledByStatus_RefusesAlreadyQueued(t *testing.T) {
	f := newDefaultFixture(t)
	bAttack := mustEnqueue(t, f.session, f.playerB, attackBy(f.b, f.c))
	setEffects(f.session, f.b, kindOnly(f.b, statuseffect.KindStunned))
	removed := f.session.DropRefusedByStatus(f.b)
	setEffects(f.session, f.b)
	if restored, _ := f.session.RestoreCancelledByStatus(removed); len(restored) != 1 {
		t.Fatalf("first restore gave back %v, want the attack", idsOf(restored))
	}

	restored, refused := f.session.RestoreCancelledByStatus(removed)
	if len(restored) != 0 {
		t.Fatalf("second restore gave back %v, want nothing", idsOf(restored))
	}
	if got := refusedIDs(refused); !sameIDs(got, []uuid.UUID{bAttack.GetID()}) || refused[0].Reason == "" {
		t.Fatalf("refused = %+v, want the attack with a reason", refused)
	}
	if got := pendingIDsOf(f.session); len(got) != 1 {
		t.Fatalf("queue = %v, want the attack once", got)
	}
}

func TestRestoreCancelledByStatus_RefusesWhenAnotherStatusStillBlocks(t *testing.T) {
	f := newDefaultFixture(t)
	bAttack := mustEnqueue(t, f.session, f.playerB, attackBy(f.b, f.c))
	bMove := mustEnqueue(t, f.session, f.playerB, moveBy(f.b))
	setEffects(f.session, f.b, kindOnly(f.b, statuseffect.KindStunned), paralyzedIn(f.b, statuseffect.DomainAttack))
	removed := f.session.DropRefusedByStatus(f.b)

	// Sai o Atordoado; o Paralisado no ataque continua.
	setEffects(f.session, f.b, paralyzedIn(f.b, statuseffect.DomainAttack))
	restored, refused := f.session.RestoreCancelledByStatus(removed)
	if got := idsOf(restored); !sameIDs(got, []uuid.UUID{bMove.GetID()}) {
		t.Fatalf("restored = %v, want only the move", got)
	}
	if len(refused) != 1 || refused[0].ActionID != bAttack.GetID() {
		t.Fatalf("refused = %+v, want the attack", refused)
	}
	if r := refused[0].Reason; !strings.Contains(r, "ataque") || !strings.Contains(r, "paralyzed") {
		t.Errorf("reason = %q, want what still blocks and by which kind", r)
	}
	if got := pendingIDsOf(f.session); !sameIDs(got, []uuid.UUID{bMove.GetID()}) {
		t.Fatalf("queue = %v, want only the move", got)
	}
}

func TestRestoreCancelledByStatus_RestoresStackedChainWhole(t *testing.T) {
	f := newDefaultFixture(t)
	first := mustEnqueue(t, f.session, f.playerB, moveBy(f.b))
	second := mustEnqueue(t, f.session, f.playerB, moveBy(f.b))
	if second.Move.StackedOn != first.GetID() {
		t.Fatalf("setup: second.StackedOn = %v, want %v", second.Move.StackedOn, first.GetID())
	}
	setEffects(f.session, f.b, kindOnly(f.b, statuseffect.KindStunned))
	removed := f.session.DropRefusedByStatus(f.b)
	setEffects(f.session, f.b)

	restored, refused := f.session.RestoreCancelledByStatus(removed)
	if len(refused) != 0 {
		t.Fatalf("refused = %+v, want none", refused)
	}
	if got := idsOf(restored); !sameIDs(got, []uuid.UUID{first.GetID(), second.GetID()}) {
		t.Fatalf("restored = %v, want the chain in order", got)
	}
	if second.Move.StackedOn != first.GetID() {
		t.Errorf("second.StackedOn = %v, want still the first", second.Move.StackedOn)
	}
	if tail := f.session.MoveChainTail(f.b); tail == nil || tail.GetID() != second.GetID() {
		t.Errorf("chain tail = %v, want the second move", tail)
	}
}

func TestRestoreCancelledByStatus_RefusesStackedWithoutBase(t *testing.T) {
	t.Run("the base is not given back", func(t *testing.T) {
		f := newDefaultFixture(t)
		mustEnqueue(t, f.session, f.playerB, moveBy(f.b))
		second := mustEnqueue(t, f.session, f.playerB, moveBy(f.b))
		setEffects(f.session, f.b, kindOnly(f.b, statuseffect.KindStunned))
		f.session.DropRefusedByStatus(f.b)
		setEffects(f.session, f.b)

		restored, refused := f.session.RestoreCancelledByStatus([]*action.Action{second})
		if len(restored) != 0 {
			t.Fatalf("restored = %v, want nothing (half a chain)", idsOf(restored))
		}
		if len(refused) != 1 || refused[0].ActionID != second.GetID() || refused[0].Reason == "" {
			t.Fatalf("refused = %+v, want the stacked move with a reason", refused)
		}
		if got := pendingIDsOf(f.session); len(got) != 0 {
			t.Fatalf("queue = %v, want empty", got)
		}
	})

	t.Run("the chain moved on without it", func(t *testing.T) {
		f := newDefaultFixture(t)
		first := mustEnqueue(t, f.session, f.playerB, moveBy(f.b))
		second := mustEnqueue(t, f.session, f.playerB, moveBy(f.b))
		setEffects(f.session, f.b, kindOnly(f.b, statuseffect.KindStunned))
		removed := f.session.DropRefusedByStatus(f.b)
		setEffects(f.session, f.b)
		// Um movimento novo chegou no meio: é ele a cabeça da cadeia agora.
		fresh := mustEnqueue(t, f.session, f.playerB, moveBy(f.b))

		restored, refused := f.session.RestoreCancelledByStatus(removed)
		if len(restored) != 0 {
			t.Fatalf("restored = %v, want nothing (a fork in the chain)", idsOf(restored))
		}
		if got := refusedIDs(refused); !sameIDs(got, []uuid.UUID{first.GetID(), second.GetID()}) {
			t.Fatalf("refused = %v, want both moves", got)
		}
		if got := pendingIDsOf(f.session); !sameIDs(got, []uuid.UUID{fresh.GetID()}) {
			t.Fatalf("queue = %v, want only the fresh move", got)
		}
	})
}
