package scenario_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/application/scenario"
)

func TestSentinelErrors_ReadInPortuguese(t *testing.T) {
	if got, want := scenario.ErrScenarioNotFound.Error(), "Cenário não encontrado."; got != want {
		t.Errorf("ErrScenarioNotFound = %q, want %q", got, want)
	}
}
