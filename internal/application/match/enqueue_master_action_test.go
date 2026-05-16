package match_test

import (
	"context"
	"errors"
	"testing"

	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

func TestEnqueueMasterActionUC(t *testing.T) {
	masterUUID := uuid.New()

	t.Run("returns ErrNotMatchMaster when caller is not master", func(t *testing.T) {
		s := matchsession.NewMatchSession(uuid.New(), nil, nil)
		uc := appmatch.NewEnqueueMasterActionUC()
		caller := uuid.New()
		ma := action.NewMasterAction()

		err := uc.Execute(context.Background(), s, masterUUID, caller, ma)
		if !errors.Is(err, appmatch.ErrNotMatchMaster) {
			t.Errorf("expected ErrNotMatchMaster, got %v", err)
		}
	})

	t.Run("returns ErrNoActiveTurn when no open turn", func(t *testing.T) {
		s := matchsession.NewMatchSession(uuid.New(), nil, nil)
		uc := appmatch.NewEnqueueMasterActionUC()
		ma := action.NewMasterAction()

		err := uc.Execute(context.Background(), s, masterUUID, masterUUID, ma)
		if !errors.Is(err, matchsession.ErrNoActiveTurn) {
			t.Errorf("expected ErrNoActiveTurn, got %v", err)
		}
	})
}
