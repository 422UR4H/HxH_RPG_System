package action

import "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"

type Move struct {
	Category   enum.MoveCategory
	From       [3]int // source grid position [col, row, z]; zero = not provided
	Position   [3]int // x, y, z
	Speed      *RollCheck
	Charge     *RollCheck
	FinalSpeed int
}
