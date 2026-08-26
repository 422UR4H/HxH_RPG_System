package game_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// This file is Phase 5's headline done-criterion, over a real WebSocket: the SAME settled
// turn resolution, read by two different players, comes back as two DIFFERENT payloads — the
// owner of the closed dodge sees closedDodge, everyone else sees the plain dodge it must be
// indistinguishable from (combat-engine.md § Visibilidade).
//
// The fixture is reaction_chain_e2e_test.go's areaFixture, trimmed to the three-character,
// three-player shape this scenario needs: an attacker, a defender who answers with a closed
// dodge, and a bystander who is neither actor nor target — present only to prove the
// resolution reaches someone with no stake in the exchange at all.

// ─── fixture ────────────────────────────────────────────────────────────────

type visibilityFixture struct {
	server                                          *httptest.Server
	session                                         *matchsession.MatchSession
	source                                          *scriptedFaces
	matchUUID                                       uuid.UUID
	masterUUID                                      uuid.UUID
	attackerPlayer, defenderPlayer, bystanderPlayer uuid.UUID
	attackerID, defenderID, bystanderID             uuid.UUID

	// attackerConn is stood up by connect() but not among its return values — this scenario
	// never inspects the attacker's own view, only that the attack lands and gets answered.
	attackerConn *websocket.Conn

	// masterConn/masterMsgs are the SINGLE reader this fixture ever installs on the master's
	// socket. gorilla's Conn allows exactly one concurrent reader — attachClosedDodge needs to
	// observe master-only confirmations (attach and open both message the master unconditionally;
	// see room.go's handleReaction and the MsgTypeOpenReaction case), so every helper that needs
	// master's traffic shares this one collector instead of each starting its own.
	masterConn *websocket.Conn
	masterMsgs *collector
}

// visibilityFaces is this scenario's exact dice budget, face by face — computed from
// rollActionDice's real consumption order (matchsession/match_session.go), not guessed from
// the reaction table alone:
//
//	Attack.Hit (2D10 test)                  → primary 2 + secondary 2 = 4
//	Sword damage (D10+D4, single set)       → 2
//	closedDodge's Evasion entry (2D10 test) → primary 2 + secondary 2 = 4 (Skills are rolled
//	                                           BEFORE Dodge — rollActionDice walks a.Skills,
//	                                           then a.Dodge)
//	closedDodge's Dodge (2D10 test)         → primary 2 + secondary 2 = 4
//
// 14 faces total. See docs/dev/match/combat-engine.md § Quantos dados cada reação consome:
// closedDodge alone is 8 faces, not 4 — Evasion is a second full 2D10 test, not a modifier on
// the first. open_reaction rolls nothing new (matchsession.OpenReaction only flips a flag and
// re-derives), so the budget does not grow between attach and open.
func visibilityFaces() []int {
	return []int{
		6, 4, 1, 1, // Attack.Hit: primary 6,4 (→10 total); secondary 1,1 unused filler
		7, 3, // Sword damage: D10=7, D4=3, raw 7+3+2=12
		5, 5, 1, 1, // Evasion: primary 5,5; secondary 1,1 unused filler
		9, 2, 1, 1, // Dodge: primary 9,2; secondary 1,1 unused filler
	}
}

// newVisibilityFixture stands the server and seeds the session with three participants, each
// owned by a distinct player, and installs the scripted source sized for exactly one closed
// dodge against one attack.
func newVisibilityFixture(t *testing.T) *visibilityFixture {
	t.Helper()

	matchUUID := uuid.New()
	masterUUID := uuid.New()
	attackerPlayer := uuid.New()
	defenderPlayer := uuid.New()
	bystanderPlayer := uuid.New()

	attackerID := uuid.New()
	defenderID := uuid.New()
	bystanderID := uuid.New()

	sheets := map[uuid.UUID]*csSheet.CharacterSheet{
		attackerID:  newCombatSheet(t),
		defenderID:  newCombatSheet(t),
		bystanderID: newCombatSheet(t),
	}
	participants := []*match.Participant{
		{UUID: uuid.New(), MatchUUID: matchUUID, Sheet: csEntity.Summary{UUID: attackerID, PlayerUUID: &attackerPlayer}},
		{UUID: uuid.New(), MatchUUID: matchUUID, Sheet: csEntity.Summary{UUID: defenderID, PlayerUUID: &defenderPlayer}},
		{UUID: uuid.New(), MatchUUID: matchUUID, Sheet: csEntity.Summary{UUID: bystanderID, PlayerUUID: &bystanderPlayer}},
	}

	session := matchsession.NewMatchSession(matchUUID, sheets, participants)
	source := &scriptedFaces{faces: visibilityFaces()}
	session.SetRollSource(source)

	hub := game.NewHub()
	go hub.Run()
	t.Cleanup(hub.Stop)

	writer := &recordingStatusWriter{}
	roundRepo := &mockRoundRepoHandler{}
	handler := game.NewHandler(
		hub,
		&fogMatchRepo{masterUUID: masterUUID, started: true},
		&mockEnrollmentChecker{enrolled: true},
		&mockStartMatchUC{},
		&mockKickPlayerUC{},
		&combatSessionUC{session: session},
		appmatch.NewOpenNextActionUC(writer, appmatch.NewCloseRoundUC(roundRepo)),
		appmatch.NewPullActionUC(writer, appmatch.NewCloseRoundUC(roundRepo)),
		appmatch.NewEnqueueActionUC(),
		appmatch.NewAttachReactionUC(),
		appmatch.NewOpenReactionUC(),
		appmatch.NewCloseTurnUC(writer),
		&mockChangeSceneUCHandler{},
		roundRepo,
		&mockEnqueueMasterActionUCHandler{},
		appmatch.NewChangeRoundModeUC(),
		appmatch.NewEditActionUC(),
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handler.HandleWebSocket)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return &visibilityFixture{
		server:          server,
		session:         session,
		source:          source,
		matchUUID:       matchUUID,
		masterUUID:      masterUUID,
		attackerPlayer:  attackerPlayer,
		defenderPlayer:  defenderPlayer,
		bystanderPlayer: bystanderPlayer,
		attackerID:      attackerID,
		defenderID:      defenderID,
		bystanderID:     bystanderID,
	}
}

// connect dials the master first — same reason as combatFixture.connect: the room refuses a
// player with lobby_not_open until the master has opened it, and it is what rehydrates the
// already-started session for everyone who joins after. The attacker's connection is kept on
// the fixture rather than returned; this scenario never reads the attacker's own traffic.
//
// The master's collector is installed here, once, and kept on the fixture — see masterMsgs'
// own comment for why every later helper must share it rather than starting a second reader.
func (f *visibilityFixture) connect(t *testing.T) (master, defenderConn, bystanderConn *websocket.Conn) {
	t.Helper()
	master = connectWS(t, f.server.URL, f.masterUUID, f.matchUUID)
	readMessage(t, master) // room_state
	f.masterConn = master
	f.masterMsgs = newCollector(master)

	f.attackerConn = connectWS(t, f.server.URL, f.attackerPlayer, f.matchUUID)
	readMessage(t, f.attackerConn) // room_state

	defenderConn = connectWS(t, f.server.URL, f.defenderPlayer, f.matchUUID)
	readMessage(t, defenderConn) // room_state

	bystanderConn = connectWS(t, f.server.URL, f.bystanderPlayer, f.matchUUID)
	readMessage(t, bystanderConn) // room_state

	return master, defenderConn, bystanderConn
}

// enqueueAttack sends the attacker's opening move — a Sword attack against the defender only.
// The bystander is never targeted: whatever they learn about this turn, they learn purely by
// being at the table when it settles, not by being hit.
func (f *visibilityFixture) enqueueAttack(t *testing.T, conn *websocket.Conn) {
	t.Helper()
	sword := "Sword"
	sendWS(t, conn, "enqueue_action", game.ActionPayload{
		ActorID:  f.attackerID,
		TargetID: []uuid.UUID{f.defenderID},
		Attack: &game.AttackPayload{
			Weapon: &sword,
			Hit:    game.RollCheckPayload{SkillName: "Accuracy"},
			Damage: game.RollCheckPayload{},
		},
	})
	if !newCollector(conn).await(game.MsgTypeActionEnqueued, 2*time.Second) {
		t.Fatal("the attack was never acknowledged as enqueued")
	}
}

// attachClosedDodge answers the open turn's attack with a closed dodge — the Evasion skill
// folded into the Reflex dodge, indistinguishable from a plain dodge to anyone but its owner
// and the master (RequiresEvasionSkill — see entity/action/reaction_kind.go) — and then has
// the master open it.
//
// Both steps are required for the reaction to reach the CLOSED resolution at all:
// buildChainOrder (service/attack_chain.go) only turns OPENED reactions into a chain step;
// left merely attached, the target gets the passive default instead and CharacterResult never
// carries a Reaction. attach_reaction alone is not enough, confirmed by running this fixture
// without the open step: the defender's dodgeTotal read back as 11 (the passive average) with
// no reaction field at all, not the scripted closed dodge.
//
// The action ID is read straight off the fixture's own session, the same shortcut
// reaction_chain_e2e_test.go's awaitTurnOpened uses and justifies: no wire field carries an
// action's own ID, and this read happens in the same quiet window that comment describes —
// after turn_opened has already been confirmed and before anything else is in flight.
//
// Both attach and open are confirmed via the MASTER's socket, never the reactor's own
// connection: handleReaction and the open_reaction handler (room.go) only ever message the
// master. A refused attach (e.g. a malformed closed dodge) surfaces here as a loud, immediate
// timeout instead of silently letting the turn close on the passive default — this is
// deliberately NOT a fixed sleep: the earlier version of this helper slept and moved on, which
// is exactly how the missing-open bug above went unnoticed.
func (f *visibilityFixture) attachClosedDodge(t *testing.T, conn *websocket.Conn) {
	t.Helper()
	turn := f.session.GetActiveRound().CurrentTurn()
	if turn == nil {
		t.Fatal("no current turn on the session to attach the closed dodge to")
	}
	act := turn.GetAction()
	actionID := act.GetID()

	before := f.masterMsgs.count(game.MsgTypeResolutionUpdate)
	sendWS(t, conn, "attach_reaction", game.ActionPayload{
		ActorID: f.defenderID, ReactToID: actionID, ReactionKind: "closedDodge",
		Dodge:  &game.DodgePayload{RollCheck: &game.RollCheckPayload{SkillName: enum.Reflex.String()}},
		Skills: []game.ActionSkillPayload{{SkillName: enum.Evasion.String()}},
	})
	if !awaitCount(f.masterMsgs, game.MsgTypeResolutionUpdate, before+1, 2*time.Second) {
		t.Fatal("attach_reaction never produced a fresh resolution_updated for the master — " +
			"the closed dodge was likely refused at the boundary (missing Dodge or Evasion entry)")
	}

	// The reaction's own ID lives only in the master's PendingReactions — see
	// ResolutionUpdatedPayload.PendingReactions' own comment for why nothing else names it.
	reactionID := uuid.Nil
	msgs := f.masterMsgs.snapshotMessages()
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Type != game.MsgTypeResolutionUpdate {
			continue
		}
		var resolved game.ResolutionUpdatedPayload
		if err := json.Unmarshal(msgs[i].Payload, &resolved); err != nil {
			t.Fatalf("unmarshal resolution_updated: %v", err)
		}
		for _, pr := range resolved.PendingReactions {
			if pr.ActorID == f.defenderID {
				reactionID = pr.ReactionID
			}
		}
		break
	}
	if reactionID == uuid.Nil {
		t.Fatal("the closed dodge never showed up in the master's PendingReactions")
	}

	beforeOpen := f.masterMsgs.count(game.MsgTypeResolutionUpdate)
	sendWS(t, f.masterConn, "open_reaction", game.OpenReactionPayload{ReactionID: reactionID})
	if !awaitCount(f.masterMsgs, game.MsgTypeResolutionUpdate, beforeOpen+1, 2*time.Second) {
		t.Fatal("open_reaction never produced a fresh resolution_updated for the master")
	}
}

// kindOfFirstTarget reads the reaction kind off the first settled resolution_updated in c that
// actually carries one — the owner and the bystander each get exactly one qualifying message
// in these tests, but scanning rather than indexing survives a stray earlier message.
func kindOfFirstTarget(t *testing.T, c *collector) string {
	t.Helper()
	for _, m := range c.snapshotMessages() {
		if m.Type != game.MsgTypeResolutionUpdate {
			continue
		}
		var p game.ResolutionUpdatedPayload
		if err := json.Unmarshal(m.Payload, &p); err != nil {
			t.Fatalf("unmarshal resolution_updated: %v", err)
		}
		if !p.IsSettled || len(p.Targets) == 0 || p.Targets[0].Reaction == nil {
			continue
		}
		return p.Targets[0].Reaction.Kind
	}
	t.Fatal("no settled resolution with a reaction arrived")
	return ""
}

// ─── the tests ──────────────────────────────────────────────────────────────

// TestE2E_AClosedDodgeReachesAThirdPartyAsAPlainDodge is the phase's headline criterion:
// two clients connected as different players receive DIFFERENT payloads for the SAME turn.
//
// The dice are scripted. Budget four faces per 2D10 test — RollCalculator.Roll always rolls
// Primary AND Secondary — plus two for the sword's damage set.
func TestE2E_AClosedDodgeReachesAThirdPartyAsAPlainDodge(t *testing.T) {
	f := newVisibilityFixture(t)
	master, defenderConn, bystanderConn := f.connect(t)
	defenderMsgs := newCollector(defenderConn)
	bystanderMsgs := newCollector(bystanderConn)

	f.enqueueAttack(t, f.attackerConn)
	sendWS(t, master, "open_next_action", struct{}{})
	if !f.masterMsgs.await(game.MsgTypeTurnOpened, 2*time.Second) {
		t.Fatal("the turn never opened")
	}

	f.attachClosedDodge(t, defenderConn)
	sendWS(t, master, "close_turn", map[string]any{"confirm": true})

	if !defenderMsgs.await(game.MsgTypeResolutionUpdate, 2*time.Second) {
		t.Fatal("the defender never received the settled resolution")
	}
	if !bystanderMsgs.await(game.MsgTypeResolutionUpdate, 2*time.Second) {
		t.Fatal("the bystander never received the settled resolution")
	}
	// A refused or malformed reaction answers with an error straight to its own sender — if the
	// closed dodge had been rejected, this is where it would show up instead of failing silently.
	if n := defenderMsgs.count(game.MsgTypeError); n != 0 {
		t.Fatalf("the defender's own connection received %d error message(s); the closed dodge was not accepted cleanly", n)
	}

	defKind := kindOfFirstTarget(t, defenderMsgs)
	bysKind := kindOfFirstTarget(t, bystanderMsgs)

	if defKind != "closedDodge" {
		t.Fatalf("the owner sees %q, want closedDodge — they were lied to about their own reaction", defKind)
	}
	if bysKind != "dodge" {
		t.Fatalf("the bystander sees %q, want dodge — the label is the leak", bysKind)
	}
	if defKind == bysKind {
		t.Fatal("both clients received the same payload; nothing was projected")
	}
}

// TestE2E_TheCalculationIsTheMastersUntilTheTurnCloses guards the time axis: while the turn
// is open, no player receives a resolution at all.
func TestE2E_TheCalculationIsTheMastersUntilTheTurnCloses(t *testing.T) {
	f := newVisibilityFixture(t)
	master, defenderConn, _ := f.connect(t)
	defenderMsgs := newCollector(defenderConn)

	f.enqueueAttack(t, f.attackerConn)
	sendWS(t, master, "open_next_action", struct{}{})

	if !f.masterMsgs.await(game.MsgTypeResolutionUpdate, 2*time.Second) {
		t.Fatal("the master did not receive the open turn's projection")
	}
	// Deliberately not a timeout-based negative: the collector is drained in the background,
	// so a blocking read is never needed and the connection stays usable.
	time.Sleep(300 * time.Millisecond)
	if n := defenderMsgs.count(game.MsgTypeResolutionUpdate); n != 0 {
		t.Fatalf("a player received %d resolution_updated while the turn was still open", n)
	}
}
