package match

import (
	"encoding/json"
	"strings"
	"testing"

	matchUC "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

// The REST mapping applies the use case's per-reader verdicts AFTER actionwire.From — the same
// order room.go applies the live gate in — and nothing else: the recorded views never reach the
// wire, and a hidden landing keeps the escape's public verdict (awaitsMaster included). Every
// verdict defaults to hidden: a HistoryTurn that never went through the use case reveals nothing.
func TestHistoryTurnResponseAppliesTheReadersMoveSight(t *testing.T) {
	escaper := uuid.MustParse("00000000-0000-0000-0000-0000000000e1")
	escapeID := uuid.MustParse("00000000-0000-0000-0000-0000000000e2")
	landing := [3]int{7, 6, 0}
	withEscape := func(tu matchUC.HistoryTurn) matchUC.HistoryTurn {
		tu.Resolution = &service.TurnResolution{IsSettled: true, CharacterResults: []service.CharacterResult{{
			TargetID: escaper, Escape: &service.EscapeResult{DodgePassed: true, Landing: &landing},
		}}}
		escape := action.NewAction(escaper, nil, tu.Action.GetID(), nil, action.ActionSpeed{}, nil,
			&action.Move{Category: enum.Dash, Position: [3]int{9, 9, 0}}, nil, nil, &action.Dodge{}, nil, nil,
			action.WithReconstructedID(escapeID))
		escape.ReactionKind = action.ReactEscape
		tu.Reactions = []action.Action{*escape}
		return tu
	}
	shown := map[uuid.UUID]bool{escaper: true, escapeID: true}

	tests := []struct {
		name           string
		sight          matchUC.MoveSight
		shown          map[uuid.UUID]bool
		from, position bool
		landing        bool
		reactionMove   bool
	}{
		{name: "whole", sight: matchUC.MoveSightWhole, shown: shown, from: true, position: true, landing: true, reactionMove: true},
		{name: "origin", sight: matchUC.MoveSightOrigin, shown: shown, from: true, landing: true, reactionMove: true},
		{name: "none, nothing shown", sight: matchUC.MoveSightNone},
		{name: "zero values: built without the use case, nothing revealed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tu := withEscape(fixtureGoldenHistoryTurn())
			tu.MoveSight, tu.ShownLandings, tu.ShownReactionMoves = tt.sight, tt.shown, tt.shown
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
				Reactions []struct {
					Move map[string]json.RawMessage `json:"move"`
				} `json:"reactions"`
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
			react := out.Reactions[0].Move
			if _, ok := react["category"]; !ok {
				t.Error("reactions[0].move.category is missing")
			}
			if _, ok := react["position"]; ok != tt.reactionMove {
				t.Errorf("reactions[0].move.position on the wire = %v, want %v", ok, tt.reactionMove)
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
