package model

import (
	"time"

	"github.com/google/uuid"
)

type CharacterProfile struct {
	ID   int
	UUID uuid.UUID

	NickName         string
	FullName         string
	Alignment        string
	Description      string
	BriefDescription string
	Birthday         time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}
