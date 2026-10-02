package match_test

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	matchDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	roundentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	sceneentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchboard"
	"github.com/google/uuid"
)

// noopRoundRepo is a minimal IRoundRepository that returns no active session.
type noopRoundRepo struct{}

func (m *noopRoundRepo) FindActiveSession(_ context.Context, _ uuid.UUID) (*matchsession.ActiveSessionData, error) {
	return nil, nil
}
func (m *noopRoundRepo) PersistTurnClose(_ context.Context, _ match.TurnCloseData) error {
	return nil
}
func (m *noopRoundRepo) CloseSceneAndRound(_ context.Context, _, _ uuid.UUID, _ time.Time) error {
	return nil
}
func (m *noopRoundRepo) EnsureSceneAndRound(_ context.Context, _ uuid.UUID, _ *sceneentity.Scene, _ *roundentity.Round) error {
	return nil
}
func (m *noopRoundRepo) PersistRoundClose(_ context.Context, _ uuid.UUID, _ *sceneentity.Scene, _, _ *roundentity.Round) error {
	return nil
}
func (m *noopRoundRepo) FindMatchHistory(_ context.Context, _ uuid.UUID) ([]match.HistoryScene, error) {
	return nil, nil
}

// mockRoundRepo allows controlling FindActiveSession per test.
type mockRoundRepo struct {
	findActiveFn func(ctx context.Context, matchUUID uuid.UUID) (*matchsession.ActiveSessionData, error)
}

func (m *mockRoundRepo) FindActiveSession(ctx context.Context, matchUUID uuid.UUID) (*matchsession.ActiveSessionData, error) {
	if m.findActiveFn != nil {
		return m.findActiveFn(ctx, matchUUID)
	}
	return nil, nil
}
func (m *mockRoundRepo) PersistTurnClose(_ context.Context, _ match.TurnCloseData) error {
	return nil
}
func (m *mockRoundRepo) CloseSceneAndRound(_ context.Context, _, _ uuid.UUID, _ time.Time) error {
	return nil
}
func (m *mockRoundRepo) EnsureSceneAndRound(_ context.Context, _ uuid.UUID, _ *sceneentity.Scene, _ *roundentity.Round) error {
	return nil
}
func (m *mockRoundRepo) PersistRoundClose(_ context.Context, _ uuid.UUID, _ *sceneentity.Scene, _, _ *roundentity.Round) error {
	return nil
}
func (m *mockRoundRepo) FindMatchHistory(_ context.Context, _ uuid.UUID) ([]match.HistoryScene, error) {
	return nil, nil
}

func TestInitMatchSession(t *testing.T) {
	matchUUID := uuid.New()
	playerUUID := uuid.New()
	sheetUUID := uuid.New()
	noop := &noopRoundRepo{}

	t.Run("creates session with loaded char sheets", func(t *testing.T) {
		pUUID := playerUUID
		repo := &mockMatchRepo{
			participants: []*matchDomain.Participant{
				{
					UUID:      uuid.New(),
					MatchUUID: matchUUID,
					Sheet: csEntity.Summary{
						UUID:       sheetUUID,
						PlayerUUID: &pUUID,
					},
				},
			},
		}
		sheet := &csSheet.CharacterSheet{}
		// An intact sheet: nothing to repair, so wasCorrected is false. This is the normal
		// case, and it must still land in the session.
		loader := &mockSheetLoader{sheet: sheet, wasCorrected: false}

		uc := match.NewInitMatchSessionUC(repo, loader, noop, nil, nil)
		session, err := uc.Init(context.Background(), matchUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if session == nil {
			t.Fatal("expected non-nil session")
		}
		got, err := session.GetCharSheet(sheetUUID)
		if err != nil {
			t.Fatalf("the intact sheet is missing from the session: %v", err)
		}
		if got != sheet {
			t.Error("expected the session to hold the loaded sheet")
		}
	})

	// Regression: the loader's second return is wasCorrected, not found. Reading it as
	// "found" kept only the sheets that had to be repaired and silently dropped every
	// intact one, leaving the session unable to resolve a single collision.
	t.Run("an intact sheet is not dropped as if it were missing", func(t *testing.T) {
		pUUID := playerUUID
		repo := &mockMatchRepo{
			participants: []*matchDomain.Participant{{
				UUID:      uuid.New(),
				MatchUUID: matchUUID,
				Sheet:     csEntity.Summary{UUID: sheetUUID, PlayerUUID: &pUUID},
			}},
		}
		loader := &mockSheetLoader{sheet: &csSheet.CharacterSheet{}, wasCorrected: false}

		uc := match.NewInitMatchSessionUC(repo, loader, noop, nil, nil)
		session, err := uc.Init(context.Background(), matchUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := session.GetCharSheet(sheetUUID); err != nil {
			t.Errorf("an intact sheet must be in the session, got %v", err)
		}
	})

	t.Run("loads the sheet of an NPC participant", func(t *testing.T) {
		npcSheetUUID := uuid.New()
		repo := &mockMatchRepo{
			participants: []*matchDomain.Participant{
				{
					UUID:      uuid.New(),
					MatchUUID: matchUUID,
					// NPC: no PlayerUUID. It used to be skipped before the loader ran.
					Sheet: csEntity.Summary{UUID: npcSheetUUID},
				},
			},
		}
		loader := &mockSheetLoader{sheet: &csSheet.CharacterSheet{}, wasCorrected: true}

		uc := match.NewInitMatchSessionUC(repo, loader, noop, nil, nil)
		session, err := uc.Init(context.Background(), matchUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := session.GetCharSheet(npcSheetUUID); err != nil {
			t.Errorf("expected the NPC sheet in the session, got %v", err)
		}
		if _, err := session.GetCharacterStatus(npcSheetUUID); err != nil {
			t.Errorf("expected a CharacterStatus for the NPC, got %v", err)
		}
	})

	t.Run("skips a participant whose sheet does not exist", func(t *testing.T) {
		repo := &mockMatchRepo{
			participants: []*matchDomain.Participant{
				{
					UUID:      uuid.New(),
					MatchUUID: matchUUID,
					Sheet:     csEntity.Summary{UUID: uuid.New()},
				},
			},
		}
		// A missing sheet is an error from the gateway, not a false.
		loader := &mockSheetLoader{err: charactersheet.ErrCharacterSheetNotFound}

		uc := match.NewInitMatchSessionUC(repo, loader, noop, nil, nil)
		session, err := uc.Init(context.Background(), matchUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if session == nil {
			t.Fatal("expected a non-nil session")
		}
	})
}

func TestInitMatchSessionUC_Recovery(t *testing.T) {
	emptyMatchRepo := &mockMatchRepo{participants: []*matchDomain.Participant{}}
	// No participants, so the loader is never called.
	emptyLoader := &mockSheetLoader{}

	t.Run("uses NewMatchSessionWithState when active session found", func(t *testing.T) {
		sceneID := uuid.New()
		roundID := uuid.New()
		now := time.Now()

		rr := &mockRoundRepo{
			findActiveFn: func(_ context.Context, _ uuid.UUID) (*matchsession.ActiveSessionData, error) {
				return &matchsession.ActiveSessionData{
					SceneID:        sceneID,
					Category:       string(enum.Battle),
					BriefInitDesc:  "Forest",
					SceneCreatedAt: now,
					RoundID:        roundID,
					Mode:           string(enum.Free),
					RoundCreatedAt: now,
				}, nil
			},
		}

		uc := match.NewInitMatchSessionUC(emptyMatchRepo, emptyLoader, rr, nil, nil)
		session, err := uc.Init(context.Background(), uuid.New())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !session.IsRoundPersisted() {
			t.Error("expected IsRoundPersisted true when recovering")
		}
		if session.GetActiveScene().GetID() != sceneID {
			t.Errorf("expected scene ID %v, got %v", sceneID, session.GetActiveScene().GetID())
		}
		if session.GetActiveRound().GetID() != roundID {
			t.Errorf("expected round ID %v, got %v", roundID, session.GetActiveRound().GetID())
		}
	})

	t.Run("uses NewMatchSession when no active session found", func(t *testing.T) {
		rr := &mockRoundRepo{}

		uc := match.NewInitMatchSessionUC(emptyMatchRepo, emptyLoader, rr, nil, nil)
		session, err := uc.Init(context.Background(), uuid.New())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if session.IsRoundPersisted() {
			t.Error("expected IsRoundPersisted false for fresh session")
		}
	})
}

// ── mocks ────────────────────────────────────────────────────────────────────

type mockMatchRepo struct {
	participants []*matchDomain.Participant
	err          error
	// masterUUID/getMatchErr back GetMatch, which B11's board-NPC enrollment needs to learn
	// who the requester is for AddMatchNPCInput.
	masterUUID  uuid.UUID
	getMatchErr error
	// embed the full IRepository to satisfy the interface without implementing all methods
	match.IRepository
}

func (m *mockMatchRepo) ListParticipantsByMatchUUID(_ context.Context, _ uuid.UUID) ([]*matchDomain.Participant, error) {
	return m.participants, m.err
}

func (m *mockMatchRepo) GetMatch(_ context.Context, _ uuid.UUID) (*matchDomain.Match, error) {
	if m.getMatchErr != nil {
		return nil, m.getMatchErr
	}
	return &matchDomain.Match{MasterUUID: m.masterUUID}, nil
}

// fakeBoardReader stands in for match.IBoardReader.
type fakeBoardReader struct {
	board *matchboard.Board
	err   error
	calls int
}

func (f *fakeBoardReader) Load(_ context.Context, _ uuid.UUID) (*matchboard.Board, error) {
	f.calls++
	return f.board, f.err
}

// fakeNPCEnroller stands in for match.INPCEnroller. addFn lets each test script exactly what
// Add does — including, when a test needs to simulate the roster already having moved (a
// race, or ErrNPCAlreadyInMatch), mutating the SAME participants slice a mockMatchRepo in the
// same test holds, so the re-fetch inside enrollBoardNPCs sees it.
type fakeNPCEnroller struct {
	mu    sync.Mutex
	calls []match.AddMatchNPCInput
	addFn func(in *match.AddMatchNPCInput) (*matchDomain.Participant, error)
}

func (f *fakeNPCEnroller) Add(
	_ context.Context, in *match.AddMatchNPCInput,
) (*matchDomain.Participant, error) {
	f.mu.Lock()
	f.calls = append(f.calls, *in)
	f.mu.Unlock()
	if f.addFn != nil {
		return f.addFn(in)
	}
	return &matchDomain.Participant{
		UUID: uuid.New(), MatchUUID: in.MatchUUID, Sheet: csEntity.Summary{UUID: in.SheetUUID},
	}, nil
}

func (f *fakeNPCEnroller) snapshot() []match.AddMatchNPCInput {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]match.AddMatchNPCInput(nil), f.calls...)
}

type mockSheetLoader struct {
	sheet *csSheet.CharacterSheet
	// wasCorrected mirrors the gateway's real second return: whether hydrating the sheet
	// had to repair it. It is NOT "found" — a missing sheet comes back as an error.
	wasCorrected bool
	err          error
}

func (m *mockSheetLoader) GetCharacterSheetByUUID(_ context.Context, _ string) (*csSheet.CharacterSheet, bool, error) {
	return m.sheet, m.wasCorrected, m.err
}

// TestInitMatchSessionUC_NPCRosterSurvivesRestart is the proof of done for this slice (D6):
// cmd/api and cmd/game are separate processes with no shared memory, so a match already in
// progress never sees an NPC the master just added — the room's live MatchSession is not
// touched. What actually has to be true is narrower and does not depend on either process
// being up: the NPC lands in match_participants, and the NEXT time a room is built —
// Room.StartMatch or the rehydrate path — InitMatchSessionUC reads that row back and turns
// it into a full participant: a CharacterStatus with both bars, and an entry in
// charToPlayer authorizing the master to act through it. This test exercises exactly that
// path, unmodified (G3), with a roster that mixes a player character and an NPC.
func TestInitMatchSessionUC_NPCRosterSurvivesRestart(t *testing.T) {
	matchUUID := uuid.New()
	playerUUID := uuid.New()
	masterUUID := uuid.New()
	playerSheetUUID := uuid.New()
	npcSheetUUID := uuid.New()
	noop := &noopRoundRepo{}

	repo := &mockMatchRepo{
		participants: []*matchDomain.Participant{
			{
				UUID:      uuid.New(),
				MatchUUID: matchUUID,
				Sheet:     csEntity.Summary{UUID: playerSheetUUID, PlayerUUID: &playerUUID},
			},
			{
				UUID:      uuid.New(),
				MatchUUID: matchUUID,
				// NPC: PlayerUUID nil, MasterUUID set — the master plays it.
				Sheet: csEntity.Summary{UUID: npcSheetUUID, MasterUUID: &masterUUID},
			},
		},
	}
	loader := &mockSheetLoader{sheet: &csSheet.CharacterSheet{}, wasCorrected: false}

	uc := match.NewInitMatchSessionUC(repo, loader, noop, nil, nil)
	session, err := uc.Init(context.Background(), matchUUID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session == nil {
		t.Fatal("expected a non-nil session")
	}

	status, err := session.GetCharacterStatus(npcSheetUUID)
	if err != nil {
		t.Fatalf("expected a CharacterStatus for the NPC, got error: %v", err)
	}
	if status == nil {
		t.Fatal("expected a non-nil CharacterStatus for the NPC")
	}
	// Both bars must be the fresh zero value NewCharacterStatus() builds — proving they
	// exist as part of the status the NPC got, not just that the struct field is typed.
	freshBar := matchDomain.ResourceBar{}
	if !reflect.DeepEqual(status.ActionBar, freshBar) {
		t.Errorf("expected a fresh ActionBar for the NPC, got %+v", status.ActionBar)
	}
	if !reflect.DeepEqual(status.MoveBar, freshBar) {
		t.Errorf("expected a fresh MoveBar for the NPC, got %+v", status.MoveBar)
	}

	charToPlayer := session.GetCharToPlayer()
	gotMaster, ok := charToPlayer[npcSheetUUID.String()]
	if !ok {
		t.Fatal("expected the NPC sheet UUID to be present in charToPlayer")
	}
	if gotMaster != masterUUID {
		t.Errorf("expected the NPC to map to the master %v, got %v", masterUUID, gotMaster)
	}

	gotPlayer, ok := charToPlayer[playerSheetUUID.String()]
	if !ok {
		t.Fatal("expected the player character sheet UUID to be present in charToPlayer")
	}
	if gotPlayer != playerUUID {
		t.Errorf("expected the player character to map to %v, got %v", playerUUID, gotPlayer)
	}
}

// TestInitMatchSessionUC_BoardNPCEnrollment is B11 (spec §4.3): when the match session is
// born — start_match and rehydration alike — every piece on the board whose character is not
// yet a participant is enrolled through the same AddMatchNPCUC.Add the REST POST /npcs and the
// WS add_npc verb already use. A player's piece that was never enrolled is skipped, never
// aborting Init.
func TestInitMatchSessionUC_BoardNPCEnrollment(t *testing.T) {
	noop := &noopRoundRepo{}

	t.Run("enrolls the master's NPC whose piece is on the board, landing in the session", func(t *testing.T) {
		matchUUID := uuid.New()
		masterUUID := uuid.New()
		npcSheetUUID := uuid.New()

		repo := &mockMatchRepo{masterUUID: masterUUID}
		board := &fakeBoardReader{board: &matchboard.Board{
			Pieces: []mapentity.Piece{{ID: "p1", CharacterID: npcSheetUUID.String()}},
		}}
		enroller := &fakeNPCEnroller{addFn: func(in *match.AddMatchNPCInput) (*matchDomain.Participant, error) {
			p := &matchDomain.Participant{
				UUID: uuid.New(), MatchUUID: in.MatchUUID,
				Sheet: csEntity.Summary{UUID: in.SheetUUID, MasterUUID: &masterUUID},
			}
			repo.participants = append(repo.participants, p)
			return p, nil
		}}
		loader := &mockSheetLoader{sheet: &csSheet.CharacterSheet{}}

		uc := match.NewInitMatchSessionUC(repo, loader, noop, board, enroller)
		session, err := uc.Init(context.Background(), matchUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		calls := enroller.snapshot()
		if len(calls) != 1 {
			t.Fatalf("Add called %d time(s), want 1", len(calls))
		}
		want := match.AddMatchNPCInput{RequesterUUID: masterUUID, MatchUUID: matchUUID, SheetUUID: npcSheetUUID}
		if calls[0] != want {
			t.Errorf("Add called with %+v, want %+v", calls[0], want)
		}
		if _, err := session.GetCharSheet(npcSheetUUID); err != nil {
			t.Errorf("expected the enrolled NPC's sheet in the session, got %v", err)
		}
	})

	t.Run("ErrNPCAlreadyInMatch is treated as success — the roster re-fetch picks it up", func(t *testing.T) {
		matchUUID := uuid.New()
		masterUUID := uuid.New()
		npcSheetUUID := uuid.New()

		repo := &mockMatchRepo{masterUUID: masterUUID}
		board := &fakeBoardReader{board: &matchboard.Board{
			Pieces: []mapentity.Piece{{ID: "p1", CharacterID: npcSheetUUID.String()}},
		}}
		enroller := &fakeNPCEnroller{addFn: func(in *match.AddMatchNPCInput) (*matchDomain.Participant, error) {
			// Simulates a race: some other writer already inserted the row between the
			// FIRST listing (empty, above) and this call — the re-fetch below must see it.
			repo.participants = append(repo.participants, &matchDomain.Participant{
				UUID: uuid.New(), MatchUUID: in.MatchUUID,
				Sheet: csEntity.Summary{UUID: in.SheetUUID, MasterUUID: &masterUUID},
			})
			return nil, match.ErrNPCAlreadyInMatch
		}}
		loader := &mockSheetLoader{sheet: &csSheet.CharacterSheet{}}

		uc := match.NewInitMatchSessionUC(repo, loader, noop, board, enroller)
		session, err := uc.Init(context.Background(), matchUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := session.GetCharSheet(npcSheetUUID); err != nil {
			t.Errorf("expected the NPC in the session despite ErrNPCAlreadyInMatch, got %v", err)
		}
	})

	// Review focus 3: a player's character with a piece on the board that was never enrolled
	// is skipped — Add's ErrSheetNotNPC says so — and the character must stay OUT of the
	// session, not just "Init doesn't crash".
	t.Run("a player's unenrolled piece is skipped and the character never enters the session", func(t *testing.T) {
		matchUUID := uuid.New()
		masterUUID := uuid.New()
		playerCharUUID := uuid.New()

		repo := &mockMatchRepo{masterUUID: masterUUID}
		board := &fakeBoardReader{board: &matchboard.Board{
			Pieces: []mapentity.Piece{{ID: "p1", CharacterID: playerCharUUID.String()}},
		}}
		enroller := &fakeNPCEnroller{addFn: func(_ *match.AddMatchNPCInput) (*matchDomain.Participant, error) {
			return nil, match.ErrSheetNotNPC
		}}
		loader := &mockSheetLoader{} // never reached for this character

		uc := match.NewInitMatchSessionUC(repo, loader, noop, board, enroller)
		session, err := uc.Init(context.Background(), matchUUID)
		if err != nil {
			t.Fatalf("Init must not fail when a piece cannot be enrolled: %v", err)
		}
		if session == nil {
			t.Fatal("expected a non-nil session")
		}
		if _, err := session.GetCharSheet(playerCharUUID); err == nil {
			t.Error("the unenrolled player's character must NOT be in the session")
		}
	})

	t.Run("ErrCharacterSheetNotFound and ErrSheetNotOwnedByMaster are skipped with a log, session still born", func(t *testing.T) {
		for _, addErr := range []error{match.ErrCharacterSheetNotFound, match.ErrSheetNotOwnedByMaster} {
			t.Run(addErr.Error(), func(t *testing.T) {
				matchUUID := uuid.New()
				masterUUID := uuid.New()
				charUUID := uuid.New()

				repo := &mockMatchRepo{masterUUID: masterUUID}
				board := &fakeBoardReader{board: &matchboard.Board{
					Pieces: []mapentity.Piece{{ID: "p1", CharacterID: charUUID.String()}},
				}}
				enroller := &fakeNPCEnroller{addFn: func(_ *match.AddMatchNPCInput) (*matchDomain.Participant, error) {
					return nil, addErr
				}}
				loader := &mockSheetLoader{}

				uc := match.NewInitMatchSessionUC(repo, loader, noop, board, enroller)
				session, err := uc.Init(context.Background(), matchUUID)
				if err != nil {
					t.Fatalf("Init must not fail: %v", err)
				}
				if session == nil {
					t.Fatal("expected a non-nil session")
				}
				if _, err := session.GetCharSheet(charUUID); err == nil {
					t.Error("the skipped character must not be in the session")
				}
			})
		}
	})

	t.Run("a piece whose character is already a participant never calls Add", func(t *testing.T) {
		matchUUID := uuid.New()
		masterUUID := uuid.New()
		sheetUUID := uuid.New()
		playerUUID := uuid.New()

		repo := &mockMatchRepo{
			masterUUID: masterUUID,
			participants: []*matchDomain.Participant{
				{UUID: uuid.New(), MatchUUID: matchUUID, Sheet: csEntity.Summary{UUID: sheetUUID, PlayerUUID: &playerUUID}},
			},
		}
		board := &fakeBoardReader{board: &matchboard.Board{
			Pieces: []mapentity.Piece{{ID: "p1", CharacterID: sheetUUID.String()}},
		}}
		enroller := &fakeNPCEnroller{}
		loader := &mockSheetLoader{sheet: &csSheet.CharacterSheet{}}

		uc := match.NewInitMatchSessionUC(repo, loader, noop, board, enroller)
		if _, err := uc.Init(context.Background(), matchUUID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if calls := enroller.snapshot(); len(calls) != 0 {
			t.Errorf("Add called %d time(s), want 0 — the character is already a participant", len(calls))
		}
	})

	t.Run("a piece with a non-UUID CharacterID is skipped without calling Add", func(t *testing.T) {
		matchUUID := uuid.New()
		repo := &mockMatchRepo{masterUUID: uuid.New()}
		board := &fakeBoardReader{board: &matchboard.Board{
			Pieces: []mapentity.Piece{{ID: "p1", CharacterID: "not-a-uuid"}},
		}}
		enroller := &fakeNPCEnroller{}
		loader := &mockSheetLoader{}

		uc := match.NewInitMatchSessionUC(repo, loader, noop, board, enroller)
		session, err := uc.Init(context.Background(), matchUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if session == nil {
			t.Fatal("expected a non-nil session")
		}
		if calls := enroller.snapshot(); len(calls) != 0 {
			t.Errorf("Add called %d time(s), want 0 — the CharacterID does not parse as a UUID", len(calls))
		}
	})

	t.Run("no board attached (Load returns nil, nil) behaves exactly as before B11", func(t *testing.T) {
		matchUUID := uuid.New()
		repo := &mockMatchRepo{masterUUID: uuid.New()}
		board := &fakeBoardReader{board: nil}
		enroller := &fakeNPCEnroller{}
		loader := &mockSheetLoader{}

		uc := match.NewInitMatchSessionUC(repo, loader, noop, board, enroller)
		if _, err := uc.Init(context.Background(), matchUUID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if calls := enroller.snapshot(); len(calls) != 0 {
			t.Errorf("Add called %d time(s), want 0 — no board to scan", len(calls))
		}
	})

	t.Run("calling Init twice over the same state is idempotent", func(t *testing.T) {
		matchUUID := uuid.New()
		masterUUID := uuid.New()
		npcSheetUUID := uuid.New()

		repo := &mockMatchRepo{masterUUID: masterUUID}
		board := &fakeBoardReader{board: &matchboard.Board{
			Pieces: []mapentity.Piece{{ID: "p1", CharacterID: npcSheetUUID.String()}},
		}}
		enroller := &fakeNPCEnroller{addFn: func(in *match.AddMatchNPCInput) (*matchDomain.Participant, error) {
			p := &matchDomain.Participant{
				UUID: uuid.New(), MatchUUID: in.MatchUUID,
				Sheet: csEntity.Summary{UUID: in.SheetUUID, MasterUUID: &masterUUID},
			}
			repo.participants = append(repo.participants, p)
			return p, nil
		}}
		loader := &mockSheetLoader{sheet: &csSheet.CharacterSheet{}}

		uc := match.NewInitMatchSessionUC(repo, loader, noop, board, enroller)

		first, err := uc.Init(context.Background(), matchUUID)
		if err != nil {
			t.Fatalf("first Init: unexpected error: %v", err)
		}
		if _, err := first.GetCharSheet(npcSheetUUID); err != nil {
			t.Fatalf("first Init: expected the NPC in the session: %v", err)
		}
		if calls := enroller.snapshot(); len(calls) != 1 {
			t.Fatalf("after first Init: Add called %d time(s), want 1", len(calls))
		}

		second, err := uc.Init(context.Background(), matchUUID)
		if err != nil {
			t.Fatalf("second Init: unexpected error: %v", err)
		}
		if _, err := second.GetCharSheet(npcSheetUUID); err != nil {
			t.Fatalf("second Init: expected the NPC in the session: %v", err)
		}
		if calls := enroller.snapshot(); len(calls) != 1 {
			t.Fatalf("after second Init: Add called %d time(s) total, want still 1 — the NPC was "+
				"already a participant by then", len(calls))
		}
	})
}
