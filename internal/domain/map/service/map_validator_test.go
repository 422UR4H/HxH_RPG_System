package service_test

import (
	"errors"
	"testing"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/map/service"
)

func validGrid() entity.GridShape {
	return entity.DefaultGrid()
}

func TestValidateMap_ValidMap(t *testing.T) {
	err := service.ValidateMap("Forest", validGrid())
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidateMap_EmptyName(t *testing.T) {
	err := service.ValidateMap("", validGrid())
	if !errors.Is(err, service.ErrEmptyName) {
		t.Errorf("expected ErrEmptyName, got %v", err)
	}
}

func TestValidateMap_InvalidCellSize(t *testing.T) {
	g := validGrid()
	g.CellSize = 0
	err := service.ValidateMap("Forest", g)
	if !errors.Is(err, service.ErrInvalidCellSize) {
		t.Errorf("expected ErrInvalidCellSize, got %v", err)
	}
}

func TestValidateMap_InvalidCols(t *testing.T) {
	g := validGrid()
	g.Cols = 0
	err := service.ValidateMap("Forest", g)
	if !errors.Is(err, service.ErrInvalidCols) {
		t.Errorf("expected ErrInvalidCols, got %v", err)
	}
}

func TestValidateMap_InvalidRows(t *testing.T) {
	g := validGrid()
	g.Rows = 0
	err := service.ValidateMap("Forest", g)
	if !errors.Is(err, service.ErrInvalidRows) {
		t.Errorf("expected ErrInvalidRows, got %v", err)
	}
}

func TestValidateMap_SkewRatioOutOfRange(t *testing.T) {
	g := validGrid()
	g.SkewRatio = 1.5
	err := service.ValidateMap("Forest", g)
	if !errors.Is(err, service.ErrInvalidSkewRatio) {
		t.Errorf("expected ErrInvalidSkewRatio, got %v", err)
	}
}
