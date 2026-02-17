package action

import "github.com/google/uuid"

type MasterAction struct {
	TargetID    []uuid.UUID
	skills      []Skill
	Move        *Move
	Attack      *Attack
	ActionSpeed *RollCheck
	// Initiative *Initiative ?
	// Penalidade *Penalty ?
}

func NewMasterAction() *MasterAction {
	return &MasterAction{}
}
