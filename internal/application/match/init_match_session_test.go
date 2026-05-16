package match_test

import (
	"context"
	"testing"
	"time"

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
		loader := &mockSheetLoader{
			sheet: &csSheet.CharacterSheet{},
			found: true,
		}

		uc := match.NewInitMatchSessionUC(repo, loader, noop)
		session, err := uc.Init(context.Background(), matchUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if session == nil {
			t.Fatal("expected non-nil session")
		}
	})

	t.Run("creates session even when sheet not found (NPC case)", func(t *testing.T) {
		repo := &mockMatchRepo{
			participants: []*matchDomain.Participant{
				{
					UUID:      uuid.New(),
					MatchUUID: matchUUID,
					Sheet:     csEntity.Summary{UUID: sheetUUID}, // no PlayerUUID
				},
			},
		}
		loader := &mockSheetLoader{found: false}

		uc := match.NewInitMatchSessionUC(repo, loader, noop)
		session, err := uc.Init(context.Background(), matchUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if session == nil {
			t.Fatal("expected non-nil session")
		}
	})
}

func TestInitMatchSessionUC_Recovery(t *testing.T) {
	emptyMatchRepo := &mockMatchRepo{participants: []*matchDomain.Participant{}}
	emptyLoader := &mockSheetLoader{found: false}

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
	found bool
	err   error
}

func (m *mockSheetLoader) GetCharacterSheetByUUID(_ context.Context, _ string) (*csSheet.CharacterSheet, bool, error) {
	return m.sheet, m.found, m.err
}
