package match_test

import (
	"context"
	"errors"
	"testing"

	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	matchDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

// sessionWithPlayers builds a session and returns it together with the sheet UUID of each
// player's character, in the order the players were given. An action's actorID carries the
// sheet UUID, not the player's.
func sessionWithPlayers(playerUUIDs ...uuid.UUID) (*matchsession.MatchSession, []uuid.UUID) {
	matchUUID := uuid.New()
	participants := make([]*matchDomain.Participant, len(playerUUIDs))
	charIDs := make([]uuid.UUID, len(playerUUIDs))
	for i, id := range playerUUIDs {
		pID := id
		participants[i] = &matchDomain.Participant{
			UUID:      uuid.New(),
			MatchUUID: matchUUID,
			Sheet:     csEntity.Summary{UUID: uuid.New(), PlayerUUID: &pID},
		}
		charIDs[i] = participants[i].Sheet.UUID
	}
	return matchsession.NewMatchSession(matchUUID, nil, participants), charIDs
}

func TestAttachReactionUC(t *testing.T) {
	t.Run("returns AttachReactionResult with non-nil Resolution on valid reaction", func(t *testing.T) {
		playerA, playerB := uuid.New(), uuid.New()
		session, chars := sessionWithPlayers(playerA, playerB)

		aAct := action.NewAction(chars[0], nil, uuid.Nil, nil, action.ActionSpeed{RollCheck: action.RollCheck{Result: 10}}, nil, nil, nil, nil, nil, nil, nil)
		session.EnqueueAction(playerA, aAct) //nolint:errcheck
		tr, _ := session.OpenNextAction()
		openedAction := tr.Opened.GetAction()
		actionID := openedAction.GetID()

		reaction := action.NewAction(chars[1], nil, actionID, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil)
		uc := match.NewAttachReactionUC()
		result, err := uc.Execute(context.Background(), session, playerB, reaction)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil AttachReactionResult")
		}
		if result.Resolution == nil {
			t.Error("expected non-nil Resolution")
		}
	})

	t.Run("returns ErrReactionNotCompatible for wrong target", func(t *testing.T) {
		playerA := uuid.New()
		session, chars := sessionWithPlayers(playerA)

		aAct := action.NewAction(chars[0], nil, uuid.Nil, nil, action.ActionSpeed{RollCheck: action.RollCheck{Result: 5}}, nil, nil, nil, nil, nil, nil, nil)
		session.EnqueueAction(playerA, aAct) //nolint:errcheck
		session.OpenNextAction()             //nolint:errcheck

		reaction := action.NewAction(chars[0], nil, uuid.New(), nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil)
		uc := match.NewAttachReactionUC()
		_, err := uc.Execute(context.Background(), session, playerA, reaction)
		if !errors.Is(err, service.ErrReactionNotCompatible) {
			t.Errorf("expected ErrReactionNotCompatible, got %v", err)
		}
	})
}
