package statuseffect_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

var fixedNow = time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)

func counterID() func() uuid.UUID {
	n := byte(0)
	return func() uuid.UUID {
		n++
		return uuid.UUID{15: n}
	}
}

func kinds(es []statuseffect.Effect) []statuseffect.Kind {
	out := make([]statuseffect.Kind, 0, len(es))
	for _, e := range es {
		out = append(out, e.Kind)
	}
	return out
}

func mustApply(t *testing.T, cur []statuseffect.Effect, char uuid.UUID, d statuseffect.Draft, newID func() uuid.UUID) []statuseffect.Effect {
	t.Helper()
	next, _, err := statuseffect.Apply(cur, char, d, newID, fixedNow)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	return next
}

func TestApply(t *testing.T) {
	char := uuid.New()
	stunned := statuseffect.Draft{Kind: statuseffect.KindStunned, Params: statuseffect.ParamsInput{Points: ip(2)}}
	sense := func(s statuseffect.Sense) statuseffect.Draft {
		return statuseffect.Draft{Kind: statuseffect.KindSense, Sense: s,
			Params: statuseffect.ParamsInput{Level: statuseffect.SenseNullified}}
	}
	weak := func(a statuseffect.Aptitude) statuseffect.Draft {
		return statuseffect.Draft{Kind: statuseffect.KindWeakened, Aptitude: a}
	}

	t.Run("adds a normalized effect", func(t *testing.T) {
		aud := &statuseffect.Audience{Everyone: true}
		d := statuseffect.Draft{Kind: statuseffect.KindBleeding, Audience: aud}
		next, added, err := statuseffect.Apply(nil, char, d, counterID(), fixedNow)
		if err != nil {
			t.Fatal(err)
		}
		if len(next) != 1 || len(added) != 1 {
			t.Fatalf("next=%d added=%d", len(next), len(added))
		}
		e := added[0]
		if e.CharacterID != char || e.Kind != statuseffect.KindBleeding || e.Params.HPPerRound != 1 {
			t.Errorf("unexpected effect %+v", e)
		}
		if e.ID == uuid.Nil || !e.AppliedAt.Equal(fixedNow) || !e.UpdatedAt.Equal(fixedNow) {
			t.Errorf("id/timestamps wrong: %+v", e)
		}
		if e.Audience == nil || !e.Audience.Everyone {
			t.Errorf("audience not stored")
		}
	})
	t.Run("refuses the same identity", func(t *testing.T) {
		cur := mustApply(t, nil, char, stunned, counterID())
		next, _, err := statuseffect.Apply(cur, char, stunned, counterID(), fixedNow)
		if !errors.Is(err, statuseffect.ErrAlreadyApplied) || next != nil {
			t.Fatalf("err=%v next=%v", err, next)
		}
	})
	t.Run("different senses coexist", func(t *testing.T) {
		cur := mustApply(t, nil, char, sense(statuseffect.SenseVision), counterID())
		next, _, err := statuseffect.Apply(cur, char, sense(statuseffect.SenseHearing), counterID(), fixedNow)
		if err != nil || len(next) != 2 {
			t.Fatalf("err=%v len=%d", err, len(next))
		}
	})
	t.Run("same sense refused", func(t *testing.T) {
		cur := mustApply(t, nil, char, sense(statuseffect.SenseVision), counterID())
		_, _, err := statuseffect.Apply(cur, char, sense(statuseffect.SenseVision), counterID(), fixedNow)
		if !errors.Is(err, statuseffect.ErrAlreadyApplied) {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("one weakened per aptitude", func(t *testing.T) {
		cur := mustApply(t, nil, char, weak(statuseffect.AptitudePhysical), counterID())
		if _, _, err := statuseffect.Apply(cur, char, weak(statuseffect.AptitudePhysical), counterID(), fixedNow); !errors.Is(err, statuseffect.ErrAlreadyApplied) {
			t.Fatalf("same aptitude: %v", err)
		}
		if _, _, err := statuseffect.Apply(cur, char, weak(statuseffect.AptitudeMental), counterID(), fixedNow); err != nil {
			t.Fatalf("other aptitude: %v", err)
		}
	})
	t.Run("unconscious brings prone", func(t *testing.T) {
		next, added, err := statuseffect.Apply(nil, char, statuseffect.Draft{Kind: statuseffect.KindUnconscious}, counterID(), fixedNow)
		if err != nil {
			t.Fatal(err)
		}
		want := []statuseffect.Kind{statuseffect.KindUnconscious, statuseffect.KindProne}
		if !reflect.DeepEqual(kinds(added), want) || len(next) != 2 {
			t.Fatalf("added=%v next=%d", kinds(added), len(next))
		}
		p := added[1]
		if p.CharacterID != char || p.Alias != "" || p.Audience != nil || p.ID == added[0].ID {
			t.Errorf("prone not default: %+v", p)
		}
	})
	t.Run("unconscious keeps an existing prone", func(t *testing.T) {
		cur := mustApply(t, nil, char, statuseffect.Draft{Kind: statuseffect.KindProne}, counterID())
		next, added, err := statuseffect.Apply(cur, char, statuseffect.Draft{Kind: statuseffect.KindUnconscious}, counterID(), fixedNow)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(kinds(added), []statuseffect.Kind{statuseffect.KindUnconscious}) || len(next) != 2 {
			t.Fatalf("added=%v next=%d", kinds(added), len(next))
		}
	})
	t.Run("alias over 60 runes refused", func(t *testing.T) {
		d := statuseffect.Draft{Kind: statuseffect.KindProne, Alias: strings.Repeat("é", 61)}
		if _, _, err := statuseffect.Apply(nil, char, d, counterID(), fixedNow); !errors.Is(err, statuseffect.ErrInvalidParams) {
			t.Fatalf("err=%v", err)
		}
		d.Alias = strings.Repeat("é", 60)
		if _, _, err := statuseffect.Apply(nil, char, d, counterID(), fixedNow); err != nil {
			t.Fatalf("60 runes: %v", err)
		}
	})
	t.Run("alias is trimmed", func(t *testing.T) {
		_, added, err := statuseffect.Apply(nil, char, statuseffect.Draft{Kind: statuseffect.KindProne, Alias: "  Tropeçou  "}, counterID(), fixedNow)
		if err != nil || added[0].Alias != "Tropeçou" {
			t.Fatalf("err=%v alias=%q", err, added[0].Alias)
		}
	})
	t.Run("does not mutate the input slice", func(t *testing.T) {
		cur := mustApply(t, nil, char, stunned, counterID())
		cur = append(make([]statuseffect.Effect, 0, 10), cur...)
		snapshot := []statuseffect.Effect{cur[0].Clone()}
		next, _, err := statuseffect.Apply(cur, char, statuseffect.Draft{Kind: statuseffect.KindProne}, counterID(), fixedNow)
		if err != nil || len(next) != 2 {
			t.Fatal(err)
		}
		if len(cur) != 1 || !reflect.DeepEqual(cur[0], snapshot[0]) {
			t.Errorf("input mutated")
		}
		if cur[:2][1].Kind != "" {
			t.Errorf("shared backing array was written")
		}
	})
}

func TestEdit(t *testing.T) {
	char := uuid.New()
	later := fixedNow.Add(time.Hour)
	base := func(t *testing.T) []statuseffect.Effect {
		return mustApply(t, nil, char, statuseffect.Draft{
			Kind: statuseffect.KindStunned, Alias: "Zonzo", Audience: &statuseffect.Audience{Everyone: true},
			Params: statuseffect.ParamsInput{Points: ip(2)},
		}, counterID())
	}

	t.Run("replaces params whole", func(t *testing.T) {
		cur := base(t)
		next, edited, err := statuseffect.Edit(cur, cur[0].ID, statuseffect.Patch{Params: &statuseffect.ParamsInput{Points: ip(5)}}, later)
		if err != nil {
			t.Fatal(err)
		}
		if edited.Params.Points != 5 || !edited.UpdatedAt.Equal(later) || !edited.AppliedAt.Equal(fixedNow) {
			t.Errorf("edited=%+v", edited)
		}
		if next[0].Params.Points != 5 || cur[0].Params.Points != 2 {
			t.Errorf("next/cur wrong")
		}
	})
	t.Run("nil params keep params", func(t *testing.T) {
		cur := base(t)
		alias := "Novo"
		_, edited, err := statuseffect.Edit(cur, cur[0].ID, statuseffect.Patch{Alias: &alias}, later)
		if err != nil || edited.Params.Points != 2 || edited.Alias != "Novo" {
			t.Fatalf("err=%v edited=%+v", err, edited)
		}
	})
	t.Run("empty alias clears", func(t *testing.T) {
		cur := base(t)
		empty := "  "
		_, edited, err := statuseffect.Edit(cur, cur[0].ID, statuseffect.Patch{Alias: &empty}, later)
		if err != nil || edited.Alias != "" {
			t.Fatalf("err=%v alias=%q", err, edited.Alias)
		}
	})
	t.Run("set audience nil returns to default", func(t *testing.T) {
		cur := base(t)
		_, edited, err := statuseffect.Edit(cur, cur[0].ID, statuseffect.Patch{SetAudience: true}, later)
		if err != nil || edited.Audience != nil {
			t.Fatalf("err=%v audience=%v", err, edited.Audience)
		}
	})
	t.Run("invalid params leave the set untouched", func(t *testing.T) {
		cur := base(t)
		before := cur[0].Clone()
		next, _, err := statuseffect.Edit(cur, cur[0].ID, statuseffect.Patch{Params: &statuseffect.ParamsInput{Points: ip(0)}}, later)
		if !errors.Is(err, statuseffect.ErrInvalidParams) || next != nil {
			t.Fatalf("err=%v next=%v", err, next)
		}
		if !reflect.DeepEqual(cur[0], before) {
			t.Errorf("set mutated")
		}
	})
	t.Run("unknown id", func(t *testing.T) {
		if _, _, err := statuseffect.Edit(base(t), uuid.New(), statuseffect.Patch{}, later); !errors.Is(err, statuseffect.ErrNotFound) {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestRemove(t *testing.T) {
	char := uuid.New()
	unconscious := func(t *testing.T) []statuseffect.Effect {
		return mustApply(t, nil, char, statuseffect.Draft{Kind: statuseffect.KindUnconscious}, counterID())
	}
	find := func(es []statuseffect.Effect, k statuseffect.Kind) statuseffect.Effect {
		for _, e := range es {
			if e.Kind == k {
				return e
			}
		}
		t.Fatalf("kind %s not found", k)
		return statuseffect.Effect{}
	}

	t.Run("removes by id", func(t *testing.T) {
		cur := mustApply(t, nil, char, statuseffect.Draft{Kind: statuseffect.KindProne}, counterID())
		next, removed, err := statuseffect.Remove(cur, cur[0].ID)
		if err != nil || len(next) != 0 || removed.ID != cur[0].ID || len(cur) != 1 {
			t.Fatalf("err=%v next=%d", err, len(next))
		}
	})
	t.Run("removing prone keeps unconscious", func(t *testing.T) {
		cur := unconscious(t)
		next, _, err := statuseffect.Remove(cur, find(cur, statuseffect.KindProne).ID)
		if err != nil || !reflect.DeepEqual(kinds(next), []statuseffect.Kind{statuseffect.KindUnconscious}) {
			t.Fatalf("err=%v next=%v", err, kinds(next))
		}
	})
	t.Run("removing unconscious keeps prone", func(t *testing.T) {
		cur := unconscious(t)
		next, _, err := statuseffect.Remove(cur, find(cur, statuseffect.KindUnconscious).ID)
		if err != nil || !reflect.DeepEqual(kinds(next), []statuseffect.Kind{statuseffect.KindProne}) {
			t.Fatalf("err=%v next=%v", err, kinds(next))
		}
	})
	t.Run("unknown id", func(t *testing.T) {
		if _, _, err := statuseffect.Remove(unconscious(t), uuid.New()); !errors.Is(err, statuseffect.ErrNotFound) {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestInvalidAudienceRefused(t *testing.T) {
	char := uuid.New()
	bad := &statuseffect.Audience{Everyone: true, Players: []uuid.UUID{uuid.New()}}

	t.Run("invalid audience refused on apply", func(t *testing.T) {
		next, added, err := statuseffect.Apply(nil, char, statuseffect.Draft{Kind: statuseffect.KindProne, Audience: bad}, counterID(), fixedNow)
		if !errors.Is(err, statuseffect.ErrInvalidAudience) || next != nil || added != nil {
			t.Fatalf("err=%v next=%v added=%v", err, next, added)
		}
	})
	t.Run("invalid audience refused on edit", func(t *testing.T) {
		cur := mustApply(t, nil, char, statuseffect.Draft{Kind: statuseffect.KindProne}, counterID())
		next, _, err := statuseffect.Edit(cur, cur[0].ID, statuseffect.Patch{SetAudience: true, Audience: bad}, fixedNow)
		if !errors.Is(err, statuseffect.ErrInvalidAudience) || next != nil {
			t.Fatalf("err=%v next=%v", err, next)
		}
	})
	t.Run("nil audience on edit resets to default", func(t *testing.T) {
		cur := mustApply(t, nil, char, statuseffect.Draft{Kind: statuseffect.KindProne, Audience: &statuseffect.Audience{Everyone: true}}, counterID())
		next, _, err := statuseffect.Edit(cur, cur[0].ID, statuseffect.Patch{SetAudience: true}, fixedNow)
		if err != nil || next[0].Audience != nil {
			t.Fatalf("err=%v", err)
		}
	})
}
