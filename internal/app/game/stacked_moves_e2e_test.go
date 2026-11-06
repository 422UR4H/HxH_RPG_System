package game_test

import (
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/map/service"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Movimentos empilhados (spec 2026-11-06 D3–D6) contra a Room de verdade: o que está sob teste
// é o wire, então toda asserção lê o que um cliente recebeu.

// queuedMoves lê, do action_queued do mestre, o id, o from e o stackedOn de cada movimento, em
// ordem de chegada.
type queuedMove struct {
	id        uuid.UUID
	from      *[3]int
	stackedOn *uuid.UUID
}

func queuedMoves(t *testing.T, c *collector) []queuedMove {
	t.Helper()
	var out []queuedMove
	for _, m := range c.snapshotMessages() {
		if m.Type != game.MsgTypeActionQueued {
			continue
		}
		var p game.ActionQueuedPayload
		if err := json.Unmarshal(m.Payload, &p); err != nil {
			t.Fatalf("unmarshal action_queued: %v", err)
		}
		if p.Action.Move == nil {
			continue
		}
		out = append(out, queuedMove{id: p.ActionID, from: p.Action.Move.From, stackedOn: p.Action.Move.StackedOn})
	}
	return out
}

// D3: a checagem de parede do enqueue parte do fim da cadeia, não da peça.
func TestE2E_StackedMoveWallCheckRunsFromTheChainTail(t *testing.T) {
	cellSize := squareGeomGrid.CellSize
	// A peça em (4,4); o primeiro dash vai a (4,6); o segundo, a um vizinho da coluna 5.
	bandBetween := func(id string, col, row int) mapentity.WallSegment {
		fx, fy := service.SlotCenterToLocal(col, row, squareGeomGrid)
		tx, _ := service.SlotCenterToLocal(col+1, row, squareGeomGrid)
		return bandWall(id, (fx+tx)/2, fy, cellSize)
	}

	setup := func(t *testing.T, wall mapentity.WallSegment) (*combatFixture, *websocket.Conn, *collector, *collector, func()) {
		f := newCombatFixture(t)
		seedGeomBoard(t, f, squareGeomGrid,
			[]mapentity.Piece{squarePiece("p-attacker", f.attackerID, 4, 4)},
			[]mapentity.WallSegment{wall})
		master, player := f.connect(t)
		mc, pc := collectFrom(master), collectFrom(player)
		if !mc.await(game.MsgTypeMapFullState, 2*time.Second) {
			t.Fatal("the master never got the board — the fixture never started")
		}
		sendGeomDash(t, player, f.attackerID, [3]int{9, 9, 0}, [3]int{4, 6, 0})
		if !pc.await(game.MsgTypeActionEnqueued, 2*time.Second) {
			t.Fatalf("the first dash was never enqueued; errors: %v", errorCodes(t, pc))
		}
		return f, player, mc, pc, func() { master.Close(); player.Close() } //nolint:errcheck
	}

	t.Run("a wall between the tail and the new destination refuses", func(t *testing.T) {
		// (4,6)→(5,6) cruza a parede; (4,4)→(5,6) passa uma linha inteira longe dela.
		f, player, _, pc, done := setup(t, bandBetween("tail-wall", 4, 6))
		defer done()
		sendGeomDash(t, player, f.attackerID, [3]int{9, 9, 0}, [3]int{5, 6, 0})
		awaitErrorWithCode(t, pc, "move_blocked", 2*time.Second)
	})

	t.Run("a wall between the piece and the new destination does not", func(t *testing.T) {
		// (4,4)→(5,4) cruzaria a parede; o segundo dash parte de (4,6) e passa longe dela.
		f, player, mc, pc, done := setup(t, bandBetween("piece-wall", 4, 4))
		defer done()
		sendGeomDash(t, player, f.attackerID, [3]int{9, 9, 0}, [3]int{5, 4, 0})
		refutesErrorAfter(t, pc, "move_blocked", game.MsgTypeActionEnqueued, 2*time.Second)
		if !awaitCount(mc, game.MsgTypeActionQueued, 2, 2*time.Second) {
			t.Fatal("the master never got the second action_queued")
		}
		moves := queuedMoves(t, mc)
		if len(moves) != 2 {
			t.Fatalf("master saw %d queued moves, want 2", len(moves))
		}
		second := moves[1]
		if second.stackedOn == nil || *second.stackedOn != moves[0].id {
			t.Errorf("second move.stackedOn = %v, want the first move %v", second.stackedOn, moves[0].id)
		}
		if second.from == nil || *second.from != [3]int{4, 6, 0} {
			t.Errorf("second move.from = %v, want the first move's destination [4 6 0]", second.from)
		}
		if moves[0].stackedOn != nil {
			t.Errorf("first move.stackedOn = %v, want absent", *moves[0].stackedOn)
		}
	})
}

// D5: cancelar a cabeça manda um action_cancelled por action — ao mestre e a quem pediu —, em
// ordem de cadeia, e só então o bars_updated; cada uma é gravada como um cancelamento pedido.
func TestE2E_CancellingAStackedChainAnnouncesEveryAction(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc, pc := collectFrom(master), collectFrom(player)

	for i, to := range [][3]int{{4, 6, 0}, {5, 6, 0}, {6, 6, 0}} {
		f.enqueueDash(t, player, [3]int{0, 0, 0}, to)
		if !awaitCount(pc, game.MsgTypeActionEnqueued, i+1, 2*time.Second) {
			t.Fatalf("dash %d was never enqueued; errors: %v", i+1, errorCodes(t, pc))
		}
	}
	chain := actionIDsInOrder(t, pc)
	if !awaitCount(mc, game.MsgTypeBarsUpdated, 3, 2*time.Second) {
		t.Fatal("the master never saw the order after the enqueues")
	}
	seqBefore := lastBarsUpdated(t, mc).Seq

	sendWS(t, player, "cancel_action", map[string]any{"actionId": chain[0].String()})
	if !awaitBarsAfter(t, mc, seqBefore, 2*time.Second) {
		t.Fatal("the master never saw the order change after the cancel")
	}
	if !awaitCount(pc, game.MsgTypeActionCancelled, 3, 2*time.Second) {
		t.Fatalf("the sender got %d action_cancelled, want 3", pc.count(game.MsgTypeActionCancelled))
	}
	for who, c := range map[string]*collector{"master": mc, "sender": pc} {
		if got := cancelledIDs(t, c); !slices.Equal(got, chain) {
			t.Errorf("%s action_cancelled ids = %v, want %v (one per action, chain order)", who, got, chain)
		}
	}
	// No mestre, todos os action_cancelled chegam antes do bars_updated do cancelamento.
	msgs := mc.snapshotMessages()
	lastCancelled, barsAfter := -1, -1
	for i, m := range msgs {
		switch m.Type {
		case game.MsgTypeActionCancelled:
			lastCancelled = i
		case game.MsgTypeBarsUpdated:
			var p game.BarsUpdatedPayload
			if err := json.Unmarshal(m.Payload, &p); err != nil {
				t.Fatalf("unmarshal bars_updated: %v", err)
			}
			if p.Seq > seqBefore && barsAfter == -1 {
				barsAfter = i
			}
		}
	}
	if lastCancelled == -1 || barsAfter < lastCancelled {
		t.Errorf("master: bars_updated at %d, last action_cancelled at %d — the bars go out after every cancel", barsAfter, lastCancelled)
	}

	recs := f.cancelled.await(t, 3, 2*time.Second)
	var recorded []uuid.UUID
	for _, r := range recs {
		recorded = append(recorded, r.UUID)
		if r.CancelledBy != f.playerUUID {
			t.Errorf("record %v cancelledBy = %v, want the player who cancelled the head", r.UUID, r.CancelledBy)
		}
	}
	if !slices.Equal(recorded, chain) {
		t.Errorf("cancelled_actions = %v, want %v", recorded, chain)
	}
}

// D4: o pull_action de um movimento empilhado chega ao mestre como game_error, e nada abre.
func TestE2E_PullingAStackedMoveIsRefused(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc, pc := collectFrom(master), collectFrom(player)

	for i, to := range [][3]int{{4, 6, 0}, {5, 6, 0}} {
		f.enqueueDash(t, player, [3]int{0, 0, 0}, to)
		if !awaitCount(pc, game.MsgTypeActionEnqueued, i+1, 2*time.Second) {
			t.Fatalf("dash %d was never enqueued; errors: %v", i+1, errorCodes(t, pc))
		}
	}
	chain := actionIDsInOrder(t, pc)

	sendWS(t, master, string(game.MsgTypePullAction), game.PullActionPayload{ActionID: chain[1]})
	e := awaitErrorWithCode(t, mc, "game_error", 2*time.Second)
	if e.Message != "action is stacked on a pending move" {
		t.Errorf("message = %q, want %q", e.Message, "action is stacked on a pending move")
	}
	if n := mc.count(game.MsgTypeTurnOpened); n != 0 {
		t.Errorf("a refused pull opened %d turn(s)", n)
	}
	// O game_error saiu depois do arm terminar: ler a sessão aqui é seguro (nada mais em voo).
	var pending []uuid.UUID
	for _, a := range f.session.PendingActions() {
		pending = append(pending, a.GetID())
	}
	if !slices.Equal(pending, chain) {
		t.Errorf("queue = %v, want unchanged %v", pending, chain)
	}
}

// D6: uma fuga que não vai para onde o primeiro movimento ia consome a cabeça e cancela o que
// estava empilhado nela — um action_cancelled ao mestre e ao dono do reator, gravado com o turno.
func TestE2E_AnEscapeElsewhereCancelsTheStackedMoves(t *testing.T) {
	f := newCombatFixture(t, withVictimPiece)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc, pc := collectFrom(master), collectFrom(player)

	attackID := f.openAttackOn(t, player, master, mc)
	// A vítima (do mesmo jogador) tinha dois movimentos na fila: (6,8) e, empilhado, (6,9).
	for i, to := range [][3]int{{6, 8, 0}, {6, 9, 0}} {
		sendGeomDash(t, player, f.victimID, [3]int{0, 0, 0}, to)
		if !awaitCount(pc, game.MsgTypeActionEnqueued, i+2, 2*time.Second) {
			t.Fatalf("victim dash %d was never enqueued; errors: %v", i+1, errorCodes(t, pc))
		}
	}
	ids := actionIDsInOrder(t, pc)
	head, stacked := ids[1], ids[2]

	f.attachDashEscape(t, player, mc, attackID, [3]int{6, 6, 0}, [3]int{8, 6, 0})
	if !mc.await(game.MsgTypeActionCancelled, 2*time.Second) {
		t.Fatalf("the master never heard of the cascaded cancel; it received: %v", messageTypes(mc.snapshotMessages()))
	}
	if !pc.await(game.MsgTypeActionCancelled, 2*time.Second) {
		t.Fatal("the reactor's owner never heard of the cascaded cancel")
	}
	for who, c := range map[string]*collector{"master": mc, "owner": pc} {
		if got := cancelledIDs(t, c); !slices.Equal(got, []uuid.UUID{stacked}) {
			t.Errorf("%s action_cancelled ids = %v, want only the stacked move %v", who, got, stacked)
		}
	}
	attached := findMessage(t, pc.snapshotMessages(), game.MsgTypeReactionAttached)
	var ack game.ReactionAttachedPayload
	if err := json.Unmarshal(attached.Payload, &ack); err != nil {
		t.Fatalf("unmarshal reaction_attached: %v", err)
	}
	if !slices.Equal(ack.ConsumedActionIDs, []uuid.UUID{head}) {
		t.Errorf("consumedActionIds = %v, want only the head %v", ack.ConsumedActionIDs, head)
	}
	// Com o turno aberto, o registro espera o fechamento dele (como todo cancelamento no turno).
	time.Sleep(100 * time.Millisecond)
	if n := len(f.cancelled.snapshot()); n != 0 {
		t.Errorf("cancelled_actions has %d record(s) mid-turn, want 0 until the close", n)
	}
}
