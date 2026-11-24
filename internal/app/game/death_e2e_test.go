package game_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Morte no fechamento do turno (Task 8 da fase 2): o golpe que o fechamento aplica e que deixa o
// HP abaixo do mínimo da barra aplica o Morto. A sala o grava na transação do turno fechado e o
// anuncia como qualquer mudança de status — a névoa decide quem recebe. Mesa: a do
// seedStatusBoard; a vítima é o NPC do mestre, e o atacante é o do jogador.

// npcAtTheMinimum põe o HP do NPC exatamente no mínimo da barra: qualquer dano o mata. Só antes
// de conectar (nada em voo).
func (f *combatFixture) npcAtTheMinimum(t *testing.T) {
	t.Helper()
	sheet, err := f.session.GetCharSheet(f.npcID)
	if err != nil {
		t.Fatalf("GetCharSheet(npc): %v", err)
	}
	bar := sheet.GetAllStatusBar()[enum.Health]
	if err := bar.SetCurrent(bar.GetMin()); err != nil {
		t.Fatalf("SetCurrent: %v", err)
	}
}

// openAttackOnNPC põe na fila o ataque do atacante do jogador no NPC e o abre; devolve o turno
// e a action dele.
func (f *combatFixture) openAttackOnNPC(t *testing.T, master, player *websocket.Conn, mc *collector) (turnID, actionID uuid.UUID) {
	t.Helper()
	f.enqueueAttackAt(t, player, f.attackerID, f.npcID)
	if !mc.await(game.MsgTypeActionQueued, 2*time.Second) {
		t.Fatal("the attack on the NPC was never queued")
	}
	sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
	if !mc.await(game.MsgTypeTurnOpened, 2*time.Second) {
		t.Fatal("no turn opened")
	}
	opened := lastTurnOpened(t, mc)
	return opened.TurnID, opened.ActionID
}

// deadUpdateFor devolve o último status_effects_updated do personagem que já lista o Morto.
func deadUpdateFor(t *testing.T, c *collector, charID uuid.UUID) (game.StatusEffectsUpdatedPayload, int, bool) {
	t.Helper()
	msgs := c.snapshotMessages()
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Type != game.MsgTypeStatusEffectsUpdated {
			continue
		}
		var u game.StatusEffectsUpdatedPayload
		if err := json.Unmarshal(msgs[i].Payload, &u); err != nil {
			t.Fatalf("unmarshal status_effects_updated: %v", err)
		}
		if u.CharacterID != charID {
			continue
		}
		for _, e := range u.StatusEffects {
			if e.Kind == string(statuseffect.KindDead) {
				return u, i, true
			}
		}
	}
	return game.StatusEffectsUpdatedPayload{}, -1, false
}

func awaitDead(t *testing.T, who string, c *collector, charID uuid.UUID) (game.StatusEffectsUpdatedPayload, int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if u, i, ok := deadUpdateFor(t, c, charID); ok {
			return u, i
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("%s: no status_effects_updated with dead for %s; got %v", who, charID, messageTypes(c.snapshotMessages()))
	return game.StatusEffectsUpdatedPayload{}, -1
}

func assertNotDead(t *testing.T, who string, c *collector, charID uuid.UUID) {
	t.Helper()
	if _, _, ok := deadUpdateFor(t, c, charID); ok {
		t.Fatalf("%s: got dead for %s, want nothing — the piece is out of sight", who, charID)
	}
}

func TestDeath(t *testing.T) {
	t.Run("death goes to the master and to who sees the piece", func(t *testing.T) {
		f := newStatusFixture(t)
		f.seedStatusBoard(t, statusNPCInSight)
		f.npcAtTheMinimum(t)
		master, player, blind, mc, pc, bc := f.connectTable(t)
		defer master.Close() //nolint:errcheck
		defer player.Close() //nolint:errcheck
		defer blind.Close()  //nolint:errcheck
		f.openAttackOnNPC(t, master, player, mc)

		start := len(mc.snapshotMessages())
		closeTurnAndWait(t, master, mc)
		u, statusAt := awaitDead(t, "master", mc, f.npcID)
		assertKinds(t, "master", u.StatusEffects, "dead")
		awaitDead(t, "player", pc, f.npcID)
		awaitDead(t, "bystander", bc, f.npcID)

		// A ordem do mestre: o HP que matou, o Morto, e só então o fim do turno.
		hp := firstIndexAfter(mc, game.MsgTypeCharacterHpChanged, start)
		closed := firstIndexAfter(mc, game.MsgTypeTurnClosed, start)
		if hp < 0 || hp > statusAt || statusAt > closed {
			t.Fatalf("master: character_hp_changed at %d, dead at %d, turn_closed at %d — want hp, dead, turn_closed", hp, statusAt, closed)
		}
	})

	t.Run("player out of sight gets dead when the piece enters sight", func(t *testing.T) {
		f := newStatusFixture(t)
		f.seedStatusBoard(t, statusNPCBehindWall, moveBoardWall)
		f.npcAtTheMinimum(t)
		master, player, blind, mc, pc, bc := f.connectTable(t)
		defer master.Close() //nolint:errcheck
		defer player.Close() //nolint:errcheck
		defer blind.Close()  //nolint:errcheck
		f.openAttackOnNPC(t, master, player, mc)

		closeTurnAndWait(t, master, mc)
		awaitDead(t, "master", mc, f.npcID)
		chatBarrier(t, master, mc, pc, bc)
		assertNotDead(t, "player", pc, f.npcID)
		assertNotDead(t, "bystander", bc, f.npcID)

		// A peça entra na visão: a lista atual, com o Morto, chega a quem passou a vê-la.
		sendMasterMove(t, master, f.npcID, [3]int{16, 4, 0})
		u, _ := awaitDead(t, "bystander", bc, f.npcID)
		assertKinds(t, "bystander", u.StatusEffects, "dead")
		awaitDead(t, "player", pc, f.npcID)
	})

	t.Run("death is in the turn close write", func(t *testing.T) {
		f := newStatusFixture(t)
		f.seedStatusBoard(t, statusNPCInSight)
		f.npcAtTheMinimum(t)
		master, player, blind, mc, pc, bc := f.connectTable(t)
		defer master.Close() //nolint:errcheck
		defer player.Close() //nolint:errcheck
		defer blind.Close()  //nolint:errcheck
		turnID, actionID := f.openAttackOnNPC(t, master, player, mc)

		// O NPC tem uma ação na fila: o Morto a tira, como qualquer status, cancelada pelo mestre.
		queuedBefore := mc.count(game.MsgTypeActionQueued)
		f.enqueueAttackAt(t, master, f.npcID, f.victimID)
		if !awaitCount(mc, game.MsgTypeActionQueued, queuedBefore+1, 2*time.Second) {
			t.Fatal("the NPC's attack was never queued")
		}

		closeTurnAndWait(t, master, mc)
		f.awaitPersistedTurn(t, turnID)
		awaitDead(t, "master", mc, f.npcID)
		chatBarrier(t, master, mc, pc, bc)

		ops := lastCloseStatusOps(t, f)
		assertOps(t, "close", ops, op(f.npcID, "dead"))
		ev := ops[0].Event
		if ev == nil || ev.Change != statuseffect.ChangeApplied {
			t.Fatalf("dead event = %+v, want ChangeApplied", ev)
		}
		if ev.TurnID == nil || *ev.TurnID != turnID {
			t.Fatalf("dead event TurnID = %v, want the closed turn %v", ev.TurnID, turnID)
		}
		if ev.ActionID == nil || *ev.ActionID != actionID {
			t.Fatalf("dead event ActionID = %v, want the closed turn's action %v", ev.ActionID, actionID)
		}
		for _, pid := range []uuid.UUID{f.playerUUID, f.bystanderUUID} {
			if _, ok := ev.Views[pid]; !ok {
				t.Fatalf("dead event Views = %v, want %v (they see the NPC)", ev.Views, pid)
			}
		}

		closes := f.roundRepo.closeData()
		cancelled := closes[len(closes)-1].CancelledActions
		if len(cancelled) != 1 || cancelled[0].ActorUUID != f.npcID || cancelled[0].CancelledBy != f.masterUUID {
			t.Fatalf("close CancelledActions = %+v, want the dead NPC's attack, cancelled by the master", cancelled)
		}
		if cancelled[0].TurnUUID == nil || *cancelled[0].TurnUUID != turnID {
			t.Fatalf("cancelled TurnUUID = %v, want the closed turn %v", cancelled[0].TurnUUID, turnID)
		}
		if mc.count(game.MsgTypeActionCancelled) != 1 {
			t.Fatalf("master: got %d action_cancelled, want 1", mc.count(game.MsgTypeActionCancelled))
		}
	})

	t.Run("death at an implicit close rides that turn and its queued action never opens", func(t *testing.T) {
		f := newStatusFixture(t)
		f.seedStatusBoard(t, statusNPCInSight)
		f.npcAtTheMinimum(t)
		master, player, blind, mc, pc, bc := f.connectTable(t)
		defer master.Close() //nolint:errcheck
		defer player.Close() //nolint:errcheck
		defer blind.Close()  //nolint:errcheck
		turnID, _ := f.openAttackOnNPC(t, master, player, mc)
		queuedBefore := mc.count(game.MsgTypeActionQueued)
		f.enqueueAttackAt(t, master, f.npcID, f.victimID)
		if !awaitCount(mc, game.MsgTypeActionQueued, queuedBefore+1, 2*time.Second) {
			t.Fatal("the NPC's attack was never queued")
		}

		// O próximo da fila era o do NPC: o Morto o tira antes da abertura, e nada abre.
		openedBefore := mc.count(game.MsgTypeTurnOpened)
		sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
		if !mc.await(game.MsgTypeTurnClosed, 3*time.Second) {
			t.Fatal("the turn never closed")
		}
		f.awaitPersistedTurn(t, turnID)
		awaitDead(t, "master", mc, f.npcID)
		awaitDead(t, "bystander", bc, f.npcID)
		chatBarrier(t, master, mc, pc, bc)
		if got := mc.count(game.MsgTypeTurnOpened); got != openedBefore {
			t.Fatalf("master: %d turn_opened after the close, want none — the dead NPC's action left the queue", got-openedBefore)
		}
		assertOps(t, "close", lastCloseStatusOps(t, f), op(f.npcID, "dead"))
		closes := f.roundRepo.closeData()
		if c := closes[len(closes)-1].CancelledActions; len(c) != 1 || c[0].ActorUUID != f.npcID {
			t.Fatalf("close CancelledActions = %+v, want the dead NPC's attack", c)
		}
	})
}
