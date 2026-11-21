package game_test

import (
	"testing"
	"time"
)

// A sessão carimba o instante real da abertura do turno e da reação (Task 15b), em µs; a
// reação nunca aberta não tem um. O fechamento os leva no Turn.
func TestOpeningTimes_StampedOnTheTurn(t *testing.T) {
	h := newHiddenTable(t)
	before := time.Now().Add(-time.Second)
	turnID := h.openPCAttackOnNPC(t)
	reactionID := h.attachNPCDodge(t, lastTurnOpened(t, h.mc).ActionID)
	h.openReaction(t, reactionID)
	closeTurnAndWait(t, h.master, h.mc)
	h.f.awaitPersistedTurn(t, turnID)
	tn := lastClose(t, h.f).Turn

	opened := tn.GetOpenedAt()
	if opened.Before(before) || !opened.Equal(opened.Truncate(time.Microsecond)) {
		t.Fatalf("turn openedAt = %v, want now truncated to µs", opened)
	}
	at, ok := tn.ReactionOpenedAt(reactionID)
	if !ok || at.Before(opened) || !at.Equal(at.Truncate(time.Microsecond)) {
		t.Fatalf("reaction openedAt = %v/%v, want after the turn's %v, in µs", at, ok, opened)
	}
	for _, r := range tn.GetReactions() {
		if r.GetID() == reactionID {
			continue
		}
		if _, ok := tn.ReactionOpenedAt(r.GetID()); ok {
			t.Fatalf("never-opened reaction %s has an opening", r.GetID())
		}
	}
}
