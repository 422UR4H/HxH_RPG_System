package turn_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
)

func TestSentinelErrors_ReadInPortuguese(t *testing.T) {
	if got, want := turn.ErrTurnIsEmpty.Error(), "O turno atual não tem ações."; got != want {
		t.Errorf("ErrTurnIsEmpty = %q, want %q", got, want)
	}
}
