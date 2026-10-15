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
	err := service.ValidateWallSegments([]entity.WallSegment{}, validGrid())
	if err != nil {
		t.Errorf("expected nil for empty walls, got %v", err)
	}
}

func TestValidateWallSegments_Valid(t *testing.T) {
	ws := validWallSegment()
	err := service.ValidateWallSegments([]entity.WallSegment{ws}, validGrid())
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidateWallSegments_SameEndpoints(t *testing.T) {
	ws := validWallSegment()
	ws.P2 = ws.P1 // same point
	err := service.ValidateWallSegments([]entity.WallSegment{ws}, validGrid())
	if !errors.Is(err, service.ErrWallSameEndpoints) {
		t.Errorf("expected ErrWallSameEndpoints, got %v", err)
	}
}

func TestValidateWallSegments_InvalidWallType(t *testing.T) {
	ws := validWallSegment()
	ws.WallType = "invalid"
	err := service.ValidateWallSegments([]entity.WallSegment{ws}, validGrid())
	if !errors.Is(err, service.ErrWallInvalidType) {
		t.Errorf("expected ErrWallInvalidType, got %v", err)
	}
}

func TestValidateWallSegments_NegativeHP(t *testing.T) {
	ws := validWallSegment()
	ws.HP = -1
	err := service.ValidateWallSegments([]entity.WallSegment{ws}, validGrid())
	if !errors.Is(err, service.ErrWallNegativeHP) {
		t.Errorf("expected ErrWallNegativeHP, got %v", err)
	}
}

func TestValidateWallSegments_OutOfBounds(t *testing.T) {
	g := validGrid()
	maxX := float64(g.Cols) * g.CellSize
	ws := validWallSegment()
	ws.P2 = [2]float64{maxX + 1, 0} // one pixel beyond grid boundary
	err := service.ValidateWallSegments([]entity.WallSegment{ws}, g)
	if !errors.Is(err, service.ErrWallOutOfBounds) {
		t.Errorf("expected ErrWallOutOfBounds, got %v", err)
	}
}

func TestValidateWallSegments_HexBoundsUseLocalOutline(t *testing.T) {
	g := entity.DefaultGrid()
	g.Kind = entity.GridKindHex
	g.Cols, g.Rows, g.CellSize = 4, 4, 32
	minX, minY, maxX, maxY := service.GridLocalBounds(g)
	if minX >= 0 || minY >= 0 || maxX <= float64(g.Cols)*g.CellSize {
		t.Fatalf("test premise broken: bounds %v %v %v %v", minX, minY, maxX, maxY)
	}

	tests := []struct {
		name    string
		p1, p2  [2]float64
		wantErr error
	}{
		{"negative x inside hex outline", [2]float64{minX + 1, 0}, [2]float64{0, 0}, nil},
		{"negative y inside hex outline", [2]float64{0, minY + 1}, [2]float64{0, 0}, nil},
		{"right of cols*cellSize inside outline", [2]float64{0, 0}, [2]float64{maxX - 1, 0}, nil},
		{"left of outline", [2]float64{minX - 1, 0}, [2]float64{0, 0}, service.ErrWallOutOfBounds},
		{"right of outline", [2]float64{0, 0}, [2]float64{maxX + 1, 0}, service.ErrWallOutOfBounds},
		{"below outline", [2]float64{0, 0}, [2]float64{0, maxY + 1}, service.ErrWallOutOfBounds},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := validWallSegment()
			ws.P1, ws.P2 = tt.p1, tt.p2
			err := service.ValidateWallSegments([]entity.WallSegment{ws}, g)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("want %v, got %v", tt.wantErr, err)
			}
		})
	}
}

// The numbers below are the left edge of hex (0,0) exactly as the front's
// JavaScript computes it: 64*Math.cos(150deg), 64*Math.sin(150deg) and the
// 210deg pair. The x of that vertex lands 1 ulp beyond minX = -hexW/2.
func TestValidateWallSegments_HexBorderVertexAsTheFrontComputesIt(t *testing.T) {
	g := entity.DefaultGrid()
	g.Kind = entity.GridKindHex
	g.Cols, g.Rows, g.CellSize = 10, 10, 64
	minX, _, _, _ := service.GridLocalBounds(g)

	t.Run("front border vertex is accepted", func(t *testing.T) {
		ws := validWallSegment()
		ws.P1 = [2]float64{-55.42562584220408, 31.999999999999996}
		ws.P2 = [2]float64{-55.42562584220407, -32.00000000000001}
		if err := service.ValidateWallSegments([]entity.WallSegment{ws}, g); err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("tolerance is not a visible slack", func(t *testing.T) {
		ws := validWallSegment()
		ws.P1 = [2]float64{minX - 0.5, 0}
		ws.P2 = [2]float64{0, 0}
		err := service.ValidateWallSegments([]entity.WallSegment{ws}, g)
		if !errors.Is(err, service.ErrWallOutOfBounds) {
			t.Errorf("expected ErrWallOutOfBounds, got %v", err)
		}
	})
}
