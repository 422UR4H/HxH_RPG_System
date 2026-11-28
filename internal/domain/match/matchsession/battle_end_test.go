package matchsession_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

func TestEndBattleStatusEffects(t *testing.T) {
	t.Run("removes every open guard and keeps the rest", func(t *testing.T) {
		s := matchsession.NewMatchSession(uuid.New(), nil, nil)
		a, b, c := uuid.New(), uuid.New(), uuid.New()
		guardA := statuseffect.Effect{ID: uuid.New(), CharacterID: a, Kind: statuseffect.KindOpenGuard}
		guardB := statuseffect.Effect{ID: uuid.New(), CharacterID: b, Kind: statuseffect.KindOpenGuard}
		proneB := proneOf(b)
		proneC := proneOf(c)
		s.SetStatusEffects(map[uuid.UUID][]statuseffect.Effect{
			a: {guardA}, b: {proneB, guardB}, c: {proneC},
		})

		chars, writes := s.EndBattleStatusEffects()

		if len(chars) != 2 || len(writes) != 2 {
			t.Fatalf("chars/writes = %v/%d, want 2/2", chars, len(writes))
		}
		gone := map[uuid.UUID]bool{}
		for _, w := range writes {
			if !w.Delete || w.Effect.Kind != statuseffect.KindOpenGuard {
				t.Fatalf("write = %+v, want an open guard removal", w)
			}
			if w.Event == nil || w.Event.Change != statuseffect.ChangeRemoved {
				t.Fatalf("event = %+v, want removed", w.Event)
			}
			gone[w.Effect.ID] = true
		}
		if !gone[guardA.ID] || !gone[guardB.ID] {
			t.Fatalf("removed = %v, want both guards", gone)
		}
		if got := s.StatusEffectsOf(a); len(got) != 0 {
			t.Errorf("a keeps %+v", got)
		}
		if got := s.StatusEffectsOf(b); len(got) != 1 || got[0].ID != proneB.ID {
			t.Errorf("b = %+v, want only prone", got)
		}
		if got := s.StatusEffectsOf(c); len(got) != 1 || got[0].ID != proneC.ID {
			t.Errorf("c = %+v, want only prone", got)
		}
	})

	t.Run("nothing to remove", func(t *testing.T) {
		s := matchsession.NewMatchSession(uuid.New(), nil, nil)
		c := uuid.New()
		s.SetStatusEffects(map[uuid.UUID][]statuseffect.Effect{c: {proneOf(c)}})
		chars, writes := s.EndBattleStatusEffects()
		if len(chars) != 0 || len(writes) != 0 {
			t.Fatalf("chars/writes = %v/%v, want none", chars, writes)
		}
	})
}
