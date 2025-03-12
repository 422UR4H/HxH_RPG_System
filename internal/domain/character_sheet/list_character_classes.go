package charactersheet

import (
	"sync"

	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
)

type IListCharacterClasses interface {
	ListCharacterClasses() []cc.CharacterClass
}

type ListCharacterClassesUC struct {
	characterClasses *sync.Map
}

func NewListCharacterClassesUC(
	charClasses *sync.Map,
) *ListCharacterClassesUC {
	return &ListCharacterClassesUC{
		characterClasses: charClasses,
	}
}

func (uc *ListCharacterClassesUC) ListCharacterClasses() []cc.CharacterClass {
	var classes []cc.CharacterClass
	uc.characterClasses.Range(func(_, value any) bool {
		class := value.(cc.CharacterClass)
		classes = append(classes, class)
		return true
	})
	return classes
}
