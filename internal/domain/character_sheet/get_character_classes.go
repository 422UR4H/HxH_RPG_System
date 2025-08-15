package charactersheet

import (
	"sync"

	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	cs "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type IGetCharacterClass interface {
	GetCharacterClass(name string) (cc.CharacterClass, error)
	GetClassSheet(name string) (cs.HalfSheet, error)
}

type GetCharacterClassUC struct {
	characterClasses *sync.Map
	classSheets      *sync.Map
}

func NewGetCharacterClassUC(
	charClasses *sync.Map,
	classSheets *sync.Map,
) *GetCharacterClassUC {
	return &GetCharacterClassUC{
		characterClasses: charClasses,
		classSheets:      classSheets,
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
		return cc.CharacterClass{}, ErrCharacterClassNotFound
	}
	return class.(cc.CharacterClass), nil
}

func (uc *GetCharacterClassUC) GetClassSheet(
	name string) (cs.HalfSheet, error) {

	className, err := enum.CharacterClassNameFrom(name)
	if err != nil {
		return cs.HalfSheet{}, err
	}

	class, exists := uc.classSheets.Load(className)
	if !exists {
		return cs.HalfSheet{}, ErrCharacterClassNotFound
	}
	return class.(cs.HalfSheet), nil
}
