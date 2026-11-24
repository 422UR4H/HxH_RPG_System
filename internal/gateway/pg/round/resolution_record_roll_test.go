package round

import (
	"slices"
	"strings"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
)

// O monte lido e a falha automática sobrevivem à linha; o Short (pedido de dados de um
// Derive) não é gravado.
func TestResolutionRecord_RollOutcomeRoundTrip(t *testing.T) {
	res := &service.TurnResolution{
		IsSettled: true,
		CharacterResults: []service.CharacterResult{{
			Hit:   service.RollOutcome{Dice: []int{9, 10}, Rolled: []int{3, 4, 9, 10}, DiceTotal: 19, Total: 19, Short: 2},
			Dodge: service.RollOutcome{AutoFailed: true},
		}},
	}
	raw, err := encodeResolution(res, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "short") || strings.Contains(string(raw), "Short") {
		t.Fatalf("Short was written: %s", raw)
	}
	got := DecodeResolution(raw)
	if len(got.CharacterResults) != 1 {
		t.Fatalf("character results = %+v", got.CharacterResults)
	}
	cr := got.CharacterResults[0]
	if !slices.Equal(cr.Hit.Rolled, []int{3, 4, 9, 10}) || cr.Hit.Short != 0 {
		t.Errorf("hit = %+v, want the pool back and no Short", cr.Hit)
	}
	if !cr.Dodge.AutoFailed {
		t.Errorf("dodge = %+v, want AutoFailed back", cr.Dodge)
	}
}
