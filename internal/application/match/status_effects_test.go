package match_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

// seSession monta uma sessão com dois PCs (de dois jogadores) e um NPC do mestre.
type seSession struct {
	s              *matchsession.MatchSession
	master, p1, p2 uuid.UUID
	pc1, pc2, npc  uuid.UUID
}

func newSESession(t *testing.T) seSession {
	t.Helper()
	f := seSession{master: uuid.New(), p1: uuid.New(), p2: uuid.New(),
		pc1: uuid.New(), pc2: uuid.New(), npc: uuid.New()}
	parts := []*match.Participant{
		{Sheet: csEntity.Summary{UUID: f.pc1, PlayerUUID: &f.p1}},
		{Sheet: csEntity.Summary{UUID: f.pc2, PlayerUUID: &f.p2}},
		{Sheet: csEntity.Summary{UUID: f.npc, MasterUUID: &f.master}},
	}
	sheets := map[uuid.UUID]*csSheet.CharacterSheet{
		f.pc1: {}, f.pc2: {}, f.npc: {},
	}
	f.s = matchsession.NewMatchSession(uuid.New(), sheets, parts)
	return f
}

func unconscious() statuseffect.Draft { return statuseffect.Draft{Kind: statuseffect.KindUnconscious} }
func hiddenDraft() statuseffect.Draft { return statuseffect.Draft{Kind: statuseffect.KindHidden} }

func TestStatusEffectsUC(t *testing.T) {
	now := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	uc := appmatch.NewStatusEffectsUC()

	t.Run("applies to several characters", func(t *testing.T) {
		f := newSESession(t)
		ch, err := uc.Apply(f.s, f.master, f.master,
			appmatch.ApplyStatusEffectInput{CharacterIDs: []uuid.UUID{f.pc1, f.npc}, Draft: unconscious()}, now)
		if err != nil {
			t.Fatalf("Apply: %v", err)
		}
		if len(ch.Characters) != 2 || ch.Characters[0] != f.pc1 || ch.Characters[1] != f.npc {
			t.Errorf("characters = %v", ch.Characters)
		}
		if len(ch.Writes) != 4 {
			t.Fatalf("writes = %d, want 4 (unconscious + prone per character)", len(ch.Writes))
		}
		for _, w := range ch.Writes {
			if w.Delete {
				t.Error("apply must only upsert")
			}
		}
		if len(f.s.StatusEffectsOf(f.pc1)) != 2 || len(f.s.StatusEffectsOf(f.npc)) != 2 {
			t.Error("session not updated")
		}
		if ch.HiddenChanged {
			t.Error("unconscious is not hidden")
		}
	})

	t.Run("all or nothing when one already has it", func(t *testing.T) {
		f := newSESession(t)
		if _, err := uc.Apply(f.s, f.master, f.master,
			appmatch.ApplyStatusEffectInput{CharacterIDs: []uuid.UUID{f.pc2}, Draft: unconscious()}, now); err != nil {
			t.Fatal(err)
		}
		_, err := uc.Apply(f.s, f.master, f.master,
			appmatch.ApplyStatusEffectInput{CharacterIDs: []uuid.UUID{f.pc1, f.pc2}, Draft: unconscious()}, now)
		if !errors.Is(err, statuseffect.ErrAlreadyApplied) {
			t.Fatalf("err = %v, want ErrAlreadyApplied", err)
		}
		if len(f.s.StatusEffectsOf(f.pc1)) != 0 {
			t.Error("pc1 must stay untouched")
		}
	})

	t.Run("refuses a non-master", func(t *testing.T) {
		f := newSESession(t)
		_, err := uc.Apply(f.s, f.master, f.p1,
			appmatch.ApplyStatusEffectInput{CharacterIDs: []uuid.UUID{f.pc1}, Draft: unconscious()}, now)
		if !errors.Is(err, appmatch.ErrNotMatchMaster) {
			t.Errorf("err = %v", err)
		}
		if _, err := uc.Edit(f.s, f.master, f.p1, uuid.New(), statuseffect.Patch{}, now); !errors.Is(err, appmatch.ErrNotMatchMaster) {
			t.Errorf("edit err = %v", err)
		}
		if _, err := uc.Remove(f.s, f.master, f.p1, uuid.New()); !errors.Is(err, appmatch.ErrNotMatchMaster) {
			t.Errorf("remove err = %v", err)
		}
	})

	t.Run("refuses an ended match", func(t *testing.T) {
		f := newSESession(t)
		ended := endedSession(t)
		_ = f
		_, err := uc.Apply(ended, f.master, f.master,
			appmatch.ApplyStatusEffectInput{CharacterIDs: []uuid.UUID{f.pc1}, Draft: unconscious()}, now)
		if !errors.Is(err, matchsession.ErrMatchEnded) {
			t.Errorf("err = %v", err)
		}
	})

	t.Run("stamps times truncated to microseconds", func(t *testing.T) {
		f := newSESession(t)
		precise := now.Add(123 * time.Nanosecond)
		ch, err := uc.Apply(f.s, f.master, f.master,
			appmatch.ApplyStatusEffectInput{CharacterIDs: []uuid.UUID{f.pc1}, Draft: unconscious()}, precise)
		if err != nil {
			t.Fatalf("apply: %v", err)
		}
		for _, w := range ch.Writes {
			if !w.Effect.AppliedAt.Equal(now) || !w.Effect.UpdatedAt.Equal(now) {
				t.Errorf("%s stamped %v/%v, want %v", w.Effect.Kind, w.Effect.AppliedAt, w.Effect.UpdatedAt, now)
			}
		}
	})

	t.Run("refuses an unknown character", func(t *testing.T) {
		f := newSESession(t)
		_, err := uc.Apply(f.s, f.master, f.master,
			appmatch.ApplyStatusEffectInput{CharacterIDs: []uuid.UUID{f.pc1, uuid.New()}, Draft: unconscious()}, now)
		if !errors.Is(err, matchsession.ErrCharSheetNotFound) {
			t.Errorf("err = %v", err)
		}
		if len(f.s.StatusEffectsOf(f.pc1)) != 0 {
			t.Error("nothing may be applied")
		}
	})

	t.Run("refuses empty or repeated character ids", func(t *testing.T) {
		f := newSESession(t)
		for _, ids := range [][]uuid.UUID{nil, {f.pc1, f.pc1}} {
			_, err := uc.Apply(f.s, f.master, f.master,
				appmatch.ApplyStatusEffectInput{CharacterIDs: ids, Draft: unconscious()}, now)
			if !errors.Is(err, statuseffect.ErrInvalidParams) {
				t.Errorf("ids %v: err = %v", ids, err)
			}
		}
	})

	t.Run("refuses a controller outside the match or equal to the target", func(t *testing.T) {
		f := newSESession(t)
		draft := func(c uuid.UUID) statuseffect.Draft {
			return statuseffect.Draft{Kind: statuseffect.KindControlled,
				Params: statuseffect.ParamsInput{ControllerID: &c}}
		}
		for name, c := range map[string]uuid.UUID{"outside": uuid.New(), "equal": f.pc1} {
			_, err := uc.Apply(f.s, f.master, f.master,
				appmatch.ApplyStatusEffectInput{CharacterIDs: []uuid.UUID{f.pc1}, Draft: draft(c)}, now)
			if !errors.Is(err, statuseffect.ErrInvalidParams) {
				t.Errorf("%s: err = %v", name, err)
			}
		}
		if _, err := uc.Apply(f.s, f.master, f.master,
			appmatch.ApplyStatusEffectInput{CharacterIDs: []uuid.UUID{f.pc1}, Draft: draft(f.pc2)}, now); err != nil {
			t.Errorf("valid controller refused: %v", err)
		}
	})

	t.Run("refuses an audience player who is not a player of the match", func(t *testing.T) {
		f := newSESession(t)
		for name, who := range map[string]uuid.UUID{"stranger": uuid.New(), "master": f.master} {
			d := hiddenDraft()
			d.Audience = &statuseffect.Audience{Players: []uuid.UUID{who}}
			_, err := uc.Apply(f.s, f.master, f.master,
				appmatch.ApplyStatusEffectInput{CharacterIDs: []uuid.UUID{f.pc1}, Draft: d}, now)
			if !errors.Is(err, statuseffect.ErrInvalidAudience) {
				t.Errorf("%s: err = %v", name, err)
			}
		}
		d := hiddenDraft()
		d.Audience = &statuseffect.Audience{Players: []uuid.UUID{f.p2}}
		if _, err := uc.Apply(f.s, f.master, f.master,
			appmatch.ApplyStatusEffectInput{CharacterIDs: []uuid.UUID{f.pc1}, Draft: d}, now); err != nil {
			t.Errorf("real player refused: %v", err)
		}
		// o mesmo vale para a edição
		e := f.s.StatusEffectsOf(f.pc1)[0]
		_, err := uc.Edit(f.s, f.master, f.master, e.ID, statuseffect.Patch{
			SetAudience: true, Audience: &statuseffect.Audience{Players: []uuid.UUID{uuid.New()}}}, now)
		if !errors.Is(err, statuseffect.ErrInvalidAudience) {
			t.Errorf("edit err = %v", err)
		}
	})

	t.Run("edit returns an upsert", func(t *testing.T) {
		f := newSESession(t)
		mustApply(t, uc, f, f.pc1, statuseffect.Draft{Kind: statuseffect.KindBleeding}, now)
		e := f.s.StatusEffectsOf(f.pc1)[0]
		alias := "corte"
		ch, err := uc.Edit(f.s, f.master, f.master, e.ID, statuseffect.Patch{Alias: &alias}, now.Add(time.Minute))
		if err != nil {
			t.Fatal(err)
		}
		if len(ch.Writes) != 1 || ch.Writes[0].Delete || ch.Writes[0].Effect.Alias != "corte" {
			t.Errorf("writes = %+v", ch.Writes)
		}
		if len(ch.Characters) != 1 || ch.Characters[0] != f.pc1 {
			t.Errorf("characters = %v", ch.Characters)
		}
		if got := f.s.StatusEffectsOf(f.pc1)[0]; got.Alias != "corte" {
			t.Errorf("session alias = %q", got.Alias)
		}
	})

	t.Run("remove returns a delete", func(t *testing.T) {
		f := newSESession(t)
		mustApply(t, uc, f, f.pc1, statuseffect.Draft{Kind: statuseffect.KindBleeding}, now)
		e := f.s.StatusEffectsOf(f.pc1)[0]
		ch, err := uc.Remove(f.s, f.master, f.master, e.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(ch.Writes) != 1 || !ch.Writes[0].Delete || ch.Writes[0].Effect.ID != e.ID {
			t.Errorf("writes = %+v", ch.Writes)
		}
		if len(f.s.CharactersWithStatusEffects()) != 0 {
			t.Error("session not updated")
		}
	})

	t.Run("unknown id", func(t *testing.T) {
		f := newSESession(t)
		if _, err := uc.Edit(f.s, f.master, f.master, uuid.New(), statuseffect.Patch{}, now); !errors.Is(err, statuseffect.ErrNotFound) {
			t.Errorf("edit err = %v", err)
		}
		if _, err := uc.Remove(f.s, f.master, f.master, uuid.New()); !errors.Is(err, statuseffect.ErrNotFound) {
			t.Errorf("remove err = %v", err)
		}
	})

	t.Run("hidden changes are flagged", func(t *testing.T) {
		f := newSESession(t)
		ch, err := uc.Apply(f.s, f.master, f.master,
			appmatch.ApplyStatusEffectInput{CharacterIDs: []uuid.UUID{f.pc1}, Draft: hiddenDraft()}, now)
		if err != nil || !ch.HiddenChanged {
			t.Fatalf("apply hidden: changed=%v err=%v", ch.HiddenChanged, err)
		}
		e := f.s.StatusEffectsOf(f.pc1)[0]

		alias := "sombra"
		ch, err = uc.Edit(f.s, f.master, f.master, e.ID, statuseffect.Patch{Alias: &alias}, now)
		if err != nil || ch.HiddenChanged {
			t.Errorf("alias edit: changed=%v err=%v", ch.HiddenChanged, err)
		}
		ch, err = uc.Edit(f.s, f.master, f.master, e.ID, statuseffect.Patch{
			SetAudience: true, Audience: &statuseffect.Audience{Players: []uuid.UUID{f.p2}}}, now)
		if err != nil || !ch.HiddenChanged {
			t.Errorf("audience edit: changed=%v err=%v", ch.HiddenChanged, err)
		}
		ch, err = uc.Remove(f.s, f.master, f.master, e.ID)
		if err != nil || !ch.HiddenChanged {
			t.Errorf("remove hidden: changed=%v err=%v", ch.HiddenChanged, err)
		}
	})
	t.Run("writes carry their change", func(t *testing.T) {
		f := newSESession(t)
		check := func(label string, ws []statuseffect.Write, want statuseffect.Change, n int) {
			t.Helper()
			if len(ws) != n {
				t.Fatalf("%s: %d writes, want %d", label, len(ws), n)
			}
			for i, w := range ws {
				if w.Event == nil {
					t.Fatalf("%s[%d]: no event", label, i)
				}
				// Só a mudança vem do caso de uso; o resto do evento é carimbado pela sala.
				if !reflect.DeepEqual(*w.Event, statuseffect.Event{Change: want}) {
					t.Errorf("%s[%d]: event = %+v, want only Change=%s", label, i, *w.Event, want)
				}
			}
		}
		ch, err := uc.Apply(f.s, f.master, f.master,
			appmatch.ApplyStatusEffectInput{CharacterIDs: []uuid.UUID{f.pc1}, Draft: unconscious()}, now)
		if err != nil {
			t.Fatal(err)
		}
		check("apply unconscious", ch.Writes, statuseffect.ChangeApplied, 2)
		e := f.s.StatusEffectsOf(f.pc1)[0]
		alias := "apagado"
		ch, err = uc.Edit(f.s, f.master, f.master, e.ID, statuseffect.Patch{Alias: &alias}, now)
		if err != nil {
			t.Fatal(err)
		}
		check("edit", ch.Writes, statuseffect.ChangeEdited, 1)
		ch, err = uc.Remove(f.s, f.master, f.master, e.ID)
		if err != nil {
			t.Fatal(err)
		}
		check("remove", ch.Writes, statuseffect.ChangeRemoved, 1)
	})
}

func mustApply(t *testing.T, uc *appmatch.StatusEffectsUC, f seSession, char uuid.UUID, d statuseffect.Draft, now time.Time) {
	t.Helper()
	if _, err := uc.Apply(f.s, f.master, f.master,
		appmatch.ApplyStatusEffectInput{CharacterIDs: []uuid.UUID{char}, Draft: d}, now); err != nil {
		t.Fatalf("setup apply: %v", err)
	}
}

// endedSession devolve uma sessão já encerrada, com os mesmos personagens do newSESession.
func endedSession(t *testing.T) *matchsession.MatchSession {
	t.Helper()
	s := matchsession.NewMatchSession(uuid.New(), nil, nil)
	at := time.Date(2000, 1, 1, 9, 0, 0, 0, time.UTC)
	s.GetActiveRound().MarkCounted()
	if _, err := s.EndMatchAt(at, false, at); err != nil {
		t.Fatalf("end match: %v", err)
	}
	return s
}
