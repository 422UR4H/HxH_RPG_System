package game_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
)

// A player can win the race against the master's rehydration after a server restart: the
// handler creates the room (GetOrCreateRoom) and only then reads the database to rebuild the
// session, so a player arriving in that window registers in a room with no session and is
// greeted with no match_full_state. When the master's register lands, the table must hear it
// — the same way it already hears the re-pushed map_full_state.

// lateMasterTable stands up a hub whose room already exists, sessionless — the state right
// after GetOrCreateRoom on a freshly started server — behind the real handler.
func lateMasterTable(t *testing.T, f *combatFixture) {
	t.Helper()
	hub := game.NewHub()
	go hub.Run()
	handler := game.NewHandler(
		hub,
		&fogMatchRepo{masterUUID: f.masterUUID, started: !f.lobby},
		&mockEnrollmentChecker{enrolled: true},
		f.roomDeps(f.session, f.roundRepo),
	)
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handler.HandleWebSocket)
	srv := httptest.NewServer(mux)
	t.Cleanup(func() {
		srv.Close()
		hub.Stop()
	})
	f.server = srv
	hub.GetOrCreateRoom(f.matchUUID, f.masterUUID, f.roomDeps(f.session, f.roundRepo))
}

func TestE2E_PlayerThereBeforeTheMasterRehydratesGetsMatchFullState(t *testing.T) {
	f := newCombatFixture(t)
	lateMasterTable(t, f)

	player := connectWS(t, f.server.URL, f.playerUUID, f.matchUUID)
	defer player.Close() //nolint:errcheck
	readMessage(t, player) // room_state
	pc := collectFrom(player)

	master := connectWS(t, f.server.URL, f.masterUUID, f.matchUUID)
	defer master.Close() //nolint:errcheck

	if !pc.await(game.MsgTypeMasterJoined, 2*time.Second) {
		t.Fatalf("the player never saw the master join; got %v", messageTypes(pc.snapshotMessages()))
	}
	if !pc.await(game.MsgTypeMatchFullState, 2*time.Second) {
		t.Fatalf("the player never got match_full_state once the session came alive; got %v",
			messageTypes(pc.snapshotMessages()))
	}
}

func TestE2E_MasterReconnectInALobbySendsNoMatchFullState(t *testing.T) {
	f := newCombatFixture(t, inLobby)
	lateMasterTable(t, f)

	player := connectWS(t, f.server.URL, f.playerUUID, f.matchUUID)
	defer player.Close() //nolint:errcheck
	readMessage(t, player) // room_state
	pc := collectFrom(player)

	master := connectWS(t, f.server.URL, f.masterUUID, f.matchUUID)
	defer master.Close() //nolint:errcheck

	if !pc.await(game.MsgTypeMasterJoined, 2*time.Second) {
		t.Fatalf("the player never saw the master join; got %v", messageTypes(pc.snapshotMessages()))
	}
	if n := pc.count(game.MsgTypeMatchFullState); n != 0 {
		t.Fatalf("a lobby sent %d match_full_state to the player, want none", n)
	}
}
