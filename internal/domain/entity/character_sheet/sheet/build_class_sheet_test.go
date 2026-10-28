package sheet_test

import (
	"testing"

	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func classProfile() sheet.CharacterProfile {
	return sheet.CharacterProfile{NickName: "Class", Alignment: "Neutral"}
}

func TestBuildClassSheet_withSpiritualSkillsExp(t *testing.T) {
	profile := *cc.NewClassProfile(enum.Monk, "", "test monk", "")
	charClass := cc.NewCharacterClass(
		profile, nil,
		map[enum.SkillName]int{enum.Focus: 210, enum.Mop: 127},
		nil, nil, nil, nil, nil, nil,
	)

	cs, err := sheet.NewCharacterSheetFactory().BuildClassSheet(classProfile(), charClass)
	if err != nil {
		t.Fatalf("BuildClassSheet error: %v", err)
	}

	spiritSkills := cs.GetSpiritualSkills()
	if got := spiritSkills[enum.Focus].GetExpPoints(); got != 210 {
		t.Errorf("Focus exp = %d, want 210", got)
	}
	if got := spiritSkills[enum.Mop].GetExpPoints(); got != 127 {
		t.Errorf("Mop exp = %d, want 127", got)
	}
	spiritAttrs := cs.GetSpiritualAttributes()
	for _, name := range []enum.AttributeName{enum.Flame, enum.Conscience} {
		if got := spiritAttrs[name].GetPower(); got <= 0 {
			t.Errorf("%s power = %d, want > 0", name, got)
		}
	}
	if cs.GetAbilities()[enum.Spirituals].GetExpPoints() <= 0 {
		t.Error("expected Spirituals ability to receive exp from spiritual skills")
	}
}

func TestBuildClassSheet_hasNoOwner(t *testing.T) {
	charClass := cc.BuildSwordsman()
	if _, err := sheet.NewCharacterSheetFactory().BuildClassSheet(classProfile(), &charClass); err != nil {
		t.Fatalf("BuildClassSheet must not require an owner: %v", err)
	}
}

// Prova que a migração HalfSheet -> CharacterSheet não muda nada nas classes atuais.
func TestBuildClassSheet_matchesHalfSheetForAllClasses(t *testing.T) {
	factory := sheet.NewCharacterSheetFactory()
	classes := cc.NewCharacterClassFactory().Build()
	if len(classes) != 12 {
		t.Fatalf("expected 12 classes, got %d", len(classes))
	}

	for name, class := range classes {
		t.Run(name.String(), func(t *testing.T) {
			ref := class
			half, err := factory.BuildHalfSheet(classProfile(), &ref)
			if err != nil {
				t.Fatalf("BuildHalfSheet error: %v", err)
			}
			ref2 := class
			full, err := factory.BuildClassSheet(classProfile(), &ref2)
			if err != nil {
				t.Fatalf("BuildClassSheet error: %v", err)
			}

			for n, a := range half.GetAbilities() {
				b := full.GetAbilities()[n]
				if a.GetLevel() != b.GetLevel() || a.GetExpPoints() != b.GetExpPoints() ||
					a.GetBonus() != b.GetBonus() {
					t.Errorf("ability %s differs", n)
				}
			}
			if full.GetAbilities()[enum.Spirituals].GetExpPoints() != 0 {
				t.Error("Spirituals must have no exp for current classes")
			}
			if len(full.GetAbilities()) != len(half.GetAbilities())+1 {
				t.Errorf("full sheet should only add Spirituals ability")
			}

			if len(half.GetPhysicalAttributes()) != len(full.GetPhysicalAttributes()) {
				t.Error("physical attributes count differs")
			}
			for n, a := range half.GetPhysicalAttributes() {
				b := full.GetPhysicalAttributes()[n]
				if a.GetLevel() != b.GetLevel() || a.GetExpPoints() != b.GetExpPoints() ||
					a.GetPower() != b.GetPower() || a.GetPoints() != b.GetPoints() {
					t.Errorf("physical attribute %s differs", n)
				}
			}
			for n, a := range half.GetMentalAttributes() {
				b := full.GetMentalAttributes()[n]
				if a.GetLevel() != b.GetLevel() || a.GetExpPoints() != b.GetExpPoints() ||
					a.GetPower() != b.GetPower() || a.GetPoints() != b.GetPoints() {
					t.Errorf("mental attribute %s differs", n)
				}
			}
			for n, a := range half.GetPhysicalSkills() {
				b := full.GetPhysicalSkills()[n]
				if a.GetLevel() != b.GetLevel() || a.GetExpPoints() != b.GetExpPoints() ||
					a.GetValueForTest() != b.GetValueForTest() {
					t.Errorf("physical skill %s differs", n)
				}
			}
			for n, a := range half.GetMentalSkills() {
				b := full.GetMentalSkills()[n]
				if a.GetLevel() != b.GetLevel() || a.GetExpPoints() != b.GetExpPoints() ||
					a.GetValueForTest() != b.GetValueForTest() {
					t.Errorf("mental skill %s differs", n)
				}
			}
			for n, s := range full.GetSpiritualSkills() {
				if s.GetExpPoints() != 0 {
					t.Errorf("spiritual skill %s should have no exp, got %d", n, s.GetExpPoints())
				}
			}
			for n, p := range half.GetCommonProficiencies() {
				q := full.GetCommonProficiencies()[n]
				if p.GetLevel() != q.GetLevel() || p.GetExpPoints() != q.GetExpPoints() {
					t.Errorf("proficiency %s differs", n)
				}
			}
			if len(half.GetCommonProficiencies()) != len(full.GetCommonProficiencies()) {
				t.Error("common proficiencies count differs")
			}
			if len(half.GetJointProficiencies()) != len(full.GetJointProficiencies()) {
				t.Error("joint proficiencies count differs")
			}
			if len(half.GetPhysJointSkills()) != len(full.GetPhysJointSkills()) {
				t.Error("joint skills count differs")
			}
		})
	}
}
