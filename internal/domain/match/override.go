package match

import (
	"time"

	"github.com/google/uuid"
)

// OverrideOrigin says where the displaced value came from. Two origins, one overwriter —
// the master. There is no Source field naming the overwriter, because if only the master
// overrides, it would have nothing to discriminate. The bias the SYSTEM applies is a
// Modifier in the ledger and was never an override.
type OverrideOrigin string

const (
	OriginPlayer OverrideOrigin = "player" // the value the player sent
	OriginSystem OverrideOrigin = "system" // the value the engine computed
)

// OverriddenValue is what a master's edit DISPLACED — never the edit itself.
//
// The action already carries the new value; storing both would be duplication that diverges.
// The purpose is not to log what the master did, it is to not lose what the player sent and
// what the system calculated, so the original can be reconstructed by reading backwards from
// the corresponding row of actions.
//
// ONE ROW PER FIELD, holding the ORIGINAL — not one per edit. Two good properties fall out:
// reverting is free (edit back and nothing is stored at all), and a mistake plus its fix
// leave no noise behind.
type OverriddenValue struct {
	ActionID   uuid.UUID
	Field      string
	Origin     OverrideOrigin
	MasterUUID uuid.UUID
	At         time.Time
	// Original is the displaced value in its domain shape — an int, a []action.Skill, a set
	// of target UUIDs. The gateway marshals it to JSONB; the format genuinely varies and
	// nobody is going to query inside it.
	Original any
}
