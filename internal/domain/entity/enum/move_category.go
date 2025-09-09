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
