package statuseffect_test

import (
	"reflect"
	"testing"

	se "github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
)

func TestSummarize(t *testing.T) {
	t.Run("empty is the zero Summary", func(t *testing.T) {
		if got := se.Summarize(nil); !reflect.DeepEqual(got, se.Summary{}) {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("stunned", func(t *testing.T) {
		got := se.Summarize([]se.Effect{{Kind: se.KindStunned, Params: se.Params{Points: 1}}})
		if !got.CannotAct || !got.CannotReact || !got.NoDisplacement {
			t.Errorf("flags: %+v", got)
		}
		if len(got.LockedDomains) != 4 {
			t.Errorf("locked: %+v", got.LockedDomains)
		}
		if !reflect.DeepEqual(got.CannotActBy, []se.Kind{se.KindStunned}) {
			t.Errorf("CannotActBy: %+v", got.CannotActBy)
		}
	})

	t.Run("paralyzed attack plus restrained", func(t *testing.T) {
		got := se.Summarize([]se.Effect{
			{Kind: se.KindParalyzed, Params: se.Params{Domains: []se.Domain{se.DomainAttack}, Points: 1}},
			{Kind: se.KindRestrained},
		})
		wantLocked := map[se.Domain][]se.Kind{se.DomainAttack: {se.KindParalyzed}}
		if !reflect.DeepEqual(got.LockedDomains, wantLocked) {
			t.Errorf("locked: %+v", got.LockedDomains)
		}
		if !reflect.DeepEqual(got.NoDisplacementBy, []se.Kind{se.KindRestrained}) || !got.NoDisplacement {
			t.Errorf("displacement: %+v", got)
		}
		wantF := map[se.Reaction][]se.Kind{se.ReactionEscape: {se.KindRestrained}}
		if !reflect.DeepEqual(got.ForbiddenReactions, wantF) {
			t.Errorf("forbidden: %+v", got.ForbiddenReactions)
		}
	})

	t.Run("openGuard plus airborne forbids only forbidden", func(t *testing.T) {
		got := se.Summarize([]se.Effect{{Kind: se.KindOpenGuard}, {Kind: se.KindAirborne}})
		want := map[se.Reaction][]se.Kind{
			se.ReactionReflexDodge: {se.KindAirborne, se.KindOpenGuard},
			se.ReactionRepel:       {se.KindOpenGuard},
			se.ReactionEscape:      {se.KindAirborne},
		}
		if !reflect.DeepEqual(got.ForbiddenReactions, want) {
			t.Errorf("forbidden: %+v", got.ForbiddenReactions)
		}
	})

	t.Run("injured disabled movement warns", func(t *testing.T) {
		got := se.Summarize([]se.Effect{{Kind: se.KindInjured, Params: se.Params{
			Injuries: map[se.Domain]se.InjuryLevel{se.DomainMovement: se.InjuryDisabled}}}})
		want := []se.Note{{Kind: se.KindInjured, Domain: se.DomainMovement}}
		if !reflect.DeepEqual(got.Notes, want) {
			t.Errorf("notes: %+v", got.Notes)
		}
		if got.CannotAct || got.LockedDomains != nil {
			t.Errorf("injured must not block: %+v", got)
		}
	})

	t.Run("compelled note", func(t *testing.T) {
		got := se.Summarize([]se.Effect{{Kind: se.KindCompelled, Params: se.Params{Conduct: "conduct", Irresistible: true}}})
		want := []se.Note{{Kind: se.KindCompelled, Conduct: "conduct", Irresistible: true}}
		if !reflect.DeepEqual(got.Notes, want) {
			t.Errorf("notes: %+v", got.Notes)
		}
	})
}
