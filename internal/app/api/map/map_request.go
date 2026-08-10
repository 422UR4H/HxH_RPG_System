// internal/app/api/map/map_request.go
package mapapi

import (
	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
)

type GridShapeRequest struct {
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

type BgImageRequest struct {
	URL      string  `json:"url"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
	Rotation float64 `json:"rotation"`
	Opacity  float64 `json:"opacity"`
}

type PieceCoordRequest struct {
	Slot any     `json:"slot"` // SquareCoord | HexCoord — serialised as-is
	Z    float64 `json:"z"`
}

type PieceRequest struct {
	ID          string            `json:"id"`
	CharacterID string            `json:"characterId"`
	Coord       PieceCoordRequest `json:"coord"`
	Visible     bool              `json:"visible"`
}

type WallSegmentRequest struct {
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

func toEntityGridShape(g GridShapeRequest) entity.GridShape {
	return entity.GridShape{
		Kind:      entity.GridKind(g.Kind),
		Cols:      g.Cols,
		Rows:      g.Rows,
		CellSize:  g.CellSize,
		SkewRatio: g.SkewRatio,
		Rotation:  g.Rotation,
		Color:     g.Color,
		Opacity:   g.Opacity,
		LineStyle: entity.LineStyle(g.LineStyle),
	}
}

func toEntityBgImage(b BgImageRequest) entity.BgImage {
	return entity.BgImage{
		URL:      b.URL,
		X:        b.X,
		Y:        b.Y,
		Width:    b.Width,
		Height:   b.Height,
		Rotation: b.Rotation,
		Opacity:  b.Opacity,
	}
}

func toEntityPiece(p PieceRequest) entity.Piece {
	return entity.Piece{
		ID:          p.ID,
		CharacterID: p.CharacterID,
		Coord: entity.PieceCoord{
			Slot: p.Coord.Slot,
			Z:    p.Coord.Z,
		},
		Visible: p.Visible,
	}
}

func toEntityWallSegment(w WallSegmentRequest) entity.WallSegment {
	seg := entity.WallSegment{
		ID:         w.ID,
		P1:         w.P1,
		P2:         w.P2,
		WallType:   entity.WallType(w.WallType),
		Material:   entity.WallMaterial(w.Material),
		Move:       w.Move,
		Sense:      entity.SenseKind(w.Sense),
		Direction:  entity.WallDirection(w.Direction),
		Open:       w.Open,
		Locked:     w.Locked,
		HP:         w.HP,
		MaxHP:      w.MaxHP,
		Resistance: w.Resistance,
		Destroyed:  w.Destroyed,
		Revealed:   w.Revealed,
	}
	if w.DoorSubtype != nil {
		d := entity.DoorSubtype(*w.DoorSubtype)
		seg.DoorSubtype = &d
	}
	if w.WindowSubtype != nil {
		wi := entity.WindowSubtype(*w.WindowSubtype)
		seg.WindowSubtype = &wi
	}
	return seg
}
