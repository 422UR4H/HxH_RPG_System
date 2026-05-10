package action

import (
	"time"

	"github.com/google/uuid"
)

type MasterAction struct {
	TargetID    []uuid.UUID
	skills      []Skill //nolint:unused // WIP: match system under development
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

func (ma *MasterAction) GetSkills() []Skill {
	// 1. create a new slice with the exact same length
	skillsCopy := make([]Skill, len(ma.skills))
	// 2. Copy the data from the original slice to the new one
	copy(skillsCopy, ma.skills)
	return skillsCopy
}
