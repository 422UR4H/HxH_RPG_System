package action_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
)

func TestReactionKindFrom_UnknownKindReadsInPortuguese(t *testing.T) {
	_, err := action.ReactionKindFrom("nope")
	if err == nil || err.Error() != "Esse tipo de reação não existe." {
		t.Errorf("err = %v", err)
	}
}
