package sheet_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/sheet"
	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func TestNewCharacterClassResponse_distributionLevels(t *testing.T) {
	profsAllowed := []enum.WeaponName{enum.Dagger, enum.Scimitar}
	dist := &cc.Distribution{
		ProficiencyPoints:    []int{210, 127},
		ProficienciesAllowed: profsAllowed,
	}
	profile := *cc.NewClassProfile(enum.Mercenary, "", "test mercenary", "")
	charClass := *cc.NewCharacterClass(profile, dist, nil, nil, nil, nil, nil, nil, nil)
	hs := buildTestHalfSheet(t)

	resp := sheet.NewCharacterClassResponse(hs, charClass)

	if resp.Distribution == nil {
		t.Fatal("expected non-nil distribution")
	}
	if len(resp.Distribution.ProficiencyPoints) != 2 {
		t.Fatalf("expected 2 proficiency_points, got %d", len(resp.Distribution.ProficiencyPoints))
	}

	wantExp := []int{210, 127}
	for i, pt := range resp.Distribution.ProficiencyPoints {
		if pt.Exp != wantExp[i] {
			t.Errorf("point[%d].Exp = %d, want %d", i, pt.Exp, wantExp[i])
		}
		if pt.Level <= 0 {
			t.Errorf("point[%d].Level = %d, want > 0 (level must be computed)", i, pt.Level)
		}
	}
	// maior XP deve dar nível maior ou igual
	if resp.Distribution.ProficiencyPoints[0].Level < resp.Distribution.ProficiencyPoints[1].Level {
		t.Errorf("expected first point (210 XP) to have level >= second point (127 XP)")
	}
	if len(resp.Distribution.ProficienciesAllowed) != 2 {
		t.Fatalf("expected 2 proficiencies_allowed, got %d", len(resp.Distribution.ProficienciesAllowed))
	}
}

func TestNewCharacterClassResponse_nilDistribution(t *testing.T) {
	profile := *cc.NewClassProfile(enum.Samurai, "", "test samurai", "")
	charClass := *cc.NewCharacterClass(profile, nil, nil, nil, nil, nil, nil, nil, nil)
	hs := buildTestHalfSheet(t)

	resp := sheet.NewCharacterClassResponse(hs, charClass)

	if resp.Distribution != nil {
		t.Errorf("expected nil distribution for class without distribution, got %+v", resp.Distribution)
	}
}
