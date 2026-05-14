package action

import (
	"time"

	"github.com/google/uuid"
)

type MasterAction struct {
	TargetID    []uuid.UUID
	Skills      []Skill
	Move        *Move
	Attack      *Attack
	ActionSpeed *RollCheck
	happenedAt  time.Time
	// Initiative *Initiative ?
	// Penalidade *Penalty ?
}

func NewMasterAction() *MasterAction {
	return &MasterAction{}
}

func (ma *MasterAction) GetHappenedAt() time.Time {
	return ma.happenedAt
}

func (ma *MasterAction) SetHappenedAt(t time.Time) {
	ma.happenedAt = t
}

func (ma *MasterAction) GetSkills() []Skill {
	skillsCopy := make([]Skill, len(ma.Skills))
	copy(skillsCopy, ma.Skills)
	return skillsCopy
}
