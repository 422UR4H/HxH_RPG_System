package game_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/google/uuid"
)

// This file is the e2e guarantee for B14's server-side piece ownership check (spec §4.3,
// "Quem move o quê"): `piece_moved`/`piece_removed` are a LOBBY-ONLY pair, and even in the
// lobby a player only moves a piece that is already theirs. Every other case (master in the
// lobby, either role once a match has a session) was already covered by the fixture's
// existing tests before this file existed; what is new here is the refusal paths themselves.
//
// combatFixture.sheets (handler_test.go's fakeSheetOwnership) backs RoomDeps.SheetOwnership —
// the lobby has no charToPlayer, so this is what handlePieceMoved reads instead.

// awaitError waits for an `error` message on the collector and returns its payload.
func awaitError(t *testing.T, c *collector) game.ErrorPayload {
	t.Helper()
	if !c.await(game.MsgTypeError, 2*time.Second) {
		t.Fatal("no error message arrived")
	}
	msg := findMessage(t, c.snapshotMessages(), game.MsgTypeError)
	var p game.ErrorPayload
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	return p
}

// O mestre arrasta QUALQUER peça no lobby — inclusive uma que não é do personagem dele —
// e ela é aplicada, retransmitida para a mesa e persistida exatamente uma vez.
func TestLobbyPieceMoved_MasterMovesAnyPiece(t *testing.T) {
	f := newCombatFixture(t, inLobby)
	f.seedBoard(t)

	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	masterMsgs := collectFrom(master)
	playerMsgs := collectFrom(player)

	if !playerMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the player never got the board — the fixture never started")
	}
	baseline := f.boards.saveCount()

	// The attacker piece belongs to f.playerUUID, not the master — proving "any piece",
	// not just one the master happens to own.
	sendPieceMoved(t, master, attackerPieceID, f.attackerID.String(), 6, 4)

	if !playerMsgs.await(game.MsgTypePieceMoved, 2*time.Second) {
		t.Fatal("the master's lobby move was never relayed to the player")
	}
	moved := findMessage(t, playerMsgs.snapshotMessages(), game.MsgTypePieceMoved)
	var mp game.PieceMovedPayload
	if err := json.Unmarshal(moved.Payload, &mp); err != nil {
		t.Fatalf("unmarshal piece_moved: %v", err)
	}
	if mp.Slot.Col == nil || *mp.Slot.Col != 6 || mp.Slot.Row == nil || *mp.Slot.Row != 4 {
		t.Fatalf("piece landed on %+v, want (6,4)", mp.Slot)
	}
	if n := masterMsgs.count(game.MsgTypePieceMoved); n != 0 {
		t.Fatalf("the master was echoed their own drag back %d time(s)", n)
	}
	if codes := errorCodes(t, masterMsgs); len(codes) != 0 {
		t.Fatalf("the master, who runs the lobby, was refused: %v", codes)
	}
	if got := f.boards.saveCount(); got != baseline+1 {
		t.Fatalf("saveCount = %d, want %d — a lobby move persists exactly once", got, baseline+1)
	}
}

// O jogador arrasta a peça JÁ EXISTENTE do próprio personagem, no lobby: aplicado, o mestre
// é avisado, e o remetente não recebe erro nenhum.
func TestLobbyPieceMoved_PlayerMovesOwnExistingPiece(t *testing.T) {
	f := newCombatFixture(t, inLobby)
	f.seedBoard(t)

	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	masterMsgs := collectFrom(master)
	playerMsgs := collectFrom(player)

	if !playerMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the player never got the board — the fixture never started")
	}
	baseline := f.boards.saveCount()

	sendPieceMoved(t, player, attackerPieceID, f.attackerID.String(), 6, 4)

	if !masterMsgs.await(game.MsgTypePieceMoved, 2*time.Second) {
		t.Fatal("the master was never told the player moved their own piece")
	}
	moved := findMessage(t, masterMsgs.snapshotMessages(), game.MsgTypePieceMoved)
	if moved.SenderID != f.playerUUID {
		t.Fatalf("piece_moved senderId = %s, want the mover's own UUID %s",
			moved.SenderID, f.playerUUID)
	}
	if codes := errorCodes(t, playerMsgs); len(codes) != 0 {
		t.Fatalf("the player, moving their own existing piece, was refused: %v", codes)
	}
	if got := f.boards.saveCount(); got != baseline+1 {
		t.Fatalf("saveCount = %d, want %d — a lobby move persists exactly once", got, baseline+1)
	}
}

// O jogador tenta arrastar a peça do personagem de OUTRO jogador (o "bystander"): forbidden,
// nada muda — nem a peça, nem a persistência.
func TestLobbyPieceMoved_PlayerMovingAnotherCharactersPieceIsForbidden(t *testing.T) {
	f := newCombatFixture(t, inLobby, withBystander)
	f.seedBoard(t)

	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	masterMsgs := collectFrom(master)
	playerMsgs := collectFrom(player)

	if !playerMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the player never got the board — the fixture never started")
	}
	baseline := f.boards.saveCount()

	// f.bystanderID belongs to f.bystanderUUID, not f.playerUUID — the sender here.
	sendPieceMoved(t, player, bystanderPieceID, f.bystanderID.String(), 6, 4)

	p := awaitError(t, playerMsgs)
	if p.Code != "forbidden" {
		t.Fatalf("error code = %q, want %q", p.Code, "forbidden")
	}
	if n := masterMsgs.count(game.MsgTypePieceMoved); n != 0 {
		t.Fatalf("a refused move was relayed to the master anyway (%d time(s))", n)
	}
	if got := f.boards.saveCount(); got != baseline {
		t.Fatalf("saveCount = %d, want %d — a refused move must not persist", got, baseline)
	}
}

// O jogador manda piece_moved com um pieceId que o tabuleiro não conhece — na prática, uma
// tentativa de CRIAR uma peça por esse caminho: forbidden. Só o mestre "põe" peça nova
// (Task 5, enqueue_master_action).
func TestLobbyPieceMoved_PlayerCannotCreateAPieceViaAnUnknownPieceID(t *testing.T) {
	f := newCombatFixture(t, inLobby)
	f.seedBoard(t)

	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	masterMsgs := collectFrom(master)
	playerMsgs := collectFrom(player)

	if !playerMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the player never got the board — the fixture never started")
	}
	baseline := f.boards.saveCount()

	unknownPieceID := uuid.New().String()
	sendPieceMoved(t, player, unknownPieceID, f.attackerID.String(), 6, 4)

	p := awaitError(t, playerMsgs)
	if p.Code != "forbidden" {
		t.Fatalf("error code = %q, want %q", p.Code, "forbidden")
	}
	if n := masterMsgs.count(game.MsgTypePieceMoved); n != 0 {
		t.Fatalf("a piece was relayed for an id the board never had (%d time(s))", n)
	}
	if got := f.boards.saveCount(); got != baseline {
		t.Fatalf("saveCount = %d, want %d — a refused move must not persist", got, baseline)
	}
}

// No lobby, remover é só do mestre: o jogador é recusado sem tirar nada do tabuleiro; o
// mestre remove e o tabuleiro persiste.
func TestLobbyPieceRemoved_OnlyTheMasterMayRemove(t *testing.T) {
	t.Run("player is forbidden", func(t *testing.T) {
		f := newCombatFixture(t, inLobby)
		f.seedBoard(t)
		master, player := f.connect(t)
		defer master.Close() //nolint:errcheck
		defer player.Close() //nolint:errcheck
		masterMsgs := collectFrom(master)
		playerMsgs := collectFrom(player)
		if !playerMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
			t.Fatal("the player never got the board — the fixture never started")
		}
		baseline := f.boards.saveCount()

		sendWS(t, player, string(game.MsgTypePieceRemoved),
			game.PieceRemovedPayload{PieceID: attackerPieceID})

		p := awaitError(t, playerMsgs)
		if p.Code != "forbidden" {
			t.Fatalf("error code = %q, want %q", p.Code, "forbidden")
		}
		if n := masterMsgs.count(game.MsgTypePieceRemoved); n != 0 {
			t.Fatalf("a player's piece_removed reached the master anyway (%d time(s))", n)
		}
		if got := f.boards.saveCount(); got != baseline {
			t.Fatalf("saveCount = %d, want %d — a refused removal must not persist", got, baseline)
		}
	})

	t.Run("master is applied and persisted", func(t *testing.T) {
		f := newCombatFixture(t, inLobby)
		f.seedBoard(t)
		master, player := f.connect(t)
		defer master.Close() //nolint:errcheck
		defer player.Close() //nolint:errcheck
		masterMsgs := collectFrom(master)
		playerMsgs := collectFrom(player)
		if !playerMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
			t.Fatal("the player never got the board — the fixture never started")
		}
		baseline := f.boards.saveCount()

		sendWS(t, master, string(game.MsgTypePieceRemoved),
			game.PieceRemovedPayload{PieceID: attackerPieceID})

		if !playerMsgs.await(game.MsgTypePieceRemoved, 2*time.Second) {
			t.Fatal("the master's removal was never relayed to the player")
		}
		if codes := errorCodes(t, masterMsgs); len(codes) != 0 {
			t.Fatalf("the master's own removal was refused: %v", codes)
		}
		if got := f.boards.saveCount(); got != baseline+1 {
			t.Fatalf("saveCount = %d, want %d — a lobby removal persists exactly once",
				got, baseline+1)
		}
	})
}

// Uma vez com sessão viva (partida, não lobby), piece_moved/piece_removed são recusados
// PARA OS DOIS papéis — o mestre passa a mover/pôr/tirar peça por enqueue_master_action
// (Task 5), e o jogador só move agindo (spec §4.3, "Quem move o quê"). O brief só lista os
// dois casos de piece_moved; os dois de piece_removed entram pela mesma linha da tabela
// ("partida | jogador | só por ação | piece_moved/piece_removed recusados") e reaproveitam
// as duas mesmas mensagens — não há um texto exato distinto dado para "remove" e
// enqueue_master_action é o mesmo mecanismo citado para os dois verbos.
func TestPieceMovedAndRemoved_ForbiddenDuringAMatch(t *testing.T) {
	const (
		masterMustUseMasterAction = "during a match the master moves pieces with enqueue_master_action"
		playersMoveByAction       = "players move by action"
	)
	cases := []struct {
		name     string
		msgType  game.MessageType
		asMaster bool
		wantMsg  string
	}{
		{"master piece_moved", game.MsgTypePieceMoved, true, masterMustUseMasterAction},
		{"player piece_moved", game.MsgTypePieceMoved, false, playersMoveByAction},
		{"master piece_removed", game.MsgTypePieceRemoved, true, masterMustUseMasterAction},
		{"player piece_removed", game.MsgTypePieceRemoved, false, playersMoveByAction},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// No inLobby opt: the default fixture is an already-started match, so the
			// session is live the moment the master registers (combatFixture.connect).
			f := newCombatFixture(t)
			f.seedBoard(t)

			master, player := f.connect(t)
			defer master.Close() //nolint:errcheck
			defer player.Close() //nolint:errcheck
			masterMsgs := collectFrom(master)
			playerMsgs := collectFrom(player)
			if !playerMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
				t.Fatal("the player never got the board — the fixture never started")
			}
			baseline := f.boards.saveCount()

			sender, senderMsgs, otherMsgs := master, masterMsgs, playerMsgs
			if !tc.asMaster {
				sender, senderMsgs, otherMsgs = player, playerMsgs, masterMsgs
			}

			if tc.msgType == game.MsgTypePieceMoved {
				sendPieceMoved(t, sender, attackerPieceID, f.attackerID.String(), 6, 4)
			} else {
				sendWS(t, sender, string(game.MsgTypePieceRemoved),
					game.PieceRemovedPayload{PieceID: attackerPieceID})
			}

			p := awaitError(t, senderMsgs)
			if p.Code != "forbidden" {
				t.Fatalf("error code = %q, want %q", p.Code, "forbidden")
			}
			if p.Message != tc.wantMsg {
				t.Fatalf("error message = %q, want %q", p.Message, tc.wantMsg)
			}
			if n := otherMsgs.count(tc.msgType); n != 0 {
				t.Fatalf("a refused %s reached the other party anyway (%d time(s))",
					tc.msgType, n)
			}
			if got := f.boards.saveCount(); got != baseline {
				t.Fatalf("saveCount = %d, want %d — a refused %s must not persist",
					got, baseline, tc.msgType)
			}
		})
	}
}

// O jogador manda um piece_moved válido para a peça própria — a checagem de posse passa —,
// mas ENQUANTO essa leitura (I/O, sem r.mu) ainda está em voo, o MESTRE remove essa mesma
// peça. A aprovação ficou stale: sem o recheck atômico de applyAndRelayPieceMoveIfOwned, o
// movimento do jogador ressuscitaria a peça que o mestre acabou de tirar, retransmitindo e
// persistindo como se a remoção nunca tivesse acontecido (review round 1, Important 1).
func TestLobbyPieceMoved_ARemovalDuringTheOwnershipCheckIsNotResurrected(t *testing.T) {
	f := newCombatFixture(t, inLobby)
	f.seedBoard(t)

	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	masterMsgs := collectFrom(master)
	playerMsgs := collectFrom(player)

	if !playerMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the player never got the board — the fixture never started")
	}
	baseline := f.boards.saveCount()

	entered, release := f.sheets.armBlock(f.attackerID)

	// Starts the player's move — its ownership check for the attacker's sheet blocks.
	sendPieceMoved(t, player, attackerPieceID, f.attackerID.String(), 6, 4)

	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("the player's ownership check never started blocking — " +
			"this test would prove nothing about the race")
	}

	// While that check is stuck in flight, the MASTER removes the exact same piece.
	sendWS(t, master, string(game.MsgTypePieceRemoved),
		game.PieceRemovedPayload{PieceID: attackerPieceID})
	if !playerMsgs.await(game.MsgTypePieceRemoved, 2*time.Second) {
		t.Fatal("the master's removal was never relayed to the player")
	}
	afterRemoval := f.boards.saveCount()
	if afterRemoval != baseline+1 {
		t.Fatalf("saveCount after the master's removal = %d, want %d", afterRemoval, baseline+1)
	}

	// Now the player's stale-approved move is allowed to proceed.
	release()

	p := awaitError(t, playerMsgs)
	if p.Code != "forbidden" {
		t.Fatalf("error code = %q, want %q — a move approved before the removal must not "+
			"resurrect the piece", p.Code, "forbidden")
	}
	if n := masterMsgs.count(game.MsgTypePieceMoved); n != 0 {
		t.Fatalf("the stale-approved move was relayed to the master anyway (%d time(s)) — "+
			"the removed piece came back", n)
	}
	if got := f.boards.saveCount(); got != afterRemoval {
		t.Fatalf("saveCount = %d, want %d — the refused, stale-approved move must not persist "+
			"a resurrection on top of the removal", got, afterRemoval)
	}
}
