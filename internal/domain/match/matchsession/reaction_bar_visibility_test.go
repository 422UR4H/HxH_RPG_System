package matchsession_test

import (
	"slices"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
)

func TestMatchSession_ReactionSpeedIsSecretUntilOpenedOrClosed(t *testing.T) {
	// A escape reaction charges the action bar; its speed is the one that must stay hidden.
	attached := func(t *testing.T) (secretSpeed int, bs func() ([]int, []int), open func(), closeTurn func()) {
		s, playerB, b, actID := swapSetup(t, dashAction)
		r := escapeTo(b, actID, 3, 2)
		if _, err := s.AttachReaction(playerB, r); err != nil {
			t.Fatalf("AttachReaction: %v", err)
		}
		speed := r.SpeedOn(action.BarAction)
		bs = func() ([]int, []int) {
			_, all := s.BarState(b, action.BarAction)
			_, public := s.PublicBarState(b, action.BarAction)
			return all, public
		}
		open = func() {
			if _, _, err := s.OpenReaction(r.GetID()); err != nil {
				t.Fatalf("OpenReaction: %v", err)
			}
		}
		closeTurn = func() {
			if _, err := s.CloseOpenTurn(); err != nil {
				t.Fatalf("CloseOpenTurn: %v", err)
			}
		}
		return speed, bs, open, closeTurn
	}

	// publicCount compares occurrences, so a speed that equals an opened action's is handled.
	count := func(xs []int, v int) int {
		n := 0
		for _, x := range xs {
			if x == v {
				n++
			}
		}
		return n
	}

	t.Run("attached: BarState has it, PublicBarState does not", func(t *testing.T) {
		speed, bs, _, _ := attached(t)
		all, public := bs()
		if len(public) != len(all)-1 {
			t.Fatalf("all=%v public=%v, want exactly one speed hidden", all, public)
		}
		if count(all, speed) != count(public, speed)+1 {
			t.Errorf("speed %d not hidden: all=%v public=%v", speed, all, public)
		}
	})

	t.Run("opened: PublicBarState shows it", func(t *testing.T) {
		_, bs, open, _ := attached(t)
		open()
		all, public := bs()
		if !slices.Equal(all, public) {
			t.Errorf("all=%v public=%v, want equal after opening", all, public)
		}
	})

	t.Run("closed unopened: PublicBarState shows it", func(t *testing.T) {
		_, bs, _, closeTurn := attached(t)
		closeTurn()
		all, public := bs()
		if !slices.Equal(all, public) {
			t.Errorf("all=%v public=%v, want equal after closing", all, public)
		}
	})
}
