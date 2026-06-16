package fog

// CellCoord is grid-agnostic: (A,B) = (Col,Row) for square, (Q,R) for hex axial.
type CellCoord struct {
	A int
	B int
}
