package matchsession_test

import (
	"errors"
	"testing"

	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

func TestNewMatchSession(t *testing.T) {
	matchUUID := uuid.New()
	playerUUID := uuid.New()

	participant := makeParticipant(matchUUID, &playerUUID)
	sheet := &csSheet.CharacterSheet{}
	sheets := map[uuid.UUID]*csSheet.CharacterSheet{playerUUID: sheet}

	s := matchsession.NewMatchSession(matchUUID, sheets, []*match.Participant{participant})

	if s == nil {
		t.Fatal("expected non-nil MatchSession")
	}
	if s.GetActiveRound() == nil {
		t.Error("expected non-nil activeRound on new session")
	}
	if s.GetActiveRound().GetMode() != enum.Free {
		t.Error("expected initial round mode to be Free")
	}
}

func TestMatchSession_GetCharSheet(t *testing.T) {
	matchUUID := uuid.New()
	playerUUID := uuid.New()
	participant := makeParticipant(matchUUID, &playerUUID)
	sheet := &csSheet.CharacterSheet{}
	sheets := map[uuid.UUID]*csSheet.CharacterSheet{playerUUID: sheet}
	s := matchsession.NewMatchSession(matchUUID, sheets, []*match.Participant{participant})

	t.Run("returns sheet for known player", func(t *testing.T) {
		got, err := s.GetCharSheet(playerUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != sheet {
			t.Error("expected same sheet pointer")
		}
	})

	t.Run("returns ErrCharSheetNotFound for unknown player", func(t *testing.T) {
		_, err := s.GetCharSheet(uuid.New())
		if !errors.Is(err, matchsession.ErrCharSheetNotFound) {
			t.Errorf("expected ErrCharSheetNotFound, got %v", err)
		}
	})
}

func TestNewMatchSession_NPCParticipantSkipped(t *testing.T) {
	matchUUID := uuid.New()
	// NPC participant: Sheet.PlayerUUID is nil
	npcParticipant := makeParticipant(matchUUID, nil)
	s := matchsession.NewMatchSession(matchUUID, nil, []*match.Participant{npcParticipant})
	if s == nil {
		t.Fatal("expected non-nil MatchSession even with NPC participant")
	}
	// Attempting to get a char sheet for any UUID should fail (nothing was loaded)
	_, err := s.GetCharSheet(uuid.New())
	if !errors.Is(err, matchsession.ErrCharSheetNotFound) {
		t.Errorf("expected ErrCharSheetNotFound, got %v", err)
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

func makeParticipant(matchUUID uuid.UUID, playerUUID *uuid.UUID) *match.Participant {
	return &match.Participant{
		UUID:      uuid.New(),
		MatchUUID: matchUUID,
		Sheet: csEntity.Summary{
			UUID:       uuid.New(),
			PlayerUUID: playerUUID,
		},
	}
}
