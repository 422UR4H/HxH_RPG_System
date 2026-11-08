package action

import "github.com/google/uuid"

// TypeName is the action's type as actions.type stores it, derived from which payload field is
// set — the one discriminator of an action's type the system has (the combat summary reports it
// as actionKind).
//
// A reaction is never classified this way: its kind is declared, not inferred, and inferring
// it from the shape is exactly what ReactionKind exists to stop — the three escapes carry the
// same fields. The row says "reaction" and reaction_kind says which one.
func (a *Action) TypeName() string {
	if a.ReactToID != uuid.Nil {
		return "reaction"
	}
	switch {
	case a.Attack != nil:
		return "attack"
	case a.Move != nil:
		return "move"
	case a.Defense != nil:
		return "defense"
	case a.Dodge != nil:
		return "dodge"
	case a.Feint != nil:
		return "feint"
	case a.Interact != nil:
		// Ahead of the skills row on purpose: a lockpick carries BOTH an Interact and a
		// Skills entry, and what it is is an interaction — the skill is how it is attempted.
		return "interact"
	case len(a.Skills) > 0:
		return "skill"
	default:
		return "unspecified"
	}
}
