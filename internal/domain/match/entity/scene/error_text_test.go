package scene_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
)

func TestSentinelErrors_ReadInPortuguese(t *testing.T) {
	if got, want := scene.ErrSceneIsFinished.Error(), "A cena já terminou."; got != want {
		t.Errorf("ErrSceneIsFinished = %q, want %q", got, want)
	}
}
