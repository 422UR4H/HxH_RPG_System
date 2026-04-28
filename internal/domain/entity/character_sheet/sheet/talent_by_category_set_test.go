package sheet_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func TestTalentByCategorySet_NewWithZeroActive(t *testing.T) {
	categories := map[enum.CategoryName]bool{
		enum.Reinforcement:   false,
		enum.Transmutation:   false,
		enum.Materialization: false,
		enum.Specialization:  false,
		enum.Manipulation:    false,
		enum.Emission:        false,
	}
	_, err := sheet.NewTalentByCategorySet(categories, nil)
	if err == nil {
		t.Error("should reject 0 active categories")
	}
}

func TestTalentByCategorySet_GetTalentLvl(t *testing.T) {
	tests := []struct {
		name        string
		active      int
		hasHexValue bool
		want        int
	}{
		// No hex value: bonus = (active-1)*2, special: if bonus==0 then bonus=1
		{"1 active no hex", 1, false, 21},  // BASE(20) + bonus: (1-1)*2=0 → special: 1
		{"2 active no hex", 2, false, 22},  // BASE(20) + (2-1)*2 = 22
		{"3 active no hex", 3, false, 24},  // BASE(20) + (3-1)*2 = 24
		{"6 active no hex", 6, false, 30},  // BASE(20) + (6-1)*2 = 30
		// With hex value: bonus = (active-1)*1
		{"1 active with hex", 1, true, 20}, // BASE(20) + (1-1)*1 = 20
		{"2 active with hex", 2, true, 21}, // BASE(20) + (2-1)*1 = 21
		{"6 active with hex", 6, true, 25}, // BASE(20) + (6-1)*1 = 25
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allCategories := enum.AllNenCategoryNames()
			categories := make(map[enum.CategoryName]bool)
			for i, name := range allCategories {
				categories[name] = i < tt.active
			}

			var hexVal *int
			if tt.hasHexValue {
				v := 0
				hexVal = &v
			}

			tcs, err := sheet.NewTalentByCategorySet(categories, hexVal)
			if err != nil {
				t.Fatalf("NewTalentByCategorySet error: %v", err)
			}

			if got := tcs.GetTalentLvl(); got != tt.want {
				t.Errorf("GetTalentLvl() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestTalentByCategorySet_Getters(t *testing.T) {
	categories := map[enum.CategoryName]bool{
		enum.Reinforcement: true,
		enum.Emission:      true,
	}
	hexVal := 100
	tcs, err := sheet.NewTalentByCategorySet(categories, &hexVal)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if len(tcs.GetCategories()) != 2 {
		t.Errorf("categories count = %d, want 2", len(tcs.GetCategories()))
	}
	if *tcs.GetInitialHexValue() != 100 {
		t.Errorf("initial hex = %d, want 100", *tcs.GetInitialHexValue())
	}
}
