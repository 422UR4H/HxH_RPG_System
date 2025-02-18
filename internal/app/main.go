package main

import (
	"fmt"

	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/sheet"
)

var characterClasses map[enum.CharacterClassName]cc.CharacterClass
var charClassSheets map[enum.CharacterClassName]*sheet.CharacterSheet

func main() {
	characterClasses = make(map[enum.CharacterClassName]cc.CharacterClass)
	charClassSheets = make(map[enum.CharacterClassName]*sheet.CharacterSheet)
	InitCharacterClasses()
}

func InitCharacterClasses() {
	factory := sheet.NewCharacterSheetFactory()
	ccFactory := cc.NewCharacterClassFactory()
	characterClasses = ccFactory.Build()

	for name, class := range characterClasses {
		profile := sheet.CharacterProfile{
			NickName:         name.String(),
			Alignment:        class.Profile.Alignment,
			Description:      class.Profile.Description,
			BriefDescription: class.Profile.BriefDescription,
		}
		set, err := sheet.NewTalentByCategorySet(
			map[enum.CategoryName]bool{
				enum.Reinforcement:   true,
				enum.Transmutation:   true,
				enum.Materialization: true,
				enum.Specialization:  true,
				enum.Manipulation:    true,
				enum.Emission:        true,
			},
			nil,
		)
		if err != nil {
			fmt.Println(err)
			continue
		}
		newClass := factory.Build(profile, set, &class)
		charClassSheets[name] = newClass
		// uncomment to print all character classes
		// fmt.Println(newClass.ToString())
	}
}

func GetCharacterClass(name enum.CharacterClassName) (cc.CharacterClass, error) {
	class, exists := characterClasses[name]
	if !exists {
		return cc.CharacterClass{}, fmt.Errorf("character class %s not found", name)
	}
	return class, nil
}

func GetAllCharacterClasses() map[enum.CharacterClassName]cc.CharacterClass {
	return characterClasses
}
