package scenario_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/application/scenario"
	"github.com/422UR4H/HxH_RPG_System/internal/application/testutil"
	"github.com/google/uuid"
)

// Emoji simples conta 1 caractere (não 2 em UTF-16 nem 4 em bytes).
func TestCreateScenario_EmojiCountsAsOneCharacter(t *testing.T) {
	run := func(n int) error {
		_, err := scenario.NewCreateScenarioUC(&testutil.MockScenarioRepo{}).CreateScenario(
			context.Background(),
			&scenario.CreateScenarioInput{UserUUID: uuid.New(), Name: strings.Repeat("😀", n)})
		return err
	}
	if err := run(32); err != nil {
		t.Fatalf("32 emojis must be accepted: %v", err)
	}
	if err := run(33); !errors.Is(err, scenario.ErrMaxNameLength) {
		t.Fatalf("33 emojis: got %v", err)
	}
}
