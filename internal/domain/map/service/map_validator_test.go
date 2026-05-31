package service_test

import (
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
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestValidateMap_InvalidCellSize(t *testing.T) {
	g := validGrid()
	g.CellSize = 0
	err := service.ValidateMap("Forest", g)
	if err == nil {
		t.Error("expected error for cell_size=0")
	}
}

func TestValidateMap_InvalidCols(t *testing.T) {
	g := validGrid()
	g.Cols = 0
	err := service.ValidateMap("Forest", g)
	if err == nil {
		t.Error("expected error for cols=0")
	}
}

func TestValidateMap_InvalidRows(t *testing.T) {
	g := validGrid()
	g.Rows = 0
	err := service.ValidateMap("Forest", g)
	if err == nil {
		t.Error("expected error for rows=0")
	}
}

func TestValidateMap_SkewRatioOutOfRange(t *testing.T) {
	g := validGrid()
	g.SkewRatio = 1.5
	err := service.ValidateMap("Forest", g)
	if err == nil {
		t.Error("expected error for skew_ratio > 1")
	}
}
