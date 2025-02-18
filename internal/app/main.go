package main

import (
	"fmt"

	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

// por conta de estar na camada app, o tratamento deve ser feito aqui, em um uc
var characterClasses map[enum.CharacterClassName]cc.CharacterClass

func main() {
	characterClasses = make(map[enum.CharacterClassName]cc.CharacterClass)
	InitCharacterClasses()
}

func InitCharacterClasses() {
	// factory := sheet.NewCharacterSheetFactory()
	ccFactory := cc.NewCharacterClassFactory()
	characterClasses = ccFactory.Build()
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
