package match_test

import (
	"context"
	"testing"
	"time"

	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	matchDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	roundentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	sceneentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
	turnentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

// noopRoundRepo is a minimal IRoundRepository that returns no active session.
type noopRoundRepo struct{}

func (m *noopRoundRepo) FindActiveSession(_ context.Context, _ uuid.UUID) (*matchsession.ActiveSessionData, error) {
	return nil, nil
}
func (m *noopRoundRepo) PersistTurnClose(_ context.Context, _ *sceneentity.Scene, _ *roundentity.Round, _ *turnentity.Turn, _ *action.Action, _ uuid.UUID) error {
	return nil
}
func (m *noopRoundRepo) CloseSceneAndRound(_ context.Context, _, _ uuid.UUID, _ time.Time) error {
	return nil
}
func (m *noopRoundRepo) CloseRound(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
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
func (m *mockRoundRepo) PersistTurnClose(_ context.Context, _ *sceneentity.Scene, _ *roundentity.Round, _ *turnentity.Turn, _ *action.Action, _ uuid.UUID) error {
	return nil
}
func (m *mockRoundRepo) CloseSceneAndRound(_ context.Context, _, _ uuid.UUID, _ time.Time) error {
	return nil
}
func (m *mockRoundRepo) CloseRound(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
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

		uc := match.NewInitMatchSessionUC(repo, loader, noop)
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

		uc := match.NewInitMatchSessionUC(repo, loader, noop)
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

		uc := match.NewInitMatchSessionUC(repo, loader, noop)
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

		uc := match.NewInitMatchSessionUC(repo, loader, noop)
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

		uc := match.NewInitMatchSessionUC(emptyMatchRepo, emptyLoader, rr)
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

		uc := match.NewInitMatchSessionUC(emptyMatchRepo, emptyLoader, rr)
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
	// embed the full IRepository to satisfy the interface without implementing all methods
	match.IRepository
}

func (m *mockMatchRepo) ListParticipantsByMatchUUID(_ context.Context, _ uuid.UUID) ([]*matchDomain.Participant, error) {
	return m.participants, m.err
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
