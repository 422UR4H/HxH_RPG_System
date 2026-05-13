package matchsession_test

import (
	"errors"
	"testing"

	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
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

func makeAction(actorID uuid.UUID) *action.Action {
	return action.NewAction(actorID, nil, uuid.Nil, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil)
}

func sessionWithParticipants(playerUUIDs ...uuid.UUID) *matchsession.MatchSession {
	matchUUID := uuid.New()
	participants := make([]*match.Participant, len(playerUUIDs))
	for i, id := range playerUUIDs {
		pID := id
		participants[i] = makeParticipant(matchUUID, &pID)
	}
	return matchsession.NewMatchSession(matchUUID, nil, participants)
}

func makeActionWithSpeed(actorID uuid.UUID, speed int) *action.Action {
	return action.NewAction(actorID, nil, uuid.Nil, nil, action.ActionSpeed{RollCheck: action.RollCheck{Result: speed}}, nil, nil, nil, nil, nil, nil)
}

func TestMatchSession_EnqueueAction(t *testing.T) {
	matchUUID := uuid.New()
	playerUUID := uuid.New()
	participant := makeParticipant(matchUUID, &playerUUID)
	s := matchsession.NewMatchSession(matchUUID, nil, []*match.Participant{participant})

	t.Run("enqueues action for known participant", func(t *testing.T) {
		a := makeAction(playerUUID)
		if err := s.EnqueueAction(playerUUID, a); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns ErrParticipantNotFound for unknown player", func(t *testing.T) {
		a := makeAction(uuid.New())
		err := s.EnqueueAction(uuid.New(), a)
		if !errors.Is(err, matchsession.ErrParticipantNotFound) {
			t.Errorf("expected ErrParticipantNotFound, got %v", err)
		}
	})

	t.Run("returns ErrActionActorMismatch when actorID does not match playerUUID", func(t *testing.T) {
		a := makeAction(uuid.New()) // actorID is a different UUID
		err := s.EnqueueAction(playerUUID, a)
		if !errors.Is(err, matchsession.ErrActionActorMismatch) {
			t.Errorf("expected ErrActionActorMismatch, got %v", err)
		}
	})
}

func TestMatchSession_OpenNextAction(t *testing.T) {
	t.Run("opens Turn from highest-priority action in queue", func(t *testing.T) {
		playerA := uuid.New()
		playerB := uuid.New()
		s := sessionWithParticipants(playerA, playerB)

		aHigh := makeActionWithSpeed(playerA, 10)
		aLow := makeActionWithSpeed(playerB, 3)
		s.EnqueueAction(playerA, aHigh) //nolint:errcheck
		s.EnqueueAction(playerB, aLow)  //nolint:errcheck

		closed, opened, err := s.OpenNextAction()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if closed != nil {
			t.Error("expected nil closed turn on first OpenNextAction")
		}
		if opened == nil {
			t.Fatal("expected non-nil opened turn")
		}
		if opened.GetAction().Speed.Result != 10 {
			t.Errorf("expected speed 10, got %d", opened.GetAction().Speed.Result)
		}
	})

	t.Run("closes previous open turn before opening next", func(t *testing.T) {
		playerA, playerB := uuid.New(), uuid.New()
		s := sessionWithParticipants(playerA, playerB)
		s.EnqueueAction(playerA, makeActionWithSpeed(playerA, 10)) //nolint:errcheck
		s.EnqueueAction(playerB, makeActionWithSpeed(playerB, 5))  //nolint:errcheck

		_, first, _ := s.OpenNextAction()
		closed, _, err := s.OpenNextAction()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if closed == nil {
			t.Fatal("expected closed turn to be non-nil on second call")
		}
		if closed != first {
			t.Error("expected closed turn to be the first opened turn")
		}
		if first.GetFinishedAt() == nil {
			t.Error("expected first turn to be closed")
		}
	})

	t.Run("returns service.ErrQueueEmpty when queue is empty", func(t *testing.T) {
		s := matchsession.NewMatchSession(uuid.New(), nil, nil)
		_, _, err := s.OpenNextAction()
		if !errors.Is(err, service.ErrQueueEmpty) {
			t.Errorf("expected ErrQueueEmpty, got %v", err)
		}
	})
}

func makeReactionTo(actorID, targetActionID uuid.UUID) *action.Action {
	return action.NewAction(actorID, nil, targetActionID, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil)
}

func TestMatchSession_AttachReaction(t *testing.T) {
	t.Run("attaches reaction to current turn and returns resolution", func(t *testing.T) {
		playerA, playerB := uuid.New(), uuid.New()
		s := sessionWithParticipants(playerA, playerB)
		s.EnqueueAction(playerA, makeActionWithSpeed(playerA, 10)) //nolint:errcheck
		_, opened, _ := s.OpenNextAction()
		act := opened.GetAction()
		actionID := act.GetID()

		reaction := makeReactionTo(playerB, actionID)
		res, err := s.AttachReaction(reaction)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res == nil {
			t.Fatal("expected non-nil TurnResolution")
		}
		if len(opened.GetReactions()) != 1 {
			t.Errorf("expected 1 reaction, got %d", len(opened.GetReactions()))
		}
	})

	t.Run("returns ErrReactionNotCompatible for wrong target", func(t *testing.T) {
		playerA := uuid.New()
		s := sessionWithParticipants(playerA)
		s.EnqueueAction(playerA, makeActionWithSpeed(playerA, 5)) //nolint:errcheck
		s.OpenNextAction()                                         //nolint:errcheck

		reaction := makeReactionTo(playerA, uuid.New()) // wrong target
		_, err := s.AttachReaction(reaction)
		if !errors.Is(err, service.ErrReactionNotCompatible) {
			t.Errorf("expected ErrReactionNotCompatible, got %v", err)
		}
	})
}

func TestMatchSession_CloseTurn(t *testing.T) {
	t.Run("closes current open turn", func(t *testing.T) {
		playerA := uuid.New()
		s := sessionWithParticipants(playerA)
		s.EnqueueAction(playerA, makeActionWithSpeed(playerA, 5)) //nolint:errcheck
		_, opened, _ := s.OpenNextAction()

		closed, err := s.CloseTurn()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if closed == nil {
			t.Fatal("expected non-nil closed turn")
		}
		if closed != opened {
			t.Error("expected closed turn to be the opened turn")
		}
		if closed.GetFinishedAt() == nil {
			t.Error("expected finishedAt to be set")
		}
	})

	t.Run("returns ErrNoCurrentTurn when no turns exist", func(t *testing.T) {
		s := matchsession.NewMatchSession(uuid.New(), nil, nil)
		_, err := s.CloseTurn()
		if !errors.Is(err, service.ErrNoCurrentTurn) {
			t.Errorf("expected ErrNoCurrentTurn, got %v", err)
		}
	})
}

func TestMatchSession_CloseRound(t *testing.T) {
	t.Run("closes round and starts a new one with same mode", func(t *testing.T) {
		playerA := uuid.New()
		s := sessionWithParticipants(playerA)
		s.EnqueueAction(playerA, makeActionWithSpeed(playerA, 5)) //nolint:errcheck
		s.OpenNextAction()                                         //nolint:errcheck
		s.CloseTurn()                                              //nolint:errcheck

		closedRound, err := s.CloseRound()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if closedRound == nil {
			t.Fatal("expected non-nil closed round")
		}
		if closedRound.GetFinishedAt() == nil {
			t.Error("expected finishedAt to be set on closed round")
		}
		if s.GetActiveRound() == closedRound {
			t.Error("expected activeRound to be a new round after CloseRound")
		}
	})

	t.Run("returns ErrRoundHasOpenTurn when a turn is still open", func(t *testing.T) {
		playerA := uuid.New()
		s := sessionWithParticipants(playerA)
		s.EnqueueAction(playerA, makeActionWithSpeed(playerA, 5)) //nolint:errcheck
		s.OpenNextAction()                                         //nolint:errcheck
		// turn is still open — no CloseTurn called

		_, err := s.CloseRound()
		if !errors.Is(err, matchsession.ErrRoundHasOpenTurn) {
			t.Errorf("expected ErrRoundHasOpenTurn, got %v", err)
		}
	})
}

func TestMatchSession_PullAction(t *testing.T) {
	t.Run("opens Turn for specific action UUID", func(t *testing.T) {
		playerA, playerB := uuid.New(), uuid.New()
		s := sessionWithParticipants(playerA, playerB)
		aTarget := makeActionWithSpeed(playerA, 3)
		aOther := makeActionWithSpeed(playerB, 10)
		s.EnqueueAction(playerA, aTarget) //nolint:errcheck
		s.EnqueueAction(playerB, aOther)  //nolint:errcheck
		targetID := aTarget.GetID()

		_, opened, err := s.PullAction(targetID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := opened.GetAction()
		if got.GetID() != targetID {
			t.Errorf("expected action %v, got %v", targetID, got.GetID())
		}
	})

	t.Run("returns service.ErrActionNotFound for unknown UUID", func(t *testing.T) {
		s := matchsession.NewMatchSession(uuid.New(), nil, nil)
		_, _, err := s.PullAction(uuid.New())
		if !errors.Is(err, service.ErrActionNotFound) {
			t.Errorf("expected ErrActionNotFound, got %v", err)
		}
	})
}
