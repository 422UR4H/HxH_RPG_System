package ability_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/ability"
)

func TestSentinelErrors_ReadInPortuguese(t *testing.T) {
	if got, want := ability.ErrAbilityNotFound.Error(), "Aptidão não encontrada."; got != want {
		t.Errorf("ErrAbilityNotFound = %q, want %q", got, want)
	}
}
