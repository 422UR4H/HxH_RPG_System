package match

import (
	"encoding/json"
	"strings"
	"testing"

	matchUC "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

// The REST mapping applies the use case's per-reader verdicts AFTER actionwire.From — the same
// order room.go applies the live gate in — and nothing else: the recorded views never reach the
// wire, and a hidden landing keeps the escape's public verdict (awaitsMaster included).
func TestHistoryTurnResponseAppliesTheReadersMoveSight(t *testing.T) {
	escaper := uuid.MustParse("00000000-0000-0000-0000-0000000000e1")
	landing := [3]int{7, 6, 0}
	withEscape := func(tu matchUC.HistoryTurn) matchUC.HistoryTurn {
		tu.Resolution = &service.TurnResolution{IsSettled: true, CharacterResults: []service.CharacterResult{{
			TargetID: escaper, Escape: &service.EscapeResult{DodgePassed: true, Landing: &landing},
		}}}
		return tu
	}

	tests := []struct {
		name           string
		sight          matchUC.MoveSight
		hidden         map[uuid.UUID]bool
		from, position bool
		landing        bool
	}{
		{name: "whole", sight: matchUC.MoveSightWhole, from: true, position: true, landing: true},
		{name: "origin", sight: matchUC.MoveSightOrigin, from: true, landing: true},
		{name: "none, landing hidden", sight: matchUC.MoveSightNone, hidden: map[uuid.UUID]bool{escaper: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tu := withEscape(fixtureGoldenHistoryTurn())
			tu.MoveSight, tu.HiddenLandings = tt.sight, tt.hidden
			// Even if the views were still there, the mapping must not carry them.
			tu.MoveViews = map[uuid.UUID]masteraction.View{uuid.New(): masteraction.ViewFull}
			tu.LandingViews = map[uuid.UUID]map[uuid.UUID]masteraction.View{escaper: {uuid.New(): masteraction.ViewFull}}

			raw, err := json.Marshal(toHistoryTurnResponse(tu))
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			for _, key := range []string{`"moveViews"`, `"landingViews"`, `"views"`, `"moveSight"`, `"hiddenLandings"`} {
				if strings.Contains(string(raw), key) {
					t.Errorf("%s is on the wire: %s", key, raw)
				}
			}

			var out struct {
				Action struct {
					Move map[string]json.RawMessage `json:"move"`
				} `json:"action"`
				Resolution struct {
					Targets []struct {
						Escape map[string]json.RawMessage `json:"escape"`
					} `json:"targets"`
				} `json:"resolution"`
			}
			if err := json.Unmarshal(raw, &out); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if _, ok := out.Action.Move["category"]; !ok {
				t.Error("move.category is missing — that the actor moved is public")
			}
			if _, ok := out.Action.Move["from"]; ok != tt.from {
				t.Errorf("move.from on the wire = %v, want %v", ok, tt.from)
			}
			if _, ok := out.Action.Move["position"]; ok != tt.position {
				t.Errorf("move.position on the wire = %v, want %v", ok, tt.position)
			}
			esc := out.Resolution.Targets[0].Escape
			if _, ok := esc["landing"]; ok != tt.landing {
				t.Errorf("escape.landing on the wire = %v, want %v", ok, tt.landing)
			}
			// A withheld landing is not a landing the master never chose.
			if string(esc["awaitsMaster"]) != "false" {
				t.Errorf("escape.awaitsMaster = %s, want false — the master did choose", esc["awaitsMaster"])
			}
		})
	}
}
