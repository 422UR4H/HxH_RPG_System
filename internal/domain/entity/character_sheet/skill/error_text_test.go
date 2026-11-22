package skill_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/skill"
)

func TestSentinelErrors_ReadInPortuguese(t *testing.T) {
	if got, want := skill.ErrSkillsAlreadyInitialized.Error(), "As perícias já foram iniciadas."; got != want {
		t.Errorf("ErrSkillsAlreadyInitialized = %q, want %q", got, want)
	}
}
