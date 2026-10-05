package action

import "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"

type Move struct {
	Category enum.MoveCategory
	// From is the actor's OWN PIECE POSITION on the server's board at enqueue time — never
	// the client's (B6, spec §4.3 "B5, B6 e B10"). room.go's enqueue_action arm derives it
	// from r.pieceSlotOf and writes it here; buildAction never sets it, and whatever the
	// client sent under "from" is parsed and discarded. nil means the actor has no piece on
	// the board, so there was nothing to check and nothing to report.
	//
	// Coordinate convention: [a, b, z], where (a, b) is (col, row) on a square grid or (q, r)
	// axial on a hex grid, and z is not read by the server — applyMove preserves the piece's
	// own virtual height instead (see its own doc).
	From       *[3]int
	Position   [3]int // [a, b, z] — same convention as From
	Speed      *RollCheck
	Charge     *RollCheck
	FinalSpeed int
}
