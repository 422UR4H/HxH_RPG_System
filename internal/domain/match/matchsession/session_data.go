package matchsession

import (
	"time"

	"github.com/google/uuid"
)

// ActiveSessionData is the DB-hydration DTO returned by the round repository.
// Defined in the matchsession package so neither the gateway nor the application
// layer needs to import each other for this type.
type ActiveSessionData struct {
	SceneID        uuid.UUID
	Category       string
	BriefInitDesc  string
	SceneCreatedAt time.Time
	RoundID        uuid.UUID
	Mode           string
	RoundCreatedAt time.Time
	// RoundCounted/RoundDrained são o que o round carrega do relógio da história.
	RoundCounted bool
	RoundDrained bool
	// SceneStoryStartedAt é o horário da história em que a cena começou (nil: não registrado).
	SceneStoryStartedAt *time.Time
}
