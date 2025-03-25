package model

import (
	"time"

	"github.com/google/uuid"
)

type Proficiency struct {
	ID   int
	UUID uuid.UUID

	Weapon string
	Exp    int

	CreatedAt time.Time
	UpdatedAt time.Time
}
