package match_test

import (
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
)

func TestNewGameEvent(t *testing.T) {
	t.Run("with categories", func(t *testing.T) {
		categories := []enum.GameEventCategory{enum.Death, enum.Achievement}
		desc := "A character died"
		event := match.NewGameEvent(categories, "Character Death", &desc, nil)

		if event == nil {
			t.Fatal("NewGameEvent returned nil")
		}
	})

	t.Run("nil categories defaults to Other", func(t *testing.T) {
		event := match.NewGameEvent(nil, "Some event", nil, nil)
		if event == nil {
			t.Fatal("NewGameEvent returned nil")
		}
	})

	t.Run("with date change", func(t *testing.T) {
		newDate := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
		categories := []enum.GameEventCategory{enum.DateChange}
		event := match.NewGameEvent(categories, "Time Skip", nil, &newDate)

		if event == nil {
			t.Fatal("NewGameEvent returned nil")
		}
	})
}
