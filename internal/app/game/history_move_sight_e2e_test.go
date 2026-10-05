package game_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	apiAuth "github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	apiMatch "github.com/422UR4H/HxH_RPG_System/internal/app/api/match"
	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/application/testutil"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/google/uuid"
)

// GET /history shows each reader a move only as they saw it live (owner decision, 2026-10-01).
// The live gate's verdict, per session player, is recorded when the move is shown — at the
// opening for the turn's move, at the settled resolution for an escape's landing — persisted
// with the turn's close, and the history projects by it. Driven over real sockets; the history
// is read back through the real use case and the real REST handler from what the close handed
// PersistTurnClose.

type historyNoParticipationCheck struct{}

func (historyNoParticipationCheck) ExistsSheetInCampaign(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return true, nil
}

// historyTurnJSON is turns[0] of GET /history as reader reads it, raw.
func (f *combatFixture) historyTurnJSON(t *testing.T, reader uuid.UUID) map[string]json.RawMessage {
	t.Helper()
	participants := []*match.Participant{
		{Sheet: csEntity.Summary{UUID: f.attackerID, PlayerUUID: &f.playerUUID}},
		{Sheet: csEntity.Summary{UUID: f.victimID, PlayerUUID: &f.playerUUID}},
		{Sheet: csEntity.Summary{UUID: f.bystanderID, PlayerUUID: &f.bystanderUUID}},
	}
	uc := appmatch.NewGetMatchHistoryUC(
		&testutil.MockMatchRepo{
			GetMatchFn: func(context.Context, uuid.UUID) (*match.Match, error) {
				return &match.Match{UUID: f.matchUUID, MasterUUID: f.masterUUID, IsPublic: true}, nil
			},
			ListParticipantsByMatchUUIDFn: func(context.Context, uuid.UUID) ([]*match.Participant, error) {
				return participants, nil
			},
		},
		f.roundRepo, historyNoParticipationCheck{}, nil, nil,
	)
	ctx := context.WithValue(context.Background(), apiAuth.UserIDKey, reader)
	resp, err := apiMatch.GetMatchHistoryHandler(uc)(ctx, &apiMatch.GetMatchHistoryRequest{UUID: f.matchUUID})
	if err != nil {
		t.Fatalf("GET /history as %s: %v", reader, err)
	}
	raw, err := json.Marshal(resp.Body)
	if err != nil {
		t.Fatalf("marshal history: %v", err)
	}
	var body struct {
		Scenes []struct {
			Rounds []struct {
				Turns []map[string]json.RawMessage `json:"turns"`
			} `json:"rounds"`
		} `json:"scenes"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal history: %v (%s)", err, raw)
	}
	if len(body.Scenes) != 1 || len(body.Scenes[0].Rounds) != 1 || len(body.Scenes[0].Rounds[0].Turns) != 1 {
		t.Fatalf("history is not the one closed turn: %s", raw)
	}
	return body.Scenes[0].Rounds[0].Turns[0]
}

// awaitPersisted waits until the close has reached PersistTurnClose.
func (f *combatFixture) awaitPersisted(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for len(f.roundRepo.persistedTurnIDs()) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("the closed turn never reached PersistTurnClose")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestE2E_TheHistoryShowsEachReaderTheMoveAsTheySawItLive(t *testing.T) {
	full := moveShape{from: true, position: true}
	tests := []struct {
		name      string
		at        [2]int
		to        [3]int
		walls     []mapentity.WallSegment
		bystander moveShape
	}{
		{
			name: "out of sight during the dash: no from, no position",
			at:   [2]int{4, 4}, to: [3]int{6, 4, 0},
			walls:     []mapentity.WallSegment{moveBoardWall},
			bystander: moveShape{},
		},
		{
			name: "saw only the piece leave: from, no position",
			at:   [2]int{14, 4}, to: [3]int{14, 8, 0},
			walls:     []mapentity.WallSegment{originOnlyWall},
			bystander: moveShape{from: true},
		},
		{
			name: "saw the destination: the whole move",
			at:   [2]int{14, 4}, to: [3]int{16, 4, 0},
			walls:     []mapentity.WallSegment{moveBoardWall},
			bystander: full,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newCombatFixture(t, withBystander)
			f.seedMoverBoard(t, tt.at, true, tt.walls...)
			from := [3]int{tt.at[0], tt.at[1], 0}

			master, player := f.connect(t)
			defer master.Close() //nolint:errcheck
			defer player.Close() //nolint:errcheck
			bystander := connectWS(t, f.server.URL, f.bystanderUUID, f.matchUUID)
			defer bystander.Close()   //nolint:errcheck
			readMessage(t, bystander) // room_state
			masterMsgs := collectFrom(master)
			playerMsgs := collectFrom(player)
			bystanderMsgs := collectFrom(bystander)
			if !bystanderMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
				t.Fatal("the bystander never got a board — they are not really at the table")
			}

			f.enqueueDash(t, player, from, tt.to)
			if !playerMsgs.await(game.MsgTypeActionEnqueued, 2*time.Second) {
				t.Fatalf("the dash was never enqueued; the player received: %v",
					messageTypes(playerMsgs.snapshotMessages()))
			}
			sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
			if !bystanderMsgs.await(game.MsgTypeTurnOpened, 2*time.Second) {
				t.Fatal("the turn never opened for the bystander")
			}
			// The premise: what the bystander's history must say is what turn_opened said.
			live := rawMoveAt(t, findMessage(t, bystanderMsgs.snapshotMessages(), game.MsgTypeTurnOpened).Payload,
				"action", "move")
			assertMoveShape(t, "bystander, live", live, tt.bystander, from, tt.to)

			sendWS(t, master, string(game.MsgTypeCloseTurn), game.CloseTurnPayload{Confirm: true})
			if !masterMsgs.await(game.MsgTypeTurnClosed, 2*time.Second) {
				t.Fatalf("the turn never closed; the master received: %v", messageTypes(masterMsgs.snapshotMessages()))
			}
			f.awaitPersisted(t)

			moveIn := func(turn map[string]json.RawMessage) map[string]json.RawMessage {
				return rawMoveAt(t, turn["action"], "move")
			}
			assertMoveShape(t, "bystander, history", moveIn(f.historyTurnJSON(t, f.bystanderUUID)), tt.bystander, from, tt.to)
			assertMoveShape(t, "master, history", moveIn(f.historyTurnJSON(t, f.masterUUID)), full, from, tt.to)
			assertMoveShape(t, "owner, history", moveIn(f.historyTurnJSON(t, f.playerUUID)), full, from, tt.to)
		})
	}
}

func TestE2E_TheHistoryShowsAnEscapeLandingOnlyToWhoSawItLive(t *testing.T) {
	tests := []struct {
		name  string
		walls []mapentity.WallSegment
		sees  bool
	}{
		{name: "landing out of sight: no landing", walls: []mapentity.WallSegment{moveBoardWall}, sees: false},
		{name: "landing in sight: the landing", walls: nil, sees: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newCombatFixture(t, withVictimPiece, withBystander)
			f.seedBoardWith(t, tt.walls...)
			master, player := f.connect(t)
			defer master.Close() //nolint:errcheck
			defer player.Close() //nolint:errcheck
			bystander := connectWS(t, f.server.URL, f.bystanderUUID, f.matchUUID)
			defer bystander.Close()   //nolint:errcheck
			readMessage(t, bystander) // room_state
			masterMsgs := collectFrom(master)
			playerMsgs := collectFrom(player)
			bystanderMsgs := collectFrom(bystander)
			if !bystanderMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
				t.Fatal("the bystander never got a board — they are not really at the table")
			}

			reactionID := f.escapeStage(t, master, player, masterMsgs, playerMsgs, false, false, true)
			pos := escapeLanding
			f.chooseLanding(t, master, masterMsgs, reactionID, &pos)
			sendWS(t, master, string(game.MsgTypeCloseTurn), game.CloseTurnPayload{Confirm: true})
			if !awaitSettled(bystanderMsgs, 3*time.Second) {
				t.Fatal("the bystander never received the settled resolution")
			}
			// The premise: the live settled resolution decided what the history must say.
			if live := f.escapeOf(t, findSettledResolution(t, bystanderMsgs)); (live.Landing != nil) != tt.sees {
				t.Fatalf("premise: the live landing reached the bystander = %v, want %v", live.Landing != nil, tt.sees)
			}
			f.awaitPersisted(t)

			escapeIn := func(turn map[string]json.RawMessage) map[string]json.RawMessage {
				var res struct {
					Targets []struct {
						TargetID uuid.UUID                  `json:"targetId"`
						Escape   map[string]json.RawMessage `json:"escape"`
					} `json:"targets"`
				}
				if err := json.Unmarshal(turn["resolution"], &res); err != nil {
					t.Fatalf("unmarshal resolution: %v (%s)", err, turn["resolution"])
				}
				for _, tr := range res.Targets {
					if tr.TargetID == f.victimID && tr.Escape != nil {
						return tr.Escape
					}
				}
				t.Fatalf("no escape for the victim in %s", turn["resolution"])
				return nil
			}
			for who, reader := range map[string]uuid.UUID{"master": f.masterUUID, "owner": f.playerUUID} {
				if _, ok := escapeIn(f.historyTurnJSON(t, reader))["landing"]; !ok {
					t.Errorf("%s: the history has no escape.landing — they always see it", who)
				}
			}
			esc := escapeIn(f.historyTurnJSON(t, f.bystanderUUID))
			if _, ok := esc["landing"]; ok != tt.sees {
				t.Errorf("bystander: escape.landing in the history = %v, want %v (as live)", ok, tt.sees)
			}
			if string(esc["awaitsMaster"]) != "false" {
				t.Errorf("bystander: awaitsMaster = %s, want false — the master chose a landing", esc["awaitsMaster"])
			}
		})
	}
}

// An escape's REACTION carries a move too, and its move.position is the same news: where the
// escaping piece went. The reaction's action never reaches the table live, so the only way a
// bystander learned that destination is the piece_moved of an escape that ESCAPED — judged at
// the close, recorded per player, and the history shows the reaction's position by that and
// nothing else. A failed escape's position is a destination the piece never reached: never
// shown to anyone but the master and the owner, even to a bystander who saw where it landed.
func TestE2E_TheHistoryShowsAnEscapesMoveOnlyToWhoSawThePieceGoThere(t *testing.T) {
	tests := []struct {
		name    string
		walls   []mapentity.WallSegment
		escapes bool
		// moved is how many piece_moved the bystander gets for the escape at the close.
		moved int
		// sees is whether the bystander's history keeps the reaction's move.position.
		sees bool
	}{
		{name: "escaped, destination out of sight", walls: []mapentity.WallSegment{moveBoardWall}, escapes: true},
		{name: "escaped, destination in sight", escapes: true, moved: 1, sees: true},
		{name: "failed, the landing in sight: the attempted destination never", moved: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newCombatFixture(t, withVictimPiece, withBystander)
			f.seedBoardWith(t, tt.walls...)
			master, player := f.connect(t)
			defer master.Close() //nolint:errcheck
			defer player.Close() //nolint:errcheck
			bystander := connectWS(t, f.server.URL, f.bystanderUUID, f.matchUUID)
			defer bystander.Close()   //nolint:errcheck
			readMessage(t, bystander) // room_state
			masterMsgs := collectFrom(master)
			playerMsgs := collectFrom(player)
			bystanderMsgs := collectFrom(bystander)
			if !bystanderMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
				t.Fatal("the bystander never got a board — they are not really at the table")
			}

			reactionID := f.escapeStage(t, master, player, masterMsgs, playerMsgs, false, tt.escapes, true)
			if !tt.escapes {
				pos := escapeLanding
				f.chooseLanding(t, master, masterMsgs, reactionID, &pos)
			}
			sendWS(t, master, string(game.MsgTypeCloseTurn), game.CloseTurnPayload{Confirm: true})
			if !awaitSettled(bystanderMsgs, 3*time.Second) {
				t.Fatal("the bystander never received the settled resolution")
			}
			// The premise: what the live relay told the bystander about the escaping piece.
			if n := bystanderMsgs.count(game.MsgTypePieceMoved); n != tt.moved {
				t.Fatalf("premise: the bystander got %d piece_moved, want %d", n, tt.moved)
			}
			f.awaitPersisted(t)

			reactionMove := func(turn map[string]json.RawMessage) map[string]json.RawMessage {
				var reactions []struct {
					UUID uuid.UUID                  `json:"uuid"`
					Move map[string]json.RawMessage `json:"move"`
				}
				if err := json.Unmarshal(turn["reactions"], &reactions); err != nil {
					t.Fatalf("unmarshal reactions: %v (%s)", err, turn["reactions"])
				}
				for _, r := range reactions {
					if r.UUID == reactionID {
						if r.Move == nil {
							t.Fatalf("the escape carries no move: %s", turn["reactions"])
						}
						return r.Move
					}
				}
				t.Fatalf("no escape %s in %s", reactionID, turn["reactions"])
				return nil
			}
			for who, reader := range map[string]uuid.UUID{"master": f.masterUUID, "owner": f.playerUUID} {
				if _, ok := reactionMove(f.historyTurnJSON(t, reader))["position"]; !ok {
					t.Errorf("%s: the escape's move.position is missing — they always see it", who)
				}
			}
			move := reactionMove(f.historyTurnJSON(t, f.bystanderUUID))
			if _, ok := move["category"]; !ok {
				t.Error("bystander: the escape's move.category is missing — that it moved is public")
			}
			if _, ok := move["position"]; ok != tt.sees {
				t.Errorf("bystander: the escape's move.position in the history = %v, want %v", ok, tt.sees)
			}
		})
	}
}
