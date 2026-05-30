package sheet_test

// TestToPrivateOnlyResponse_* pins the exp computation in ToPrivateOnlyResponse
// against known values from three real characters. These characters cover two
// distinct failure modes that have broken the listing in the past:
//
//  1. Wrong coefficient (NewDefaultExpTable instead of NewExpTable(CHARACTER_COEFF)):
//     curr_exp and next_lvl_base_exp are both 10x off.
//  2. Stale DB level column: the level stored in the DB is higher than what
//     char_exp justifies, causing curr_exp to go negative when the stale
//     level is used to look up aggregate exp.
//
// The fix for both: derive level from char_exp via charExpTable.GetLvlByExp,
// using CHARACTER_COEFF=10.0. deriveCurrExp and deriveNxtLvlBaseExp already
// do this; Level field uses the same derivation.

import (
	"testing"
	"time"

	appSheet "github.com/422UR4H/HxH_RPG_System/internal/app/api/sheet"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	"github.com/google/uuid"
)

type summaryFixture struct {
	name           string
	uuid           string
	dbLevel        int // stale level stored in character_sheets.level
	charExp        int // char_exp after correct backfill / self-heal
	wantLevel      int // level derived from char_exp
	wantCurrExp    int // curr_exp shown in character sheet view
	wantNextLvlExp int // next_lvl_base_exp shown in character sheet view
}

var listingExpFixtures = []summaryFixture{
	{
		// Baseline: DB level matches derived level. Verifies correct coefficient.
		name:           "644188d4 level matches char_exp",
		uuid:           "644188d4-99ec-4fa2-9f3b-0ea51342dcf5",
		dbLevel:        6,
		charExp:        5471,
		wantLevel:      6,
		wantCurrExp:    501,
		wantNextLvlExp: 2310,
	},
	{
		// Stale DB level=6 but char_exp=4933 only justifies level=5.
		// Without the fix: curr_exp = 4933 - aggregate[6] = -37 (negative).
		// With the fix:    curr_exp = 4933 - aggregate[5] = 1628.
		name:           "72293eac stale DB level too high by 1",
		uuid:           "72293eac-0f7b-4145-86fa-2c271ba1d58c",
		dbLevel:        6,
		charExp:        4933,
		wantLevel:      5,
		wantCurrExp:    1628,
		wantNextLvlExp: 1665,
	},
	{
		// Stale DB level=7 but char_exp=6728 only justifies level=6.
		// Without the fix: curr_exp = 6728 - aggregate[7] = -552 (negative).
		// With the fix:    curr_exp = 6728 - aggregate[6] = 1758.
		name:           "8e79bdd3 stale DB level too high by 1",
		uuid:           "8e79bdd3-8aee-46c7-965d-3a2b1acecd44",
		dbLevel:        7,
		charExp:        6728,
		wantLevel:      6,
		wantCurrExp:    1758,
		wantNextLvlExp: 2310,
	},
}

func makeSummary(f summaryFixture) *csEntity.Summary {
	return &csEntity.Summary{
		UUID:           uuid.MustParse(f.uuid),
		NickName:       "test",
		FullName:       "test",
		Alignment:      "neutral",
		CharacterClass: "test",
		Birthday:       time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		CategoryName:   "test",
		Level:          f.dbLevel,
		CharExp:        f.charExp,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

func TestToPrivateOnlyResponse_Level(t *testing.T) {
	for _, f := range listingExpFixtures {
		t.Run(f.name, func(t *testing.T) {
			resp := appSheet.ToPrivateOnlyResponse(makeSummary(f))
			if resp.Level != f.wantLevel {
				t.Errorf("Level = %d, want %d", resp.Level, f.wantLevel)
			}
		})
	}
}

func TestToPrivateOnlyResponse_CurrExp(t *testing.T) {
	for _, f := range listingExpFixtures {
		t.Run(f.name, func(t *testing.T) {
			resp := appSheet.ToPrivateOnlyResponse(makeSummary(f))
			if resp.CurrExp != f.wantCurrExp {
				t.Errorf("CurrExp = %d, want %d", resp.CurrExp, f.wantCurrExp)
			}
		})
	}
}

func TestToPrivateOnlyResponse_NxtLvlBaseExp(t *testing.T) {
	for _, f := range listingExpFixtures {
		t.Run(f.name, func(t *testing.T) {
			resp := appSheet.ToPrivateOnlyResponse(makeSummary(f))
			if resp.NxtLvlBaseExp != f.wantNextLvlExp {
				t.Errorf("NxtLvlBaseExp = %d, want %d", resp.NxtLvlBaseExp, f.wantNextLvlExp)
			}
		})
	}
}
