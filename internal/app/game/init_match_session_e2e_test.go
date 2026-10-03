package game_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	matchboarduc "github.com/422UR4H/HxH_RPG_System/internal/application/matchboard"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	matchDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchboard"
	"github.com/google/uuid"
)

// This file is B11's end-to-end guarantee (spec §4.3, Task 9): starting a match with an NPC's
// piece already on the lobby's board must enroll that NPC into the roster, through the REAL
// InitMatchSessionUC, over a real Room and a real start_match — the path the rest of Phase 6's
// verification never exercised (it always drove combat against a session the test itself
// built by hand, via combatSessionUC).
//
// The only fakes here are repositories: matchNPCFixtureRepo doubles as BOTH IRepository
// (ListParticipantsByMatchUUID, GetMatch) and the local INPCEnroller (Add) InitMatchSessionUC
// needs — the same shape the production wiring in cmd/game/main.go gives it, where
// AddMatchNPCUC's roster write and the match read live behind the same matchRepository.

// ─── fakes ──────────────────────────────────────────────────────────────────

// matchNPCFixtureRepo is the fixture's own tiny "database": a mutable participants slice a
// real InitMatchSessionUC reads and writes through, exactly the way a real Postgres roster
// would. Add appends directly to it, so the SECOND ListParticipantsByMatchUUID call
// enrollBoardNPCs makes after enrolling sees the new NPC — the very thing this test proves.
type matchNPCFixtureRepo struct {
	mu           sync.Mutex
	masterUUID   uuid.UUID
	participants []*matchDomain.Participant
	addCalls     []appmatch.AddMatchNPCInput

	appmatch.IRepository // embedded nil: only GetMatch/ListParticipantsByMatchUUID are used
}

func (r *matchNPCFixtureRepo) GetMatch(_ context.Context, _ uuid.UUID) (*matchDomain.Match, error) {
	return &matchDomain.Match{MasterUUID: r.masterUUID}, nil
}

func (r *matchNPCFixtureRepo) ListParticipantsByMatchUUID(
	_ context.Context, _ uuid.UUID,
) ([]*matchDomain.Participant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]*matchDomain.Participant(nil), r.participants...), nil
}

// Add is the fixture's stand-in for AddMatchNPCUC.Add: no ownership rules, it simply rosters
// the sheet as an NPC of the requester (the master, always, since enrollBoardNPCs is the only
// caller here) and records the call for the test to assert against.
func (r *matchNPCFixtureRepo) Add(
	_ context.Context, in *appmatch.AddMatchNPCInput,
) (*matchDomain.Participant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.addCalls = append(r.addCalls, *in)
	p := &matchDomain.Participant{
		UUID:      uuid.New(),
		MatchUUID: in.MatchUUID,
		Sheet:     csEntity.Summary{UUID: in.SheetUUID, MasterUUID: &in.RequesterUUID},
		JoinedAt:  time.Now(),
	}
	r.participants = append(r.participants, p)
	return p, nil
}

func (r *matchNPCFixtureRepo) addCallSnapshot() []appmatch.AddMatchNPCInput {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]appmatch.AddMatchNPCInput(nil), r.addCalls...)
}

// fakeInitSheetLoader stands in for appmatch.ICharSheetLoader: any UUID gets a fresh, valid
// sheet — this test cares about the roster and the bars, not the sheet's own contents.
type fakeInitSheetLoader struct{ t *testing.T }

func (f *fakeInitSheetLoader) GetCharacterSheetByUUID(
	_ context.Context, _ string,
) (*csSheet.CharacterSheet, bool, error) {
	return newCombatSheet(f.t), false, nil
}

// ─── the test ───────────────────────────────────────────────────────────────

// TestE2E_StartingTheMatchEnrollsTheNPCsOnTheBoard drives:
//
//	master  → connect                (lobby; loadBoard picks up the seeded NPC piece)
//	master  → start_match            (persists the lobby board, then runs the REAL Init)
//	master  → reconnect              (match_full_state.bars.characters proves the NPC is in)
func TestE2E_StartingTheMatchEnrollsTheNPCsOnTheBoard(t *testing.T) {
	matchUUID := uuid.New()
	masterUUID := uuid.New()
	npcSheetUUID := uuid.New()
	mapUUID := uuid.New()

	// The lobby's board already carries the NPC's piece — B11 reads what T3 persists at
	// start_match, which is whatever is actually on screen right now.
	boards := newFakeBoardStore()
	boards.seed(matchUUID, &matchboard.Board{
		MatchUUID: matchUUID,
		MapUUID:   mapUUID,
		Grid:      mapentity.DefaultGrid(),
		Pieces: []mapentity.Piece{{
			ID:          "piece-1",
			CharacterID: npcSheetUUID.String(),
			Coord: mapentity.PieceCoord{
				Slot: mapentity.SquareCoord{Kind: "square", Col: 1, Row: 1},
			},
			Visible: true,
		}},
	})
	memories := newFakeMemoryStore()

	repo := &matchNPCFixtureRepo{masterUUID: masterUUID}
	sheetLoader := &fakeInitSheetLoader{t: t}
	roundRepo := &mockRoundRepoHandler{}
	// The REAL InitMatchSessionUC — this is the whole point of the test — wired exactly as
	// cmd/game/main.go wires it: boards.Load satisfies IBoardReader, repo.Add satisfies
	// INPCEnroller.
	initUC := appmatch.NewInitMatchSessionUC(repo, sheetLoader, roundRepo, boards, repo)

	hub := game.NewHub()
	go hub.Run()
	handler := game.NewHandler(
		hub,
		&fogMatchRepo{masterUUID: masterUUID, started: false},
		&mockEnrollmentChecker{enrolled: true},
		game.RoomDeps{
			StartMatchUC:  &mockStartMatchUC{},
			KickPlayerUC:  &mockKickPlayerUC{},
			InitSessionUC: initUC,
			RoundRepo:     roundRepo,
			LoadBoardUC:   boards,
			SaveBoardUC:   matchboarduc.NewSaveMatchBoardUC(boards, memories),
			MemoryLoader:  memories,
		},
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handler.HandleWebSocket)
	server := httptest.NewServer(mux)
	defer server.Close()
	defer hub.Stop()

	// Master connects to the lobby: loadBoard runs on register and picks up the seeded piece.
	master := connectWS(t, server.URL, masterUUID, matchUUID)
	_ = readMessage(t, master) // room_state
	master.Close()             //nolint:errcheck

	// A fresh connection drives start_match — using a collector so the reconnect below cannot
	// race whatever start_match's own direct sends produce on this same connection.
	starter := connectWS(t, server.URL, masterUUID, matchUUID)
	defer starter.Close()       //nolint:errcheck
	_ = readMessage(t, starter) // room_state

	sendWS(t, starter, string(game.MsgTypeStartMatch), map[string]any{})
	starterMsgs := newCollector(starter)
	if !starterMsgs.await(game.MsgTypeMatchStarted, 2*time.Second) {
		t.Fatal("start_match never produced match_started")
	}

	t.Run("the real InitMatchSessionUC enrolled the NPC exactly once", func(t *testing.T) {
		calls := repo.addCallSnapshot()
		if len(calls) != 1 {
			t.Fatalf("Add called %d time(s), want 1", len(calls))
		}
		want := appmatch.AddMatchNPCInput{RequesterUUID: masterUUID, MatchUUID: matchUUID, SheetUUID: npcSheetUUID}
		if calls[0] != want {
			t.Errorf("Add called with %+v, want %+v", calls[0], want)
		}
	})

	t.Run("the NPC appears in match_full_state.bars.characters", func(t *testing.T) {
		// A reconnect is what re-triggers buildMatchFullState for the master (Room.Run's
		// register branch sends it whenever session != nil) — the same thing a browser
		// reload after start does.
		reconnected := connectWS(t, server.URL, masterUUID, matchUUID)
		defer reconnected.Close() //nolint:errcheck
		msgs := newCollector(reconnected)

		if !msgs.await(game.MsgTypeMatchFullState, 2*time.Second) {
			t.Fatalf("no match_full_state after reconnect; got: %v", messageTypes(msgs.snapshotMessages()))
		}
		var found bool
		for _, m := range msgs.snapshotMessages() {
			if m.Type != game.MsgTypeMatchFullState {
				continue
			}
			var p game.MatchFullStatePayload
			if err := json.Unmarshal(m.Payload, &p); err != nil {
				t.Fatalf("unmarshal match_full_state: %v", err)
			}
			for _, c := range p.Bars.Characters {
				if c.CharacterID == npcSheetUUID {
					found = true
				}
			}
		}
		if !found {
			t.Error("the NPC's CharacterID never showed up in match_full_state.bars.characters")
		}
	})
}
