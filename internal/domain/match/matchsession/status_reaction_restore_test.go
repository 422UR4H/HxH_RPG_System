package matchsession_test

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

// A reação que um status desfeito no turno trocou volta como estava (D3, revisão final): o
// mesmo id, os mesmos dados, o mesmo lugar — revalidada contra o turno de agora, nunca repetida.

func reactionIDsOf(t *testing.T, f *defaultFixture) []uuid.UUID {
	t.Helper()
	var out []uuid.UUID
	for _, r := range f.turn.GetReactions() {
		out = append(out, r.GetID())
	}
	return out
}

// stunAndDropB atordoa B e refaz as pendências dele; devolve as reações trocadas.
func stunAndDropB(t *testing.T, f *defaultFixture, extra ...statuseffect.Effect) []matchsession.ReactionDrop {
	t.Helper()
	setEffects(f.session, f.b, append([]statuseffect.Effect{kindOnly(f.b, statuseffect.KindStunned)}, extra...)...)
	_, swapped := f.session.DropRefusedByStatusWithReactions(f.b)
	return swapped
}

// undoStun tira o Atordoado de B (os outros efeitos ficam) e refaz as pendências, como a
// remoção faz antes do desfazer.
func undoStun(f *defaultFixture, keep ...statuseffect.Effect) {
	setEffects(f.session, f.b, keep...)
	f.session.DropRefusedByStatusWithReactions(f.b)
}

func TestRestoreSwappedReactions_ComesBackWithTheSameIDAndValues(t *testing.T) {
	f := newDefaultFixture(t)
	bDodge, err := f.react(t, f.playerB, f.b, action.ReactDodge)
	if err != nil {
		t.Fatalf("B dodge: %v", err)
	}
	if _, err := f.react(t, f.playerC, f.c, action.ReactDodge); err != nil {
		t.Fatalf("C dodge: %v", err)
	}
	live := f.turn.ReactionRef(bDodge.GetID())
	if live == nil || live.Dodge.Attempts.IsEmpty() {
		t.Fatalf("B's dodge = %+v, want rolled", live)
	}
	want := *live
	order := reactionIDsOf(t, f)

	swapped := stunAndDropB(t, f)
	if len(swapped) != 1 || swapped[0].Reaction.GetID() != bDodge.GetID() {
		t.Fatalf("swapped = %+v, want B's dodge", swapped)
	}
	if f.turn.ReactionRef(bDodge.GetID()) != nil {
		t.Fatal("B's dodge is still on the turn under the stun")
	}
	undoStun(f)
	seat := f.turn.SeatedFor(f.b)
	if seat == nil || seat.ReactionKind != action.ReactDefault {
		t.Fatalf("after the removal B seated = %+v, want the default", seat)
	}
	// SeatedFor aponta para dentro do turno: o id é lido agora, antes de o turno mudar.
	seatID := seat.GetID()
	f.session.TakeAnchorMoves()

	restored, refused := f.session.RestoreSwappedReactions(swapped)
	if len(refused) != 0 {
		t.Fatalf("refused = %+v, want none", refused)
	}
	if len(restored) != 1 || restored[0].Reaction.GetID() != bDodge.GetID() || restored[0].Replaced != seatID {
		t.Fatalf("restored = %+v, want B's dodge in place of the seat %s", restored, seatID)
	}
	got := f.turn.ReactionRef(bDodge.GetID())
	if got == nil {
		t.Fatal("B's dodge did not come back")
	}
	if f.turn.IsSeated(got.GetID()) {
		t.Fatal("B's own dodge came back as a seat")
	}
	if !reflect.DeepEqual(*got, want) {
		t.Fatalf("restored = %+v, want exactly %+v", *got, want)
	}
	if f.turn.SeatedFor(f.b) != nil {
		t.Fatal("the seat stayed next to the restored reaction")
	}
	if now := reactionIDsOf(t, f); !slices.Equal(now, order) {
		t.Fatalf("reactions = %v, want the original order %v", now, order)
	}
	// A âncora que tinha seguido a reação para a sentada volta com ela.
	moves := f.session.TakeAnchorMoves()
	if len(moves) != 1 || moves[0].From != seatID || moves[0].To != bDodge.GetID() || moves[0].TurnID != f.turn.GetID() {
		t.Fatalf("anchor moves = %+v, want the seat → B's dodge", moves)
	}
	if again, refused := f.session.RestoreSwappedReactions(swapped); len(again) != 0 || len(refused) != 1 {
		t.Fatalf("second restore = %+v / %+v, want one refusal (idempotent)", again, refused)
	}
	if _, err := f.session.CloseOpenTurn(); err != nil {
		t.Fatalf("CloseOpenTurn: %v", err)
	}
}

func TestRestoreSwappedReactions_ASeatComesBackWithTheMastersEdits(t *testing.T) {
	f := newDefaultFixture(t)
	seat := f.turn.SeatedFor(f.b)
	if seat == nil {
		t.Fatal("B has no seat")
	}
	seatID := seat.GetID()
	swapped := stunAndDropB(t, f)
	if len(swapped) != 1 || swapped[0].Reaction.GetID() != seatID {
		t.Fatalf("swapped = %+v, want B's seated default", swapped)
	}
	undoStun(f)
	if s := f.turn.SeatedFor(f.b); s == nil || s.GetID() == seatID {
		t.Fatalf("after the removal B seated = %+v, want a fresh default", s)
	}
	restored, refused := f.session.RestoreSwappedReactions(swapped)
	if len(refused) != 0 || len(restored) != 1 {
		t.Fatalf("restore = %+v / %+v, want the seat back", restored, refused)
	}
	if s := f.turn.SeatedFor(f.b); s == nil || s.GetID() != seatID {
		t.Fatalf("B seated = %+v, want the original seat %s back as a seat", s, seatID)
	}
}

func TestRestoreSwappedReactions_RefusedWhenTheReactorSentAnother(t *testing.T) {
	f := newDefaultFixture(t)
	bDodge, err := f.react(t, f.playerB, f.b, action.ReactDodge)
	if err != nil {
		t.Fatalf("B dodge: %v", err)
	}
	swapped := stunAndDropB(t, f)
	// Atordoado, B só pode mandar um nothing — que é dele, não da sessão.
	resent, err := f.react(t, f.playerB, f.b, action.ReactNothing)
	if err != nil {
		t.Fatalf("B nothing: %v", err)
	}
	undoStun(f)
	restored, refused := f.session.RestoreSwappedReactions(swapped)
	if len(restored) != 0 || len(refused) != 1 || refused[0].ReactionID != bDodge.GetID() {
		t.Fatalf("restore = %+v / %+v, want B's dodge refused", restored, refused)
	}
	if !strings.Contains(refused[0].Reason, "outra reação") {
		t.Fatalf("reason = %q, want it to say B sent another reaction", refused[0].Reason)
	}
	if f.turn.ReactionRef(resent.GetID()) == nil || f.turn.ReactionRef(bDodge.GetID()) != nil {
		t.Fatal("the reaction B re-sent must stay, the old one must not come back")
	}
}

func TestRestoreSwappedReactions_RefusedWhileAnotherStatusBlocksIt(t *testing.T) {
	f := newDefaultFixture(t)
	bDodge, err := f.react(t, f.playerB, f.b, action.ReactDodge)
	if err != nil {
		t.Fatalf("B dodge: %v", err)
	}
	para := paralyzedIn(f.b, statuseffect.DomainDodge)
	swapped := stunAndDropB(t, f, para)
	undoStun(f, para)
	restored, refused := f.session.RestoreSwappedReactions(swapped)
	if len(restored) != 0 || len(refused) != 1 {
		t.Fatalf("restore = %+v / %+v, want one refusal", restored, refused)
	}
	if !slices.Equal(refused[0].BlockedBy, []statuseffect.Kind{statuseffect.KindParalyzed}) {
		t.Fatalf("blocked by = %v, want [paralyzed]", refused[0].BlockedBy)
	}
	if refused[0].Drop.Reaction.GetID() != bDodge.GetID() {
		t.Fatalf("the refusal must carry the drop back, got %+v", refused[0].Drop)
	}
	// Sem o Paralisado, o mesmo registro devolve a esquiva.
	undoStun(f)
	if restored, _ := f.session.RestoreSwappedReactions([]matchsession.ReactionDrop{refused[0].Drop}); len(restored) != 1 {
		t.Fatalf("restore after the second removal = %+v, want B's dodge", restored)
	}
}

func TestRestoreSwappedReactions_RefusedOnceTheTurnIsNoLongerOpen(t *testing.T) {
	f := newDefaultFixture(t)
	if _, err := f.react(t, f.playerB, f.b, action.ReactDodge); err != nil {
		t.Fatalf("B dodge: %v", err)
	}
	swapped := stunAndDropB(t, f)
	undoStun(f)
	if _, err := f.session.CloseOpenTurn(); err != nil {
		t.Fatalf("CloseOpenTurn: %v", err)
	}
	restored, refused := f.session.RestoreSwappedReactions(swapped)
	if len(restored) != 0 || len(refused) != 1 || !strings.Contains(refused[0].Reason, "turno") {
		t.Fatalf("restore = %+v / %+v, want a refusal naming the turn", restored, refused)
	}
}

func TestRestoreSwappedReactions_RefusedWhenTheSeatInItsPlaceWasOpened(t *testing.T) {
	f := newDefaultFixture(t)
	if _, err := f.react(t, f.playerB, f.b, action.ReactDodge); err != nil {
		t.Fatalf("B dodge: %v", err)
	}
	swapped := stunAndDropB(t, f)
	seat := f.turn.SeatedFor(f.b)
	if _, _, err := f.session.OpenReaction(seat.GetID()); err != nil {
		t.Fatalf("OpenReaction: %v", err)
	}
	undoStun(f)
	restored, refused := f.session.RestoreSwappedReactions(swapped)
	if len(restored) != 0 || len(refused) != 1 || !strings.Contains(refused[0].Reason, "abriu") {
		t.Fatalf("restore = %+v / %+v, want a refusal: the master opened the seat", restored, refused)
	}
}
