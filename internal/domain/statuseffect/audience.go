package statuseffect

import "github.com/google/uuid"

// Audience é a lista de quem vê um efeito (comportamento na Task 4).
type Audience struct {
	Everyone bool
	Players  []uuid.UUID
}
