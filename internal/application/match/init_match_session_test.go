package match_test

import (
	"context"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	matchDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/google/uuid"
)

func TestInitMatchSession(t *testing.T) {
	matchUUID := uuid.New()
	playerUUID := uuid.New()
	sheetUUID := uuid.New()

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

		uc := match.NewInitMatchSessionUC(repo, loader)
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

		uc := match.NewInitMatchSessionUC(repo, loader)
		session, err := uc.Init(context.Background(), matchUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if session == nil {
			t.Fatal("expected non-nil session")
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
