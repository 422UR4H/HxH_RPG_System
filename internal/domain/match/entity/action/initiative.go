package action

import "github.com/google/uuid"

type Initiative struct {
	targetID    []uuid.UUID //nolint:unused // WIP: match system under development
	skills      []Skill     //nolint:unused // WIP: match system under development
	FinalResult int
}
