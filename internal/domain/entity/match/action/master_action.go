package action

import "github.com/google/uuid"

type MasterAction struct {
	TargetID    []uuid.UUID
	skills      []Skill //nolint:unused // WIP: match system under development
	Move        *Move
	Attack      *Attack
	ActionSpeed *RollCheck
	// Initiative *Initiative ?
	// Penalidade *Penalty ?
}

func NewMasterAction() *MasterAction {
	return &MasterAction{}
}
