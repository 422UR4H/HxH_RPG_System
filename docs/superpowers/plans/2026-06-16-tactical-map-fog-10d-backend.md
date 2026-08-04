# Tactical Map Fog of War (10-D) — Backend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Backend line-of-sight and fog-of-war for the tactical map: per-player filtering of walls/pieces (REST + WS), secret-door masking, and the fog-state persistence scaffold.

**Architecture:** A pure-domain geometric angular-sweep computes each player's visibility polygon in world coords. A single `FilterMapState` domain service applies all visibility policy, reused by the REST handler and the per-player WS dispatch in `room.go`. `MatchSession` owns the live fog state and visibility cache. Fog exploration persistence is seeded at `start_match` (concrete) with a turn-close wire-up left as a documented TODO.

**Tech Stack:** Go, goose migrations, Postgres JSONB, gorilla/websocket, existing DDD-lite layout (`domain` / `application` / `gateway` / `app`).

**Spec:** `docs/superpowers/specs/2026-06-16-tactical-map-walls-10d-design.md`

---

## Conventions for every task

- Run tests from repo root `System_X_System/`.
- Unit tests: `go test ./internal/...`. Single package: `go test ./internal/domain/match/service/ -run TestName -v`.
- After each implementation task: `go vet ./...` must pass (project rule — never poll CI).
- Commit messages end with the project's `Co-Authored-By` trailer (see CLAUDE.md). Commits below omit it for brevity — add it.
- Branch: `feat/tactical-map-walls-10d` (already created; the design commit lives here).

---

## File structure (created / modified)

**Created:**
- `internal/domain/match/entity/fog/fog_mode.go` — `FogMode` enum
- `internal/domain/match/entity/fog/cell_coord.go` — `CellCoord`
- `internal/domain/match/entity/fog/player_fog_state.go` — `PlayerFogState` + `AddExplored`
- `internal/domain/match/entity/fog/player_fog_state_test.go`
- `internal/domain/map/service/coords.go` — `SlotToWorld` / `WorldToSlot` (square, hex, iso)
- `internal/domain/map/service/coords_test.go`
- `internal/domain/match/service/visibility.go` — sweep, `ToLOSWalls`, `PointInPolygon`, `IsVisible`, `CellsInPolygon`
- `internal/domain/match/service/visibility_test.go`
- `internal/domain/match/service/mask_secret_door.go` — `MaskSecretDoorForPlayer`
- `internal/domain/match/service/mask_secret_door_test.go`
- `internal/domain/match/service/filter_map_state.go` — `PieceVisibility`, `FilterMapState`
- `internal/domain/match/service/filter_map_state_test.go`
- `internal/application/fog/i_fog_state_repository.go` — repo interface
- `internal/gateway/pg/fog/fog_state_repository.go` — PG stub (TODO impl)
- `migrations/NNNN_fog_state.sql` (goose) — ALTER maps + CREATE player_fog_states

**Modified:**
- `internal/domain/map/entity/wall_segment.go` — `Revealed bool`
- `internal/domain/map/entity/map.go` — `FogMode` on `TacticalMap`
- `internal/gateway/pg/maps_mapper.go` — (de)serialize `fog_mode`; `Revealed` rides existing walls JSONB
- `internal/domain/map/service/map_validator.go` — validate `fog_mode`
- `internal/domain/match/matchsession/match_session.go` — fog fields, `grid GridShape`, methods
- `internal/domain/match/matchsession/match_session_test.go`
- `internal/app/game/message.go` — new message types/payloads, extended `MapFullStatePayload`
- `internal/app/game/room.go` — `dispatchPerPlayer`, start_match fog steps, filtered sends, per-receiver piece moves, secret-door masking, `wall_revealed`
- `internal/app/game/game_test.go` — room behavior tests
- `internal/app/api/maps_handler.go` — role-aware `GET /maps/:id` filtering
- `docs/dev/api/maps.md` — filtering + new WS events
- `docs/documentation-map.yaml`

---

## Task 1: Fog domain entities

**Files:**
- Create: `internal/domain/match/entity/fog/fog_mode.go`
- Create: `internal/domain/match/entity/fog/cell_coord.go`
- Create: `internal/domain/match/entity/fog/player_fog_state.go`
- Test: `internal/domain/match/entity/fog/player_fog_state_test.go`

- [ ] **Step 1: Write the failing test**

`internal/domain/match/entity/fog/player_fog_state_test.go`:
```go
package fog

import (
	"testing"

	"github.com/google/uuid"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
)

func TestAddExplored_ReturnsOnlyNewCells(t *testing.T) {
	s := NewPlayerFogState(uuid.New(), uuid.New(), uuid.New(), mapentity.GridKindSquare)

	delta := s.AddExplored([]CellCoord{{A: 0, B: 0}, {A: 1, B: 0}})
	if len(delta) != 2 {
		t.Fatalf("first add: want 2 new cells, got %d", len(delta))
	}

	delta = s.AddExplored([]CellCoord{{A: 1, B: 0}, {A: 2, B: 0}})
	if len(delta) != 1 {
		t.Fatalf("second add: want 1 new cell, got %d", len(delta))
	}
	if delta[0] != (CellCoord{A: 2, B: 0}) {
		t.Fatalf("want new cell {2,0}, got %+v", delta[0])
	}
	if len(s.ExploredCells) != 3 {
		t.Fatalf("want 3 total explored, got %d", len(s.ExploredCells))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/match/entity/fog/ -run TestAddExplored -v`
Expected: FAIL — package/symbols undefined.

- [ ] **Step 3: Write the implementation**

`internal/domain/match/entity/fog/fog_mode.go`:
```go
package fog

type FogMode string

const (
	FogModeLive     FogMode = "live"
	FogModeExplored FogMode = "explored"
)

// IsValid reports whether m is a known fog mode.
func (m FogMode) IsValid() bool {
	return m == FogModeLive || m == FogModeExplored
}
```

`internal/domain/match/entity/fog/cell_coord.go`:
```go
package fog

// CellCoord is grid-agnostic: (A,B) = (Col,Row) for square, (Q,R) for hex axial.
type CellCoord struct {
	A int
	B int
}
```

`internal/domain/match/entity/fog/player_fog_state.go`:
```go
package fog

import (
	"time"

	"github.com/google/uuid"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
)

// PlayerFogState is the accumulated explored area for one (player, match, map).
type PlayerFogState struct {
	PlayerID      uuid.UUID
	MatchID       uuid.UUID
	MapID         uuid.UUID
	GridKind      mapentity.GridKind
	ExploredCells map[CellCoord]struct{}
	UpdatedAt     time.Time
}

func NewPlayerFogState(playerID, matchID, mapID uuid.UUID, kind mapentity.GridKind) *PlayerFogState {
	return &PlayerFogState{
		PlayerID:      playerID,
		MatchID:       matchID,
		MapID:         mapID,
		GridKind:      kind,
		ExploredCells: make(map[CellCoord]struct{}),
		UpdatedAt:     time.Now(),
	}
}

// AddExplored unions cells into the explored set and returns only the newly added
// cells (the delta). Order of the delta follows the input order.
func (s *PlayerFogState) AddExplored(cells []CellCoord) []CellCoord {
	delta := make([]CellCoord, 0, len(cells))
	for _, c := range cells {
		if _, ok := s.ExploredCells[c]; ok {
			continue
		}
		s.ExploredCells[c] = struct{}{}
		delta = append(delta, c)
	}
	if len(delta) > 0 {
		s.UpdatedAt = time.Now()
	}
	return delta
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/domain/match/entity/fog/ -run TestAddExplored -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/match/entity/fog/
git commit -m "feat(fog): add fog domain entities (FogMode, CellCoord, PlayerFogState)"
```

---

## Task 2: Grid coordinate math in Go

`CellsInPolygon` needs cell-center↔world conversion for square and hex, with the iso
transform (skew + rotation) applied. This mirrors the frontend `utils/coords.ts` + `hex.ts`.

**Files:**
- Create: `internal/domain/map/service/coords.go`
- Test: `internal/domain/map/service/coords_test.go`

- [ ] **Step 1: Write the failing test**

`internal/domain/map/service/coords_test.go`:
```go
package service

import (
	"math"
	"testing"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func TestSquareSlotToWorld_CellCenter(t *testing.T) {
	g := entity.GridShape{Kind: entity.GridKindSquare, CellSize: 64, SkewRatio: 1, Rotation: 0}
	x, y := SlotCenterToWorld(2, 1, g)
	if !approx(x, 2.5*64) || !approx(y, 1.5*64) {
		t.Fatalf("want (160,96), got (%v,%v)", x, y)
	}
}

func TestSquareWorldToSlot_RoundTrip(t *testing.T) {
	g := entity.GridShape{Kind: entity.GridKindSquare, CellSize: 64, SkewRatio: 1, Rotation: 0}
	x, y := SlotCenterToWorld(5, 3, g)
	a, b := WorldToSlot(x, y, g)
	if a != 5 || b != 3 {
		t.Fatalf("round trip want (5,3), got (%d,%d)", a, b)
	}
}

func TestIsoSkew_RoundTrip(t *testing.T) {
	g := entity.GridShape{Kind: entity.GridKindSquare, CellSize: 64, SkewRatio: 0.5, Rotation: 30}
	x, y := SlotCenterToWorld(4, 6, g)
	a, b := WorldToSlot(x, y, g)
	if a != 4 || b != 6 {
		t.Fatalf("iso round trip want (4,6), got (%d,%d)", a, b)
	}
}

func TestHexSlotToWorld_RoundTrip(t *testing.T) {
	g := entity.GridShape{Kind: entity.GridKindHex, CellSize: 64, SkewRatio: 1, Rotation: 0}
	x, y := SlotCenterToWorld(3, -2, g) // (q=3, r=-2)
	a, b := WorldToSlot(x, y, g)
	if a != 3 || b != -2 {
		t.Fatalf("hex round trip want (3,-2), got (%d,%d)", a, b)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/map/service/ -run "Slot|Iso|Hex|World" -v`
Expected: FAIL — `SlotCenterToWorld`/`WorldToSlot` undefined.

- [ ] **Step 3: Write the implementation**

`internal/domain/map/service/coords.go`:
```go
package service

import (
	"math"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
)

// applyTransform applies the grid's iso skew (Y scaling) then rotation, in that order.
// Mirrors the frontend utils/coords.ts applyTransform.
func applyTransform(x, y float64, g entity.GridShape) (float64, float64) {
	ys := y * g.SkewRatio
	if g.Rotation == 0 {
		return x, ys
	}
	rad := g.Rotation * math.Pi / 180
	cos, sin := math.Cos(rad), math.Sin(rad)
	return x*cos - ys*sin, x*sin + ys*cos
}

// reverseTransform inverts applyTransform.
func reverseTransform(x, y float64, g entity.GridShape) (float64, float64) {
	rx, ry := x, y
	if g.Rotation != 0 {
		rad := g.Rotation * math.Pi / 180
		cos, sin := math.Cos(rad), math.Sin(rad)
		rx = x*cos + y*sin
		ry = -x*sin + y*cos
	}
	if g.SkewRatio != 0 {
		ry /= g.SkewRatio
	}
	return rx, ry
}

// baseCenter returns the untransformed world center of cell (a,b).
func baseCenter(a, b int, g entity.GridShape) (float64, float64) {
	if g.Kind == entity.GridKindHex {
		// pointy-top axial → pixel (RedBlobGames), size = CellSize/2.
		size := g.CellSize / 2
		x := size * math.Sqrt(3) * (float64(a) + float64(b)/2)
		y := size * 1.5 * float64(b)
		return x, y
	}
	return (float64(a) + 0.5) * g.CellSize, (float64(b) + 0.5) * g.CellSize
}

// SlotCenterToWorld returns the transformed world position of cell (a,b)'s center.
func SlotCenterToWorld(a, b int, g entity.GridShape) (float64, float64) {
	bx, by := baseCenter(a, b, g)
	return applyTransform(bx, by, g)
}

// WorldToSlot returns the cell (a,b) whose region contains world point (x,y).
func WorldToSlot(x, y float64, g entity.GridShape) (int, int) {
	bx, by := reverseTransform(x, y, g)
	if g.Kind == entity.GridKindHex {
		size := g.CellSize / 2
		q := (math.Sqrt(3)/3*bx - 1.0/3*by) / size
		r := (2.0 / 3 * by) / size
		return hexRound(q, r)
	}
	return int(math.Floor(bx / g.CellSize)), int(math.Floor(by / g.CellSize))
}

// hexRound rounds fractional axial coords to the nearest hex (cube rounding).
func hexRound(q, r float64) (int, int) {
	s := -q - r
	rq, rr, rs := math.Round(q), math.Round(r), math.Round(s)
	dq, dr, ds := math.Abs(rq-q), math.Abs(rr-r), math.Abs(rs-s)
	if dq > dr && dq > ds {
		rq = -rr - rs
	} else if dr > ds {
		rr = -rq - rs
	}
	return int(rq), int(rr)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/domain/map/service/ -run "Slot|Iso|Hex|World" -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/map/service/coords.go internal/domain/map/service/coords_test.go
git commit -m "feat(map): add Go slot<->world coords (square, hex, iso) for fog cell math"
```

---

## Task 3: Visibility types + ToLOSWalls

**Files:**
- Create: `internal/domain/match/service/visibility.go`
- Test: `internal/domain/match/service/visibility_test.go`

- [ ] **Step 1: Write the failing test**

`internal/domain/match/service/visibility_test.go`:
```go
package service

import (
	"testing"

	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
)

func wall(p1, p2 [2]float64, sense mapentity.SenseKind, open, destroyed bool) mapentity.WallSegment {
	return mapentity.WallSegment{
		ID: "w", P1: p1, P2: p2, Sense: sense, Open: open, Destroyed: destroyed,
		Direction: mapentity.WallDirectionBoth,
	}
}

func TestToLOSWalls_Excludes(t *testing.T) {
	walls := []mapentity.WallSegment{
		wall([2]float64{0, 0}, [2]float64{1, 0}, mapentity.SenseFull, false, false),  // keep
		wall([2]float64{0, 1}, [2]float64{1, 1}, mapentity.SenseSight, false, false), // keep
		wall([2]float64{0, 2}, [2]float64{1, 2}, mapentity.SenseNone, false, false),  // drop: none
		wall([2]float64{0, 3}, [2]float64{1, 3}, mapentity.SenseFull, true, false),   // drop: open
		wall([2]float64{0, 4}, [2]float64{1, 4}, mapentity.SenseFull, false, true),   // drop: destroyed
	}
	got := ToLOSWalls(walls)
	if len(got) != 2 {
		t.Fatalf("want 2 LOS walls, got %d", len(got))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/match/service/ -run TestToLOSWalls -v`
Expected: FAIL — `ToLOSWalls` undefined.

- [ ] **Step 3: Write the implementation**

`internal/domain/match/service/visibility.go` (types + ToLOSWalls; sweep added in Task 5):
```go
package service

import (
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
)

type Point2D struct{ X, Y float64 }

// VisibilityPolygon is the sweep result for one origin (one piece). World coords.
type VisibilityPolygon struct {
	Origin   Point2D
	Vertices []Point2D
}

// WallSegmentLOS is the minimal projection of a vision-blocking wall.
type WallSegmentLOS struct {
	ID        string
	P1, P2    Point2D
	Direction mapentity.WallDirection
}

// ToLOSWalls keeps only walls that block vision: excludes sense=none, destroyed, open.
func ToLOSWalls(walls []mapentity.WallSegment) []WallSegmentLOS {
	out := make([]WallSegmentLOS, 0, len(walls))
	for _, w := range walls {
		if w.Sense == mapentity.SenseNone || w.Destroyed || w.Open {
			continue
		}
		out = append(out, WallSegmentLOS{
			ID:        w.ID,
			P1:        Point2D{w.P1[0], w.P1[1]},
			P2:        Point2D{w.P2[0], w.P2[1]},
			Direction: w.Direction,
		})
	}
	return out
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/domain/match/service/ -run TestToLOSWalls -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/match/service/visibility.go internal/domain/match/service/visibility_test.go
git commit -m "feat(fog): add visibility types and ToLOSWalls filter"
```

---

## Task 4: PointInPolygon + IsVisible

**Files:**
- Modify: `internal/domain/match/service/visibility.go`
- Test: `internal/domain/match/service/visibility_test.go`

- [ ] **Step 1: Write the failing test (append)**

```go
func TestPointInPolygon(t *testing.T) {
	square := []Point2D{{0, 0}, {10, 0}, {10, 10}, {0, 10}}
	if !PointInPolygon(Point2D{5, 5}, square) {
		t.Fatal("center should be inside")
	}
	if PointInPolygon(Point2D{15, 5}, square) {
		t.Fatal("outside point should be outside")
	}
}

func TestIsVisible_AnyPolygon(t *testing.T) {
	polys := []VisibilityPolygon{
		{Vertices: []Point2D{{0, 0}, {10, 0}, {10, 10}, {0, 10}}},
	}
	if !IsVisible(Point2D{5, 5}, polys) {
		t.Fatal("should be visible")
	}
	if IsVisible(Point2D{50, 50}, polys) {
		t.Fatal("should not be visible")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/match/service/ -run "PointInPolygon|IsVisible" -v`
Expected: FAIL — undefined.

- [ ] **Step 3: Write the implementation (append to visibility.go)**

```go
// PointInPolygon does ray casting; O(V).
func PointInPolygon(p Point2D, poly []Point2D) bool {
	inside := false
	n := len(poly)
	for i, j := 0, n-1; i < n; j, i = i, i+1 {
		yi, yj := poly[i].Y, poly[j].Y
		xi, xj := poly[i].X, poly[j].X
		if (yi > p.Y) != (yj > p.Y) &&
			p.X < (xj-xi)*(p.Y-yi)/(yj-yi)+xi {
			inside = !inside
		}
	}
	return inside
}

// IsVisible reports whether target lies in any of the player's polygons.
func IsVisible(target Point2D, polys []VisibilityPolygon) bool {
	for _, poly := range polys {
		if PointInPolygon(target, poly.Vertices) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/domain/match/service/ -run "PointInPolygon|IsVisible" -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/match/service/visibility.go internal/domain/match/service/visibility_test.go
git commit -m "feat(fog): add PointInPolygon and IsVisible"
```

---

## Task 5: ComputeVisibilityPolygon (angular sweep)

The core. Casts rays toward every wall endpoint (plus two epsilon-offset rays per endpoint
to slip past corners), finds the nearest wall hit along each ray, and assembles the hit
points in angular order into a polygon. One-way walls only occlude from their blocking side.

**Files:**
- Modify: `internal/domain/match/service/visibility.go`
- Test: `internal/domain/match/service/visibility_test.go`

- [ ] **Step 1: Write the failing test (append)**

```go
func TestComputeVisibility_WallOccludes(t *testing.T) {
	// Vertical wall at x=10 from y=-5..5. Origin at (0,0).
	// A point behind the wall (20,0) must NOT be visible; (5,0) in front must be.
	walls := []WallSegmentLOS{
		{ID: "w", P1: Point2D{10, -5}, P2: Point2D{10, 5}, Direction: "both"},
	}
	poly := ComputeVisibilityPolygon(Point2D{0, 0}, walls)
	if PointInPolygon(Point2D{20, 0}, poly.Vertices) {
		t.Fatal("point behind wall must be occluded")
	}
	if !PointInPolygon(Point2D{5, 0}, poly.Vertices) {
		t.Fatal("point in front of wall must be visible")
	}
}

func TestComputeVisibility_NoWalls_SeesFar(t *testing.T) {
	poly := ComputeVisibilityPolygon(Point2D{0, 0}, nil)
	if !PointInPolygon(Point2D{100, 100}, poly.Vertices) {
		t.Fatal("with no walls, far point should be visible")
	}
}

func TestComputeVisibility_OneWay_BlocksFromOneSide(t *testing.T) {
	// One-way wall blocking only its "left" side relative to P1->P2.
	// P1->P2 points +Y; left side (by (P2-P1) x (origin-P1) sign) is -X.
	// Origin on the blocking side should be occluded; on the other side, not.
	wallLeft := []WallSegmentLOS{
		{ID: "w", P1: Point2D{0, -5}, P2: Point2D{0, 5}, Direction: "left"},
	}
	// Origin at +X looking at the wall: depending on side it blocks or not.
	polyA := ComputeVisibilityPolygon(Point2D{10, 0}, wallLeft)
	polyB := ComputeVisibilityPolygon(Point2D{-10, 0}, wallLeft)
	behindFromA := PointInPolygon(Point2D{-1, 0}, polyA.Vertices)
	behindFromB := PointInPolygon(Point2D{1, 0}, polyB.Vertices)
	// Exactly one side must be blocked (XOR): the wall is one-way.
	if behindFromA == behindFromB {
		t.Fatalf("one-way wall must block from exactly one side (A=%v B=%v)", behindFromA, behindFromB)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/match/service/ -run TestComputeVisibility -v`
Expected: FAIL — `ComputeVisibilityPolygon` undefined.

- [ ] **Step 3: Write the implementation (append to visibility.go)**

```go
import "math" // add to the existing import block

const visRadius = 1e7 // effective "infinite" ray length in world units

// blocksFor reports whether a one-way wall occludes for an origin on its given side.
// For Direction "both" it always blocks. For "left"/"right" it blocks only when the
// origin is on that side of the directed segment P1->P2 (sign of the 2D cross product).
func (w WallSegmentLOS) blocksFor(origin Point2D) bool {
	if w.Direction == mapentity.WallDirectionBoth || w.Direction == "" {
		return true
	}
	cross := (w.P2.X-w.P1.X)*(origin.Y-w.P1.Y) - (w.P2.Y-w.P1.Y)*(origin.X-w.P1.X)
	if w.Direction == mapentity.WallDirectionLeft {
		return cross > 0
	}
	return cross < 0 // right
}

// rayHit returns the distance t>=0 along the ray (origin + t*dir, dir unit) at which it
// first crosses segment [a,b], and whether it hits. Uses standard segment intersection.
func rayHit(origin, dir, a, b Point2D) (float64, bool) {
	// Ray: origin + t*dir, t in [0, +inf). Segment: a + u*(b-a), u in [0,1].
	sx, sy := b.X-a.X, b.Y-a.Y
	denom := dir.X*sy - dir.Y*sx
	if math.Abs(denom) < 1e-12 {
		return 0, false // parallel
	}
	ax, ay := a.X-origin.X, a.Y-origin.Y
	t := (ax*sy - ay*sx) / denom
	u := (ax*dir.Y - ay*dir.X) / denom
	if t >= 1e-9 && u >= -1e-9 && u <= 1+1e-9 {
		return t, true
	}
	return 0, false
}

// ComputeVisibilityPolygon performs an angular sweep from origin over the blocking walls.
func ComputeVisibilityPolygon(origin Point2D, walls []WallSegmentLOS) VisibilityPolygon {
	// Active blockers for this origin (respect one-way).
	active := make([]WallSegmentLOS, 0, len(walls))
	for _, w := range walls {
		if w.blocksFor(origin) {
			active = append(active, w)
		}
	}

	// Candidate angles: toward every endpoint, plus +/- epsilon to round corners.
	const eps = 1e-5
	angles := make([]float64, 0, len(active)*6)
	for _, w := range active {
		for _, p := range [2]Point2D{w.P1, w.P2} {
			base := math.Atan2(p.Y-origin.Y, p.X-origin.X)
			angles = append(angles, base-eps, base, base+eps)
		}
	}
	if len(angles) == 0 {
		// No walls: return a large diamond so everything is "visible".
		return VisibilityPolygon{Origin: origin, Vertices: []Point2D{
			{origin.X + visRadius, origin.Y},
			{origin.X, origin.Y + visRadius},
			{origin.X - visRadius, origin.Y},
			{origin.X, origin.Y - visRadius},
		}}
	}

	sortFloats(angles)

	verts := make([]Point2D, 0, len(angles))
	for _, ang := range angles {
		dir := Point2D{math.Cos(ang), math.Sin(ang)}
		best := visRadius
		for _, w := range active {
			if t, ok := rayHit(origin, dir, w.P1, w.P2); ok && t < best {
				best = t
			}
		}
		verts = append(verts, Point2D{origin.X + dir.X*best, origin.Y + dir.Y*best})
	}
	return VisibilityPolygon{Origin: origin, Vertices: verts}
}

// sortFloats sorts ascending without pulling in sort.Slice closures in a hot path.
func sortFloats(a []float64) {
	for i := 1; i < len(a); i++ {
		v := a[i]
		j := i - 1
		for j >= 0 && a[j] > v {
			a[j+1] = a[j]
			j--
		}
		a[j+1] = v
	}
}
```

> Note: vertices are already in angular order (angles sorted), so the polygon winds
> correctly for `PointInPolygon`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/domain/match/service/ -run TestComputeVisibility -v`
Expected: PASS (all three).

- [ ] **Step 5: Commit**

```bash
git add internal/domain/match/service/visibility.go internal/domain/match/service/visibility_test.go
git commit -m "feat(fog): implement angular-sweep visibility polygon with one-way support"
```

---

## Task 6: CellsInPolygon

**Files:**
- Modify: `internal/domain/match/service/visibility.go`
- Test: `internal/domain/match/service/visibility_test.go`

- [ ] **Step 1: Write the failing test (append)**

```go
import (
	// add alongside existing imports:
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
)

func TestCellsInPolygon_Square(t *testing.T) {
	g := mapentity.GridShape{Kind: mapentity.GridKindSquare, Cols: 10, Rows: 10, CellSize: 64, SkewRatio: 1}
	// A polygon covering roughly cells (0,0),(1,0),(0,1),(1,1) centers.
	poly := VisibilityPolygon{Vertices: []Point2D{{0, 0}, {128, 0}, {128, 128}, {0, 128}}}
	cells := CellsInPolygon(poly, g)
	if len(cells) != 4 {
		t.Fatalf("want 4 cells, got %d (%+v)", len(cells), cells)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/match/service/ -run TestCellsInPolygon -v`
Expected: FAIL — undefined.

- [ ] **Step 3: Write the implementation (append to visibility.go)**

```go
import mapservice "github.com/422UR4H/HxH_RPG_System/internal/domain/map/service" // add to import block

// CellsInPolygon returns the grid cells whose center falls inside poly.
// Bounds are derived from the polygon's world AABB mapped back to slot space.
func CellsInPolygon(poly VisibilityPolygon, grid mapentity.GridShape) []fog.CellCoord {
	if len(poly.Vertices) == 0 {
		return nil
	}
	minX, minY := poly.Vertices[0].X, poly.Vertices[0].Y
	maxX, maxY := minX, minY
	for _, v := range poly.Vertices {
		minX = math.Min(minX, v.X)
		minY = math.Min(minY, v.Y)
		maxX = math.Max(maxX, v.X)
		maxY = math.Max(maxY, v.Y)
	}
	// Convert the four AABB corners to slot space and take the integer envelope.
	aLo, bLo, aHi, bHi := slotEnvelope(minX, minY, maxX, maxY, grid)

	out := make([]fog.CellCoord, 0, 32)
	for a := aLo; a <= aHi; a++ {
		for b := bLo; b <= bHi; b++ {
			cx, cy := mapservice.SlotCenterToWorld(a, b, grid)
			if PointInPolygon(Point2D{cx, cy}, poly.Vertices) {
				out = append(out, fog.CellCoord{A: a, B: b})
			}
		}
	}
	return out
}

func slotEnvelope(minX, minY, maxX, maxY float64, g mapentity.GridShape) (aLo, bLo, aHi, bHi int) {
	corners := [4][2]float64{{minX, minY}, {maxX, minY}, {minX, maxY}, {maxX, maxY}}
	first := true
	for _, c := range corners {
		a, b := mapservice.WorldToSlot(c[0], c[1], g)
		if first {
			aLo, aHi, bLo, bHi, first = a, a, b, b, false
			continue
		}
		aLo, aHi = minInt(aLo, a), maxInt(aHi, a)
		bLo, bHi = minInt(bLo, b), maxInt(bHi, b)
	}
	// Pad by 1 to cover cells whose center lies just inside near the AABB edge.
	return aLo - 1, bLo - 1, aHi + 1, bHi + 1
}

func minInt(a, b int) int { if a < b { return a }; return b }
func maxInt(a, b int) int { if a > b { return a }; return b }
```

> Import check: this introduces a `match/service` → `map/service` dependency. That is
> acceptable (match depends on map). Verify no import cycle with `go vet`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/domain/match/service/ -run TestCellsInPolygon -v && go vet ./internal/domain/match/service/`
Expected: PASS, vet clean.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/match/service/visibility.go internal/domain/match/service/visibility_test.go
git commit -m "feat(fog): add CellsInPolygon for explored-cell tracking"
```

---

## Task 7: WallSegment.Revealed + TacticalMap.FogMode + mapper + validator

**Files:**
- Modify: `internal/domain/map/entity/wall_segment.go`
- Modify: `internal/domain/map/entity/map.go`
- Modify: `internal/gateway/pg/maps_mapper.go`
- Modify: `internal/domain/map/service/map_validator.go`
- Test: add to existing `maps` mapper/validator test files (or create `maps_mapper_fog_test.go`)

- [ ] **Step 1: Write the failing test**

Create `internal/gateway/pg/maps_mapper_fog_test.go`:
```go
package pg

import (
	"testing"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
)

func TestMapper_FogModeRoundTrip(t *testing.T) {
	m := entity.TacticalMap{FogMode: fog.FogModeExplored, Grid: entity.DefaultGrid()}
	row := mapToRow(m) // existing mapper entry point — adjust name to match codebase
	back := rowToMap(row)
	if back.FogMode != fog.FogModeExplored {
		t.Fatalf("want explored, got %q", back.FogMode)
	}
}
```
> If the mapper entry-point names differ, align the test to the real functions in
> `maps_mapper.go`. The assertion (fog_mode survives a round trip) is what matters.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/gateway/pg/ -run TestMapper_FogModeRoundTrip -v`
Expected: FAIL — `TacticalMap.FogMode` undefined / not mapped.

- [ ] **Step 3: Write the implementation**

In `internal/domain/map/entity/wall_segment.go`, add to the struct:
```go
	Revealed bool // secret_door revealed to all players by the master
	// TODO(future): RevealedTo map[uuid.UUID]bool for per-player reveal via examine
```

In `internal/domain/map/entity/map.go`, add to `TacticalMap`:
```go
	FogMode fog.FogMode // default FogModeExplored when zero
```
Add the import `"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"`.
> Import-cycle check: `map/entity` importing `match/entity/fog` must not cycle. If
> `match/entity/fog` imports `map/entity` (it does — for `GridKind`), this WOULD cycle.
> **Resolution:** move `FogMode` constants reference out — store `FogMode` on the map as a
> plain `string` typed via a local alias, OR keep `fog.FogMode` but ensure `fog` does not
> import `map/entity`. Re-check Task 1: `PlayerFogState` imports `map/entity` for `GridKind`.
> To avoid the cycle, define `FogMode` and `CellCoord` WITHOUT importing `map/entity`
> (they don't need it), and keep `GridKind` on `PlayerFogState` as `string`. Apply this
> adjustment now: change Task 1 files so package `fog` imports nothing from `map/entity`
> (use `string` for GridKind), letting `map/entity` import `fog` safely.

Apply the cycle fix: in `player_fog_state.go` replace `mapentity.GridKind` with `string`
and drop the `mapentity` import; update `NewPlayerFogState` signature to take `gridKind string`.
Update Task 1 test accordingly (`mapentity.GridKindSquare` → `string(entity.GridKindSquare)`
or the literal `"square"`).

In `internal/gateway/pg/maps_mapper.go`: serialize/deserialize `fog_mode`. When reading,
default empty → `fog.FogModeExplored`:
```go
// when building the entity from the row:
m.FogMode = fog.FogMode(row.FogMode)
if !m.FogMode.IsValid() {
	m.FogMode = fog.FogModeExplored
}
// when building the row from the entity:
row.FogMode = string(m.FogMode)
```
Add a `FogMode string` column field to the row struct and include it in the SQL
INSERT/UPDATE/SELECT column lists in the maps repository.

In `internal/domain/map/service/map_validator.go`, add:
```go
if m.FogMode != "" && !m.FogMode.IsValid() {
	return ErrInvalidFogMode // define alongside existing validator errors
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/gateway/pg/ -run TestMapper_FogModeRoundTrip -v && go vet ./...`
Expected: PASS, no import cycle.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/map/ internal/gateway/pg/
git commit -m "feat(map): add WallSegment.Revealed and TacticalMap.FogMode with mapper/validator"
```

---

## Task 8: MaskSecretDoorForPlayer

**Files:**
- Create: `internal/domain/match/service/mask_secret_door.go`
- Test: `internal/domain/match/service/mask_secret_door_test.go`

- [ ] **Step 1: Write the failing test**

```go
package service

import (
	"testing"

	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
)

func TestMaskSecretDoor_LooksLikeWallKeepsCombatFields(t *testing.T) {
	sub := mapentity.DoorSubtypeBasic
	sd := mapentity.WallSegment{
		ID: "w1", WallType: mapentity.WallTypeSecretDoor, Material: mapentity.WallMaterialStone,
		DoorSubtype: &sub, Open: true, Locked: true, HP: 80, MaxHP: 100, Resistance: 5,
	}
	m := MaskSecretDoorForPlayer(sd)
	if m.WallType != mapentity.WallTypeWall {
		t.Fatal("masked type must be wall")
	}
	if m.DoorSubtype != nil || m.Open || m.Locked {
		t.Fatal("masked door must not leak subtype/open/locked")
	}
	if m.ID != "w1" || m.HP != 80 || m.MaxHP != 100 || m.Resistance != 5 || m.Material != mapentity.WallMaterialStone {
		t.Fatal("masked wall must keep id/material/hp for combat parity")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/match/service/ -run TestMaskSecretDoor -v`
Expected: FAIL — undefined.

- [ ] **Step 3: Write the implementation**

```go
package service

import mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"

// MaskSecretDoorForPlayer returns a copy of a secret door that looks like a plain wall,
// preserving all combat-relevant fields (id, material, hp, resistance, move, sense, direction).
func MaskSecretDoorForPlayer(w mapentity.WallSegment) mapentity.WallSegment {
	masked := w
	masked.WallType = mapentity.WallTypeWall
	masked.DoorSubtype = nil
	masked.WindowSubtype = nil
	masked.Open = false
	masked.Locked = false
	return masked
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/domain/match/service/ -run TestMaskSecretDoor -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/match/service/mask_secret_door.go internal/domain/match/service/mask_secret_door_test.go
git commit -m "feat(fog): add MaskSecretDoorForPlayer"
```

---

## Task 9: FilterMapState

**Files:**
- Create: `internal/domain/match/service/filter_map_state.go`
- Test: `internal/domain/match/service/filter_map_state_test.go`

- [ ] **Step 1: Write the failing test**

```go
package service

import (
	"testing"

	"github.com/google/uuid"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
)

func TestFilterMapState_PlayerSeesOnlyVisibleAndOwn(t *testing.T) {
	player := uuid.New()
	grid := mapentity.GridShape{Kind: mapentity.GridKindSquare, CellSize: 64, SkewRatio: 1}
	// Visibility polygon covering a small box near origin.
	polys := []VisibilityPolygon{{Vertices: []Point2D{{0, 0}, {100, 0}, {100, 100}, {0, 100}}}}

	pieces := []PieceVisibility{
		{ID: "own", CharacterID: "cOwn", Pos: Point2D{500, 500}, Visible: true},   // far but own → seen
		{ID: "seen", CharacterID: "cE1", Pos: Point2D{50, 50}, Visible: true},     // in poly → seen
		{ID: "hiddenGeom", CharacterID: "cE2", Pos: Point2D{500, 500}, Visible: true}, // far, not own → not seen
		{ID: "invisible", CharacterID: "cE3", Pos: Point2D{50, 50}, Visible: false},    // visible=false → never
	}
	charToPlayer := map[string]uuid.UUID{"cOwn": player}

	sd := mapentity.WallSegment{ID: "sd", WallType: mapentity.WallTypeSecretDoor, P1: [2]float64{10, 10}, P2: [2]float64{20, 10}}
	normal := mapentity.WallSegment{ID: "n", WallType: mapentity.WallTypeWall, P1: [2]float64{10, 20}, P2: [2]float64{20, 20}}
	far := mapentity.WallSegment{ID: "f", WallType: mapentity.WallTypeWall, P1: [2]float64{500, 500}, P2: [2]float64{520, 500}}

	walls, visIDs := FilterMapState(
		[]mapentity.WallSegment{sd, normal, far}, pieces, polys, nil,
		fog.FogModeLive, grid, player, charToPlayer, false,
	)

	if !visIDs["own"] || !visIDs["seen"] {
		t.Fatal("own and in-LOS pieces must be visible")
	}
	if visIDs["hiddenGeom"] || visIDs["invisible"] {
		t.Fatal("out-of-LOS non-own and invisible pieces must be hidden")
	}
	// Walls: sd masked to wall, normal kept, far dropped.
	byID := map[string]mapentity.WallSegment{}
	for _, w := range walls {
		byID[w.ID] = w
	}
	if _, ok := byID["f"]; ok {
		t.Fatal("far wall out of LOS must be dropped")
	}
	if byID["sd"].WallType != mapentity.WallTypeWall {
		t.Fatal("secret door must be masked as wall for player")
	}
	if _, ok := byID["n"]; !ok {
		t.Fatal("visible normal wall must be kept")
	}
}

func TestFilterMapState_MasterSeesEverythingUnmasked(t *testing.T) {
	grid := mapentity.GridShape{Kind: mapentity.GridKindSquare, CellSize: 64, SkewRatio: 1}
	sd := mapentity.WallSegment{ID: "sd", WallType: mapentity.WallTypeSecretDoor}
	pieces := []PieceVisibility{{ID: "p", CharacterID: "c", Pos: Point2D{9, 9}, Visible: false}}
	walls, visIDs := FilterMapState([]mapentity.WallSegment{sd}, pieces, nil, nil,
		fog.FogModeLive, grid, uuid.New(), nil, true)
	if walls[0].WallType != mapentity.WallTypeSecretDoor {
		t.Fatal("master must see real secret door type")
	}
	if !visIDs["p"] {
		t.Fatal("master must see invisible pieces")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/match/service/ -run TestFilterMapState -v`
Expected: FAIL — undefined.

- [ ] **Step 3: Write the implementation**

```go
package service

import (
	"github.com/google/uuid"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	mapservice "github.com/422UR4H/HxH_RPG_System/internal/domain/map/service"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
)

// PieceVisibility is the domain projection of a piece for filtering (no delivery types).
type PieceVisibility struct {
	ID          string
	CharacterID string
	Pos         Point2D
	Visible     bool
}

// FilterMapState applies visibility policy. Returns walls filtered/masked for the viewer
// and the set of piece IDs the viewer may see. Master gets everything unmasked.
func FilterMapState(
	allWalls []mapentity.WallSegment,
	pieces []PieceVisibility,
	polys []VisibilityPolygon,
	explored map[fog.CellCoord]struct{},
	fogMode fog.FogMode,
	grid mapentity.GridShape,
	playerID uuid.UUID,
	charToPlayer map[string]uuid.UUID,
	isMaster bool,
) (walls []mapentity.WallSegment, visiblePieceIDs map[string]bool) {
	visiblePieceIDs = make(map[string]bool, len(pieces))

	if isMaster {
		walls = append(walls, allWalls...)
		for _, p := range pieces {
			visiblePieceIDs[p.ID] = true
		}
		return walls, visiblePieceIDs
	}

	// Pieces: visible=false never; else in-LOS or own.
	for _, p := range pieces {
		if !p.Visible {
			continue
		}
		own := charToPlayer[p.CharacterID] == playerID && charToPlayer[p.CharacterID] != uuid.Nil
		if own || IsVisible(p.Pos, polys) {
			visiblePieceIDs[p.ID] = true
		}
	}

	// Walls: in-LOS or (explored mode and midpoint cell explored).
	for _, w := range allWalls {
		mid := Point2D{(w.P1[0] + w.P1[0]) / 2, (w.P1[1] + w.P2[1]) / 2}
		mid = Point2D{(w.P1[0] + w.P2[0]) / 2, (w.P1[1] + w.P2[1]) / 2}
		seen := IsVisible(mid, polys)
		if !seen && fogMode == fog.FogModeExplored && explored != nil {
			a, b := mapservice.WorldToSlot(mid.X, mid.Y, grid)
			if _, ok := explored[fog.CellCoord{A: a, B: b}]; ok {
				seen = true
			}
		}
		if !seen {
			continue
		}
		if w.WallType == mapentity.WallTypeSecretDoor && !w.Revealed {
			walls = append(walls, MaskSecretDoorForPlayer(w))
		} else {
			walls = append(walls, w)
		}
	}
	return walls, visiblePieceIDs
}
```
> Remove the duplicated `mid` line — keep only the `(w.P1[0]+w.P2[0])/2` version. (The
> first line is shown only to flag the common typo; delete it.)

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/domain/match/service/ -run TestFilterMapState -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/match/service/filter_map_state.go internal/domain/match/service/filter_map_state_test.go
git commit -m "feat(fog): add FilterMapState visibility policy service"
```

---

## Task 10: Fog repository interface + PG stub

**Files:**
- Create: `internal/application/fog/i_fog_state_repository.go`
- Create: `internal/gateway/pg/fog/fog_state_repository.go`

- [ ] **Step 1: Write the interface (no test — interface only)**

`internal/application/fog/i_fog_state_repository.go`:
```go
package fog

import (
	"context"

	"github.com/google/uuid"
	fogentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
)

// IFogStateRepository persists per-player explored-area state.
// Implementation is a Phase 10-D TODO (see gateway/pg/fog).
type IFogStateRepository interface {
	Upsert(ctx context.Context, state fogentity.PlayerFogState) error
	FindByMatchMap(ctx context.Context, matchID, mapID uuid.UUID) ([]fogentity.PlayerFogState, error)
	DeleteByMatch(ctx context.Context, matchID uuid.UUID) error
}
```

- [ ] **Step 2: Write the PG stub**

`internal/gateway/pg/fog/fog_state_repository.go`:
```go
package fog

import (
	"context"

	"github.com/google/uuid"
	fogentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
)

// FogStateRepository is the Postgres-backed fog persistence.
// TODO(10-D persistence): implement Upsert/FindByMatchMap/DeleteByMatch against
// player_fog_states (serialize ExploredCells as JSONB [[a,b],...]). Until then the
// session keeps fog state in memory; start_match seeding works via the in-memory path.
type FogStateRepository struct {
	// db *pgxpool.Pool  // wire when implementing
}

func NewFogStateRepository( /* db *pgxpool.Pool */ ) *FogStateRepository {
	return &FogStateRepository{}
}

func (r *FogStateRepository) Upsert(ctx context.Context, state fogentity.PlayerFogState) error {
	// TODO: INSERT ... ON CONFLICT (match_id, map_id, player_id) DO UPDATE.
	return nil
}

func (r *FogStateRepository) FindByMatchMap(ctx context.Context, matchID, mapID uuid.UUID) ([]fogentity.PlayerFogState, error) {
	// TODO: SELECT ... WHERE match_id=$1 AND map_id=$2.
	return nil, nil
}

func (r *FogStateRepository) DeleteByMatch(ctx context.Context, matchID uuid.UUID) error {
	// TODO: DELETE FROM player_fog_states WHERE match_id=$1.
	return nil
}
```

- [ ] **Step 3: Verify it builds**

Run: `go build ./internal/application/fog/ ./internal/gateway/pg/fog/ && go vet ./internal/application/fog/ ./internal/gateway/pg/fog/`
Expected: builds clean.

- [ ] **Step 4: Commit**

```bash
git add internal/application/fog/ internal/gateway/pg/fog/
git commit -m "feat(fog): add fog repository interface and PG stub (impl TODO)"
```

---

## Task 11: Migration

**Files:**
- Create: `migrations/<next-number>_fog_state.sql` (match the repo's goose naming/dir)

- [ ] **Step 1: Inspect existing migrations for naming/dir**

Run: `ls migrations/ | tail -5`
Note the numbering scheme and goose directive style used by neighbors.

- [ ] **Step 2: Write the migration**

```sql
-- +goose Up
ALTER TABLE maps ADD COLUMN fog_mode VARCHAR(10) NOT NULL DEFAULT 'explored';

CREATE TABLE player_fog_states (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id    UUID        NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    map_id      UUID        NOT NULL REFERENCES maps(id)    ON DELETE CASCADE,
    player_id   UUID        NOT NULL,
    grid_kind   VARCHAR(10) NOT NULL,
    explored    JSONB       NOT NULL DEFAULT '[]',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(match_id, map_id, player_id)
);
CREATE INDEX idx_player_fog_states_match_map ON player_fog_states(match_id, map_id);

-- +goose Down
DROP TABLE IF EXISTS player_fog_states;
ALTER TABLE maps DROP COLUMN IF EXISTS fog_mode;
```

- [ ] **Step 3: Apply locally and verify**

Run the project's migration command (see AGENTS.md; e.g. `make migrate-up` or the goose
invocation used elsewhere). Verify `\d maps` shows `fog_mode` and `\d player_fog_states`
exists.
Expected: migration applies cleanly; `down` then `up` round-trips.

- [ ] **Step 4: Commit**

```bash
git add migrations/
git commit -m "feat(fog): migration for maps.fog_mode and player_fog_states"
```

---

## Task 12: MatchSession fog state + grid + methods

**Files:**
- Modify: `internal/domain/match/matchsession/match_session.go`
- Test: `internal/domain/match/matchsession/match_session_test.go`

- [ ] **Step 1: Write the failing test (append)**

```go
func TestSession_RecomputeVisibility_SeedsExploredAndReturnsDelta(t *testing.T) {
	// Build a session with one participant whose piece sits at world (50,50),
	// fog mode explored, a square grid, and no walls (sees a wide area).
	s := newTestSessionForFog(t) // helper: 1 participant (playerID known), grid square 64
	playerID := s.testPlayerID   // expose via helper

	s.SyncMapState(nil, mapentity.GridShape{Kind: mapentity.GridKindSquare, Cols: 20, Rows: 20, CellSize: 64, SkewRatio: 1})
	s.SyncFogStates(nil, fog.FogModeExplored)
	s.SetTestPiece(playerID, service.Point2D{X: 50, Y: 50}) // helper seeds a piece position source

	polys, delta, err := s.RecomputeVisibility(playerID)
	if err != nil {
		t.Fatal(err)
	}
	if len(polys) == 0 || len(delta) == 0 {
		t.Fatalf("want non-empty polys and explored delta, got polys=%d delta=%d", len(polys), len(delta))
	}
	// Second recompute from the same spot yields no new cells.
	_, delta2, _ := s.RecomputeVisibility(playerID)
	if len(delta2) != 0 {
		t.Fatalf("re-recompute from same position should add no cells, got %d", len(delta2))
	}
}
```
> This test needs a small amount of session test scaffolding (a way to inject a piece
> position per player). How the session reads piece positions depends on where pieces
> live. **Decision for this task:** the session reads piece world-positions through a
> `PiecePositionSource` it is given, decoupling it from `room.go`'s payloads. Define it in
> Step 3 and provide the helper. Keep the test aligned to the real helper you write.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/match/matchsession/ -run TestSession_RecomputeVisibility -v`
Expected: FAIL — methods/fields undefined.

- [ ] **Step 3: Write the implementation**

Add fields (replace `gridSize float64` with `grid GridShape`; keep `GetGridSize`):
```go
import (
	mapservice "github.com/422UR4H/HxH_RPG_System/internal/domain/map/service"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
	// service already imported as the match service package
)

// Inside MatchSession struct:
	grid         mapentity.GridShape
	fogMode      fog.FogMode
	fogStates    map[uuid.UUID]*fog.PlayerFogState
	visCache     map[uuid.UUID][]service.VisibilityPolygon
	charToPlayer map[string]uuid.UUID
	pieceSource  PiecePositionSource
```

> If `match_session.go` cannot import `match/service` due to an existing cycle (service
> imports session?), invert: define `VisibilityPolygon` usage via the `service` package
> only (session → service is the natural direction; service must NOT import session).
> Verify with `go vet`. The 10-C `TargetReader` already has session implementing a
> service interface, so session→service imports are fine.

```go
// PiecePositionSource yields the current world positions of a player's pieces.
// Implemented by room.go (which owns the live piece payloads), keeping the session
// decoupled from delivery types.
type PiecePositionSource interface {
	PlayerPiecePositions(playerID uuid.UUID) []service.Point2D
}

func (s *MatchSession) SetPieceSource(src PiecePositionSource) { s.pieceSource = src }

func (s *MatchSession) SyncMapState(walls []mapentity.WallSegment, grid mapentity.GridShape) {
	s.walls = make(map[string]mapentity.WallSegment, len(walls))
	for _, w := range walls {
		s.walls[w.ID] = w
	}
	s.grid = grid
}

func (s *MatchSession) GetGrid() mapentity.GridShape { return s.grid }
func (s *MatchSession) GetGridSize() float64         { return s.grid.CellSize }

func (s *MatchSession) SyncFogStates(states []fog.PlayerFogState, mode fog.FogMode) {
	s.fogMode = mode
	s.fogStates = make(map[uuid.UUID]*fog.PlayerFogState, len(states))
	for i := range states {
		st := states[i]
		s.fogStates[st.PlayerID] = &st
	}
	s.visCache = make(map[uuid.UUID][]service.VisibilityPolygon)
}

func (s *MatchSession) fogStateFor(playerID uuid.UUID) *fog.PlayerFogState {
	if s.fogStates == nil {
		s.fogStates = make(map[uuid.UUID]*fog.PlayerFogState)
	}
	st, ok := s.fogStates[playerID]
	if !ok {
		st = fog.NewPlayerFogState(playerID, s.matchUUID, uuid.Nil, string(s.grid.Kind))
		s.fogStates[playerID] = st
	}
	return st
}

// RecomputeVisibility recomputes a player's LOS, caches polygons, unions explored cells
// (explored mode only), and returns the polygons and the newly explored delta.
func (s *MatchSession) RecomputeVisibility(playerID uuid.UUID) ([]service.VisibilityPolygon, []fog.CellCoord, error) {
	losWalls := service.ToLOSWalls(s.GetWalls())
	var positions []service.Point2D
	if s.pieceSource != nil {
		positions = s.pieceSource.PlayerPiecePositions(playerID)
	}
	polys := make([]service.VisibilityPolygon, 0, len(positions))
	var delta []fog.CellCoord
	for _, pos := range positions {
		poly := service.ComputeVisibilityPolygon(pos, losWalls)
		polys = append(polys, poly)
		if s.fogMode == fog.FogModeExplored {
			cells := service.CellsInPolygon(poly, s.grid)
			delta = append(delta, s.fogStateFor(playerID).AddExplored(cells)...)
		}
	}
	if s.visCache == nil {
		s.visCache = make(map[uuid.UUID][]service.VisibilityPolygon)
	}
	s.visCache[playerID] = polys
	return polys, delta, nil
}

func (s *MatchSession) RecomputeAllVisibility() error {
	for pid := range s.participants {
		if _, _, err := s.RecomputeVisibility(pid); err != nil {
			return err
		}
	}
	return nil
}

func (s *MatchSession) GetVisibility(playerID uuid.UUID) []service.VisibilityPolygon {
	return s.visCache[playerID]
}

func (s *MatchSession) InvalidateVisibilityCache() {
	s.visCache = make(map[uuid.UUID][]service.VisibilityPolygon)
}

func (s *MatchSession) RevealSecretDoor(wallID string) {
	if w, ok := s.walls[wallID]; ok {
		w.Revealed = true
		s.walls[wallID] = w
	}
	s.InvalidateVisibilityCache()
}

func (s *MatchSession) GetFogMode() fog.FogMode { return s.fogMode }
func (s *MatchSession) GetCharToPlayer() map[string]uuid.UUID { return s.charToPlayer }

func (s *MatchSession) GetFogState(playerID uuid.UUID) (*fog.PlayerFogState, bool) {
	st, ok := s.fogStates[playerID]
	return st, ok
}

func (s *MatchSession) GetAllFogStates() []fog.PlayerFogState {
	out := make([]fog.PlayerFogState, 0, len(s.fogStates))
	for _, st := range s.fogStates {
		out = append(out, *st)
	}
	return out
}
```

Build `charToPlayer` in both `NewMatchSession` constructors from `participants`:
```go
charToPlayer := make(map[string]uuid.UUID)
for _, p := range participants {
	if p.Sheet.PlayerUUID != nil && p.Sheet.CharacterID != uuid.Nil { // adjust field names to the real Participant/Sheet
		charToPlayer[p.Sheet.CharacterID.String()] = *p.Sheet.PlayerUUID
	}
}
// assign s.charToPlayer = charToPlayer
```
> Adjust `p.Sheet.CharacterID` to the actual character-id field on the participant's sheet.
> Inspect the `match.Participant` / `CharacterSheet` structs and use the correct accessor.

Update all `SyncMapState` callers in `room.go` to pass a `GridShape` (Task 15 handles the
call sites). Update `NewMatchSession` / `NewMatchSessionWithState` to init the new maps.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/domain/match/matchsession/ -run TestSession_RecomputeVisibility -v && go vet ./internal/domain/match/...`
Expected: PASS, vet clean.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/match/matchsession/
git commit -m "feat(fog): add fog state, grid, and visibility recompute to MatchSession"
```

---

## Task 13: WS messages

**Files:**
- Modify: `internal/app/game/message.go`

- [ ] **Step 1: Add message types and payloads**

Add to the const block:
```go
	MsgTypeVisibilityUpdated MessageType = "visibility_updated"
	MsgTypeWallRevealed      MessageType = "wall_revealed"
```

Add payload types:
```go
type Point2DPayload struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type VisibilityUpdatedPayload struct {
	VisiblePolygons [][]Point2DPayload `json:"visible_polygons"`
	ExploredDelta   [][2]int           `json:"explored_delta,omitempty"`
}

type WallRevealedPayload struct {
	Wall mapentity.WallSegment `json:"wall"`
}

// MapFullStatePayload replaces MapPiecesPayload for map_full_state (server→client).
type MapFullStatePayload struct {
	Pieces          []PieceMovedPayload     `json:"pieces"`
	Walls           []mapentity.WallSegment `json:"walls,omitempty"`
	VisiblePolygons [][]Point2DPayload      `json:"visible_polygons,omitempty"`
	ExploredCells   [][2]int                `json:"explored_cells,omitempty"`
	FogMode         string                  `json:"fog_mode"`
}
```

- [ ] **Step 2: Verify it builds**

Run: `go build ./internal/app/game/`
Expected: builds (existing `map_full_state` send still uses `MapPiecesPayload` until Task 15
rewires it — that's fine; both types compile).

- [ ] **Step 3: Commit**

```bash
git add internal/app/game/message.go
git commit -m "feat(fog): add visibility_updated, wall_revealed, MapFullStatePayload"
```

---

## Task 14: room.go — per-player dispatch + fog wiring

This is the integration task. Sub-steps are sequenced; build after each.

**Files:**
- Modify: `internal/app/game/room.go`
- Test: `internal/app/game/game_test.go`

- [ ] **Step 1: Add `dispatchPerPlayer` + `sendToClient` helpers**

```go
// dispatchPerPlayer sends a per-player-built message to each client. build returns nil to skip.
func (r *Room) dispatchPerPlayer(build func(playerID uuid.UUID, isMaster bool) *Message) {
	r.mu.RLock()
	type entry struct {
		c        *Client
		isMaster bool
	}
	entries := make(map[uuid.UUID]entry, len(r.clients))
	for id, c := range r.clients {
		entries[id] = entry{c: c, isMaster: id == r.masterUUID}
	}
	r.mu.RUnlock()

	for id, e := range entries {
		if msg := build(id, e.isMaster); msg != nil {
			e.c.SendMessage(*msg)
		}
	}
}

func (r *Room) sendToClient(playerID uuid.UUID, msg Message) {
	r.mu.RLock()
	c, ok := r.clients[playerID]
	r.mu.RUnlock()
	if ok {
		c.SendMessage(msg)
	}
}
```

- [ ] **Step 2: Implement `PlayerPiecePositions` (session's PiecePositionSource)**

room.go owns `r.pieces`. Implement the interface so the session can read positions:
```go
// PlayerPiecePositions implements matchsession.PiecePositionSource.
func (r *Room) PlayerPiecePositions(playerID uuid.UUID) []service.Point2D {
	r.mu.RLock()
	defer r.mu.RUnlock()
	grid := r.gridShape()
	out := make([]service.Point2D, 0)
	for _, p := range r.pieces {
		cid := p.CharacterID
		if r.session != nil && r.session.GetCharToPlayer()[cid] != playerID {
			continue
		}
		x, y := slotPayloadToWorld(p.Slot, grid)
		out = append(out, service.Point2D{X: x, Y: y})
	}
	return out
}
```
Add helpers in room.go:
```go
func (r *Room) gridShape() mapentity.GridShape {
	if r.session != nil {
		return r.session.GetGrid()
	}
	return mapentity.GridShape{Kind: mapentity.GridKindSquare, CellSize: r.gridSize, SkewRatio: 1}
}

func slotPayloadToWorld(s SlotPayload, g mapentity.GridShape) (float64, float64) {
	a, b := 0, 0
	if s.Kind == "hex" {
		if s.Q != nil { a = *s.Q }
		if s.R != nil { b = *s.R }
	} else {
		if s.Col != nil { a = *s.Col }
		if s.Row != nil { b = *s.Row }
	}
	return mapservice.SlotCenterToWorld(a, b, g)
}
```
Wire `session.SetPieceSource(r)` right after the session is created in `StartMatch` and in
`RehydrateSession`.
> Keep `r.gridSize float64` as a fallback for pre-session phases, OR replace it with a
> `r.grid mapentity.GridShape` field. Replacing is cleaner — update `map_state_sync`
> handling and `NewRoom` default to set `r.grid = DefaultGrid()` and read
> `r.grid.CellSize` where `r.gridSize` was used (movement blocking). Choose replacement;
> update the `MapStateSyncPayload` handling to set `r.grid` from `payload.Grid`.

- [ ] **Step 3: Extend `MapStateSyncPayload` + `GridSyncEntry` to carry the full grid**

In message.go change `GridSyncEntry` to the full grid (or embed `mapentity.GridShape`):
```go
type MapStateSyncPayload struct {
	Pieces []PieceMovedPayload     `json:"pieces"`
	Walls  []mapentity.WallSegment `json:"walls,omitempty"`
	Grid   *mapentity.GridShape    `json:"grid,omitempty"`
}
```
Update the `MsgTypeMapStateSync` handler in room.go to store `r.grid = *payload.Grid` (guard
nil) and to pass `r.grid` into `session.SyncMapState(walls, r.grid)`.
> Frontend already sends grid on sync (`map_state_sync`); the frontend plan ensures it now
> sends the full grid shape (kind/cellSize/skew/rotation).

- [ ] **Step 4: Filtered `map_full_state` on register and a shared builder**

Add a builder that produces the filtered payload for one viewer, reused by register and
start_match:
```go
func (r *Room) buildMapFullState(playerID uuid.UUID, isMaster bool) *Message {
	r.mu.RLock()
	allWalls := make([]mapentity.WallSegment, 0, len(r.walls))
	for _, w := range r.walls {
		allWalls = append(allWalls, w)
	}
	allPieces := make([]PieceMovedPayload, 0, len(r.pieces))
	pieceProj := make([]service.PieceVisibility, 0, len(r.pieces))
	grid := r.gridShape()
	for _, p := range r.pieces {
		allPieces = append(allPieces, p)
		x, y := slotPayloadToWorld(p.Slot, grid)
		visible := true
		if p.Visible != nil {
			visible = *p.Visible
		}
		pieceProj = append(pieceProj, service.PieceVisibility{
			ID: p.PieceID, CharacterID: p.CharacterID, Pos: service.Point2D{X: x, Y: y}, Visible: visible,
		})
	}
	var polys []service.VisibilityPolygon
	var fogMode fogentity.FogMode = fogentity.FogModeLive
	var explored map[fogentity.CellCoord]struct{}
	charToPlayer := map[string]uuid.UUID{}
	if r.session != nil {
		polys = r.session.GetVisibility(playerID)
		fogMode = r.session.GetFogMode()
		charToPlayer = r.session.GetCharToPlayer()
		if st, ok := r.session.GetFogState(playerID); ok {
			explored = st.ExploredCells
		}
	}
	r.mu.RUnlock()

	walls, visIDs := service.FilterMapState(allWalls, pieceProj, polys, explored, fogMode, grid, playerID, charToPlayer, isMaster)

	pieces := make([]PieceMovedPayload, 0, len(allPieces))
	for _, p := range allPieces {
		if isMaster || visIDs[p.PieceID] {
			pieces = append(pieces, p)
		}
	}
	payload := MapFullStatePayload{Pieces: pieces, Walls: walls, FogMode: string(fogMode)}
	if !isMaster {
		payload.VisiblePolygons = polysToPayload(polys)
		if fogMode == fogentity.FogModeExplored && explored != nil {
			payload.ExploredCells = cellsToPayload(explored)
		}
	}
	msg := NewServerMessage(MsgTypeMapFullState, payload)
	return &msg
}

func polysToPayload(polys []service.VisibilityPolygon) [][]Point2DPayload {
	out := make([][]Point2DPayload, 0, len(polys))
	for _, poly := range polys {
		pts := make([]Point2DPayload, 0, len(poly.Vertices))
		for _, v := range poly.Vertices {
			pts = append(pts, Point2DPayload{X: v.X, Y: v.Y})
		}
		out = append(out, pts)
	}
	return out
}

func cellsToPayload(set map[fogentity.CellCoord]struct{}) [][2]int {
	out := make([][2]int, 0, len(set))
	for c := range set {
		out = append(out, [2]int{c.A, c.B})
	}
	return out
}
```
Replace the existing `sendMapFullState(client)` call in `register` with the filtered path:
```go
if hasPieces {
	r.dispatchPerPlayer(func(pid uuid.UUID, isMaster bool) *Message {
		// only send to the just-registered client here; for register, target that client
		return nil
	})
}
```
> Simpler: on register, send only to the new client:
> `msg := r.buildMapFullState(client.userUUID, r.IsMaster(client.userUUID)); client.SendMessage(*msg)`.
> Replace `sendMapFullState` usage accordingly and delete the old `sendMapFullState` if
> unused. Add imports: `mapservice`, `service` (match service), `fogentity` aliases.

- [ ] **Step 5: start_match fog seeding + filtered broadcast**

In `StartMatch`, after `r.session.SyncMapState(...)` and `r.session.SetPieceSource(r)`:
```go
r.session.SyncFogStates(loadInitialFogStates(), mapFogMode(r))
// loadInitialFogStates(): return nil for now (repo TODO); seeds empty states.
// mapFogMode(r): return the attached map's FogMode; default explored.
for pid := range r.sessionParticipants() {
	if _, _, err := r.session.RecomputeVisibility(pid); err != nil {
		log.Printf("recompute visibility for %s: %v", pid, err)
	}
	// TODO(10-D persistence): fogStateRepo.Upsert(session.GetFogState(pid))
}
```
After broadcasting `match_started`, push filtered state to everyone:
```go
r.dispatchPerPlayer(func(pid uuid.UUID, isMaster bool) *Message {
	return r.buildMapFullState(pid, isMaster)
})
```
> `mapFogMode`: the room needs the attached map's `FogMode`. If the room already loads the
> attached map (Phase 6 `match_maps`), read it there; otherwise default to
> `fogentity.FogModeExplored`. Add `sessionParticipants()` returning the player IDs (from
> session) — or expose `session.PlayerIDs()`. Add that accessor to MatchSession if absent.

- [ ] **Step 6: Per-receiver `piece_moved` / `piece_removed` + visible=false → master only**

Replace the relay loops in `MsgTypePieceMoved` / `MsgTypePieceRemoved` so that:
- `visible=false` pieces (a `*bool` that is non-nil and false) → send only to master.
- otherwise dispatch per-player: send `piece_moved` if the receiver can see the NEW position;
  else if they could see the OLD position, send `piece_removed` (vanish); else nothing.

```go
case MsgTypePieceMoved:
	var payload PieceMovedPayload
	if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
		client.SendMessage(NewErrorMessage("invalid_payload", "invalid piece_moved payload"))
		return
	}
	r.mu.Lock()
	old, hadOld := r.pieces[payload.PieceID]
	r.pieces[payload.PieceID] = payload
	r.mu.Unlock()

	grid := r.gridShape()
	newX, newY := slotPayloadToWorld(payload.Slot, grid)
	var oldX, oldY float64
	if hadOld {
		oldX, oldY = slotPayloadToWorld(old.Slot, grid)
	}
	hidden := payload.Visible != nil && !*payload.Visible

	moved := NewClientMessage(MsgTypePieceMoved, client.userUUID, payload)
	removed := NewClientMessage(MsgTypePieceRemoved, client.userUUID, PieceRemovedPayload{PieceID: payload.PieceID})

	r.dispatchPerPlayer(func(pid uuid.UUID, isMaster bool) *Message {
		if pid == client.userUUID {
			return nil // mover already has it locally
		}
		if isMaster {
			m := moved
			return &m
		}
		if hidden {
			return nil // hidden pieces never go to players
		}
		polys := r.visibilityFor(pid)
		seesNew := service.IsVisible(service.Point2D{X: newX, Y: newY}, polys)
		seesOld := hadOld && service.IsVisible(service.Point2D{X: oldX, Y: oldY}, polys)
		switch {
		case seesNew:
			m := moved
			return &m
		case seesOld:
			m := removed
			return &m
		default:
			return nil
		}
	})
```
Add:
```go
func (r *Room) visibilityFor(pid uuid.UUID) []service.VisibilityPolygon {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.session == nil {
		return nil
	}
	return r.session.GetVisibility(pid)
}
```
Apply the analogous `visible=false → master only` treatment to `MsgTypePieceRemoved`.
> When the mover is a player moving their OWN piece, recompute that player's LOS and resend
> them a filtered full state:
> ```go
> if r.session != nil {
>     if _, _, err := r.session.RecomputeVisibility(client.userUUID); err == nil {
>         msg := r.buildMapFullState(client.userUUID, r.IsMaster(client.userUUID))
>         client.SendMessage(*msg)
>     }
> }
> ```
> Place this after updating `r.pieces`, guarded so it only runs when the moved piece belongs
> to `client.userUUID` (check `charToPlayer[payload.CharacterID] == client.userUUID`).

- [ ] **Step 7: Wall events — recompute, mask secret doors, wall_revealed**

In `broadcastWallResults` and the `applyWallInteract` master path:
- After a wall changes (`UpdateWall`), call `r.session.InvalidateVisibilityCache()` and
  `r.session.RecomputeAllVisibility()`, then push `visibility_updated` to each player.
- `wall_hp_changed`: dispatch per-player — send to a player only if they can see the wall
  (midpoint visible); never leaks type (payload has no type). Master always.
- `wall_state_changed` for an unrevealed secret door: send only to master.
- New: when the master interaction is a reveal (`InteractKind` reveal/examine-success),
  call `session.RevealSecretDoor(wallID)` and `r.broadcast` a `wall_revealed` with the full
  `WallSegment`.

```go
func (r *Room) pushVisibilityUpdates() {
	r.dispatchPerPlayer(func(pid uuid.UUID, isMaster bool) *Message {
		if isMaster {
			return nil
		}
		polys, delta, err := func() ([]service.VisibilityPolygon, []fogentity.CellCoord, error) {
			r.mu.Lock()
			defer r.mu.Unlock()
			if r.session == nil {
				return nil, nil, nil
			}
			return r.session.RecomputeVisibility(pid)
		}()
		if err != nil || polys == nil {
			return nil
		}
		payload := VisibilityUpdatedPayload{VisiblePolygons: polysToPayload(polys)}
		for _, c := range delta {
			payload.ExploredDelta = append(payload.ExploredDelta, [2]int{c.A, c.B})
		}
		msg := NewServerMessage(MsgTypeVisibilityUpdated, payload)
		return &msg
	})
}
```
Add a `wall_hp_changed` visibility gate in `broadcastWallResults`:
```go
// replace the unconditional broadcast of wall_hp_changed with:
mid := service.Point2D{
	X: (wr.UpdatedWall.P1[0] + wr.UpdatedWall.P2[0]) / 2,
	Y: (wr.UpdatedWall.P1[1] + wr.UpdatedWall.P2[1]) / 2,
}
r.dispatchPerPlayer(func(pid uuid.UUID, isMaster bool) *Message {
	if !isMaster && !service.IsVisible(mid, r.visibilityFor(pid)) {
		return nil
	}
	m := NewServerMessage(MsgTypeWallHpChanged, WallHpChangedPayload{
		WallID: wr.UpdatedWall.ID, HP: wr.UpdatedWall.HP, MaxHP: wr.UpdatedWall.MaxHP, Destroyed: wr.UpdatedWall.Destroyed,
	})
	return &m
})
// then call r.pushVisibilityUpdates() once after applying all wall results
```
For `wall_state_changed` of a secret door: gate to master only when `!w.Revealed`.

- [ ] **Step 8: Write room behavior tests**

Append to `internal/app/game/game_test.go` tests that:
- a player NOT in LOS of a moved enemy piece receives no `piece_moved` (or a `piece_removed`
  if it left their view);
- the master always receives `piece_moved`;
- `wall_hp_changed` reaches a player who can see the wall.

Model these on existing `game_test.go` patterns (in-process room with fake clients). Use a
session seeded so one player's visibility excludes a known position.
> If `game_test.go`'s harness makes per-player LOS hard to set up, assert the smaller unit:
> that `buildMapFullState` masks a secret door for a non-master and returns it real for the
> master (call it directly on a constructed Room). Keep at least one test that exercises the
> per-player branch.

- [ ] **Step 9: Build, vet, test**

Run: `go build ./... && go vet ./... && go test ./internal/app/game/ -v`
Expected: builds, vet clean, tests pass.

- [ ] **Step 10: Commit**

```bash
git add internal/app/game/
git commit -m "feat(fog): per-player WS dispatch, LOS filtering, secret-door masking in room"
```

---

## Task 15: REST GET /maps/:id role-aware filtering

**Files:**
- Modify: `internal/app/api/maps_handler.go`
- Test: `internal/app/api/maps_handler_test.go` (or the existing handler test file)

- [ ] **Step 1: Write the failing test**

```go
func TestGetMap_PlayerGetsMaskedSecretDoor(t *testing.T) {
	// Arrange a map with one secret_door; request as a non-master user.
	// Assert the response wall has type "wall" (masked), and as master it stays "secret_door".
	// Use the handler's existing test scaffolding (httptest + mock use case).
}
```
> Fill the test body using the patterns already in the maps handler test file (mocked
> use case returning a map with a secret_door; two requests differing by the auth/role).
> The assertion: player response masks secret_door → wall; master response keeps it.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/app/api/ -run TestGetMap_PlayerGetsMaskedSecretDoor -v`
Expected: FAIL.

- [ ] **Step 3: Write the implementation**

In the `GET /maps/:id` handler, after loading the map, determine the requester's role
(campaign master vs player — reuse the existing campaign/role lookup used elsewhere in the
handlers) and apply filtering before serialization:
```go
isMaster := /* existing role check: requester owns/masters the campaign of this map */
if !isMaster {
	// Build piece projections from the snapshot and the player's LOS from snapshot positions.
	grid := m.Grid
	pieceProj := make([]service.PieceVisibility, 0, len(m.Pieces))
	for _, p := range m.Pieces {
		x, y := mapservice.SlotCenterToWorld(p.Coord.A(), p.Coord.B(), grid) // adapt to Piece coord accessor
		pieceProj = append(pieceProj, service.PieceVisibility{
			ID: p.ID, CharacterID: p.CharacterID, Pos: service.Point2D{X: x, Y: y}, Visible: p.Visible,
		})
	}
	// LOS from the requesting player's own pieces in the snapshot.
	losWalls := service.ToLOSWalls(m.Walls)
	var polys []service.VisibilityPolygon
	for _, proj := range pieceProj {
		if charToPlayer[proj.CharacterID] == requesterID {
			polys = append(polys, service.ComputeVisibilityPolygon(proj.Pos, losWalls))
		}
	}
	// Explored from persisted fog state (empty until repo implemented).
	var explored map[fog.CellCoord]struct{}
	if states, _ := fogRepo.FindByMatchMap(ctx, matchID, m.ID); states != nil {
		for _, st := range states {
			if st.PlayerID == requesterID {
				explored = st.ExploredCells
			}
		}
	}
	walls, visIDs := service.FilterMapState(m.Walls, pieceProj, polys, explored, m.FogMode, grid, requesterID, charToPlayer, false)
	m.Walls = walls
	m.Pieces = filterPiecesByID(m.Pieces, visIDs) // keep only visible piece IDs
}
return m
```
> Several adapters depend on the real shapes:
> - `charToPlayer` for a not-yet-started map: derive from the campaign's enrollments
>   (character→player). If unavailable at REST, pass an empty map — then the player sees
>   only LOS-based geometry from their own pieces if those pieces' ownership can be resolved;
>   at minimum, secret_door masking and visible=false removal still apply.
> - `matchID`: if the map isn't attached to a live match, skip the explored lookup (pass nil).
> - `m.Pieces` coord accessor: adapt `p.Coord.A()/B()` to the real `PieceCoord` shape.
>
> The non-negotiable guarantees this task must deliver, even if LOS-at-REST is partial:
> **secret_door is masked and visible=false pieces are removed for non-masters.** LOS
> refinement is best-effort at REST; WS delivers exact live fog.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/app/api/ -run TestGetMap -v && go vet ./...`
Expected: PASS, vet clean.

- [ ] **Step 5: Commit**

```bash
git add internal/app/api/
git commit -m "feat(fog): role-aware filtering on GET /maps/:id (mask secret doors, hide invisible)"
```

---

## Task 16: Contract docs + documentation map

**Files:**
- Modify: `docs/dev/api/maps.md`
- Modify: `docs/documentation-map.yaml`

- [ ] **Step 1: Document the filtering and WS events**

In `maps.md` add a "Fog of War / visibility" section:
- `GET /maps/:id` returns role-filtered content: master = full; player = secret_door masked
  as `wall`, `visible=false` pieces removed, out-of-LOS walls/pieces removed.
- `maps.fog_mode`: `live | explored`, default `explored`.
- WS events: `map_full_state` (extended: `walls`, `visible_polygons`, `explored_cells`,
  `fog_mode`), `visibility_updated` (`visible_polygons`, `explored_delta`), `wall_revealed`
  (`wall`). Note that `wall_hp_changed` is per-player gated; `wall_state_changed` for an
  unrevealed secret door is master-only.

- [ ] **Step 2: Register in documentation-map.yaml**

Add entries mapping the new code paths (`visibility.go`, `filter_map_state.go`,
`mask_secret_door.go`, `coords.go`, `fog/`, `player_fog_states` migration, new WS payloads)
to `docs/dev/api/maps.md`.

- [ ] **Step 3: Commit**

```bash
git add docs/
git commit -m "docs(fog): document visibility filtering, fog_mode, and new WS events"
```

---

## Task 17: Full verification

- [ ] **Step 1: Vet + full test suite**

Run: `go vet ./... && go test ./...`
Expected: all pass.

- [ ] **Step 2: Smoke test (per CLAUDE.md delivery rule)**

From `System_X_System_Project/`: `./dev-checkout.sh feat/tactical-map-walls-10d`, then curl
`GET /maps/:id` as a player and as the master for a map containing a secret_door; confirm the
player response masks it as `wall` and the master sees `secret_door`.

- [ ] **Step 3: Final commit if any docs/cleanup remain**

```bash
git add -A
git commit -m "chore(fog): backend 10-D cleanup and verification"
```

---

## Self-review notes (already folded into tasks)

- **Import cycle** `map/entity` ↔ `match/entity/fog`: resolved in Task 7 Step 3 — package
  `fog` imports nothing from `map/entity` (`GridKind` stored as `string`), so `map/entity`
  may import `fog`.
- **Domain ↔ delivery decoupling**: `FilterMapState` takes `PieceVisibility` and returns a
  visible-ID set; `room.go` maps payloads ↔ projections (Task 9, Task 14).
- **Session ↔ delivery decoupling**: session reads positions via `PiecePositionSource`
  implemented by `room.go` (Task 12, Task 14).
- **`gridSize float64` → grid shape**: session uses `GridShape`; room replaces `r.gridSize`
  with `r.grid` (Task 12, Task 14 Steps 2–3). `GetGridSize()` preserved for movement blocking.
- **Spec coverage**: data model (T1,7,11), coords (T2), visibility (T3–6), masking (T8),
  filter (T9), persistence scaffold (T10,11,12), WS contract (T13), per-player dispatch +
  fog wiring (T14), REST filter (T15), docs (T16).
- **Out of scope (left as TODO, per spec §10)**: PG repo impl + turn-close upsert; per-player
  reveal via examine; vision radius; in-match turn→move fog integration beyond what is wired.
