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
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// This file is Phase 4's done-criterion, over a real WebSocket: one attack, three targets,
// three different answers, and a master who opens those answers in two different orders —
// producing two different outcomes for the same scripted dice. That difference IS the phase.
//
// It reuses combat_e2e_test.go's shape (combatSessionUC, recordingStatusWriter, scriptedFaces,
// newCombatSheet, sendWS, connectWS, readMessage, collector, awaitCount) and extends it to four
// characters, each driven by its own player, so AttachReaction's ownership check is exercised
// for real: only a target may react, and only through a character its own player owns.

// ─── wsConn ─────────────────────────────────────────────────────────────────

// wsConn pairs one WebSocket connection with a background collector, so the fixture can send
// on it and, independently, ask what it has received so far.
type wsConn struct {
	conn *websocket.Conn
	msgs *collector
}

// dialAreaConn connects, drains the initial room_state, and starts collecting.
func dialAreaConn(t *testing.T, serverURL string, userUUID, matchUUID uuid.UUID) *wsConn {
	t.Helper()
	conn := connectWS(t, serverURL, userUUID, matchUUID)
	readMessage(t, conn) // room_state — confirms the client is registered in the room
	return &wsConn{conn: conn, msgs: newCollector(conn)}
}

// send marshals a client message and writes it, failing the test on any error.
func (c *wsConn) send(t *testing.T, mt game.MessageType, payload any) {
	t.Helper()
	sendWS(t, c.conn, string(mt), payload)
}

// sawAny reports whether this connection ever received a message of that type.
func (c *wsConn) sawAny(mt game.MessageType) bool {
	return c.msgs.count(mt) > 0
}

// ─── fixture ────────────────────────────────────────────────────────────────

// areaFixture is combatFixture with three targets instead of one. Each character belongs to
// a different player, so the ownership check in AttachReaction is genuinely crossed.
type areaFixture struct {
	server     *httptest.Server
	session    *matchsession.MatchSession
	source     *scriptedFaces
	master     *wsConn
	attacker   *wsConn // and the three below, one connection per player
	pA, pB, pC *wsConn
	attackerID uuid.UUID // sheet UUIDs
	a, b, c    uuid.UUID
	sheets     map[uuid.UUID]*csSheet.CharacterSheet
}

// newAreaFixture stands the server, seeds the session with four participants (one attacker,
// three targets, four distinct players) and installs the scripted source.
func newAreaFixture(t *testing.T, faces []int) *areaFixture {
	t.Helper()

	matchUUID := uuid.New()
	masterUUID := uuid.New()
	attackerPlayer := uuid.New()
	playerA := uuid.New()
	playerB := uuid.New()
	playerC := uuid.New()

	attackerID := uuid.New()
	aID := uuid.New()
	bID := uuid.New()
	cID := uuid.New()

	sheets := map[uuid.UUID]*csSheet.CharacterSheet{
		attackerID: newCombatSheet(t),
		aID:        newCombatSheet(t),
		bID:        newCombatSheet(t),
		cID:        newCombatSheet(t),
	}
	participants := []*match.Participant{
		{UUID: uuid.New(), MatchUUID: matchUUID, Sheet: csEntity.Summary{UUID: attackerID, PlayerUUID: &attackerPlayer}},
		{UUID: uuid.New(), MatchUUID: matchUUID, Sheet: csEntity.Summary{UUID: aID, PlayerUUID: &playerA}},
		{UUID: uuid.New(), MatchUUID: matchUUID, Sheet: csEntity.Summary{UUID: bID, PlayerUUID: &playerB}},
		{UUID: uuid.New(), MatchUUID: matchUUID, Sheet: csEntity.Summary{UUID: cID, PlayerUUID: &playerC}},
	}

	session := matchsession.NewMatchSession(matchUUID, sheets, participants)
	source := &scriptedFaces{faces: append([]int(nil), faces...)}
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
		&mockChangeSceneUCHandler{},
		roundRepo,
		&mockEnqueueMasterActionUCHandler{},
		appmatch.NewChangeRoundModeUC(),
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handler.HandleWebSocket)
	server := httptest.NewServer(mux)

	f := &areaFixture{
		server:     server,
		session:    session,
		source:     source,
		attackerID: attackerID,
		a:          aID,
		b:          bID,
		c:          cID,
		sheets:     sheets,
	}

	// The master connects first — it is what rehydrates/serves the already-started session for
	// everyone who joins after.
	f.master = dialAreaConn(t, server.URL, masterUUID, matchUUID)
	f.attacker = dialAreaConn(t, server.URL, attackerPlayer, matchUUID)
	f.pA = dialAreaConn(t, server.URL, playerA, matchUUID)
	f.pB = dialAreaConn(t, server.URL, playerB, matchUUID)
	f.pC = dialAreaConn(t, server.URL, playerC, matchUUID)

	return f
}

// awaitTurnOpened blocks until turn_opened arrives and returns the turn ID and the action ID
// the reactions must point at.
//
// The wire has no field carrying the action's own ID — RoundOrchestrator.AttachReaction checks
// reactToId against the ACTION's ID, which is a different UUID from the turn's on purpose (see
// turn.NewTurn / action.NewAction). No message a real player receives ever names it. Reading it
// straight off the fixture's own session is the same move combatFixture.victimHP makes to read
// live sheet state: safe in this exact quiet window, with nothing else in flight yet.
func (f *areaFixture) awaitTurnOpened(t *testing.T) (turnID, actionID uuid.UUID) {
	t.Helper()
	if !f.master.msgs.await(game.MsgTypeTurnOpened, 2*time.Second) {
		t.Fatal("turn_opened never arrived")
	}
	opened := lastTurnOpened(t, f.master.msgs)
	turn := f.session.GetActiveRound().CurrentTurn()
	if turn == nil {
		t.Fatal("no current turn on the session right after turn_opened")
	}
	act := turn.GetAction()
	return opened.TurnID, act.GetID()
}

// attachReaction sends one reaction and returns the master's resulting projection, which is
// how the test learns the reaction's own ID for the later open_reaction.
func (f *areaFixture) attachReaction(t *testing.T, c *wsConn, p game.ActionPayload) uuid.UUID {
	t.Helper()
	before := f.master.msgs.count(game.MsgTypeResolutionUpdate)
	c.send(t, game.MsgTypeAttachReaction, p)
	if !awaitCount(f.master.msgs, game.MsgTypeResolutionUpdate, before+1, 2*time.Second) {
		t.Fatalf("attach_reaction for actor %v: master never got a fresh resolution_updated", p.ActorID)
	}
	resolved := lastResolutionUpdated(t, f.master.msgs)
	for _, tgt := range resolved.Targets {
		if tgt.TargetID == p.ActorID && tgt.Reaction != nil {
			return tgt.Reaction.ReactionID
		}
	}
	t.Fatalf("attach_reaction for actor %v: no reaction entry in the master's resolution_updated", p.ActorID)
	return uuid.Nil
}

// openReaction sends open_reaction and waits for the fresh resolution_updated it produces.
func (f *areaFixture) openReaction(t *testing.T, reactionID uuid.UUID) {
	t.Helper()
	before := f.master.msgs.count(game.MsgTypeResolutionUpdate)
	f.master.send(t, game.MsgTypeOpenReaction, game.OpenReactionPayload{ReactionID: reactionID})
	if !awaitCount(f.master.msgs, game.MsgTypeResolutionUpdate, before+1, 2*time.Second) {
		t.Fatalf("open_reaction %v: master never got a fresh resolution_updated", reactionID)
	}
}

// projectedDamage reads the master's most recent resolution_updated for turnID and returns
// the projected damage per target.
func (f *areaFixture) projectedDamage(t *testing.T, turnID uuid.UUID) map[uuid.UUID]int {
	t.Helper()
	msgs := f.master.msgs.snapshotMessages()
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Type != game.MsgTypeResolutionUpdate {
			continue
		}
		var p game.ResolutionUpdatedPayload
		if err := json.Unmarshal(msgs[i].Payload, &p); err != nil {
			t.Fatalf("unmarshal resolution_updated: %v", err)
		}
		if p.TurnID != turnID {
			continue
		}
		out := make(map[uuid.UUID]int, len(p.Targets))
		for _, tgt := range p.Targets {
			out[tgt.TargetID] = tgt.ProjectedDamage
		}
		return out
	}
	t.Fatal("no resolution_updated for that turn in the master's collected messages")
	return nil
}

// lastResolutionUpdated pulls the most recent resolution_updated payload out of what the
// collector has gathered so far.
func lastResolutionUpdated(t *testing.T, c *collector) game.ResolutionUpdatedPayload {
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
		return p
	}
	t.Fatal("no resolution_updated in the collected messages")
	return game.ResolutionUpdatedPayload{}
}

// ─── commit 1: the fixture compiles and stands a live turn ────────────────────

// TestE2E_AreaAttackFixtureStandsATurn is commit 1's trivial proof: the four-character, four-
// player fixture connects, an attack enqueues, the master opens it, and a turn actually opens.
// Nothing here depends on face counts or exact damage numbers yet — that is commit 2.
func TestE2E_AreaAttackFixtureStandsATurn(t *testing.T) {
	f := newAreaFixture(t, []int{6, 4, 1, 1, 7, 3})
	defer f.server.Close()

	sword := "Sword"
	f.attacker.send(t, game.MsgTypeEnqueueAction, game.ActionPayload{
		ActorID:  f.attackerID,
		TargetID: []uuid.UUID{f.a, f.b, f.c},
		Attack: &game.AttackPayload{
			Weapon: &sword,
			Hit:    game.RollCheckPayload{SkillName: "Accuracy"},
			Damage: game.RollCheckPayload{},
		},
	})
	if !f.attacker.msgs.await(game.MsgTypeActionEnqueued, 2*time.Second) {
		t.Fatal("the attack was never acknowledged as enqueued")
	}
	f.master.send(t, game.MsgTypeOpenNextAction, struct{}{})

	turnID, actionID := f.awaitTurnOpened(t)
	if turnID == uuid.Nil {
		t.Error("turnID must not be nil")
	}
	if actionID == uuid.Nil {
		t.Error("actionID must not be nil")
	}
	if turnID == actionID {
		t.Error("the turn ID and the action ID are deliberately different UUIDs")
	}
}
