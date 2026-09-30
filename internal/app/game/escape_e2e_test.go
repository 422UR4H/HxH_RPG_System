package game_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// ─── B13: a peça do escape espera o fechamento, e a queda é do mestre ──────
//
// Escapar exige a esquiva E o movimento, os dois contra o acerto do atacante (spec §4.4,
// front-combat-phases.md §6A.5 B13). Como qualquer escape pode falhar, NENHUM desloca na
// abertura da reação — nem o de Shift, que antes pisava ali. No fechamento, pelos três verbos
// que fecham:
//
//   - passou → a peça vai ao destino pedido;
//   - falhou, e o mestre escolheu onde ela caiu (edit_action.escapeLanding) → vai para lá;
//   - falhou, sem escolha → fica onde estava.
//
// A escolha é permanente enquanto o turno está aberto, e só vale se, no fechamento, o escape
// falhou: se o mestre muda a leitura e o escape passa a passar, a peça vai ao DESTINO.
//
// Os testes decidem o escape pelas alavancas do próprio mestre (condições em moveSpeed e no
// acerto do ataque — ver setEscapeReading), não por dados roteirizados — a mesma escolha de biasReactionMoveSpeed, e
// pela mesma razão: amarrar cada asserção ao número exato de rolagens que um ataque e uma
// reação fazem hoje.

// The escape the tests below attach: from the victim's own slot to (8,6). escapeLanding is
// the other slot the master picks when it fails.
var (
	escapeFrom    = [3]int{6, 6, 0}
	escapeTo      = [3]int{8, 6, 0}
	escapeLanding = [3]int{7, 6, 0}
)

// setEscapeReading decides the escape with the master's own levers, over the real edit_action
// surface, so each half clears the attacker's hit because the TEST says so, not the dice.
//
// The movement is a condition on the reaction's moveSpeed (±5000). The dodge cannot be levered
// the same way — a condition on the reaction's dodge does not read through to the resolver's
// Reflex derivation today — so it is decided from the other side: a condition on the attack's
// own hit, −1000 (the dodge clears it) or +1000 (it cannot). ±5000 stays clear of both hits.
func (f *combatFixture) setEscapeReading(
	t *testing.T, master *websocket.Conn, masterMsgs *collector, reactionID uuid.UUID,
	movePasses, dodgePasses bool,
) {
	t.Helper()
	hit, move := 1000, -5000
	if dodgePasses {
		hit = -1000
	}
	if movePasses {
		move = 5000
	}
	for _, edit := range []game.EditActionPayload{
		// ActionID omitted: the turn's own action, the attack.
		{Conditions: []game.ConditionEditPayload{
			{Field: "hit", Modifier: hit, Description: "test lever: the CD both halves read against"},
		}},
		{ActionID: reactionID, Conditions: []game.ConditionEditPayload{
			{Field: "moveSpeed", Modifier: move, Description: "test lever: the movement"},
		}},
	} {
		before := masterMsgs.count(game.MsgTypeActionEdited)
		sendWS(t, master, string(game.MsgTypeEditAction), edit)
		if !awaitCount(masterMsgs, game.MsgTypeActionEdited, before+1, 2*time.Second) {
			t.Fatalf("the reading edit was never acknowledged; the master received: %v",
				messageTypes(masterMsgs.snapshotMessages()))
		}
	}
}

// chooseLanding sends edit_action's escapeLanding — the master's choice of where a failed
// escape leaves the piece; nil clears it — and waits for the recomputed resolution_updated
// that follows the action_edited ack, returning it.
func (f *combatFixture) chooseLanding(
	t *testing.T, master *websocket.Conn, masterMsgs *collector, reactionID uuid.UUID, pos *[3]int,
) game.ResolutionUpdatedPayload {
	t.Helper()
	edited := masterMsgs.count(game.MsgTypeActionEdited)
	resolved := masterMsgs.count(game.MsgTypeResolutionUpdate)
	sendWS(t, master, string(game.MsgTypeEditAction), game.EditActionPayload{
		ActionID:      reactionID,
		EscapeLanding: &game.EscapeLandingPayload{Position: pos},
	})
	if !awaitCount(masterMsgs, game.MsgTypeActionEdited, edited+1, 2*time.Second) {
		t.Fatalf("edit_action with escapeLanding was never acknowledged; the master received: %v",
			messageTypes(masterMsgs.snapshotMessages()))
	}
	if !awaitCount(masterMsgs, game.MsgTypeResolutionUpdate, resolved+1, 2*time.Second) {
		t.Fatal("no resolution_updated followed the landing choice")
	}
	return lastResolutionUpdated(t, masterMsgs)
}

// escapeStage seeds nothing (the caller seeds the board before connect), waits for the board,
// opens the attack, attaches the victim's escape of the given kind from escapeFrom to escapeTo,
// sets its reading and OPENS it. It returns the reaction's ID.
//
// It does not assert on piece_moved at the opening — TestE2E_OpeningAnEscapeNeverMovesThePiece
// is the test for that, and every close-side test below counts piece_moved from zero, which
// catches an opening that moved just as well.
func (f *combatFixture) escapeStage(
	t *testing.T, master, player *websocket.Conn, masterMsgs, playerMsgs *collector,
	closed bool, movePasses, dodgePasses bool,
) uuid.UUID {
	t.Helper()
	if !masterMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the master never got the board")
	}
	if !playerMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the player never got the board")
	}
	actionID := f.openAttackOn(t, player, master, masterMsgs)
	var reactionID uuid.UUID
	if closed {
		reactionID = f.attachClosedEscape(t, player, masterMsgs, actionID, escapeFrom, escapeTo)
	} else {
		reactionID = f.attachDashEscape(t, player, masterMsgs, actionID, escapeFrom, escapeTo)
	}
	f.setEscapeReading(t, master, masterMsgs, reactionID, movePasses, dodgePasses)

	resolved := masterMsgs.count(game.MsgTypeResolutionUpdate)
	sendWS(t, master, string(game.MsgTypeOpenReaction), game.OpenReactionPayload{ReactionID: reactionID})
	if !masterMsgs.await(game.MsgTypeReactionOpened, 2*time.Second) {
		t.Fatalf("the reaction never opened; the master received: %v",
			messageTypes(masterMsgs.snapshotMessages()))
	}
	// The resolution recomputed by the opening is the first one in which the escape is a step
	// of the chain (an unopened reaction is only named in pendingReactions). It travels on a
	// different lane from reaction_opened, so it is waited for on its own.
	if !awaitCount(masterMsgs, game.MsgTypeResolutionUpdate, resolved+1, 2*time.Second) {
		t.Fatal("no resolution_updated followed the opening")
	}
	return reactionID
}

// escapeOf reads the victim's escape off a resolution_updated payload.
func (f *combatFixture) escapeOf(t *testing.T, p game.ResolutionUpdatedPayload) *game.EscapeResultPayload {
	t.Helper()
	for _, tr := range p.Targets {
		if tr.TargetID == f.victimID {
			if tr.Escape == nil {
				t.Fatalf("the victim's result carries no escape: %+v", tr)
			}
			return tr.Escape
		}
	}
	t.Fatalf("no result for the victim in %+v", p.Targets)
	return nil
}

// movedTo reads the one piece_moved the master received and returns the slot it put the
// victim's piece on, asserting the invariants the shared displacement path keeps.
func movedTo(t *testing.T, c *collector) (col, row int) {
	t.Helper()
	moved := findMessage(t, c.snapshotMessages(), game.MsgTypePieceMoved)
	var mp game.PieceMovedPayload
	if err := json.Unmarshal(moved.Payload, &mp); err != nil {
		t.Fatalf("unmarshal piece_moved: %v", err)
	}
	if mp.PieceID != victimPieceID {
		t.Fatalf("piece_moved carried pieceId %q, want the escaping target's %q", mp.PieceID, victimPieceID)
	}
	if mp.Slot.Col == nil || mp.Slot.Row == nil || mp.Slot.Kind != "square" {
		t.Fatalf("piece_moved carried no square slot: %+v", mp.Slot)
	}
	// Z is metres, a Move's position[2] is a grid index; the landing path must not flatten
	// the token any more than the destination path does.
	if mp.Z != victimElevation {
		t.Fatalf("piece_moved carried z=%v, want the elevation it already had (%v)", mp.Z, victimElevation)
	}
	if moved.SenderID != uuid.Nil {
		t.Fatalf("piece_moved senderId = %s, want the zero UUID of a server message", moved.SenderID)
	}
	return *mp.Slot.Col, *mp.Slot.Row
}

// closingVerbs are the three ways room.go closes a turn. close_turn is the explicit one; the
// other two close the open turn on their way to opening the next — with the queue empty
// (open_next_action) or the action unknown (pull_action) they still close first and only then
// report the failure, which is what makes the master's error message a safe barrier.
var closingVerbs = []struct {
	name    string
	close   func(t *testing.T, master *websocket.Conn)
	barrier game.MessageType
}{
	{
		name: "close_turn",
		close: func(t *testing.T, master *websocket.Conn) {
			sendWS(t, master, string(game.MsgTypeCloseTurn), game.CloseTurnPayload{Confirm: true})
		},
		barrier: game.MsgTypeTurnClosed,
	},
	{
		name: "open_next_action on an empty queue",
		close: func(t *testing.T, master *websocket.Conn) {
			sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
		},
		barrier: game.MsgTypeError,
	},
	{
		name: "pull_action for an action that is not queued",
		close: func(t *testing.T, master *websocket.Conn) {
			sendWS(t, master, string(game.MsgTypePullAction), game.PullActionPayload{ActionID: uuid.New()})
		},
		barrier: game.MsgTypeError,
	},
}

// Any escape can fail, so none of them moves when the master gives it the floor — not even the
// Shift one, which used to step there because it rolls nothing. Both set up to PASS: only an
// escape that would clear its test can prove the gate is the opening, not the outcome.
func TestE2E_OpeningAnEscapeNeverMovesThePiece(t *testing.T) {
	for _, c := range []struct {
		name   string
		closed bool
	}{
		{"escape (Dash)", false},
		{"closedEscape (Shift)", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newCombatFixture(t, withVictimPiece)
			f.seedBoard(t)
			master, player := f.connect(t)
			defer master.Close() //nolint:errcheck
			defer player.Close() //nolint:errcheck
			masterMsgs := collectFrom(master)
			playerMsgs := collectFrom(player)

			f.escapeStage(t, master, player, masterMsgs, playerMsgs, c.closed, true, true)

			// reaction_opened is the barrier: anything the open_reaction arm was going to put on
			// the board would already be queued ahead of it.
			if n := masterMsgs.count(game.MsgTypePieceMoved); n != 0 {
				t.Fatalf("the escape displaced the piece %d time(s) when the reaction OPENED", n)
			}
			if !playerMsgs.await(game.MsgTypeReactionOpened, 2*time.Second) {
				t.Fatal("the player never saw the reaction open")
			}
			if n := playerMsgs.count(game.MsgTypePieceMoved); n != 0 {
				t.Fatalf("the escaping player's client was told the piece moved %d time(s) at the opening", n)
			}
		})
	}
}

// The whole board rule, by every verb that closes a turn. An escape whose outcome depended on
// which verb the master happened to use would be a bug, not a rule.
func TestE2E_TheCloseDecidesWhereTheEscapingPieceEndsUp(t *testing.T) {
	type want struct {
		moves    bool
		col, row int
	}
	scenarios := []struct {
		name string
		// stage runs after the escape is open and decides it; it may choose a landing.
		stage func(t *testing.T, f *combatFixture, master *websocket.Conn, masterMsgs *collector, reactionID uuid.UUID)
		want  want
	}{
		{
			name:  "passes → the destination",
			stage: func(*testing.T, *combatFixture, *websocket.Conn, *collector, uuid.UUID) {},
			want:  want{moves: true, col: escapeTo[0], row: escapeTo[1]},
		},
		{
			name: "fails, no choice → stays",
			stage: func(t *testing.T, f *combatFixture, master *websocket.Conn, masterMsgs *collector, reactionID uuid.UUID) {
				f.setEscapeReading(t, master, masterMsgs, reactionID, false, true)
			},
			want: want{moves: false},
		},
		{
			name: "fails, the master chose a landing → the landing",
			stage: func(t *testing.T, f *combatFixture, master *websocket.Conn, masterMsgs *collector, reactionID uuid.UUID) {
				f.setEscapeReading(t, master, masterMsgs, reactionID, false, true)
				pos := escapeLanding
				f.chooseLanding(t, master, masterMsgs, reactionID, &pos)
			},
			want: want{moves: true, col: escapeLanding[0], row: escapeLanding[1]},
		},
		{
			name: "fails on the dodge alone, with a landing → the landing",
			stage: func(t *testing.T, f *combatFixture, master *websocket.Conn, masterMsgs *collector, reactionID uuid.UUID) {
				f.setEscapeReading(t, master, masterMsgs, reactionID, true, false)
				pos := escapeLanding
				f.chooseLanding(t, master, masterMsgs, reactionID, &pos)
			},
			want: want{moves: true, col: escapeLanding[0], row: escapeLanding[1]},
		},
		{
			// Review focus 4: the landing is a standing choice, not a verdict. Chosen while the
			// escape was failing; the master then changes the reading and it passes — the
			// piece goes to the DESTINATION.
			name: "landing chosen, then the escape passes → the destination",
			stage: func(t *testing.T, f *combatFixture, master *websocket.Conn, masterMsgs *collector, reactionID uuid.UUID) {
				f.setEscapeReading(t, master, masterMsgs, reactionID, false, true)
				pos := escapeLanding
				f.chooseLanding(t, master, masterMsgs, reactionID, &pos)
				f.setEscapeReading(t, master, masterMsgs, reactionID, true, true)
			},
			want: want{moves: true, col: escapeTo[0], row: escapeTo[1]},
		},
		{
			name: "landing chosen, then cleared → stays",
			stage: func(t *testing.T, f *combatFixture, master *websocket.Conn, masterMsgs *collector, reactionID uuid.UUID) {
				f.setEscapeReading(t, master, masterMsgs, reactionID, false, true)
				pos := escapeLanding
				f.chooseLanding(t, master, masterMsgs, reactionID, &pos)
				f.chooseLanding(t, master, masterMsgs, reactionID, nil)
			},
			want: want{moves: false},
		},
	}

	for _, v := range closingVerbs {
		for _, sc := range scenarios {
			t.Run(v.name+"/"+sc.name, func(t *testing.T) {
				f := newCombatFixture(t, withVictimPiece)
				f.seedBoard(t)
				master, player := f.connect(t)
				defer master.Close() //nolint:errcheck
				defer player.Close() //nolint:errcheck
				masterMsgs := collectFrom(master)
				playerMsgs := collectFrom(player)

				reactionID := f.escapeStage(t, master, player, masterMsgs, playerMsgs, false, true, true)
				sc.stage(t, f, master, masterMsgs, reactionID)
				if n := masterMsgs.count(game.MsgTypePieceMoved); n != 0 {
					t.Fatalf("the piece moved %d time(s) before the turn closed", n)
				}

				v.close(t, master)
				if !masterMsgs.await(v.barrier, 3*time.Second) {
					t.Fatalf("the turn never closed; the master received: %v",
						messageTypes(masterMsgs.snapshotMessages()))
				}

				if !sc.want.moves {
					if n := masterMsgs.count(game.MsgTypePieceMoved); n != 0 {
						t.Fatalf("the piece moved %d time(s); a failed escape with no landing stays put", n)
					}
					return
				}
				if !masterMsgs.await(game.MsgTypePieceMoved, 2*time.Second) {
					t.Fatalf("no piece_moved at the close; the master received: %v",
						messageTypes(masterMsgs.snapshotMessages()))
				}
				col, row := movedTo(t, masterMsgs)
				if col != sc.want.col || row != sc.want.row {
					t.Fatalf("the piece ended on (%d,%d), want (%d,%d)", col, row, sc.want.col, sc.want.row)
				}
				if n := masterMsgs.count(game.MsgTypePieceMoved); n != 1 {
					t.Fatalf("the close moved the piece %d time(s), want exactly once", n)
				}
			})
		}
	}
}

// While the turn is open the master sees the verdict, and a failed escape with nothing chosen
// says it is waiting for them. Choosing a landing acknowledges and recomputes, like every
// other edit.
func TestE2E_AFailedEscapeAwaitsTheMastersLanding(t *testing.T) {
	f := newCombatFixture(t, withVictimPiece)
	f.seedBoard(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	masterMsgs := collectFrom(master)
	playerMsgs := collectFrom(player)

	reactionID := f.escapeStage(t, master, player, masterMsgs, playerMsgs, false, false, true)

	open := lastResolutionUpdated(t, masterMsgs)
	if open.IsSettled {
		t.Fatal("the last resolution is settled; this test needs the open turn's")
	}
	esc := f.escapeOf(t, open)
	if esc.Escaped || esc.MovePassed || !esc.DodgePassed || !esc.AwaitsMaster || esc.Landing != nil {
		t.Fatalf("escape = %+v, want escaped=false movePassed=false dodgePassed=true "+
			"awaitsMaster=true and no landing", *esc)
	}

	pos := escapeLanding
	chosen := f.chooseLanding(t, master, masterMsgs, reactionID, &pos)
	esc = f.escapeOf(t, chosen)
	if esc.AwaitsMaster || esc.Landing == nil || *esc.Landing != escapeLanding {
		t.Fatalf("escape after the choice = %+v, want landing %v and awaitsMaster=false", *esc, escapeLanding)
	}
	// The ack names the reaction the landing is for.
	var ack game.ActionEditedPayload
	msgs := masterMsgs.snapshotMessages()
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Type == game.MsgTypeActionEdited {
			if err := json.Unmarshal(msgs[i].Payload, &ack); err != nil {
				t.Fatalf("unmarshal action_edited: %v", err)
			}
			break
		}
	}
	if ack.ActionID != reactionID {
		t.Fatalf("action_edited names %s, want the escape %s", ack.ActionID, reactionID)
	}
	// The resolution of an open turn is the master's alone, landing included.
	if n := playerMsgs.count(game.MsgTypeResolutionUpdate); n != 0 {
		t.Fatalf("the player received %d resolution_updated while the turn was open", n)
	}
}

// Once the turn closes the escape's numbers are public: the settled resolution reaches the
// table with it.
func TestE2E_TheSettledResolutionCarriesTheEscapeToEveryone(t *testing.T) {
	f := newCombatFixture(t, withVictimPiece)
	f.seedBoard(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	masterMsgs := collectFrom(master)
	playerMsgs := collectFrom(player)

	reactionID := f.escapeStage(t, master, player, masterMsgs, playerMsgs, false, false, true)
	pos := escapeLanding
	f.chooseLanding(t, master, masterMsgs, reactionID, &pos)

	sendWS(t, master, string(game.MsgTypeCloseTurn), game.CloseTurnPayload{Confirm: true})
	for name, c := range map[string]*collector{"master": masterMsgs, "player": playerMsgs} {
		if !awaitSettled(c, 3*time.Second) {
			t.Fatalf("the %s never received the settled resolution; they received: %v",
				name, messageTypes(c.snapshotMessages()))
		}
		settled := findSettledResolution(t, c)
		esc := f.escapeOf(t, settled)
		if esc.Escaped || esc.MovePassed || !esc.DodgePassed || esc.AwaitsMaster ||
			esc.Landing == nil || *esc.Landing != escapeLanding {
			t.Fatalf("%s's settled escape = %+v, want the failed escape with its landing %v",
				name, *esc, escapeLanding)
		}
	}
}

// escapeLanding only means something on an escape of the open turn, on a cell the board has.
// Anything else is the master pressing a control describing something that is not there —
// refused, never a silent no-op.
func TestE2E_EscapeLandingIsRefusedWhereItMeansNothing(t *testing.T) {
	f := newCombatFixture(t, withVictimPiece)
	f.seedBoard(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	masterMsgs := collectFrom(master)
	playerMsgs := collectFrom(player)

	reactionID := f.escapeStage(t, master, player, masterMsgs, playerMsgs, false, false, true)

	cases := []struct {
		name     string
		payload  game.EditActionPayload
		contains string
	}{
		{
			name: "the turn's own action",
			payload: game.EditActionPayload{
				EscapeLanding: &game.EscapeLandingPayload{Position: &[3]int{7, 6, 0}},
			},
			contains: "escape reaction",
		},
		{
			name: "a cell off the board",
			payload: game.EditActionPayload{
				ActionID:      reactionID,
				EscapeLanding: &game.EscapeLandingPayload{Position: &[3]int{99, 6, 0}},
			},
			contains: "outside the grid",
		},
		{
			name: "something that is not on the turn",
			payload: game.EditActionPayload{
				ActionID:      uuid.New(),
				EscapeLanding: &game.EscapeLandingPayload{Position: &[3]int{7, 6, 0}},
			},
			contains: "not on the open turn",
		},
	}
	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sendWS(t, master, string(game.MsgTypeEditAction), c.payload)
			if !awaitCount(masterMsgs, game.MsgTypeError, i+1, 2*time.Second) {
				t.Fatalf("the edit was not refused; the master received: %v",
					messageTypes(masterMsgs.snapshotMessages()))
			}
			msgs := masterMsgs.snapshotMessages()
			var last game.ErrorPayload
			for j := len(msgs) - 1; j >= 0; j-- {
				if msgs[j].Type == game.MsgTypeError {
					if err := json.Unmarshal(msgs[j].Payload, &last); err != nil {
						t.Fatalf("unmarshal error: %v", err)
					}
					break
				}
			}
			if last.Code != "game_error" || !strings.Contains(last.Message, c.contains) {
				t.Fatalf("error = %+v, want game_error mentioning %q", last, c.contains)
			}
		})
	}
	if n := masterMsgs.count(game.MsgTypeActionEdited); n != 2 {
		// Two: escapeStage's own reading edits. A refused landing is never acknowledged.
		t.Fatalf("action_edited count = %d, want only the stage's reading edit", n)
	}
}

// The same refusal on a REACTION that does not displace: a plain dodge has no piece to land.
func TestE2E_EscapeLandingOnADodgeIsRefused(t *testing.T) {
	f := newCombatFixture(t, withVictimPiece)
	f.seedBoard(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	masterMsgs := collectFrom(master)
	playerMsgs := collectFrom(player)
	if !playerMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the player never got the board")
	}

	actionID := f.openAttackOn(t, player, master, masterMsgs)
	before := masterMsgs.count(game.MsgTypeResolutionUpdate)
	sendWS(t, player, string(game.MsgTypeAttachReaction), game.ActionPayload{
		ActorID:      f.victimID,
		ReactToID:    actionID,
		ReactionKind: "dodge",
		Dodge:        &game.DodgePayload{RollCheck: &game.RollCheckPayload{SkillName: enum.Reflex.String()}},
	})
	if !awaitCount(masterMsgs, game.MsgTypeResolutionUpdate, before+1, 2*time.Second) {
		t.Fatal("the dodge never attached")
	}
	pending := lastResolutionUpdated(t, masterMsgs).PendingReactions
	if len(pending) != 1 {
		t.Fatalf("pendingReactions = %+v, want the one dodge", pending)
	}

	sendWS(t, master, string(game.MsgTypeEditAction), game.EditActionPayload{
		ActionID:      pending[0].ReactionID,
		EscapeLanding: &game.EscapeLandingPayload{Position: &[3]int{7, 6, 0}},
	})
	if !masterMsgs.await(game.MsgTypeError, 2*time.Second) {
		t.Fatalf("escapeLanding on a dodge was not refused; the master received: %v",
			messageTypes(masterMsgs.snapshotMessages()))
	}
	var e game.ErrorPayload
	if err := json.Unmarshal(findMessage(t, masterMsgs.snapshotMessages(), game.MsgTypeError).Payload, &e); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if e.Code != "game_error" || !strings.Contains(e.Message, "escape reaction") {
		t.Fatalf("error = %+v, want game_error naming the escape rule", e)
	}
	if n := masterMsgs.count(game.MsgTypeActionEdited); n != 0 {
		t.Fatalf("a refused landing was acknowledged %d time(s)", n)
	}
}

// A Shift escape that passes moves at the close, and exactly once — it no longer steps at the
// opening, so there is no earlier piece_moved for the close to repeat.
func TestE2E_AClosedEscapeMovesOnceAtTheClose(t *testing.T) {
	f := newCombatFixture(t, withVictimPiece)
	f.seedBoard(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	masterMsgs := collectFrom(master)
	playerMsgs := collectFrom(player)

	f.escapeStage(t, master, player, masterMsgs, playerMsgs, true, true, true)

	sendWS(t, master, string(game.MsgTypeCloseTurn), game.CloseTurnPayload{Confirm: true})
	if !masterMsgs.await(game.MsgTypeTurnClosed, 3*time.Second) {
		t.Fatal("the turn never closed")
	}
	if !masterMsgs.await(game.MsgTypePieceMoved, 2*time.Second) {
		t.Fatal("the Shift escape that passed never moved at the close")
	}
	if col, row := movedTo(t, masterMsgs); col != escapeTo[0] || row != escapeTo[1] {
		t.Fatalf("the piece ended on (%d,%d), want (%d,%d)", col, row, escapeTo[0], escapeTo[1])
	}
	if n := masterMsgs.count(game.MsgTypePieceMoved); n != 1 {
		t.Fatalf("%d piece_moved in total, want exactly the close's one", n)
	}
	if !playerMsgs.await(game.MsgTypeTurnClosed, 2*time.Second) {
		t.Fatal("the player never saw the turn close")
	}
	if n := playerMsgs.count(game.MsgTypePieceMoved); n != 1 {
		t.Fatalf("the escaping player's client saw the piece move %d time(s), want 1", n)
	}
}

// Both roads out of the close — the destination and the master's landing — put the piece on
// the board BEFORE turn_closed: the table must never watch the turn end with the piece still
// in its old slot. Both envelopes are stamped when they are BUILT, by the same goroutine
// running the close_turn arm, so the stamps compare the order the SERVER decided; strict,
// because a tie is not evidence of the right order.
func TestE2E_TheEscapingPieceMovesBeforeTurnClosed(t *testing.T) {
	for _, c := range []struct {
		name       string
		movePasses bool
	}{
		{"the destination", true},
		{"the master's landing", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newCombatFixture(t, withVictimPiece)
			f.seedBoard(t)
			master, player := f.connect(t)
			defer master.Close() //nolint:errcheck
			defer player.Close() //nolint:errcheck
			masterMsgs := collectFrom(master)
			playerMsgs := collectFrom(player)

			reactionID := f.escapeStage(t, master, player, masterMsgs, playerMsgs, false, c.movePasses, true)
			if !c.movePasses {
				pos := escapeLanding
				f.chooseLanding(t, master, masterMsgs, reactionID, &pos)
			}

			sendWS(t, master, string(game.MsgTypeCloseTurn), game.CloseTurnPayload{Confirm: true})
			if !masterMsgs.await(game.MsgTypeTurnClosed, 3*time.Second) {
				t.Fatal("the turn never closed")
			}
			if !masterMsgs.await(game.MsgTypePieceMoved, 2*time.Second) {
				t.Fatal("nothing moved at the close, so this test measured no ordering at all")
			}
			msgs := masterMsgs.snapshotMessages()
			moved := findMessage(t, msgs, game.MsgTypePieceMoved)
			closed := findMessage(t, msgs, game.MsgTypeTurnClosed)
			if !moved.Timestamp.Before(closed.Timestamp) {
				t.Fatalf("piece_moved was built at %s, NOT strictly before turn_closed at %s",
					moved.Timestamp, closed.Timestamp)
			}
			if movedAt, closedAt := indexOfMessage(msgs, game.MsgTypePieceMoved),
				indexOfMessage(msgs, game.MsgTypeTurnClosed); movedAt > closedAt {
				t.Fatalf("turn_closed (index %d) arrived before piece_moved (index %d)", closedAt, movedAt)
			}
		})
	}
}

// The same line-of-sight gate as every other displacement: whoever sees neither end of the
// escape is not told about it — now at the close, where the escape moves.
func TestE2E_AnEscapeAtTheCloseDoesNotLeakToWhoCannotSeeIt(t *testing.T) {
	f := newCombatFixture(t, withVictimPiece, withBystander)
	f.seedBoard(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	blind := connectWS(t, f.server.URL, f.bystanderUUID, f.matchUUID)
	defer blind.Close()   //nolint:errcheck
	readMessage(t, blind) // room_state
	masterMsgs := collectFrom(master)
	playerMsgs := collectFrom(player)
	blindMsgs := collectFrom(blind)

	if !blindMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the bystander never got a board — they are not really at the table")
	}
	// The premise: the divider wall hides the escaping target from the bystander. Without
	// it the negative assertion below would be a tautology.
	var board game.MapFullStatePayload
	if err := json.Unmarshal(
		findMessage(t, blindMsgs.snapshotMessages(), game.MsgTypeMapFullState).Payload, &board,
	); err != nil {
		t.Fatalf("unmarshal map_full_state: %v", err)
	}
	sees := map[string]bool{}
	for _, p := range board.Pieces {
		sees[p.PieceID] = true
	}
	if !sees[bystanderPieceID] || sees[victimPieceID] {
		t.Fatalf("the bystander's view is not the premise (sees own=%v, sees victim=%v)",
			sees[bystanderPieceID], sees[victimPieceID])
	}

	f.escapeStage(t, master, player, masterMsgs, playerMsgs, true, true, true)
	sendWS(t, master, string(game.MsgTypeCloseTurn), game.CloseTurnPayload{Confirm: true})

	if !masterMsgs.await(game.MsgTypePieceMoved, 3*time.Second) {
		t.Fatal("no piece_moved reached the master: nothing moved, so the gate was never exercised")
	}
	// turn_closed is the barrier: it rides the same per-client lane, after the escape's move.
	if !blindMsgs.await(game.MsgTypeTurnClosed, 2*time.Second) {
		t.Fatal("the bystander received nothing at all — this connection is dead, not gated")
	}
	if n := blindMsgs.count(game.MsgTypePieceMoved); n != 0 {
		t.Fatalf("a player who sees neither end of the escape was told about it %d time(s)", n)
	}
	if n := blindMsgs.count(game.MsgTypePieceRemoved); n != 0 {
		t.Fatalf("the bystander was told the piece left a slot they never saw it in (%d time(s))", n)
	}
}

// An actor with no piece on the board is NOT an error — the escape happens on the sheet, and
// there is nothing to move. The same silent no-op as the action path, at the close.
func TestE2E_AnEscapeForAnActorWithNoPieceIsSilent(t *testing.T) {
	f := newCombatFixture(t) // no withVictimPiece: the target has no piece on the board
	f.seedBoard(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	masterMsgs := collectFrom(master)
	playerMsgs := collectFrom(player)

	f.escapeStage(t, master, player, masterMsgs, playerMsgs, true, true, true)
	sendWS(t, master, string(game.MsgTypeCloseTurn), game.CloseTurnPayload{Confirm: true})
	if !masterMsgs.await(game.MsgTypeTurnClosed, 3*time.Second) {
		t.Fatal("the turn never closed: a missing piece must not block it")
	}
	if n := masterMsgs.count(game.MsgTypeError); n != 0 {
		t.Fatalf("an actor with no piece produced %d error message(s); it is a no-op, not a fault: %v",
			n, messageTypes(masterMsgs.snapshotMessages()))
	}
	if n := masterMsgs.count(game.MsgTypePieceMoved); n != 0 {
		t.Fatalf("something moved (%d piece_moved) although the escaping actor has no piece", n)
	}
}

// awaitSettled waits for a settled resolution_updated to have arrived.
func awaitSettled(c *collector, d time.Duration) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		for _, m := range c.snapshotMessages() {
			if m.Type != game.MsgTypeResolutionUpdate {
				continue
			}
			var p game.ResolutionUpdatedPayload
			if json.Unmarshal(m.Payload, &p) == nil && p.IsSettled {
				return true
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// findSettledResolution reads the last settled resolution_updated a collector received. The
// settled one is the turn's real outcome — the projections that preceded it were dry runs.
func findSettledResolution(t *testing.T, c *collector) game.ResolutionUpdatedPayload {
	t.Helper()
	msgs := c.snapshotMessages()
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Type != game.MsgTypeResolutionUpdate {
			continue
		}
		var p game.ResolutionUpdatedPayload
		if err := json.Unmarshal(msgs[i].Payload, &p); err != nil {
			t.Fatalf("unmarshal resolution_updated: %v", err)
		}
		if p.IsSettled {
			return p
		}
	}
	t.Fatalf("no settled resolution_updated was received; got: %v", messageTypes(msgs))
	return game.ResolutionUpdatedPayload{}
}
