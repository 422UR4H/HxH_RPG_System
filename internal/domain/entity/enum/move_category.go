package enum

type MoveCategory string

const (
	Shift    MoveCategory = "Shift" // Break
	Dash     MoveCategory = "Dash"  // Accelerate
	Back     MoveCategory = "Back"  // Cait
	Roll     MoveCategory = "Roll"
	Slide    MoveCategory = "Slide" // Sneak
	Jump     MoveCategory = "Jump"
	FlatJump MoveCategory = "FlatJump"
)

// RollsSpeed reports whether this movement's speed is ROLLED, as opposed to taken passively.
//
// Shift is the controlled step: Brake measures the ability to STOP, and the character takes
// the dice set's average instead of rolling — MatchSession.rollActionDice explicitly drops no
// die for it and deriveSpeeds derives it as a passive. Every other category accelerates and
// rolls. This mirrors exactly the branch those two take rather than inventing a rule for the
// five categories nobody has decided yet: Back, Roll, Slide, Jump and FlatJump are refused at
// the WS boundary (moveSpeedSkill), so they cannot reach any caller of this.
//
// It is the discriminator of WHEN a reaction's displacement happens (design §4.1: movement
// that depends on no test displaces at the opening; movement with a difficulty to clear shows
// only the intention, and the server decides at the close). A reaction always has a
// difficulty — the action being moved against whoever reacts, i.e. the attacker's hit — but a
// movement that rolls nothing has no reading to put against it, so a Shift escape steps the
// moment it gets the floor and a Dash escape waits for the turn to close. See
// Room.applyMove and Room.applyClosedEscapes.
//
// ⚠️ It says nothing about an ACTION's move. An action has no CD coming at it, so its
// displacement is not conditioned on anything and happens at the opening whatever the
// category — the position cannot wait, because the reactions that follow depend on it.
func (m MoveCategory) RollsSpeed() bool { return m != Shift }
