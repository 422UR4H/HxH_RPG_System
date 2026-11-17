package matchsession_test

import (
	"reflect"
	"testing"

	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

func proneOf(charID uuid.UUID) statuseffect.Effect {
	return statuseffect.Effect{ID: uuid.New(), CharacterID: charID, Kind: statuseffect.KindProne}
}

func TestMatchSessionStatusEffects(t *testing.T) {
	t.Run("a character with no effects reads an empty non-nil slice", func(t *testing.T) {
		s := matchsession.NewMatchSession(uuid.New(), nil, nil)
		got := s.StatusEffectsOf(uuid.New())
		if got == nil || len(got) != 0 {
			t.Errorf("got %#v, want empty non-nil", got)
		}
	})

	t.Run("reads are clones", func(t *testing.T) {
		s := matchsession.NewMatchSession(uuid.New(), nil, nil)
		c := uuid.New()
		e := proneOf(c)
		s.SetStatusEffects(map[uuid.UUID][]statuseffect.Effect{c: {e}})

		got := s.StatusEffectsOf(c)
		got[0].Kind = statuseffect.KindAirborne
		got = append(got, proneOf(c))
		_ = got

		if again := s.StatusEffectsOf(c); len(again) != 1 || again[0].Kind != statuseffect.KindProne {
			t.Errorf("session state leaked through a read: %+v", again)
		}
		found, _ := s.FindStatusEffect(e.ID)
		found.Kind = statuseffect.KindAirborne
		if again, _ := s.FindStatusEffect(e.ID); again.Kind != statuseffect.KindProne {
			t.Errorf("session state leaked through Find: %+v", again)
		}
	})

	t.Run("set clones its input", func(t *testing.T) {
		s := matchsession.NewMatchSession(uuid.New(), nil, nil)
		c := uuid.New()
		in := map[uuid.UUID][]statuseffect.Effect{c: {proneOf(c)}}
		s.SetStatusEffects(in)
		in[c][0].Kind = statuseffect.KindAirborne
		if got := s.StatusEffectsOf(c); got[0].Kind != statuseffect.KindProne {
			t.Errorf("input aliased: %+v", got)
		}
	})

	t.Run("replacing with an empty set deletes the character", func(t *testing.T) {
		s := matchsession.NewMatchSession(uuid.New(), nil, nil)
		c := uuid.New()
		s.ReplaceStatusEffects(c, []statuseffect.Effect{proneOf(c)})
		if got := s.CharactersWithStatusEffects(); !reflect.DeepEqual(got, []uuid.UUID{c}) {
			t.Fatalf("characters = %v", got)
		}
		s.ReplaceStatusEffects(c, nil)
		if got := s.CharactersWithStatusEffects(); len(got) != 0 {
			t.Errorf("characters = %v, want none", got)
		}
	})

	t.Run("finds an effect across characters", func(t *testing.T) {
		s := matchsession.NewMatchSession(uuid.New(), nil, nil)
		a, b := uuid.New(), uuid.New()
		ea, eb := proneOf(a), proneOf(b)
		s.ReplaceStatusEffects(a, []statuseffect.Effect{ea})
		s.ReplaceStatusEffects(b, []statuseffect.Effect{eb})
		got, ok := s.FindStatusEffect(eb.ID)
		if !ok || got.CharacterID != b {
			t.Errorf("got %+v ok=%v", got, ok)
		}
		if _, ok := s.FindStatusEffect(uuid.New()); ok {
			t.Error("unknown id must not be found")
		}
	})

	t.Run("summary is the summary of the character's effects", func(t *testing.T) {
		s := matchsession.NewMatchSession(uuid.New(), nil, nil)
		c := uuid.New()
		effects := []statuseffect.Effect{{ID: uuid.New(), CharacterID: c, Kind: statuseffect.KindUnconscious}}
		s.ReplaceStatusEffects(c, effects)
		want := statuseffect.Summarize(effects)
		if got := s.StatusSummaryOf(c); !reflect.DeepEqual(got, want) {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("IsPlayerParticipant is true for players only, not for NPC masters", func(t *testing.T) {
		player, master := uuid.New(), uuid.New()
		pp := &match.Participant{Sheet: csEntity.Summary{UUID: uuid.New(), PlayerUUID: &player}}
		npc := &match.Participant{Sheet: csEntity.Summary{UUID: uuid.New(), MasterUUID: &master}}
		s := matchsession.NewMatchSession(uuid.New(), nil, []*match.Participant{pp, npc})
		if !s.IsPlayerParticipant(player) {
			t.Error("player must be a participant")
		}
		if s.IsPlayerParticipant(master) || s.IsPlayerParticipant(uuid.New()) {
			t.Error("master/stranger must not be a player participant")
		}
	})
}
