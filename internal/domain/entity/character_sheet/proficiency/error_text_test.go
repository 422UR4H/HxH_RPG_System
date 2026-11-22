package proficiency_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/proficiency"
)

func TestSentinelErrors_ReadInPortuguese(t *testing.T) {
	if got, want := proficiency.ErrProficiencyAlreadyInitialized.Error(), "As proficiências já foram iniciadas."; got != want {
		t.Errorf("ErrProficiencyAlreadyInitialized = %q, want %q", got, want)
	}
}
