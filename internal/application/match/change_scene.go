package match

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	roundentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	sceneentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

type IChangeScene interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID, category enum.SceneCategory, briefDesc string) (*sceneentity.Scene, *roundentity.Round, error)
}

type ChangeSceneUC struct{}

func NewChangeSceneUC() *ChangeSceneUC {
	return &ChangeSceneUC{}
}

func (uc *ChangeSceneUC) Execute(
	ctx context.Context,
	session *matchsession.MatchSession,
	masterUUID, callerUUID uuid.UUID,
	category enum.SceneCategory,
	briefDesc string,
) (*sceneentity.Scene, *roundentity.Round, error) {
	if callerUUID != masterUUID {
		return nil, nil, ErrNotMatchMaster
	}
	return session.ChangeScene(category, briefDesc)
}

var _ IChangeScene = (*ChangeSceneUC)(nil)
