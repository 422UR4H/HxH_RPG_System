package characterclass_test

import (
	"testing"

	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func TestCharacterClassFactory_Build(t *testing.T) {
	factory := cc.NewCharacterClassFactory()
	classes := factory.Build()

	expectedClasses := []enum.CharacterClassName{
		enum.Swordsman, enum.Samurai, enum.Ninja, enum.Rogue,
		enum.Netrunner, enum.Pirate, enum.Mercenary, enum.Terrorist,
		enum.Monk, enum.Military, enum.Hunter, enum.WeaponsMaster,
	}

	if len(classes) != len(expectedClasses) {
		t.Fatalf("Build() produced %d classes, want %d", len(classes), len(expectedClasses))
	}

	for _, name := range expectedClasses {
		class, ok := classes[name]
		if !ok {
			t.Errorf("Build() missing class: %s", name)
			continue
		}
		if class.GetName() != name {
			t.Errorf("class %s: GetName() = %v", name, class.GetName())
		}
	}
}

func TestCharacterClassFactory_Build_ClassesHaveSkills(t *testing.T) {
	factory := cc.NewCharacterClassFactory()
	classes := factory.Build()

	for name, class := range classes {
		if len(class.SkillsExps) == 0 {
			t.Errorf("class %s has no skills", name)
		}
	}
}

func TestCharacterClassFactory_Build_DistributionClasses(t *testing.T) {
	factory := cc.NewCharacterClassFactory()
	classes := factory.Build()

	classesWithDist := []enum.CharacterClassName{
		enum.Ninja, enum.Mercenary, enum.Hunter,
	}
	for _, name := range classesWithDist {
		class := classes[name]
		if class.Distribution == nil {
			t.Errorf("class %s: expected non-nil Distribution", name)
		}
	}

	classesWithoutDist := []enum.CharacterClassName{
		enum.Swordsman, enum.Samurai, enum.Rogue, enum.Pirate,
	}
	for _, name := range classesWithoutDist {
		class := classes[name]
		if class.Distribution != nil {
			t.Errorf("class %s: expected nil Distribution", name)
		}
	}
}
