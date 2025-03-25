package model

import (
	"time"

	"github.com/google/uuid"
)

type JointProficiency struct {
	ID   int
	UUID uuid.UUID

	Weapons []string
	Name    string
	Exp     int

	CreatedAt time.Time
	UpdatedAt time.Time
}
