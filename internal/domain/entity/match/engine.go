package match

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match/scene"
)

type Engine struct {
	// adicionar scene aqui?
	scenes        []*scene.Scene
	sceneCategory enum.SceneCategory
}

func NewEngine(match *Match) (*Engine, error) {
	if match == nil {
		return nil, ErrMatchIsNil
	}
	var sceneCategory enum.SceneCategory
	scenes := match.GetScenes()
	if len(scenes) > 0 {
		sceneCategory = scenes[len(scenes)-1].GetCategory()
	}

	return &Engine{
		scenes:        scenes,
		sceneCategory: sceneCategory,
	}, nil
}

// TODO: create and finish Initiative and turn.engine.ChangeMode to continue here
func (e *Engine) ChangeScene(initiative *action.Initiative) {
	if e.sceneCategory == enum.Battle {
		e.sceneCategory = enum.Roleplay
		return
	}
	e.sceneCategory = enum.Battle

	if initiative != nil { //nolint:staticcheck // TODO: process initiative actions
	}
}
