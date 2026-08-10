// internal/app/api/map/map_response.go
package mapapi

import (
	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/google/uuid"
)

type GridShapeResponse struct {
	Kind      string  `json:"kind"`
	Cols      int     `json:"cols"`
	Rows      int     `json:"rows"`
	CellSize  float64 `json:"cellSize"`
	SkewRatio float64 `json:"skewRatio"`
	Rotation  float64 `json:"rotation"`
	Color     string  `json:"color"`
	Opacity   float64 `json:"opacity"`
	LineStyle string  `json:"lineStyle"`
}

type BgImageResponse struct {
	URL      string  `json:"url"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
	Rotation float64 `json:"rotation"`
	Opacity  float64 `json:"opacity"`
}

type PieceCoordResponse struct {
	Slot any     `json:"slot"` // SquareCoord | HexCoord — serialised as-is
	Z    float64 `json:"z"`
}

type PieceResponse struct {
	ID          string             `json:"id"`
	CharacterID string             `json:"characterId"`
	Coord       PieceCoordResponse `json:"coord"`
	Visible     bool               `json:"visible"`
}

type WallSegmentResponse struct {
	ID            string     `json:"id"`
	P1            [2]float64 `json:"p1"`
	P2            [2]float64 `json:"p2"`
	WallType      string     `json:"wallType"`
	Material      string     `json:"material"`
	DoorSubtype   *string    `json:"doorSubtype,omitempty"`
	WindowSubtype *string    `json:"windowSubtype,omitempty"`
	Move          bool       `json:"move"`
	Sense         string     `json:"sense"`
	Direction     string     `json:"direction"`
	Open          bool       `json:"open"`
	Locked        bool       `json:"locked"`
	HP            int        `json:"hp"`
	MaxHP         int        `json:"maxHp"`
	Resistance    int        `json:"resistance"`
	Destroyed     bool       `json:"destroyed"`
	Revealed      bool       `json:"revealed"`
}

type DecorationResponse struct {
	ID       string  `json:"id"`
	URL      string  `json:"url"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
	Rotation float64 `json:"rotation"`
	ZOrder   int     `json:"zOrder"`
	Opacity  float64 `json:"opacity"`
}

type MapItemResponse struct {
	ID        string `json:"id"`
	ItemDefID string `json:"itemDefId"`
}

type MapResponse struct {
	ID          uuid.UUID             `json:"id"`
	CampaignID  uuid.UUID             `json:"campaignId"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Grid        GridShapeResponse     `json:"grid"`
	Bg          *BgImageResponse      `json:"bg"`
	Pieces      []PieceResponse       `json:"pieces"`
	Walls       []WallSegmentResponse `json:"walls"`
	Decorations []DecorationResponse  `json:"decorations"`
	Items       []MapItemResponse     `json:"items"`
	CreatedAt   string                `json:"createdAt"`
	UpdatedAt   string                `json:"updatedAt"`
}

func toBgImageResponse(bg entity.BgImage) BgImageResponse {
	return BgImageResponse{
		URL:      bg.URL,
		X:        bg.X,
		Y:        bg.Y,
		Width:    bg.Width,
		Height:   bg.Height,
		Rotation: bg.Rotation,
		Opacity:  bg.Opacity,
	}
}

func toPieceResponse(p entity.Piece) PieceResponse {
	return PieceResponse{
		ID:          p.ID,
		CharacterID: p.CharacterID,
		Coord: PieceCoordResponse{
			Slot: p.Coord.Slot,
			Z:    p.Coord.Z,
		},
		Visible: p.Visible,
	}
}

func toWallSegmentResponse(w entity.WallSegment) WallSegmentResponse {
	resp := WallSegmentResponse{
		ID:         w.ID,
		P1:         w.P1,
		P2:         w.P2,
		WallType:   string(w.WallType),
		Material:   string(w.Material),
		Move:       w.Move,
		Sense:      string(w.Sense),
		Direction:  string(w.Direction),
		Open:       w.Open,
		Locked:     w.Locked,
		HP:         w.HP,
		MaxHP:      w.MaxHP,
		Resistance: w.Resistance,
		Destroyed:  w.Destroyed,
		Revealed:   w.Revealed,
	}
	if w.DoorSubtype != nil {
		s := string(*w.DoorSubtype)
		resp.DoorSubtype = &s
	}
	if w.WindowSubtype != nil {
		s := string(*w.WindowSubtype)
		resp.WindowSubtype = &s
	}
	return resp
}

func toDecorationResponse(d entity.Decoration) DecorationResponse {
	return DecorationResponse{
		ID:       d.ID,
		URL:      d.URL,
		X:        d.X,
		Y:        d.Y,
		Width:    d.Width,
		Height:   d.Height,
		Rotation: d.Rotation,
		ZOrder:   d.ZOrder,
		Opacity:  d.Opacity,
	}
}

func toMapItemResponse(i entity.MapItem) MapItemResponse {
	return MapItemResponse{
		ID:        i.ID,
		ItemDefID: i.ItemDefID,
	}
}

func toMapResponse(m *entity.TacticalMap) MapResponse {
	pieces := m.Pieces
	if pieces == nil {
		pieces = []entity.Piece{}
	}
	walls := m.Walls
	if walls == nil {
		walls = []entity.WallSegment{}
	}
	decorations := m.Decorations
	if decorations == nil {
		decorations = []entity.Decoration{}
	}
	items := m.Items
	if items == nil {
		items = []entity.MapItem{}
	}

	pieceResponses := make([]PieceResponse, len(pieces))
	for i, p := range pieces {
		pieceResponses[i] = toPieceResponse(p)
	}
	wallResponses := make([]WallSegmentResponse, len(walls))
	for i, w := range walls {
		wallResponses[i] = toWallSegmentResponse(w)
	}
	decorationResponses := make([]DecorationResponse, len(decorations))
	for i, d := range decorations {
		decorationResponses[i] = toDecorationResponse(d)
	}
	itemResponses := make([]MapItemResponse, len(items))
	for i, it := range items {
		itemResponses[i] = toMapItemResponse(it)
	}

	var bg *BgImageResponse
	if m.Bg != nil {
		converted := toBgImageResponse(*m.Bg)
		bg = &converted
	}

	return MapResponse{
		ID:          m.ID,
		CampaignID:  m.CampaignID,
		Name:        m.Name,
		Description: m.Description,
		Grid: GridShapeResponse{
			Kind:      string(m.Grid.Kind),
			Cols:      m.Grid.Cols,
			Rows:      m.Grid.Rows,
			CellSize:  m.Grid.CellSize,
			SkewRatio: m.Grid.SkewRatio,
			Rotation:  m.Grid.Rotation,
			Color:     m.Grid.Color,
			Opacity:   m.Grid.Opacity,
			LineStyle: string(m.Grid.LineStyle),
		},
		Bg:          bg,
		Pieces:      pieceResponses,
		Walls:       wallResponses,
		Decorations: decorationResponses,
		Items:       itemResponses,
		CreatedAt:   m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   m.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
