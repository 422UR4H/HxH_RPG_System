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
	if !errors.Is(err, service.ErrNameTooShort) {
		t.Errorf("expected ErrNameTooShort, got %v", err)
	}
}

func TestValidateMap_NameTooShort(t *testing.T) {
	err := service.ValidateMap("Ab", validGrid())
	if !errors.Is(err, service.ErrNameTooShort) {
		t.Errorf("expected ErrNameTooShort, got %v", err)
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

func validWallSegment() entity.WallSegment {
	return entity.WallSegment{
		ID:        "00000000-0000-0000-0000-000000000001",
		P1:        [2]float64{0, 0},
		P2:        [2]float64{1, 0},
		WallType:  entity.WallTypeWall,
		Material:  entity.WallMaterialStone,
		Move:      true,
		Sense:     entity.SenseFull,
		Direction: entity.WallDirectionBoth,
		HP:        100,
		MaxHP:     100,
		Resistance: 5,
	}
}

func TestValidateWallSegments_Empty(t *testing.T) {
	err := service.ValidateWallSegments([]entity.WallSegment{})
	if err != nil {
		t.Errorf("expected nil for empty walls, got %v", err)
	}
}

func TestValidateWallSegments_Valid(t *testing.T) {
	ws := validWallSegment()
	err := service.ValidateWallSegments([]entity.WallSegment{ws})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidateWallSegments_SameEndpoints(t *testing.T) {
	ws := validWallSegment()
	ws.P2 = ws.P1 // same point
	err := service.ValidateWallSegments([]entity.WallSegment{ws})
	if !errors.Is(err, service.ErrWallSameEndpoints) {
		t.Errorf("expected ErrWallSameEndpoints, got %v", err)
	}
}

func TestValidateWallSegments_InvalidWallType(t *testing.T) {
	ws := validWallSegment()
	ws.WallType = "invalid"
	err := service.ValidateWallSegments([]entity.WallSegment{ws})
	if !errors.Is(err, service.ErrWallInvalidType) {
		t.Errorf("expected ErrWallInvalidType, got %v", err)
	}
}

func TestValidateWallSegments_NegativeHP(t *testing.T) {
	ws := validWallSegment()
	ws.HP = -1
	err := service.ValidateWallSegments([]entity.WallSegment{ws})
	if !errors.Is(err, service.ErrWallNegativeHP) {
		t.Errorf("expected ErrWallNegativeHP, got %v", err)
	}
}
