package action

// Dodge is the roll behind a dodge reaction. WHICH dodge — active, closed, escape, closed
// escape — is ReactionKind's business, on the Action itself.
//
// It used to carry an enum category of {Evasive, Close, Scape}. That was the same axis as
// ReactionKind and strictly less expressive: Scape alone covered all THREE escapes, which is
// exactly the distinction the cost table needs. Keeping both would be redundant state that can
// disagree with itself, and state like that always does, sooner or later.
type Dodge struct {
	RollCheck
}
