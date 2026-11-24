package game_test

import (
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
)

// Um status no ATOR do turno aberto (não só em quem reage) pode mudar o viés de um teste que a
// resolução lê: a sala resolve o turno de novo sob a trava de escrita, onde os dados que faltam
// caem, e manda o resolution_updated ao mestre. Sem isso, a primeira leitura seria a do
// match_full_state, sob RLock, que nunca tira dado.
func TestStatusOnTheOpenTurnsActorResolvesTheTurn(t *testing.T) {
	f := newStatusFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc := collectFrom(master)
	collectFrom(player)
	f.openAttackTurn(t, master, player, mc)
	if !mc.await(game.MsgTypeResolutionUpdate, 2*time.Second) {
		t.Fatal("the open never sent the master a resolution")
	}
	n := mc.count(game.MsgTypeResolutionUpdate)

	applyStatus(t, master, mc, f.attackerID, "openGuard", nil)
	if !awaitCount(mc, game.MsgTypeResolutionUpdate, n+1, 2*time.Second) {
		t.Fatalf("a status on the open turn's actor sent no resolution_updated (still %d)", mc.count(game.MsgTypeResolutionUpdate))
	}
}
