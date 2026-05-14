package turn_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/google/uuid"
)

func TestTurn_GetID(t *testing.T) {
	a := action.Action{}
	tRn := turn.NewTurn(a)
	id := tRn.GetID()
	if id == uuid.Nil {
		t.Error("expected non-nil UUID")
	}
}

func TestTurn_AddMasterAction(t *testing.T) {
	a := action.Action{}
	tRn := turn.NewTurn(a)
	ma := action.NewMasterAction()
	tRn.AddMasterAction(*ma)
	got := tRn.GetMasterActions()
	if len(got) != 1 {
		t.Fatalf("expected 1 master action, got %d", len(got))
	}
}
