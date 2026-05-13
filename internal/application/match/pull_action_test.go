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
		uc := match.NewPullActionUC()
		_, err := uc.Execute(context.Background(), session, masterUUID, uuid.New(), uuid.New())
		if !errors.Is(err, match.ErrNotMatchMaster) {
			t.Errorf("expected ErrNotMatchMaster, got %v", err)
		}
	})

	t.Run("returns ErrActionNotFound for unknown actionID", func(t *testing.T) {
		session := matchsession.NewMatchSession(uuid.New(), nil, nil)
		uc := match.NewPullActionUC()
		_, err := uc.Execute(context.Background(), session, masterUUID, masterUUID, uuid.New())
		if !errors.Is(err, service.ErrActionNotFound) {
			t.Errorf("expected ErrActionNotFound, got %v", err)
		}
	})
}
