package charactersheet

import (
	"sync"

	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	cs "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
)

type IListCharacterClasses interface {
	ListCharacterClasses() []cc.CharacterClass
	ListClassSheets() []cs.HalfSheet
}

type ListCharacterClassesUC struct {
	characterClasses *sync.Map
	classSheets      *sync.Map
}

func NewListCharacterClassesUC(
	charClasses *sync.Map,
	classSheets *sync.Map,
) *ListCharacterClassesUC {
	return &ListCharacterClassesUC{
		characterClasses: charClasses,
		classSheets:      classSheets,
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

func (uc *ListCharacterClassesUC) ListClassSheets() []cs.HalfSheet {
	var classes []cs.HalfSheet
	uc.classSheets.Range(func(_, value any) bool {
		class := value.(*cs.HalfSheet)
		classes = append(classes, *class)
		return true
	})
	return classes
}
