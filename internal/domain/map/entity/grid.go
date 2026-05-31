package entity

type GridKind  string
type LineStyle string

const (
	GridKindSquare GridKind  = "square"
	GridKindHex    GridKind  = "hex"
	LineStyleSolid LineStyle = "solid"
	LineStyleDashed LineStyle = "dashed"
)

type GridShape struct {
	Kind      GridKind  `json:"kind"`
	Cols      int       `json:"cols"`
	Rows      int       `json:"rows"`
	CellSize  float64   `json:"cell_size"`
	SkewRatio float64   `json:"skew_ratio"`
	Rotation  float64   `json:"rotation"`
	Color     string    `json:"color"`
	Opacity   float64   `json:"opacity"`
	LineStyle LineStyle `json:"line_style"`
}

func DefaultGrid() GridShape {
	return GridShape{
		Kind:      GridKindSquare,
		Cols:      25,
		Rows:      25,
		CellSize:  64,
		SkewRatio: 1.0,
		Rotation:  0,
		Color:     "#ffffff",
		Opacity:   0.5,
		LineStyle: LineStyleSolid,
	}
}
