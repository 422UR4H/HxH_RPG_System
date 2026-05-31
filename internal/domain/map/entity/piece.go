package entity

type SquareCoord struct {
	Kind string `json:"kind"` // "square"
	Col  int    `json:"col"`
	Row  int    `json:"row"`
}

type HexCoord struct {
	Kind string `json:"kind"` // "hex"
	Q    int    `json:"q"`
	R    int    `json:"r"`
}

type PieceCoord struct {
	Slot any     `json:"slot"` // SquareCoord | HexCoord — serialised as-is
	Z    float64 `json:"z"`
}

type Piece struct {
	ID          string     `json:"id"`
	CharacterID string     `json:"character_id"`
	Coord       PieceCoord `json:"coord"`
	Visible     bool       `json:"visible"`
}
