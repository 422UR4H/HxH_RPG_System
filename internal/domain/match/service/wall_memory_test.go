package service_test

import (
	"testing"

	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

// A room with one interior wall the viewer faces, and a second wall hidden directly
// behind it. The viewer stands to the left of both.
func memoryFixture() (viewer service.Point2D, walls []mapentity.WallSegment, grid mapentity.GridShape) {
	grid = mapentity.GridShape{
		Kind: mapentity.GridKindSquare, Cols: 20, Rows: 20, CellSize: 64, SkewRatio: 1,
	}
	near := mapentity.WallSegment{
		ID: "near", P1: [2]float64{640, 0}, P2: [2]float64{640, 1280},
		WallType: mapentity.WallTypeWall, Material: mapentity.WallMaterialStone,
		Sense: mapentity.SenseSight, HP: 100, MaxHP: 100,
	}
	// Far wall sits behind `near` from the viewer's point of view, so it can never be
	// in line of sight and must never be remembered.
	far := mapentity.WallSegment{
		ID: "far", P1: [2]float64{704, 0}, P2: [2]float64{704, 1280},
		WallType: mapentity.WallTypeWall, Material: mapentity.WallMaterialStone,
		Sense: mapentity.SenseSight, HP: 100, MaxHP: 100,
	}
	return service.Point2D{X: 160, Y: 640}, []mapentity.WallSegment{near, far}, grid
}

func polysFor(viewer service.Point2D, walls []mapentity.WallSegment, grid mapentity.GridShape) []service.VisibilityPolygon {
	losWalls := append(service.ToLOSWalls(walls), service.BoundaryLOSWalls(grid)...)
	return []service.VisibilityPolygon{
		service.ComputeVisibilityPolygon(viewer, losWalls, 4000),
	}
}

func idsOf(walls []mapentity.WallSegment) map[string]bool {
	out := make(map[string]bool, len(walls))
	for _, w := range walls {
		out[w.ID] = true
	}
	return out
}

// The invariant from the spec: whatever FilterMapState reveals through line of sight is
// exactly what SeenWalls records. If these two predicates ever drift apart, a player
// sees a wall that never reaches memory and it disappears when they walk away.
func TestSeenWallsAgreesWithFilterMapState(t *testing.T) {
	viewer, walls, grid := memoryFixture()
	polys := polysFor(viewer, walls, grid)

	// Empty memory: everything FilterMapState returns came from line of sight alone.
	revealed, _ := service.FilterMapState(
		walls, nil, polys, nil, fog.FogModeExplored, uuid.New(), nil, false,
	)
	recorded := service.SeenWalls(walls, polys)

	revealedIDs := idsOf(revealed)
	recordedIDs := make(map[string]bool, len(recorded))
	for _, r := range recorded {
		if r.Kind != fog.FeatureWall {
			t.Fatalf("SeenWalls returned a non-wall kind: %q", r.Kind)
		}
		recordedIDs[r.ID] = true
	}

	if len(revealedIDs) != len(recordedIDs) {
		t.Fatalf("revealed %d walls but recorded %d", len(revealedIDs), len(recordedIDs))
	}
	for id := range revealedIDs {
		if !recordedIDs[id] {
			t.Fatalf("wall %q is shown to the player but never recorded in memory", id)
		}
	}
}

// The false negative of the cell model: the player sees a wall, walks away, and the
// wall must still be there.
func TestRememberedWallSurvivesLeavingLineOfSight(t *testing.T) {
	viewer, walls, grid := memoryFixture()
	polys := polysFor(viewer, walls, grid)

	memory := fog.NewPlayerMemory(uuid.New(), uuid.New(), uuid.New())
	memory.Remember(service.SeenWalls(walls, polys))

	// The player is gone: no polygons at all.
	got, _ := service.FilterMapState(
		walls, nil, nil, memory, fog.FogModeExplored, uuid.New(), nil, false,
	)
	if !idsOf(got)["near"] {
		t.Fatal("a wall the player already saw disappeared once they left line of sight")
	}
}

// The false positive of the cell model: a wall the player could never see must never be
// remembered, no matter how close the lit area gets.
func TestOccludedWallIsNeverRemembered(t *testing.T) {
	viewer, walls, grid := memoryFixture()
	polys := polysFor(viewer, walls, grid)

	for _, r := range service.SeenWalls(walls, polys) {
		if r.ID == "far" {
			t.Fatal("a wall hidden behind another wall was written to the player's memory")
		}
	}

	memory := fog.NewPlayerMemory(uuid.New(), uuid.New(), uuid.New())
	memory.Remember(service.SeenWalls(walls, polys))
	got, _ := service.FilterMapState(
		walls, nil, polys, memory, fog.FogModeExplored, uuid.New(), nil, false,
	)
	if idsOf(got)["far"] {
		t.Fatal("a wall the player never saw was sent to them")
	}
}

// live mode has no memory at all: walls exist only while in view.
func TestLiveModeIgnoresMemory(t *testing.T) {
	_, walls, _ := memoryFixture()

	memory := fog.NewPlayerMemory(uuid.New(), uuid.New(), uuid.New())
	memory.Remember([]fog.FeatureRef{{Kind: fog.FeatureWall, ID: "near"}})

	got, _ := service.FilterMapState(
		walls, nil, nil, memory, fog.FogModeLive, uuid.New(), nil, false,
	)
	if len(got) != 0 {
		t.Fatalf("live mode leaked %d remembered walls", len(got))
	}
}

// A nil memory is a real, common state (lobby, master, first push). It must not panic.
func TestNilMemoryIsSafe(t *testing.T) {
	_, walls, _ := memoryFixture()
	_, _ = service.FilterMapState(
		walls, nil, nil, nil, fog.FogModeExplored, uuid.New(), nil, false,
	)
}
