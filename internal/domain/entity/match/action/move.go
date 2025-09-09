package action

import "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"

type Move struct {
	Category   enum.MoveCategory
	Position   [3]int // x, y, z
	Speed      *RollContext
	Charge     *RollContext
	FinalSpeed int
}
