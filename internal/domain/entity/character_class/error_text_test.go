package characterclass_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
)

func TestSentinelErrors_ReadInPortuguese(t *testing.T) {
	if got, want := characterclass.ErrNoSkillDistribution.Error(), "A classe não tem distribuição de perícias."; got != want {
		t.Errorf("ErrNoSkillDistribution = %q, want %q", got, want)
	}
}
