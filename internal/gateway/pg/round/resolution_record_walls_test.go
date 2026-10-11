package round

import (
	"testing"

	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
)

// A wall's outcome survives the row — and a secret door nobody had been shown comes back as one,
// so the history's projection still withholds it from the players.
func TestResolutionRecord_WallResultsRoundTrip(t *testing.T) {
	res := &service.TurnResolution{
		IsSettled: true,
		WallResults: []service.WallResult{
			{UpdatedWall: mapentity.WallSegment{ID: "door", WallType: mapentity.WallTypeDoor}, Kind: service.WallResultKindInteract, Failure: "locked"},
			{UpdatedWall: mapentity.WallSegment{ID: "secret", WallType: mapentity.WallTypeSecretDoor}, Kind: service.WallResultKindAttack, EffectiveDamage: 4},
		},
	}
	raw, err := encodeResolution(res, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := DecodeResolution(raw)
	if len(got.WallResults) != 2 {
		t.Fatalf("wall results = %+v", got.WallResults)
	}
	if got.WallResults[0].Failure != "locked" || got.WallResults[1].EffectiveDamage != 4 {
		t.Errorf("outcomes lost: %+v", got.WallResults)
	}
	player := service.ProjectResolution(got, service.Viewer{})
	if len(player.WallResults) != 1 || player.WallResults[0].UpdatedWall.ID != "door" {
		t.Errorf("a player's history shows %+v, want only the door", player.WallResults)
	}
}
