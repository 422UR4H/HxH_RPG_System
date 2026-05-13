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

func TestOpenNextActionUC(t *testing.T) {
	masterUUID := uuid.New()
	playerUUID := uuid.New()

	t.Run("returns ErrNotMatchMaster when caller is not master", func(t *testing.T) {
		session := matchsession.NewMatchSession(uuid.New(), nil, nil)
		uc := match.NewOpenNextActionUC()
		_, err := uc.Execute(context.Background(), session, masterUUID, uuid.New())
		if !errors.Is(err, match.ErrNotMatchMaster) {
			t.Errorf("expected ErrNotMatchMaster, got %v", err)
		}
	})

	t.Run("returns result with opened turn on success", func(t *testing.T) {
		pUUID := playerUUID
		matchUUID := uuid.New()
		p := &matchDomain.Participant{
			UUID:      uuid.New(),
			MatchUUID: matchUUID,
			Sheet:     csEntity.Summary{UUID: uuid.New(), PlayerUUID: &pUUID},
		}
		session := matchsession.NewMatchSession(matchUUID, nil, []*matchDomain.Participant{p})
		a := action.NewAction(playerUUID, nil, uuid.Nil, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil)
		session.EnqueueAction(playerUUID, a) //nolint:errcheck

		uc := match.NewOpenNextActionUC()
		result, err := uc.Execute(context.Background(), session, masterUUID, masterUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.OpenedTurn == nil {
			t.Error("expected non-nil OpenedTurn")
		}
		if result.Resolution == nil {
			t.Error("expected non-nil Resolution")
		}
	})

	t.Run("returns ErrQueueEmpty when queue is empty", func(t *testing.T) {
		session := matchsession.NewMatchSession(uuid.New(), nil, nil)
		uc := match.NewOpenNextActionUC()
		_, err := uc.Execute(context.Background(), session, masterUUID, masterUUID)
		if !errors.Is(err, service.ErrQueueEmpty) {
			t.Errorf("expected ErrQueueEmpty, got %v", err)
		}
	})
}
