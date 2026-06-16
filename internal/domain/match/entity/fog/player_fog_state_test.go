package fog

import (
	"testing"

	"github.com/google/uuid"
)

func TestAddExplored_ReturnsOnlyNewCells(t *testing.T) {
	s := NewPlayerFogState(uuid.New(), uuid.New(), uuid.New(), "square")

	delta := s.AddExplored([]CellCoord{{A: 0, B: 0}, {A: 1, B: 0}})
	if len(delta) != 2 {
		t.Fatalf("first add: want 2 new cells, got %d", len(delta))
	}

	delta = s.AddExplored([]CellCoord{{A: 1, B: 0}, {A: 2, B: 0}})
	if len(delta) != 1 {
		t.Fatalf("second add: want 1 new cell, got %d", len(delta))
	}
	if delta[0] != (CellCoord{A: 2, B: 0}) {
		t.Fatalf("want new cell {2,0}, got %+v", delta[0])
	}
	if len(s.ExploredCells) != 3 {
		t.Fatalf("want 3 total explored, got %d", len(s.ExploredCells))
	}
}
