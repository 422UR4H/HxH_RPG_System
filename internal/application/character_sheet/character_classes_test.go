package charactersheet_test

import (
	"errors"
	"sync"
	"testing"

	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

// armazena fichas de classe do mesmo jeito que cmd/api/main.go (ponteiro).
func newTestClassSheetMap(t *testing.T) *sync.Map {
	t.Helper()
	m := &sync.Map{}
	factory := newTestFactory()
	for name, class := range map[enum.CharacterClassName]cc.CharacterClass{
		enum.Swordsman: cc.BuildSwordsman(),
		enum.Ninja:     cc.BuildNinja(),
	} {
		ref := class
		cs, err := factory.BuildClassSheet(sheet.CharacterProfile{NickName: name.String()}, &ref)
		if err != nil {
			t.Fatalf("BuildClassSheet error: %v", err)
		}
		m.Store(name, cs)
	}
	return m
}

func TestGetClassSheet_returnsStoredSheet(t *testing.T) {
	uc := charactersheet.NewGetCharacterClassUC(newTestClassMap(), newTestClassSheetMap(t))

	cs, err := uc.GetClassSheet("Swordsman")
	if err != nil {
		t.Fatalf("GetClassSheet error: %v", err)
	}
	if cs == nil || cs.GetProfile().NickName != "Swordsman" {
		t.Errorf("unexpected class sheet: %+v", cs)
	}
}

func TestGetClassSheet_errors(t *testing.T) {
	uc := charactersheet.NewGetCharacterClassUC(newTestClassMap(), &sync.Map{})

	if _, err := uc.GetClassSheet("NotAClass"); err == nil {
		t.Error("expected error for invalid class name")
	}
	_, err := uc.GetClassSheet("Swordsman")
	if !errors.Is(err, charactersheet.ErrCharacterClassNotFound) {
		t.Errorf("err = %v, want ErrCharacterClassNotFound", err)
	}
}

func TestListClassSheets_listsStoredSheets(t *testing.T) {
	uc := charactersheet.NewListCharacterClassesUC(newTestClassMap(), newTestClassSheetMap(t))

	sheets := uc.ListClassSheets()
	if len(sheets) != 2 {
		t.Fatalf("len = %d, want 2", len(sheets))
	}
	for _, cs := range sheets {
		if cs == nil {
			t.Error("nil class sheet in list")
		}
	}
}
