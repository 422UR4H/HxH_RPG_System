package action

import "github.com/google/uuid"

type Initiative struct {
	targetID    []uuid.UUID
	skills      []Skill
	FinalResult int
}
