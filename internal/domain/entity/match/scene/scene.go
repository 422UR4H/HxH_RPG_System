package scene

import (
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match/turn"
)

type Scene struct {
	category                enum.SceneCategory
	BriefInitialDescription string // need to be sent by the master when starting the match
	BriefFinalDescription   *string

	turns []*turn.Turn

	createdAt  time.Time
	finishedAt *time.Time
}

func NewScene(category enum.SceneCategory, briefInitialDescription string) *Scene {
	return &Scene{
		category:                category,
		BriefInitialDescription: briefInitialDescription,
		createdAt:               time.Now(),
	}
}

func (s *Scene) GetCategory() enum.SceneCategory {
	return s.category
}

func (s *Scene) AddTurn(turn *turn.Turn) error {
	if s.finishedAt != nil {
		return ErrSceneIsFinished
	}
	s.turns = append(s.turns, turn)
	return nil
}

func (s *Scene) GetTurns() []*turn.Turn {
	return s.turns
}

func (s *Scene) FinishScene(briefFinalDescription string) error {
	if s.finishedAt != nil {
		return ErrSceneIsFinished
	}
	s.BriefFinalDescription = &briefFinalDescription
	now := time.Now()
	s.finishedAt = &now
	return nil
}

func (s *Scene) GetCreatedAt() time.Time {
	return s.createdAt
}

func (s *Scene) GetFinishedAt() *time.Time {
	return s.finishedAt
}
