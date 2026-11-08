// internal/domain/matchevent/event.go
package matchevent

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Kind identifies what happened inside a round that is not a turn and is nobody's action
// (spec §4.5, B15). Master actions have a table of their own (masteraction, §4.8) and are
// never recorded here.
type Kind string

const (
	KindRoundModeChanged Kind = "roundModeChanged"
	// Relógio da história (spec 2026-11-08 §7). Payloads, tempos como clock.FormatStory:
	// clockStarted {"at"}; clockAdjusted {"from","to","deltaSeconds"};
	// clockRealTime {"from","to","elapsedSeconds","reason"} (gravado a cada pausa).
	KindClockStarted  Kind = "clockStarted"
	KindClockAdjusted Kind = "clockAdjusted"
	KindClockRealTime Kind = "clockRealTime"
)

// Event is something that happened inside a round that is not a turn and is nobody's
// action — today only a round mode change ({"from": ..., "to": ...} in Payload).
type Event struct {
	UUID      uuid.UUID
	MatchUUID uuid.UUID
	SceneUUID uuid.UUID
	RoundUUID uuid.UUID
	Kind      Kind
	Payload   json.RawMessage
	CreatedAt time.Time
}
