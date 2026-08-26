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
// Composition rule: a row's Original is a snapshot taken at the moment of ITS OWN edit, not a
// pristine copy of whatever the player originally sent. A field edited more than once by
// different edit SURFACES can leave a later row's Original carrying an earlier row's edit
// already baked in, while still being stamped Origin: player.
//
// Concretely: if the master sets a condition on skill X (captured under
// "skill:X.condition", MatchSession.ApplyMasterAction's condition path) and later removes X
// from the list entirely (captured under "skills", MatchSession.applySkillEdit), the "skills"
// row's Original entry for X already carries that condition — a value the player never sent,
// yet Origin still reads "player" because captureOverride's origin parameter names where the
// DISPLACED value's shape came from (the action, not the edit history behind it).
//
// The player's true original is still recoverable, by composing the two rows rather than
// reading either alone: take the "skills" row's Original entry for X, then undo what the
// "skill:X.condition" row's own Original names for that field (nil, if the player had set no
// condition at all before the master's first edit). Nothing reads this table yet, so this is
// documented rather than restructured — see combat-engine.md § A edição do mestre and the
// migration comment on overridden_action_values.
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
