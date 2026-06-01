package service

import (
	"errors"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
)

var (
	ErrEmptyName        = errors.New("map name cannot be empty")
	ErrInvalidCellSize  = errors.New("cell_size must be > 0")
	ErrInvalidCols      = errors.New("cols must be > 0")
	ErrInvalidRows      = errors.New("rows must be > 0")
	ErrInvalidSkewRatio = errors.New("skew_ratio must be in [0, 1]")
)

func ValidateMap(name string, grid entity.GridShape) error {
	if name == "" {
		return ErrEmptyName
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
