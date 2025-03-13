package charactersheet

import (
	"fmt"
	"sync"

	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type IGetCharacterClass interface {
	GetCharacterClass(name string) (cc.CharacterClass, error)
}

type GetCharacterClassUC struct {
	characterClasses *sync.Map
}

func NewGetCharacterClassUC(
	charClasses *sync.Map,
) *GetCharacterClassUC {
	return &GetCharacterClassUC{
		characterClasses: charClasses,
	}
}

func (uc *GetCharacterClassUC) GetCharacterClass(
	name string) (cc.CharacterClass, error) {

	className, err := enum.CharacterClassNameFrom(name)
	if err != nil {
		return cc.CharacterClass{}, err
	}

	class, exists := uc.characterClasses.Load(className)
	if !exists {
		return cc.CharacterClass{}, fmt.Errorf("character class %s not found", name)
	}
	return class.(cc.CharacterClass), nil
}
