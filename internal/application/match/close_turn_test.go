package match_test

import (
	"context"
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

func TestCloseTurnUC(t *testing.T) {
	masterUUID := uuid.New()

	t.Run("returns ErrNotMatchMaster for non-master caller", func(t *testing.T) {
		session := matchsession.NewMatchSession(uuid.New(), nil, nil)
		uc := match.NewCloseTurnUC()
		_, err := uc.Execute(context.Background(), session, masterUUID, uuid.New())
		if !errors.Is(err, match.ErrNotMatchMaster) {
			t.Errorf("expected ErrNotMatchMaster, got %v", err)
		}
	})

	t.Run("closes the current open turn", func(t *testing.T) {
		playerA := uuid.New()
		session := sessionWithPlayers(playerA)

		aAct := action.NewAction(playerA, nil, uuid.Nil, nil, action.ActionSpeed{RollCheck: action.RollCheck{Result: 5}}, nil, nil, nil, nil, nil, nil)
		session.EnqueueAction(playerA, aAct) //nolint:errcheck
		_, opened, _ := session.OpenNextAction()

		uc := match.NewCloseTurnUC()
		closed, err := uc.Execute(context.Background(), session, masterUUID, masterUUID)
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
			t.Error("expected finishedAt to be set on closed turn")
		}
	})

	t.Run("returns ErrNoCurrentTurn when round has no turns", func(t *testing.T) {
		session := matchsession.NewMatchSession(uuid.New(), nil, nil)
		uc := match.NewCloseTurnUC()
		_, err := uc.Execute(context.Background(), session, masterUUID, masterUUID)
		if !errors.Is(err, service.ErrNoCurrentTurn) {
			t.Errorf("expected ErrNoCurrentTurn, got %v", err)
		}
	})
}
