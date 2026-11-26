package game_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

// Tirar o Morto (Task 10 da fase 2): a remoção põe o HP com que o personagem volta e esse HP é
// gravado com a remoção — fora de turno na mesma transação da gravação de status, dentro de um
// turno no fechamento dele. Vítima: o NPC do mestre, já morto antes de a mesa conectar.

// npcDead põe o NPC morto como o fechamento o deixa (HP abaixo do mínimo, com o Morto) e devolve
// o id do efeito, o mínimo e o HP de antes. Só antes de conectar (nada em voo).
func (f *combatFixture) npcDead(t *testing.T) (deadID uuid.UUID, min, before int) {
	t.Helper()
	sheet, err := f.session.GetCharSheet(f.npcID)
	if err != nil {
		t.Fatalf("GetCharSheet(npc): %v", err)
	}
	bar := sheet.GetAllStatusBar()[enum.Health]
	before = bar.GetMin() - 5
	if err := bar.SetCurrent(before); err != nil {
		t.Fatalf("SetCurrent: %v", err)
	}
	next, added, err := statuseffect.Apply(f.session.StatusEffectsOf(f.npcID), f.npcID,
		statuseffect.Draft{Kind: statuseffect.KindDead}, uuid.New, time.Now().UTC().Truncate(time.Microsecond))
	if err != nil {
		t.Fatalf("apply dead: %v", err)
	}
	f.session.ReplaceStatusEffects(f.npcID, next)
	return added[0].ID, bar.GetMin(), before
}

func hpChangesFor(t *testing.T, c *collector, charID uuid.UUID) []game.CharacterHpChangedPayload {
	t.Helper()
	var out []game.CharacterHpChangedPayload
	for _, m := range c.snapshotMessages() {
		if m.Type != game.MsgTypeCharacterHpChanged {
			continue
		}
		var p game.CharacterHpChangedPayload
		if err := json.Unmarshal(m.Payload, &p); err != nil {
			t.Fatalf("unmarshal character_hp_changed: %v", err)
		}
		if p.CharacterID == charID {
			out = append(out, p)
		}
	}
	return out
}

func barsOf(list []appmatch.SheetStatusBars, charID uuid.UUID) (appmatch.SheetStatusBars, bool) {
	for _, b := range list {
		if b.CharacterID == charID {
			return b, true
		}
	}
	return appmatch.SheetStatusBars{}, false
}

func TestDeathRevive(t *testing.T) {
	t.Run("remove dead out of turn writes status and hp together", func(t *testing.T) {
		store := &fakeStatusEffectStore{}
		f := newCombatFixture(t, withBystander, withNPCTarget, withStatusEffectStore(store))
		deadID, _, before := f.npcDead(t)
		master, player, blind, mc, pc, bc := f.connectNoBoard(t)
		defer master.Close() //nolint:errcheck
		defer player.Close() //nolint:errcheck
		defer blind.Close()  //nolint:errcheck

		sendWS(t, master, string(game.MsgTypeRemoveStatusEffect), map[string]any{"statusEffectId": deadID, "hp": 3})
		if !mc.await(game.MsgTypeCharacterHpChanged, 2*time.Second) {
			t.Fatalf("master: no character_hp_changed; got %v", messageTypes(mc.snapshotMessages()))
		}
		chatBarrier(t, master, mc, pc, bc)

		hp := hpChangesFor(t, mc, f.npcID)
		if len(hp) != 1 || hp[0].HP != 3 || hp[0].HPChange != 3-before {
			t.Fatalf("master: hp changes = %+v, want one with hp 3 and hpChange %d", hp, 3-before)
		}
		// O NPC é do mestre: nenhum jogador recebe o HP dele.
		if n := len(hpChangesFor(t, pc, f.npcID)) + len(hpChangesFor(t, bc, f.npcID)); n != 0 {
			t.Fatalf("players got %d character_hp_changed for the NPC, want none", n)
		}
		writes, _ := store.snapshot()
		if len(writes) != 1 {
			t.Fatalf("store got %d write(s), want 1", len(writes))
		}
		assertOps(t, "write", writes[0], "-"+op(f.npcID, "dead"))
		b, ok := barsOf(store.barWrites()[0], f.npcID)
		if !ok || b.Health.GetCurrent() != 3 {
			t.Fatalf("bars = %+v, want the NPC's HP 3 in the same write", store.barWrites()[0])
		}
	})

	t.Run("a failed out of turn write keeps status and hp pending together", func(t *testing.T) {
		store := &fakeStatusEffectStore{}
		f := newCombatFixture(t, withBystander, withNPCTarget, withStatusEffectStore(store))
		deadID, min, _ := f.npcDead(t)
		master, player, blind, mc, _, _ := f.connectNoBoard(t)
		defer master.Close() //nolint:errcheck
		defer player.Close() //nolint:errcheck
		defer blind.Close()  //nolint:errcheck

		store.setFail(errors.New("connection reset"))
		sendWS(t, master, string(game.MsgTypeRemoveStatusEffect), map[string]any{"statusEffectId": deadID})
		if !mc.await(game.MsgTypeCharacterHpChanged, 2*time.Second) {
			t.Fatal("master: no character_hp_changed — the HP moved at the table even if the write failed")
		}
		if hp := hpChangesFor(t, mc, f.npcID); len(hp) != 1 || hp[0].HP != min {
			t.Fatalf("master: hp changes = %+v, want one with the min %d", hp, min)
		}
		if writes, attempts := store.snapshot(); len(writes) != 0 || attempts != 1 {
			t.Fatalf("writes=%d attempts=%d, want one failed attempt", len(writes), attempts)
		}

		store.setFail(nil)
		applyProneAndWait(t, master, mc, f.victimID)
		writes, _ := store.snapshot()
		if len(writes) != 1 {
			t.Fatalf("store got %d write(s), want 1", len(writes))
		}
		assertOps(t, "retry", writes[0], "-"+op(f.npcID, "dead"), op(f.victimID, "prone"))
		if b, ok := barsOf(store.barWrites()[0], f.npcID); !ok || b.Health.GetCurrent() != min {
			t.Fatalf("retry bars = %+v, want the NPC's HP %d with the pending removal", store.barWrites()[0], min)
		}

		// Já gravado: a próxima gravação não leva a ficha de novo.
		applyProneAndWait(t, master, mc, f.bystanderID)
		writes, _ = store.snapshot()
		if len(writes) != 2 || len(store.barWrites()[1]) != 0 {
			t.Fatalf("second write bars = %+v, want none", store.barWrites())
		}
	})

	t.Run("remove dead inside a turn waits for the close", func(t *testing.T) {
		store := &fakeStatusEffectStore{}
		f := newCombatFixture(t, withBystander, withNPCTarget, withStatusEffectStore(store))
		deadID, min, _ := f.npcDead(t)
		master, player, blind, mc, _, _ := f.connectNoBoard(t)
		defer master.Close() //nolint:errcheck
		defer player.Close() //nolint:errcheck
		defer blind.Close()  //nolint:errcheck
		turnID := f.openAttackTurn(t, master, player, mc)

		sendWS(t, master, string(game.MsgTypeRemoveStatusEffect), map[string]any{"statusEffectId": deadID})
		if !mc.await(game.MsgTypeCharacterHpChanged, 2*time.Second) {
			t.Fatal("master: no character_hp_changed")
		}
		if writes, attempts := store.snapshot(); len(writes) != 0 || attempts != 0 {
			t.Fatalf("store got %d write(s) (%d attempt(s)) with the turn open, want none", len(writes), attempts)
		}

		closeTurnAndWait(t, master, mc)
		f.awaitPersistedTurn(t, turnID)
		assertOps(t, "close", lastCloseStatusOps(t, f), "-"+op(f.npcID, "dead"))
		closes := f.roundRepo.closeData()
		if b, ok := barsOf(closes[len(closes)-1].StatusBars, f.npcID); !ok || b.Health.GetCurrent() != min {
			t.Fatalf("close bars = %+v, want the NPC's HP %d", closes[len(closes)-1].StatusBars, min)
		}
		if writes, attempts := store.snapshot(); len(writes) != 0 || attempts != 0 {
			t.Fatalf("store got %d write(s) (%d attempt(s)), want none — the turn's changes ride its close", len(writes), attempts)
		}
	})

	t.Run("hp on another kind is refused", func(t *testing.T) {
		f := newStatusFixture(t)
		master, player, blind, mc, _, _ := f.connectNoBoard(t)
		defer master.Close() //nolint:errcheck
		defer player.Close() //nolint:errcheck
		defer blind.Close()  //nolint:errcheck
		applied := applyProneAndWait(t, master, mc, f.victimID)
		sendWS(t, master, string(game.MsgTypeRemoveStatusEffect), map[string]any{"statusEffectId": applied.StatusEffects[0].ID, "hp": 3})
		if !mc.await(game.MsgTypeError, 2*time.Second) {
			t.Fatal("no error")
		}
		if e := lastError(t, mc); e.Code != "status_effect_invalid" || e.Message != "Só a remoção do Morto aceita HP." {
			t.Fatalf("error = %+v", e)
		}
	})
}
