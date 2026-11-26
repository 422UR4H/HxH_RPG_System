package game_test

import (
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
)

// Uma mudança de status re-deriva a ação da fila do personagem (decisão 7) e o mestre recebe a
// ação re-descrita, como numa velocidade editada: action_queued + bars_updated.
func TestStatusChange_ResendsRederivedQueuedAction(t *testing.T) {
	f := newStatusFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc, pc := collectFrom(master), collectFrom(player)

	f.enqueueAttack(t, player)
	if !mc.await(game.MsgTypeActionQueued, 2*time.Second) {
		t.Fatalf("never queued; errors: %v", errorCodes(t, pc))
	}
	queued := lastActionQueued(t, mc)
	before := queued.Action.Speed.RollCheck.Result
	if before == nil {
		t.Fatal("the fresh action_queued has no derived actionSpeed")
	}

	nQueued := mc.count(game.MsgTypeActionQueued)
	nBars := mc.count(game.MsgTypeBarsUpdated)
	nStatus := mc.count(game.MsgTypeStatusEffectsUpdated)
	sendApplyStatus(t, master, map[string]any{
		"characterIds": []string{f.attackerID.String()},
		"kind":         "weakened", "aptitude": "physical",
		"params": map[string]any{"penalty": 3},
	})
	if !awaitCount(mc, game.MsgTypeStatusEffectsUpdated, nStatus+1, 2*time.Second) {
		t.Fatalf("the status never landed; errors: %v", errorCodes(t, mc))
	}
	if !awaitCount(mc, game.MsgTypeActionQueued, nQueued+1, 2*time.Second) {
		t.Fatal("the master never got the action_queued that re-describes the slowed action")
	}
	if !awaitCount(mc, game.MsgTypeBarsUpdated, nBars+1, 2*time.Second) {
		t.Fatal("the bars never went out after the status re-derive")
	}
	got := lastActionQueued(t, mc)
	if got.ActionID != queued.ActionID {
		t.Fatalf("action_queued for %v, want %v", got.ActionID, queued.ActionID)
	}
	after := got.Action.Speed.RollCheck.Result
	if after == nil || *after != *before-3 {
		t.Fatalf("actionSpeed after Weakened = %v, want %d", after, *before-3)
	}

	t.Run("a change that re-derives nothing resends nothing", func(t *testing.T) {
		nQueued := mc.count(game.MsgTypeActionQueued)
		nStatus := mc.count(game.MsgTypeStatusEffectsUpdated)
		sendApplyStatus(t, master, map[string]any{
			"characterIds": []string{f.victimID.String()}, "kind": "weakened", "aptitude": "physical",
			"params": map[string]any{"penalty": 1},
		})
		if !awaitCount(mc, game.MsgTypeStatusEffectsUpdated, nStatus+1, 2*time.Second) {
			t.Fatalf("the status never landed; errors: %v", errorCodes(t, mc))
		}
		chatBarrier(t, master, mc, pc)
		if n := mc.count(game.MsgTypeActionQueued); n != nQueued {
			t.Fatalf("action_queued count = %d, want %d (the victim has nothing queued)", n, nQueued)
		}
	})
}
