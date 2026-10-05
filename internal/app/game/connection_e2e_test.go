package game_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// This file is the e2e guarantee for B4 and B7 (spec §4.6, task 14):
//
//   - B4 — the same account connecting twice: the LAST connection wins. The old one is told
//     `error` `connection_replaced` and the server closes it; the new one keeps the room.
//   - B7 — Register on a room whose Run has already returned never blocks: it returns
//     ErrRoomClosed instead, and a client whose ReadPump is unblocking against a dead room
//     exits instead of hanging on an unregister nobody is draining anymore.

// waitUntil polls cond every 10ms until it reports true or d elapses. Returns whether it
// succeeded — same shape as collector.await, used across this package's other e2e files.
func waitUntil(d time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// TestConnection_LastConnectionWins is B4's core scenario: the same player account opens a
// second connection (a second tab, a reload that raced its own close) while the first is
// still registered. The OLD one is told connection_replaced and the server closes it; the NEW
// one is the one left standing in the room.
func TestConnection_LastConnectionWins(t *testing.T) {
	matchUUID := uuid.New()
	masterUUID := uuid.New()
	playerUUID := uuid.New()

	server, hub := setupTestServer(masterUUID, true)
	defer server.Close()
	defer hub.Stop()

	master := connectWS(t, server.URL, masterUUID, matchUUID)
	defer master.Close() //nolint:errcheck
	readMessage(t, master) // room_state

	oldConn := connectWS(t, server.URL, playerUUID, matchUUID)
	defer oldConn.Close() //nolint:errcheck
	readMessage(t, oldConn) // room_state

	// Same account, second tab.
	newConn := connectWS(t, server.URL, playerUUID, matchUUID)
	defer newConn.Close() //nolint:errcheck
	readMessage(t, newConn) // room_state — the new connection registers exactly like any other

	// The OLD connection's next message is the replacement notice, not silence and not a
	// generic disconnect — the client needs the code to know not to reconnect on its own (the
	// other tab is the one that is now current).
	oldMsg := readMessage(t, oldConn)
	if oldMsg.Type != game.MsgTypeError {
		t.Fatalf("old connection's next message type = %q, want %q", oldMsg.Type, game.MsgTypeError)
	}
	var p game.ErrorPayload
	if err := json.Unmarshal(oldMsg.Payload, &p); err != nil {
		t.Fatalf("unmarshal error payload: %v", err)
	}
	if p.Code != "connection_replaced" {
		t.Fatalf("error code = %q, want %q", p.Code, "connection_replaced")
	}

	// The master's next chat reaches the NEW connection — the room still has a live player.
	sendWS(t, master, string(game.MsgTypeChat), game.ChatPayload{Message: "hello"})
	newMsg := readMessage(t, newConn)
	if newMsg.Type != game.MsgTypeChatMessage {
		t.Fatalf("new connection's next message type = %q, want %q", newMsg.Type, game.MsgTypeChatMessage)
	}
}

// TestConnection_OldClosingAfterwardsDoesNotEvictTheNewConnection is review focus 5: once the
// old connection has actually finished closing (its own ReadPump exits and sends its
// unregister, same pointer, some time AFTER the new one already took the account's slot), the
// new connection must still be the one in the room — the pre-existing unregister pointer guard
// has to keep working alongside B4's new replacement path — and the room itself must not
// close.
func TestConnection_OldClosingAfterwardsDoesNotEvictTheNewConnection(t *testing.T) {
	matchUUID := uuid.New()
	masterUUID := uuid.New()
	playerUUID := uuid.New()

	server, hub := setupTestServer(masterUUID, true)
	defer server.Close()
	defer hub.Stop()

	master := connectWS(t, server.URL, masterUUID, matchUUID)
	defer master.Close() //nolint:errcheck
	readMessage(t, master) // room_state

	oldConn := connectWS(t, server.URL, playerUUID, matchUUID)
	readMessage(t, oldConn) // room_state

	room, ok := hub.GetRoom(matchUUID)
	if !ok {
		t.Fatal("room was never created")
	}
	if got := room.ClientCount(); got != 2 {
		t.Fatalf("ClientCount before the replacement = %d, want 2 (master + old player)", got)
	}

	newConn := connectWS(t, server.URL, playerUUID, matchUUID)
	defer newConn.Close() //nolint:errcheck
	readMessage(t, newConn) // room_state

	// Drain the replacement notice, then let the OLD connection actually finish closing —
	// its own ReadPump has to notice the server already closed its socket and send its
	// unregister, with the OLD pointer, strictly AFTER the new one already replaced it in
	// r.clients.
	readMessage(t, oldConn) // error connection_replaced
	if _, _, err := oldConn.ReadMessage(); err == nil {
		t.Fatal("the old connection's socket should already be closing server-side")
	}
	_ = oldConn.Close()

	// Give the server-side ReadPump/unregister a moment to actually run, then prove the guard
	// held: still exactly two clients (master + the account's one slot, now the new
	// connection), and the room never mistook this for "everyone left".
	if !waitUntil(time.Second, func() bool {
		return room.GetState() != game.RoomStateClosed
	}) {
		t.Fatal("room closed")
	}
	time.Sleep(100 * time.Millisecond) // let the old ReadPump's unregister, if any, actually land
	if got := room.ClientCount(); got != 2 {
		t.Fatalf("ClientCount after the old connection finished closing = %d, want 2 — "+
			"the stale unregister must not have evicted the new connection", got)
	}
	if room.GetState() == game.RoomStateClosed {
		t.Fatal("the room closed because of the old connection's delayed unregister")
	}

	// The new connection is still the one receiving broadcasts.
	sendWS(t, master, string(game.MsgTypeChat), game.ChatPayload{Message: "still here"})
	msg := readMessage(t, newConn)
	if msg.Type != game.MsgTypeChatMessage {
		t.Fatalf("new connection's next message type = %q, want %q", msg.Type, game.MsgTypeChatMessage)
	}
}

// TestRoomRegister_ReturnsErrRoomClosedQuickly is B7: Register against a Room whose Run has
// already returned must not block — it has to come back with ErrRoomClosed, fast.
func TestRoomRegister_ReturnsErrRoomClosedQuickly(t *testing.T) {
	room := game.NewRoom(uuid.New(), uuid.New(), game.RoomDeps{})
	go room.Run()
	room.Stop()

	if !waitUntil(time.Second, func() bool { return room.GetState() == game.RoomStateClosed }) {
		t.Fatal("room never reached RoomStateClosed after Stop")
	}

	client := game.NewClient(uuid.New(), nil, "nick")
	errCh := make(chan error, 1)
	go func() { errCh <- room.Register(client) }()

	select {
	case err := <-errCh:
		if !errors.Is(err, game.ErrRoomClosed) {
			t.Fatalf("Register error = %v, want %v", err, game.ErrRoomClosed)
		}
	case <-time.After(time.Second):
		t.Fatal("Register hung for over 1s against a closed room")
	}
}

// newBareWSConn upgrades a bare HTTP connection to a WebSocket, bypassing game.Handler
// entirely — this test needs a genuine server-side *websocket.Conn to hand to game.NewClient
// directly, with no room-registration machinery attached to it, so it can drive ReadPump in
// isolation against a room it wires up by hand.
func newBareWSConn(t *testing.T) (serverConn, clientConn *websocket.Conn) {
	t.Helper()
	var mu sync.Mutex
	var sc *websocket.Conn
	ready := make(chan struct{})

	upgrader := websocket.Upgrader{}
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		mu.Lock()
		sc = conn
		mu.Unlock()
		close(ready)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	cc, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	t.Cleanup(func() { _ = cc.Close() })

	<-ready
	mu.Lock()
	defer mu.Unlock()
	return sc, cc
}

// TestClientReadPump_ExitsWhenItsRoomHasAlreadyClosed is B7's other half: a client whose
// ReadPump is unblocking (its socket just errored) AFTER its room's Run has already
// returned must not hang trying to send its unregister — nobody is draining that channel
// anymore. Before the fix this deadlocked the goroutine forever; a `select` against the
// room's `done` is what lets it exit.
func TestClientReadPump_ExitsWhenItsRoomHasAlreadyClosed(t *testing.T) {
	serverConn, clientConn := newBareWSConn(t)

	room := game.NewRoom(uuid.New(), uuid.New(), game.RoomDeps{})
	go room.Run()
	room.Stop()
	if !waitUntil(time.Second, func() bool { return room.GetState() == game.RoomStateClosed }) {
		t.Fatal("room never reached RoomStateClosed after Stop")
	}

	client := game.NewClient(uuid.New(), serverConn, "nick")
	client.SetRoom(room)

	exited := make(chan struct{})
	go func() {
		client.ReadPump()
		close(exited)
	}()

	// Force the server-side read to error out, the same way a dropped network connection
	// would — this is what makes ReadPump reach its deferred unregister-send.
	_ = clientConn.Close()

	select {
	case <-exited:
	case <-time.After(time.Second):
		t.Fatal("ReadPump hung for over 1s trying to unregister against an already-closed room")
	}
}
