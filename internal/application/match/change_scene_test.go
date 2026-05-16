package match_test

import (
	"context"
	"errors"
	"testing"

	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

func TestChangeSceneUC(t *testing.T) {
	masterUUID := uuid.New()

	t.Run("returns ErrNotMatchMaster when caller is not master", func(t *testing.T) {
		s := matchsession.NewMatchSession(uuid.New(), nil, nil)
		uc := appmatch.NewChangeSceneUC()
		caller := uuid.New()

		_, _, err := uc.Execute(context.Background(), s, masterUUID, caller, enum.Battle, "Arena")
		if !errors.Is(err, appmatch.ErrNotMatchMaster) {
			t.Errorf("expected ErrNotMatchMaster, got %v", err)
		}
	})

	t.Run("changes scene when master calls with valid args", func(t *testing.T) {
		s := matchsession.NewMatchSession(uuid.New(), nil, nil)
		uc := appmatch.NewChangeSceneUC()

		oldScene, oldRound, err := uc.Execute(context.Background(), s, masterUUID, masterUUID, enum.Battle, "Arena fight")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if oldScene == nil {
			t.Fatal("expected non-nil old scene")
		}
		if oldRound == nil {
			t.Fatal("expected non-nil old round")
		}
		if oldScene.GetFinishedAt() == nil {
			t.Error("expected old scene to be closed")
		}
		if s.GetActiveScene().GetCategory() != enum.Battle {
			t.Errorf("expected active scene category Battle, got %v", s.GetActiveScene().GetCategory())
		}
	})
}
