package service_test

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"testing"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/map/service"
)

const goldenTol = 1e-9

type goldenGrid struct {
	Kind      string  `json:"kind"`
	Cols      int     `json:"cols"`
	Rows      int     `json:"rows"`
	CellSize  float64 `json:"cellSize"`
	SkewRatio float64 `json:"skewRatio"`
	Rotation  float64 `json:"rotation"`
}

func (g goldenGrid) shape() entity.GridShape {
	return entity.GridShape{
		Kind:      entity.GridKind(g.Kind),
		Cols:      g.Cols,
		Rows:      g.Rows,
		CellSize:  g.CellSize,
		SkewRatio: g.SkewRatio,
		Rotation:  g.Rotation,
	}
}

type goldenFile struct {
	Grids       map[string]goldenGrid `json:"grids"`
	SlotCenters []struct {
		Grid string  `json:"grid"`
		Slot [2]int  `json:"slot"`
		X    float64 `json:"x"`
		Y    float64 `json:"y"`
	} `json:"slotCenters"`
	LocalToSlot []struct {
		Grid string  `json:"grid"`
		X    float64 `json:"x"`
		Y    float64 `json:"y"`
		Slot [2]int  `json:"slot"`
	} `json:"localToSlot"`
	Bounds []struct {
		Grid string  `json:"grid"`
		MinX float64 `json:"minX"`
		MinY float64 `json:"minY"`
		MaxX float64 `json:"maxX"`
		MaxY float64 `json:"maxY"`
	} `json:"bounds"`
	SlotInGrid []struct {
		Grid   string `json:"grid"`
		Slot   [2]int `json:"slot"`
		InGrid bool   `json:"inGrid"`
	} `json:"slotInGrid"`
	WallContactSlots []struct {
		Grid  string     `json:"grid"`
		P1    [2]float64 `json:"p1"`
		P2    [2]float64 `json:"p2"`
		Slots [][2]int   `json:"slots"`
	} `json:"wallContactSlots"`
	PathBlocked []struct {
		Grid string `json:"grid"`
		From [2]int `json:"from"`
		To   [2]int `json:"to"`
		Wall struct {
			P1        [2]float64 `json:"p1"`
			P2        [2]float64 `json:"p2"`
			Direction string     `json:"direction"`
		} `json:"wall"`
		Blocked bool `json:"blocked"`
	} `json:"pathBlocked"`
}

func loadGolden(t *testing.T) goldenFile {
	t.Helper()
	raw, err := os.ReadFile("testdata/grid_geometry_golden.json")
	if err != nil {
		t.Fatalf("lendo golden: %v", err)
	}
	var f goldenFile
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("decodificando golden: %v", err)
	}
	// Seção vazia faria o teste correspondente passar sem checar nada.
	for name, n := range map[string]int{
		"grids": len(f.Grids), "slotCenters": len(f.SlotCenters), "localToSlot": len(f.LocalToSlot),
		"bounds": len(f.Bounds), "slotInGrid": len(f.SlotInGrid),
		"wallContactSlots": len(f.WallContactSlots), "pathBlocked": len(f.PathBlocked),
	} {
		if n == 0 {
			t.Fatalf("golden: seção %q vazia", name)
		}
	}
	return f
}

func (f goldenFile) grid(t *testing.T, name string) entity.GridShape {
	t.Helper()
	g, ok := f.Grids[name]
	if !ok {
		t.Fatalf("grid %q não existe no golden", name)
	}
	return g.shape()
}

func near(a, b float64) bool { return math.Abs(a-b) <= goldenTol }

// TestGridGeometryGolden confere a geometria do back contra o golden
// compartilhado com o front (mesmo arquivo, byte a byte).
func TestGridGeometryGolden(t *testing.T) {
	f := loadGolden(t)

	t.Run("slotCenters", func(t *testing.T) {
		for _, c := range f.SlotCenters {
			name := fmt.Sprintf("%s/%v", c.Grid, c.Slot)
			t.Run(name, func(t *testing.T) {
				x, y := service.SlotCenterToLocal(c.Slot[0], c.Slot[1], f.grid(t, c.Grid))
				if !near(x, c.X) || !near(y, c.Y) {
					t.Errorf("SlotCenterToLocal(%v) = (%v, %v), esperado (%v, %v)", c.Slot, x, y, c.X, c.Y)
				}
			})
		}
	})

	t.Run("localToSlot", func(t *testing.T) {
		for _, c := range f.LocalToSlot {
			name := fmt.Sprintf("%s/%v,%v", c.Grid, c.X, c.Y)
			t.Run(name, func(t *testing.T) {
				a, b := service.LocalToSlot(c.X, c.Y, f.grid(t, c.Grid))
				if got := [2]int{a, b}; got != c.Slot {
					t.Errorf("LocalToSlot(%v, %v) = %v, esperado %v", c.X, c.Y, got, c.Slot)
				}
			})
		}
	})

	t.Run("bounds", func(t *testing.T) {
		for _, c := range f.Bounds {
			t.Run(c.Grid, func(t *testing.T) {
				minX, minY, maxX, maxY := service.GridLocalBounds(f.grid(t, c.Grid))
				if !near(minX, c.MinX) || !near(minY, c.MinY) || !near(maxX, c.MaxX) || !near(maxY, c.MaxY) {
					t.Errorf("GridLocalBounds = (%v, %v, %v, %v), esperado (%v, %v, %v, %v)",
						minX, minY, maxX, maxY, c.MinX, c.MinY, c.MaxX, c.MaxY)
				}
			})
		}
	})

	t.Run("slotInGrid", func(t *testing.T) {
		for _, c := range f.SlotInGrid {
			name := fmt.Sprintf("%s/%v", c.Grid, c.Slot)
			t.Run(name, func(t *testing.T) {
				if got := service.SlotInGrid(c.Slot[0], c.Slot[1], f.grid(t, c.Grid)); got != c.InGrid {
					t.Errorf("SlotInGrid(%v) = %v, esperado %v", c.Slot, got, c.InGrid)
				}
			})
		}
	})

	t.Run("wallContactSlots", func(t *testing.T) {
		for _, c := range f.WallContactSlots {
			name := fmt.Sprintf("%s/%v-%v", c.Grid, c.P1, c.P2)
			t.Run(name, func(t *testing.T) {
				w := entity.WallSegment{P1: c.P1, P2: c.P2}
				got := service.WallContactSlots(w, f.grid(t, c.Grid))
				sort.Slice(got, func(i, j int) bool {
					if got[i][0] != got[j][0] {
						return got[i][0] < got[j][0]
					}
					return got[i][1] < got[j][1]
				})
				if fmt.Sprint(got) != fmt.Sprint(c.Slots) {
					t.Errorf("WallContactSlots = %v, esperado %v", got, c.Slots)
				}
			})
		}
	})

	t.Run("pathBlocked", func(t *testing.T) {
		for _, c := range f.PathBlocked {
			name := fmt.Sprintf("%s/%v-%v/%v-%v/%s", c.Grid, c.From, c.To, c.Wall.P1, c.Wall.P2, c.Wall.Direction)
			t.Run(name, func(t *testing.T) {
				g := f.grid(t, c.Grid)
				fx, fy := service.SlotCenterToLocal(c.From[0], c.From[1], g)
				tx, ty := service.SlotCenterToLocal(c.To[0], c.To[1], g)
				walls := []entity.WallSegment{{
					P1:        c.Wall.P1,
					P2:        c.Wall.P2,
					Move:      true,
					Direction: entity.WallDirection(c.Wall.Direction),
				}}
				got := service.IsPathBlocked([2]float64{fx, fy}, [2]float64{tx, ty}, walls)
				if got != c.Blocked {
					t.Errorf("IsPathBlocked = %v, esperado %v", got, c.Blocked)
				}
			})
		}
	})
}
