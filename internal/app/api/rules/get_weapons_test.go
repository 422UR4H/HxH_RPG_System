package rules_test

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"testing"

	rulesHandler "github.com/422UR4H/HxH_RPG_System/internal/app/api/rules"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/item"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
)

type weaponsBody struct {
	Weapons []struct {
		Name       string `json:"name"`
		Dice       []int  `json:"dice"`
		FlatDamage int    `json:"flatDamage"`
	} `json:"weapons"`
}

func getWeapons(t *testing.T, catalogue *item.WeaponsManager) (weaponsBody, http.Header) {
	t.Helper()
	_, api := humatest.New(t)
	huma.Register(api, huma.Operation{
		Method: http.MethodGet,
		Path:   "/rules/weapons",
	}, rulesHandler.GetWeaponsHandler(catalogue))

	resp := api.GetCtx(context.Background(), "/rules/weapons")
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", resp.Code, resp.Body.String())
	}
	var body weaponsBody
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	return body, resp.Header()
}

func TestGetWeaponsHandler(t *testing.T) {
	catalogue := item.NewWeaponsManagerFactory().Build()

	t.Run("lists every weapon once, in enum order", func(t *testing.T) {
		body, _ := getWeapons(t, catalogue)
		var got []string
		for _, w := range body.Weapons {
			got = append(got, w.Name)
		}
		var want []string
		for _, n := range enum.GetAllWeaponNames() {
			want = append(want, string(n))
		}
		if !slices.Equal(got, want) {
			t.Errorf("names = %v, want %v", got, want)
		}
	})

	t.Run("spot values", func(t *testing.T) {
		body, _ := getWeapons(t, catalogue)
		byName := map[string]struct {
			dice []int
			flat int
		}{}
		for _, w := range body.Weapons {
			byName[w.Name] = struct {
				dice []int
				flat int
			}{w.Dice, w.FlatDamage}
		}
		tests := []struct {
			name string
			dice []int
			flat int
		}{
			{"Katana", []int{4, 12}, 7},
			{"Bomb", []int{12, 12, 12}, 0},
			{"Fist", []int{6, 6, 4}, 0},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, ok := byName[tt.name]
				if !ok {
					t.Fatalf("%s missing", tt.name)
				}
				if !slices.Equal(got.dice, tt.dice) || got.flat != tt.flat {
					t.Errorf("got dice %v flat %d, want %v / %d", got.dice, got.flat, tt.dice, tt.flat)
				}
			})
		}
	})

	t.Run("cache header", func(t *testing.T) {
		_, h := getWeapons(t, catalogue)
		if got := h.Get("Cache-Control"); got != "public, max-age=300" {
			t.Errorf("Cache-Control = %q", got)
		}
	})

	t.Run("skips a weapon the catalogue does not carry", func(t *testing.T) {
		partial := item.NewWeaponsManager(map[enum.WeaponName]item.Weapon{
			enum.Katana: *item.NewWeapon([]int{4, 12}, 7, 0, 1, 1.3, 14, false),
		})
		body, _ := getWeapons(t, partial)
		if len(body.Weapons) != 1 || body.Weapons[0].Name != "Katana" {
			t.Errorf("weapons = %+v, want only Katana", body.Weapons)
		}
	})

	t.Run("mutating a response does not leak into the next", func(t *testing.T) {
		handler := rulesHandler.GetWeaponsHandler(catalogue)

		first, err := handler(context.Background(), nil)
		if err != nil {
			t.Fatal(err)
		}
		for i := range first.Body.Weapons {
			for j := range first.Body.Weapons[i].Dice {
				first.Body.Weapons[i].Dice[j] = 99
			}
		}
		second, _ := handler(context.Background(), nil)
		for _, w := range second.Body.Weapons {
			if slices.Contains(w.Dice, 99) {
				t.Fatalf("%s leaked mutation: %v", w.Name, w.Dice)
			}
		}
		// catalogue itself untouched
		d, _ := catalogue.GetDice(enum.Katana)
		if !slices.Equal(d, []int{4, 12}) {
			t.Errorf("catalogue Katana dice = %v", d)
		}
	})
}
