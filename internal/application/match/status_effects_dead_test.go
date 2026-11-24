package match_test

import (
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/status"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

// deadFixture é uma sessão com um PC de ficha real (barra de vida com mínimo negativo) que já
// tem o Morto, como o fechamento de turno o deixa: o HP abaixo do mínimo.
type deadFixture struct {
	s      *matchsession.MatchSession
	master uuid.UUID
	pc     uuid.UUID
	bar    status.IStatusBar
	dead   statuseffect.Effect
}

func newDeadFixture(t *testing.T) deadFixture {
	t.Helper()
	f := deadFixture{master: uuid.New(), pc: uuid.New()}
	player := uuid.New()
	sheet := buildPlainSheet(t)
	f.s = matchsession.NewMatchSession(uuid.New(),
		map[uuid.UUID]*csSheet.CharacterSheet{f.pc: sheet},
		[]*match.Participant{{Sheet: csEntity.Summary{UUID: f.pc, PlayerUUID: &player}}})
	f.bar = sheet.GetAllStatusBar()[enum.Health]
	if f.bar.GetMin() >= 0 {
		t.Fatalf("bar min = %d, want negative", f.bar.GetMin())
	}
	if err := f.bar.SetCurrent(f.bar.GetMin() - 7); err != nil {
		t.Fatal(err)
	}
	next, added, err := statuseffect.Apply(nil, f.pc, statuseffect.Draft{Kind: statuseffect.KindDead}, uuid.New,
		time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	f.s.ReplaceStatusEffects(f.pc, next)
	f.dead = added[0]
	return f
}

func intPtr(v int) *int { return &v }

func TestStatusEffectsUC_Dead(t *testing.T) {
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	uc := appmatch.NewStatusEffectsUC()

	t.Run("apply dead refused", func(t *testing.T) {
		f := newSESession(t)
		_, err := uc.Apply(f.s, f.master, f.master, appmatch.ApplyStatusEffectInput{
			CharacterIDs: []uuid.UUID{f.pc1}, Draft: statuseffect.Draft{Kind: statuseffect.KindDead},
		}, now)
		if !errors.Is(err, statuseffect.ErrInvalidParams) || !strings.Contains(err.Error(), "O Morto entra só pela regra.") {
			t.Fatalf("err = %v, want invalid params with the rule text", err)
		}
		if len(f.s.StatusEffectsOf(f.pc1)) != 0 {
			t.Fatal("session changed")
		}
	})

	t.Run("remove dead raises hp to min", func(t *testing.T) {
		f := newDeadFixture(t)
		old := f.bar.GetCurrent()
		ch, err := uc.Remove(f.s, f.master, f.master, f.dead.ID, nil)
		if err != nil {
			t.Fatal(err)
		}
		if f.bar.GetCurrent() != f.bar.GetMin() {
			t.Fatalf("hp = %d, want the min %d", f.bar.GetCurrent(), f.bar.GetMin())
		}
		if len(ch.HP) != 1 || ch.HP[0].CharacterID != f.pc || ch.HP[0].NewHP != f.bar.GetMin() ||
			ch.HP[0].Sheet == nil || ch.HP[0].Damage != old-f.bar.GetMin() {
			t.Fatalf("HP = %+v", ch.HP)
		}
		if len(ch.Writes) != 1 || !ch.Writes[0].Delete || ch.Writes[0].Effect.ID != f.dead.ID {
			t.Fatalf("writes = %+v", ch.Writes)
		}
		if f.s.IsDead(f.pc) {
			t.Fatal("still dead")
		}
	})

	t.Run("remove dead with hp", func(t *testing.T) {
		f := newDeadFixture(t)
		ch, err := uc.Remove(f.s, f.master, f.master, f.dead.ID, intPtr(5))
		if err != nil {
			t.Fatal(err)
		}
		if f.bar.GetCurrent() != 5 || len(ch.HP) != 1 || ch.HP[0].NewHP != 5 {
			t.Fatalf("hp = %d, HP = %+v, want 5", f.bar.GetCurrent(), ch.HP)
		}
	})

	t.Run("hp out of range refused", func(t *testing.T) {
		f := newDeadFixture(t)
		old := f.bar.GetCurrent()
		want := "O HP precisa ficar entre " + strconv.Itoa(f.bar.GetMin()) + " e " + strconv.Itoa(f.bar.GetMax()) + "."
		for _, hp := range []int{f.bar.GetMin() - 1, f.bar.GetMax() + 1} {
			_, err := uc.Remove(f.s, f.master, f.master, f.dead.ID, intPtr(hp))
			if !errors.Is(err, statuseffect.ErrInvalidParams) || !strings.Contains(err.Error(), want) {
				t.Fatalf("hp %d: err = %v, want %q", hp, err, want)
			}
		}
		if !f.s.IsDead(f.pc) || f.bar.GetCurrent() != old {
			t.Fatal("session changed")
		}
		for _, hp := range []int{f.bar.GetMin(), f.bar.GetMax()} {
			g := newDeadFixture(t)
			if _, err := uc.Remove(g.s, g.master, g.master, g.dead.ID, intPtr(hp)); err != nil {
				t.Fatalf("hp %d at the edge: %v", hp, err)
			}
		}
	})

	t.Run("hp on another kind refused", func(t *testing.T) {
		f := newSESession(t)
		mustApply(t, uc, f, f.pc1, statuseffect.Draft{Kind: statuseffect.KindBleeding}, now)
		e := f.s.StatusEffectsOf(f.pc1)[0]
		_, err := uc.Remove(f.s, f.master, f.master, e.ID, intPtr(3))
		if !errors.Is(err, statuseffect.ErrInvalidParams) || !strings.Contains(err.Error(), "Só a remoção do Morto aceita HP.") {
			t.Fatalf("err = %v", err)
		}
		if len(f.s.StatusEffectsOf(f.pc1)) != 1 {
			t.Fatal("session changed")
		}
	})

	t.Run("remove dead with hp equal to current changes nothing", func(t *testing.T) {
		f := newDeadFixture(t)
		if err := f.bar.SetCurrent(f.bar.GetMin() + 2); err != nil {
			t.Fatal(err)
		}
		ch, err := uc.Remove(f.s, f.master, f.master, f.dead.ID, intPtr(f.bar.GetMin()+2))
		if err != nil {
			t.Fatal(err)
		}
		if f.bar.GetCurrent() != f.bar.GetMin()+2 || len(ch.HP) != 0 {
			t.Fatalf("hp = %d, HP = %+v, want unchanged and no HP change", f.bar.GetCurrent(), ch.HP)
		}
		if f.s.IsDead(f.pc) || len(ch.Writes) != 1 {
			t.Fatalf("dead not removed: writes = %+v", ch.Writes)
		}
	})

	t.Run("remove dead with hp already above min keeps it", func(t *testing.T) {
		f := newDeadFixture(t)
		// O mestre já mexeu no HP por outro caminho: sem hp, a remoção não o rebaixa.
		if err := f.bar.SetCurrent(f.bar.GetMin() + 3); err != nil {
			t.Fatal(err)
		}
		ch, err := uc.Remove(f.s, f.master, f.master, f.dead.ID, nil)
		if err != nil {
			t.Fatal(err)
		}
		if f.bar.GetCurrent() != f.bar.GetMin()+3 || len(ch.HP) != 0 {
			t.Fatalf("hp = %d, HP = %+v, want unchanged and no HP change", f.bar.GetCurrent(), ch.HP)
		}
	})
}
