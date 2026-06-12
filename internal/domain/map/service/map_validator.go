package service

import (
	"errors"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
)

var (
	ErrNameTooShort      = errors.New("map name must be at least 3 characters")
	ErrInvalidCellSize   = errors.New("cell_size must be > 0")
	ErrInvalidCols       = errors.New("cols must be > 0")
	ErrInvalidRows       = errors.New("rows must be > 0")
	ErrInvalidSkewRatio  = errors.New("skew_ratio must be in [0, 1]")
	ErrWallSameEndpoints = errors.New("wall p1 and p2 must be different points")
	ErrWallInvalidType   = errors.New("invalid wall_type")
	ErrWallNegativeHP    = errors.New("wall hp must be >= 0")
	ErrWallOutOfBounds   = errors.New("wall segment out of grid bounds")
)

func ValidateMap(name string, grid entity.GridShape) error {
	if len(name) < 3 {
		return ErrNameTooShort
	}
	if grid.CellSize <= 0 {
		return ErrInvalidCellSize
	}
	if grid.Cols <= 0 {
		return ErrInvalidCols
	}
	if grid.Rows <= 0 {
		return ErrInvalidRows
	}
	if grid.SkewRatio < 0 || grid.SkewRatio > 1 {
		return ErrInvalidSkewRatio
	}
	return nil
}

func ValidateWallSegments(walls []entity.WallSegment, grid entity.GridShape) error {
	maxX := float64(grid.Cols) * grid.CellSize
	maxY := float64(grid.Rows) * grid.CellSize
	for _, w := range walls {
		if w.P1 == w.P2 {
			return ErrWallSameEndpoints
		}
		switch w.WallType {
		case entity.WallTypeWall, entity.WallTypeDoor, entity.WallTypeWindow,
			entity.WallTypeSecretDoor, entity.WallTypeTerrain:
		default:
			return ErrWallInvalidType
		}
		if w.HP < 0 {
			return ErrWallNegativeHP
		}
		for _, p := range [][2]float64{w.P1, w.P2} {
			if p[0] < 0 || p[0] > maxX || p[1] < 0 || p[1] > maxY {
				return ErrWallOutOfBounds
			}
		}
	}
	return nil
}
