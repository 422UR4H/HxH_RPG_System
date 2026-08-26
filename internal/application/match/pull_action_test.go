package match_test

import (
	"context"
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

func TestPullActionUC(t *testing.T) {
	masterUUID := uuid.New()

	t.Run("returns ErrNotMatchMaster when caller is not master", func(t *testing.T) {
		session := matchsession.NewMatchSession(uuid.New(), nil, nil)
		uc := match.NewPullActionUC(&fakeStatusWriter{}, nil)
		_, err := uc.Execute(context.Background(), session, masterUUID, uuid.New(), uuid.New())
		if !errors.Is(err, match.ErrNotMatchMaster) {
			t.Errorf("expected ErrNotMatchMaster, got %v", err)
		}
	})

	t.Run("returns ErrActionNotFound for unknown actionID", func(t *testing.T) {
		session := matchsession.NewMatchSession(uuid.New(), nil, nil)
		uc := match.NewPullActionUC(&fakeStatusWriter{}, nil)
		_, err := uc.Execute(context.Background(), session, masterUUID, masterUUID, uuid.New())
		if !errors.Is(err, service.ErrActionNotFound) {
			t.Errorf("expected ErrActionNotFound, got %v", err)
		}
	})
}

// TestPullActionUC_ClosedTurnSurvivesAPullFailure mirrors
// TestOpenNextActionUC_ClosedTurnSurvivesAnOpenFailure: PullAction's Execute has the identical
// shape and the identical bug. session.PullAction always closes the currently open turn
// FIRST, before it can fail to find the requested action — so an unknown actionID against a
// session with an open turn and nothing else queued must still report that turn closed.
func TestPullActionUC_ClosedTurnSurvivesAPullFailure(t *testing.T) {
	f := newTurnFixture(t)
	writer := &fakeStatusWriter{}
	uc := match.NewPullActionUC(writer, nil)

	res, err := uc.Execute(context.Background(), f.session, f.masterUUID, f.masterUUID, uuid.New())

	if !errors.Is(err, service.ErrActionNotFound) {
		t.Fatalf("err = %v, want ErrActionNotFound", err)
	}
	if res == nil {
		t.Fatal("the previous turn already closed and its damage already applied — losing the result here would strand the table")
	}
	if res.ClosedTurn == nil {
		t.Error("expected the turn that was open before this call to be reported closed")
	}
	if res.ClosedResolution == nil {
		t.Error("expected the settled resolution for the closed turn")
	}
	if len(writer.calls) != 1 || writer.calls[0] != f.victimChar.String() {
		t.Errorf("persisted = %v, want exactly the victim %s", writer.calls, f.victimChar)
	}
}
