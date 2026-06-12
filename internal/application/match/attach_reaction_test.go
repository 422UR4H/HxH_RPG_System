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

func sessionWithPlayers(playerUUIDs ...uuid.UUID) *matchsession.MatchSession {
	matchUUID := uuid.New()
	participants := make([]*matchDomain.Participant, len(playerUUIDs))
	for i, id := range playerUUIDs {
		pID := id
		participants[i] = &matchDomain.Participant{
			UUID:      uuid.New(),
			MatchUUID: matchUUID,
			Sheet:     csEntity.Summary{UUID: uuid.New(), PlayerUUID: &pID},
		}
	}
	return matchsession.NewMatchSession(matchUUID, nil, participants)
}

func TestAttachReactionUC(t *testing.T) {
	t.Run("returns AttachReactionResult with non-nil Resolution on valid reaction", func(t *testing.T) {
		playerA, playerB := uuid.New(), uuid.New()
		session := sessionWithPlayers(playerA, playerB)

		aAct := action.NewAction(playerA, nil, uuid.Nil, nil, action.ActionSpeed{RollCheck: action.RollCheck{Result: 10}}, nil, nil, nil, nil, nil, nil, nil)
		session.EnqueueAction(playerA, aAct) //nolint:errcheck
		_, opened, _ := session.OpenNextAction()
		openedAction := opened.GetAction()
		actionID := openedAction.GetID()

		reaction := action.NewAction(playerB, nil, actionID, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil)
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
		session := sessionWithPlayers(playerA)

		aAct := action.NewAction(playerA, nil, uuid.Nil, nil, action.ActionSpeed{RollCheck: action.RollCheck{Result: 5}}, nil, nil, nil, nil, nil, nil, nil)
		session.EnqueueAction(playerA, aAct) //nolint:errcheck
		session.OpenNextAction()              //nolint:errcheck

		reaction := action.NewAction(playerA, nil, uuid.New(), nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil)
		uc := match.NewAttachReactionUC()
		_, err := uc.Execute(context.Background(), session, playerA, reaction)
		if !errors.Is(err, service.ErrReactionNotCompatible) {
			t.Errorf("expected ErrReactionNotCompatible, got %v", err)
		}
	})
}
