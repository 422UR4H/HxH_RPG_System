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

// Os limites contam caracteres (como o VARCHAR do Postgres), não bytes.
func TestCreateScenario_LimitsCountCharacters(t *testing.T) {
	run := func(name, brief string) error {
		uc := scenario.NewCreateScenarioUC(&testutil.MockScenarioRepo{})
		_, err := uc.CreateScenario(context.Background(), &scenario.CreateScenarioInput{
			UserUUID: uuid.New(), Name: name, BriefDescription: brief,
		})
		return err
	}
	t.Run("name at limit with accent passes", func(t *testing.T) {
		if err := run(strings.Repeat("ã", 32), ""); err != nil {
			t.Fatalf("unexpected: %v", err)
		}
	})
	t.Run("name over limit fails", func(t *testing.T) {
		if err := run(strings.Repeat("ã", 33), ""); !errors.Is(err, scenario.ErrMaxNameLength) {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("name with 4 accented characters is too short", func(t *testing.T) {
		if err := run("ãããã", ""); !errors.Is(err, scenario.ErrMinNameLength) {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("brief at limit with accent passes", func(t *testing.T) {
		if err := run("Valid Name", strings.Repeat("ã", 64)); err != nil {
			t.Fatalf("unexpected: %v", err)
		}
	})
	t.Run("brief over limit fails", func(t *testing.T) {
		if err := run("Valid Name", strings.Repeat("ã", 65)); !errors.Is(err, scenario.ErrMaxBriefDescLength) {
			t.Fatalf("got %v", err)
		}
	})
}
