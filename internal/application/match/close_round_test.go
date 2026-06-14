package match_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	roundentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	sceneentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
	turnentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

// mockCloseRoundRepo is a minimal IRoundRepository for CloseRoundUC tests.
type mockCloseRoundRepo struct {
	closeRoundErr error
}

func (m *mockCloseRoundRepo) CloseRound(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return m.closeRoundErr
}
func (m *mockCloseRoundRepo) CloseSceneAndRound(_ context.Context, _, _ uuid.UUID, _ time.Time) error {
	return nil
}
func (m *mockCloseRoundRepo) FindActiveSession(_ context.Context, _ uuid.UUID) (*matchsession.ActiveSessionData, error) {
	return nil, nil
}
func (m *mockCloseRoundRepo) PersistTurnClose(_ context.Context, _ *sceneentity.Scene, _ *roundentity.Round, _ *turnentity.Turn, _ *action.Action, _ uuid.UUID) error {
	return nil
}

func TestCloseRoundUC(t *testing.T) {
	masterUUID := uuid.New()
	repo := &mockCloseRoundRepo{}

	t.Run("returns ErrNotMatchMaster for non-master caller", func(t *testing.T) {
		session := matchsession.NewMatchSession(uuid.New(), nil, nil)
		uc := match.NewCloseRoundUC(repo)
		_, err := uc.Execute(context.Background(), session, masterUUID, uuid.New())
		if !errors.Is(err, match.ErrNotMatchMaster) {
			t.Errorf("expected ErrNotMatchMaster, got %v", err)
		}
	})

	t.Run("closes round when no open turn", func(t *testing.T) {
		playerA := uuid.New()
		session := sessionWithPlayers(playerA)

		aAct := action.NewAction(playerA, nil, uuid.Nil, nil, action.ActionSpeed{RollCheck: action.RollCheck{Result: 5}}, nil, nil, nil, nil, nil, nil, nil)
		session.EnqueueAction(playerA, aAct) //nolint:errcheck
		session.OpenNextAction()              //nolint:errcheck
		session.CloseTurn()                   //nolint:errcheck

		uc := match.NewCloseRoundUC(repo)
		closedRound, err := uc.Execute(context.Background(), session, masterUUID, masterUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if closedRound == nil {
			t.Fatal("expected non-nil closed round")
		}
		if closedRound.GetFinishedAt() == nil {
			t.Error("expected finishedAt to be set on closed round")
		}
	})

	t.Run("returns ErrRoundHasOpenTurn when turn is still open", func(t *testing.T) {
		playerA := uuid.New()
		session := sessionWithPlayers(playerA)

		aAct := action.NewAction(playerA, nil, uuid.Nil, nil, action.ActionSpeed{RollCheck: action.RollCheck{Result: 5}}, nil, nil, nil, nil, nil, nil, nil)
		session.EnqueueAction(playerA, aAct) //nolint:errcheck
		session.OpenNextAction()              //nolint:errcheck
		// turn is still open — no CloseTurn called

		uc := match.NewCloseRoundUC(repo)
		_, err := uc.Execute(context.Background(), session, masterUUID, masterUUID)
		if !errors.Is(err, matchsession.ErrRoundHasOpenTurn) {
			t.Errorf("expected ErrRoundHasOpenTurn, got %v", err)
		}
	})
}
