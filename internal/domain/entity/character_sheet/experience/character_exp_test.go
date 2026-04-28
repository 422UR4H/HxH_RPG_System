package experience_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
)

func newTestCharacterExp() *experience.CharacterExp {
	table := experience.NewDefaultExpTable()
	exp := experience.NewExperience(table)
	return experience.NewCharacterExp(*exp)
}

func TestCharacterExp_InitialState(t *testing.T) {
	ce := newTestCharacterExp()

	if ce.GetCharacterPoints() != 0 {
		t.Errorf("initial character points: got %d, want 0", ce.GetCharacterPoints())
	}
	if ce.GetLevel() != 0 {
		t.Errorf("initial level: got %d, want 0", ce.GetLevel())
	}
	if ce.GetExpPoints() != 0 {
		t.Errorf("initial exp points: got %d, want 0", ce.GetExpPoints())
	}
	if ce.GetCurrentExp() != 0 {
		t.Errorf("initial current exp: got %d, want 0", ce.GetCurrentExp())
	}
}

func TestCharacterExp_IncreaseCharacterPoints(t *testing.T) {
	ce := newTestCharacterExp()

	ce.IncreaseCharacterPoints(5)
	if ce.GetCharacterPoints() != 5 {
		t.Errorf("character points: got %d, want 5", ce.GetCharacterPoints())
	}

	ce.IncreaseCharacterPoints(3)
	if ce.GetCharacterPoints() != 8 {
		t.Errorf("character points after second increase: got %d, want 8", ce.GetCharacterPoints())
	}
}

func TestCharacterExp_EndCascadeUpgrade(t *testing.T) {
	ce := newTestCharacterExp()

	cascade := experience.NewUpgradeCascade(100)
	ce.EndCascadeUpgrade(cascade)

	if ce.GetExpPoints() != 100 {
		t.Errorf("exp points after cascade: got %d, want 100", ce.GetExpPoints())
	}
	if cascade.CharacterExp != ce {
		t.Error("cascade.CharacterExp should reference the CharacterExp instance")
	}
}

func TestCharacterExp_EndCascadeUpgrade_MultipleCalls(t *testing.T) {
	ce := newTestCharacterExp()

	cascade1 := experience.NewUpgradeCascade(50)
	ce.EndCascadeUpgrade(cascade1)

	cascade2 := experience.NewUpgradeCascade(50)
	ce.EndCascadeUpgrade(cascade2)

	if ce.GetExpPoints() != 100 {
		t.Errorf("exp points after two cascades: got %d, want 100", ce.GetExpPoints())
	}
}

func TestCharacterExp_GetNextLvlBaseExp(t *testing.T) {
	ce := newTestCharacterExp()

	got := ce.GetNextLvlBaseExp()
	if got <= 0 {
		t.Errorf("next lvl base exp should be positive, got %d", got)
	}
}

func TestCharacterExp_GetNextLvlAggregateExp(t *testing.T) {
	ce := newTestCharacterExp()

	got := ce.GetNextLvlAggregateExp()
	if got <= 0 {
		t.Errorf("next lvl aggregate exp should be positive, got %d", got)
	}
}
