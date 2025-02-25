package main

import (
	"fmt"
	"sync"

	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/sheet"
)

var characterClasses sync.Map

// TODO: remove or handle after balancing
var charClassSheets map[enum.CharacterClassName]*sheet.CharacterSheet

func main() {
	charClassSheets = make(map[enum.CharacterClassName]*sheet.CharacterSheet)
	InitCharacterClasses()
}

func InitCharacterClasses() {
	factory := sheet.NewCharacterSheetFactory()
	ccFactory := cc.NewCharacterClassFactory()
	for name, class := range ccFactory.Build() {
		characterClasses.Store(name, class)
	}

	characterClasses.Range(func(key, value interface{}) bool {
		name := key.(enum.CharacterClassName)
		class := value.(cc.CharacterClass)
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
		}
		newClass, err := factory.Build(profile, set, &class)
		if err != nil {
			fmt.Println(err)
		}
		charClassSheets[name] = newClass
		// uncomment to print all character classes
		fmt.Println(newClass.ToString())
		return true
	})
}

func GetCharacterClass(name enum.CharacterClassName) (cc.CharacterClass, error) {

	class, exists := characterClasses.Load(name)
	if !exists {
		return cc.CharacterClass{}, fmt.Errorf("character class %s not found", name)
	}
	return class.(cc.CharacterClass), nil
}

func GetAllCharacterClasses() map[enum.CharacterClassName]cc.CharacterClass {
	charClasses := make(map[enum.CharacterClassName]cc.CharacterClass)

	characterClasses.Range(func(key, value interface{}) bool {
		name := key.(enum.CharacterClassName)
		class := value.(cc.CharacterClass)
		charClasses[name] = class
		return true
	})
	return charClasses
}
