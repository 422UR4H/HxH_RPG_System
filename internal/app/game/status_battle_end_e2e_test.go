package game_test

import (
	"slices"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Fim da batalha (spec §10, decisão do dono 2026-11-27): o mestre passa o round do turno
// disputado (Race) para o livre (Free) e o sistema tira toda Guarda aberta da mesa. Só essa
// transição; nunca é um desfazer.

// switchRoundMode troca o regime e espera o n-ésimo round_mode_changed do mestre.
func switchRoundMode(t *testing.T, master *websocket.Conn, mc *collector, mode enum.RoundMode, n int) {
	t.Helper()
	sendWS(t, master, string(game.MsgTypeChangeRoundMode), game.ChangeRoundModePayload{Mode: string(mode)})
	if !awaitCount(mc, game.MsgTypeRoundModeChanged, n, 2*time.Second) {
		t.Fatalf("round_mode_changed #%d (%s) never arrived; errors: %v", n, mode, errorCodes(t, mc))
	}
}

// statusUpdatesOf são os status_effects_updated de charID recebidos depois dos primeiros skip.
func statusUpdatesOf(t *testing.T, c *collector, charID uuid.UUID, skip int) []game.StatusEffectsUpdatedPayload {
	t.Helper()
	var out []game.StatusEffectsUpdatedPayload
	for _, p := range statusUpdates(t, c)[skip:] {
		if p.CharacterID == charID {
			out = append(out, p)
		}
	}
	return out
}

// assertGuardLeft confere que who recebeu, depois de skip, a lista nova de charID sem a Guarda
// aberta, e com want.
func assertGuardLeft(t *testing.T, who string, c *collector, charID uuid.UUID, skip int, want ...string) {
	t.Helper()
	got := statusUpdatesOf(t, c, charID, skip)
	if len(got) != 1 {
		t.Fatalf("%s: got %d status_effects_updated for %s after the battle end, want 1", who, len(got), charID)
	}
	assertKinds(t, who, got[0].StatusEffects, want...)
}

func assertNoUndo(t *testing.T, c *collector) {
	t.Helper()
	for _, m := range c.snapshotMessages() {
		if m.Type == game.MsgTypeActionRestored || m.Type == game.MsgTypeReactionRestored {
			t.Fatalf("got %s: the battle end is never an undo", m.Type)
		}
	}
	if slices.Contains(errorCodes(t, c), "restore_refused") {
		t.Fatal("got restore_refused: the battle end is never an undo")
	}
}

func TestBattleEndRemovesOpenGuard(t *testing.T) {
	store := &fakeStatusEffectStore{}
	f := newCombatFixture(t, withBystander, withNPCTarget, withStatusEffectStore(store))
	master, player, blind, mc, pc, bc := f.connectNoBoard(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	defer blind.Close()  //nolint:errcheck

	switchRoundMode(t, master, mc, enum.Race, 1)
	applyStatus(t, master, mc, f.victimID, "openGuard", nil)
	applyStatus(t, master, mc, f.npcID, "openGuard", nil)
	applyStatus(t, master, mc, f.bystanderID, "prone", nil)
	// Sem tabuleiro todos enxergam todos: cada um recebe as três listas.
	awaitStatusCount(t, "player", pc, 3)
	awaitStatusCount(t, "bystander", bc, 3)
	writesBefore, _ := store.snapshot()
	skipM, skipP, skipB := 3, 3, 3

	switchRoundMode(t, master, mc, enum.Free, 2)
	awaitStatusCount(t, "master", mc, skipM+2)
	awaitStatusCount(t, "player", pc, skipP+2)
	awaitStatusCount(t, "bystander", bc, skipB+2)
	for who, c := range map[string]*collector{"master": mc, "player": pc, "bystander": bc} {
		skip := map[string]int{"master": skipM, "player": skipP, "bystander": skipB}[who]
		assertGuardLeft(t, who, c, f.victimID, skip)
		assertGuardLeft(t, who, c, f.npcID, skip)
		if got := statusUpdatesOf(t, c, f.bystanderID, skip); len(got) != 0 {
			t.Fatalf("%s: the prone character got %d update(s), want none", who, len(got))
		}
	}
	assertNoUndo(t, mc)

	// Fora de turno: a saída é gravada já, com o evento de cada remoção, sem âncora.
	writes, _ := store.snapshot()
	if len(writes) != len(writesBefore)+1 {
		t.Fatalf("store got %d write(s) after the battle end, want 1", len(writes)-len(writesBefore))
	}
	exit := writes[len(writes)-1]
	got := opsOn(exit)
	slices.Sort(got)
	want := []string{"-" + op(f.npcID, "openGuard"), "-" + op(f.victimID, "openGuard")}
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("exit ops = %v, want %v", got, want)
	}
	for _, w := range exit {
		if w.Event == nil || w.Event.Change != statuseffect.ChangeRemoved || w.Event.ID == uuid.Nil {
			t.Fatalf("event = %+v, want a stamped removal", w.Event)
		}
		if w.Event.ActionID != nil || w.Event.TurnID != nil {
			t.Fatalf("turn/action = %v/%v, want nil outside a turn", w.Event.TurnID, w.Event.ActionID)
		}
	}
}

func TestBattleEndWithOpenTurnRecordsOnClose(t *testing.T) {
	store := &fakeStatusEffectStore{}
	f := newCombatFixture(t, withBystander, withNPCTarget, withStatusEffectStore(store))
	master, player, blind, mc, pc, _ := f.connectNoBoard(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	defer blind.Close()  //nolint:errcheck

	switchRoundMode(t, master, mc, enum.Race, 1)
	// A Guarda aberta do NPC é de antes do turno (já gravada); a da vítima, deste turno.
	applyStatus(t, master, mc, f.npcID, "openGuard", nil)
	turnID := f.openAttackTurn(t, master, player, mc)
	applyStatus(t, master, mc, f.victimID, "openGuard", nil)
	writesBefore, attemptsBefore := store.snapshot()
	skipM := mc.count(game.MsgTypeStatusEffectsUpdated)
	skipP := pc.count(game.MsgTypeStatusEffectsUpdated)

	switchRoundMode(t, master, mc, enum.Free, 2)
	// A saída vale já, para todos.
	awaitStatusCount(t, "master", mc, skipM+2)
	awaitStatusCount(t, "player", pc, skipP+2)
	assertGuardLeft(t, "master", mc, f.victimID, skipM)
	assertGuardLeft(t, "master", mc, f.npcID, skipM)
	assertGuardLeft(t, "player", pc, f.victimID, skipP)
	assertNoUndo(t, mc)
	if writes, attempts := store.snapshot(); len(writes) != len(writesBefore) || attempts != attemptsBefore {
		t.Fatalf("store got %d write(s) (%d attempt(s)) with the turn open, want none",
			len(writes)-len(writesBefore), attempts-attemptsBefore)
	}

	closeTurnAndWait(t, master, mc)
	f.awaitPersistedTurn(t, turnID)
	ops := lastCloseStatusOps(t, f)
	// A aplicação deste turno não é desfeita: ela e a saída vão juntas, nessa ordem.
	if len(ops) != 3 || opsOn(ops)[0] != op(f.victimID, "openGuard") {
		t.Fatalf("close ops = %v, want the in-turn apply then both exits", opsOn(ops))
	}
	exits := opsOn(ops[1:])
	slices.Sort(exits)
	want := []string{"-" + op(f.npcID, "openGuard"), "-" + op(f.victimID, "openGuard")}
	slices.Sort(want)
	if !slices.Equal(exits, want) {
		t.Fatalf("exit ops = %v, want %v", exits, want)
	}
	for _, w := range ops[1:] {
		if w.Event == nil || w.Event.Change != statuseffect.ChangeRemoved {
			t.Fatalf("event = %+v, want a removal", w.Event)
		}
		if w.Event.ActionID != nil || w.Event.TurnID == nil || *w.Event.TurnID != turnID {
			t.Fatalf("turn/action = %v/%v, want turn %s and no action", w.Event.TurnID, w.Event.ActionID, turnID)
		}
	}
	if writes, attempts := store.snapshot(); len(writes) != len(writesBefore) || attempts != attemptsBefore {
		t.Fatalf("store got %d write(s) (%d attempt(s)), want none — the exits ride the close",
			len(writes)-len(writesBefore), attempts-attemptsBefore)
	}
}

func TestRaceDoesNotRemoveOpenGuard(t *testing.T) {
	store := &fakeStatusEffectStore{}
	f := newCombatFixture(t, withBystander, withNPCTarget, withStatusEffectStore(store))
	master, player, blind, mc, _, _ := f.connectNoBoard(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	defer blind.Close()  //nolint:errcheck

	applyStatus(t, master, mc, f.victimID, "openGuard", nil)
	writesBefore, _ := store.snapshot()
	skip := mc.count(game.MsgTypeStatusEffectsUpdated)

	switchRoundMode(t, master, mc, enum.Race, 1) // free → race
	switchRoundMode(t, master, mc, enum.Race, 2) // race → race (nada muda)
	// Barreira: a lista de outro personagem chega depois de qualquer coisa que as trocas
	// tivessem mandado.
	applyStatus(t, master, mc, f.bystanderID, "prone", nil)
	if got := statusUpdatesOf(t, mc, f.victimID, skip); len(got) != 0 {
		t.Fatalf("the victim got %d update(s), want none — only race → free ends the battle", len(got))
	}
	writes, _ := store.snapshot()
	for _, w := range writes[len(writesBefore):] {
		for _, o := range w {
			if o.Delete {
				t.Fatalf("store got a removal %v, want none", opsOn(w))
			}
		}
	}
}
