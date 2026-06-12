# Tactical Map Fase 10-B — Backend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire interactive wall state (open/locked doors) into the game room's in-memory model, handle `MasterAction.Interact` for door toggling via WS, and validate movement paths against blocking walls.

**Architecture:** `Interact` is a new value object in the `action/` package (same pattern as `Attack`, `Move`). `Room` gains `lobbyWalls map[string]WallSegment` (same pattern as `lobbyPieces`). The master seeds walls via the extended `lobby_state_sync` payload. `enqueue_master_action` checks for `Interact` targeting walls, updates in-memory state, and broadcasts `wall_state_changed`. Movement blocking uses a pure `IsPathBlocked` function in the map service. The `MovePayload` gains a `From [3]int` field so the server can compute the path without looking up piece state.

**Tech Stack:** Go 1.23+, `encoding/json`, standard `testing`, existing patterns in `internal/app/game/`.

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `internal/domain/match/entity/action/interact.go` | **Create** | `Interact` struct + `InteractKind` enum |
| `internal/domain/match/entity/action/action.go` | **Modify** | Add `Interact *Interact` field and param to `NewAction` |
| `internal/domain/match/entity/action/master_action.go` | **Modify** | Add `Interact *Interact` field |
| `internal/domain/match/entity/action/move.go` | **Modify** | Add `From [3]int` source position field |
| `internal/app/game/message.go` | **Modify** | `InteractPayload`, `LobbyStateSyncPayload`, `MsgTypeWallStateChanged`, `WallStateChangedPayload`; extend `MasterActionPayload`, `MasterActionEnqueuedPayload`, `ActionPayload`, `MovePayload` |
| `internal/app/game/action_mapper.go` | **Modify** | Map `InteractPayload` → `action.Interact`; map `MovePayload.From` |
| `internal/domain/map/service/wall_geometry.go` | **Create** | `IsPathBlocked` pure function |
| `internal/domain/map/service/wall_geometry_test.go` | **Create** | Unit tests for `IsPathBlocked` |
| `internal/app/game/room.go` | **Modify** | `lobbyWalls`, `lobbyGridSize`; extend `lobby_state_sync` handler; wall Interact handler; movement blocking |

---

### Task 1: Create `interact.go` domain entity

**Files:**
- Create: `internal/domain/match/entity/action/interact.go`

- [ ] **Step 1: Create the file**

```go
package action

type InteractKind string

const (
	InteractOpen     InteractKind = "open"
	InteractClose    InteractKind = "close"
	InteractToggle   InteractKind = "toggle"
	InteractLockpick InteractKind = "lockpick"
	InteractExamine  InteractKind = "examine"
)

type Interact struct {
	Kind InteractKind
}
```

- [ ] **Step 2: Build**

```bash
cd /home/azzurah/Documentos/HxH_RPG_Environment_Project/System_X_System_Project/System_X_System
go build ./internal/domain/match/entity/action/...
```

Expected: success.

- [ ] **Step 3: Commit**

```bash
git add internal/domain/match/entity/action/interact.go
git commit -m "feat(action): add Interact value object and InteractKind enum"
```

---

### Task 2: Add `Interact` to `Action` and `MasterAction`; add `From` to `Move`

**Files:**
- Modify: `internal/domain/match/entity/action/action.go`
- Modify: `internal/domain/match/entity/action/master_action.go`
- Modify: `internal/domain/match/entity/action/move.go`

- [ ] **Step 1: Add `From` to `Move`**

In `internal/domain/match/entity/action/move.go`, add `From [3]int` after the existing fields:

```go
type Move struct {
	Category   enum.MoveCategory
	From       [3]int // source grid position [col, row, z]; zero = not provided
	Position   [3]int // x, y, z
	Speed      *RollCheck
	Charge     *RollCheck
	FinalSpeed int
}
```

- [ ] **Step 2: Add `Interact *Interact` to `Action`**

In `internal/domain/match/entity/action/action.go`, add the field after `Dodge`:

```go
type Action struct {
	id        uuid.UUID
	actorID   uuid.UUID
	TargetID  []uuid.UUID
	ReactToID uuid.UUID

	Speed  ActionSpeed
	Skills []Skill

	Trigger  *Trigger
	Feint    *RollCheck
	Move     *Move
	Attack   *Attack
	Defense  *Defense
	Dodge    *Dodge
	Interact *Interact

	openedAt    *time.Time //nolint:unused
	confirmedAt *time.Time //nolint:unused
}
```

Add `interact *Interact` as the last parameter of `NewAction` and wire it:

```go
func NewAction(
	actorID uuid.UUID,
	targetID []uuid.UUID,
	reactToID uuid.UUID,
	skills []Skill,
	actionSpeed ActionSpeed,
	feint *RollCheck,
	move *Move,
	attack *Attack,
	defense *Defense,
	dodge *Dodge,
	trigger *Trigger,
	interact *Interact,
) *Action {
	return &Action{
		id:        uuid.New(),
		actorID:   actorID,
		TargetID:  targetID,
		ReactToID: reactToID,
		Skills:    skills,
		Speed:     actionSpeed,
		Feint:     feint,
		Move:      move,
		Attack:    attack,
		Defense:   defense,
		Dodge:     dodge,
		Trigger:   trigger,
		Interact:  interact,
	}
}
```

- [ ] **Step 3: Add `Interact *Interact` to `MasterAction`**

In `internal/domain/match/entity/action/master_action.go`, add after `ActionSpeed`:

```go
type MasterAction struct {
	TargetID    []uuid.UUID
	Skills      []Skill
	Move        *Move
	Attack      *Attack
	ActionSpeed *RollCheck
	Interact    *Interact
	happenedAt  time.Time
}
```

- [ ] **Step 4: Build (expect action_mapper.go error)**

```bash
go build ./...
```

Expected: compilation error in `action_mapper.go` — `NewAction` called with wrong arg count. Fix in Task 4.

- [ ] **Step 5: Commit the domain changes**

```bash
git add internal/domain/match/entity/action/action.go \
        internal/domain/match/entity/action/master_action.go \
        internal/domain/match/entity/action/move.go
git commit -m "feat(action): add Interact to Action/MasterAction; add From to Move"
```

---

### Task 3: Add WS types to `message.go`

**Files:**
- Modify: `internal/app/game/message.go`

- [ ] **Step 1: Add `MsgTypeWallStateChanged` constant**

In the message type constants, add:

```go
// Server → Client (wall events)
MsgTypeWallStateChanged MessageType = "wall_state_changed"
```

- [ ] **Step 2: Add `InteractPayload`**

Append after the existing payload types:

```go
type InteractPayload struct {
	Kind string `json:"kind"` // "open" | "close" | "toggle" | "lockpick" | "examine"
}
```

- [ ] **Step 3: Add `WallStateChangedPayload`**

```go
// WallStateChangedPayload is broadcast to all clients when a wall's open/locked state changes.
type WallStateChangedPayload struct {
	WallID string `json:"wall_id"`
	Open   bool   `json:"open"`
	Locked bool   `json:"locked"`
}
```

- [ ] **Step 4: Extend `MasterActionPayload` and `MasterActionEnqueuedPayload`**

Add `Interact` field to both:

```go
type MasterActionPayload struct {
	TargetIDs   []uuid.UUID          `json:"target_ids"`
	Skills      []ActionSkillPayload `json:"skills,omitempty"`
	Move        *MovePayload         `json:"move,omitempty"`
	Attack      *AttackPayload       `json:"attack,omitempty"`
	ActionSpeed *RollCheckPayload    `json:"action_speed,omitempty"`
	Interact    *InteractPayload     `json:"interact,omitempty"`
}

type MasterActionEnqueuedPayload struct {
	TargetIDs   []uuid.UUID          `json:"target_ids"`
	Skills      []ActionSkillPayload `json:"skills,omitempty"`
	Move        *MovePayload         `json:"move,omitempty"`
	Attack      *AttackPayload       `json:"attack,omitempty"`
	ActionSpeed *RollCheckPayload    `json:"action_speed,omitempty"`
	Interact    *InteractPayload     `json:"interact,omitempty"`
}
```

- [ ] **Step 5: Extend `ActionPayload` with `Interact`**

```go
type ActionPayload struct {
	ReactToID uuid.UUID            `json:"react_to_id,omitempty"`
	TargetID  []uuid.UUID          `json:"target_id,omitempty"`
	Skills    []ActionSkillPayload `json:"skills,omitempty"`
	Speed     *ActionSpeedPayload  `json:"speed,omitempty"`
	Feint     *RollCheckPayload    `json:"feint,omitempty"`
	Move      *MovePayload         `json:"move,omitempty"`
	Attack    *AttackPayload       `json:"attack,omitempty"`
	Defense   *DefensePayload      `json:"defense,omitempty"`
	Dodge     *DodgePayload        `json:"dodge,omitempty"`
	Interact  *InteractPayload     `json:"interact,omitempty"`
}
```

- [ ] **Step 6: Add `From` to `MovePayload`**

```go
type MovePayload struct {
	Category string            `json:"category"`
	From     [3]int            `json:"from,omitempty"` // source grid position [col, row, z]; zero = not provided
	Position [3]int            `json:"position"`
	Speed    *RollCheckPayload `json:"speed,omitempty"`
	Charge   *RollCheckPayload `json:"charge,omitempty"`
}
```

- [ ] **Step 7: Add `LobbyStateSyncPayload`**

The master sends full `WallSegment` objects on connect so the room can validate movement blocking without a DB query. Import `mapentity` at the top of `message.go`:

```go
import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
)
```

Append the payload type:

```go
// LobbyStateSyncPayload extends the original LobbyPiecesPayload to include walls and grid.
// Sent by the master on WS connect to seed the room's in-memory state from the DB.
// Walls are full WallSegment objects so the room can perform movement blocking without
// additional DB queries.
type LobbyStateSyncPayload struct {
	Pieces []LobbyPieceMovedPayload `json:"pieces"`
	Walls  []mapentity.WallSegment  `json:"walls,omitempty"`
	Grid   *GridSyncEntry           `json:"grid,omitempty"`
}

// GridSyncEntry carries the cell size used to convert grid slot coords to world coords.
type GridSyncEntry struct {
	CellSize float64 `json:"cell_size"`
}
```

- [ ] **Step 8: Build**

```bash
go build ./internal/app/game/...
```

Expected: success (action_mapper.go still fails — fixed in Task 4).

- [ ] **Step 9: Commit**

```bash
git add internal/app/game/message.go
git commit -m "feat(game): add Interact/Wall WS types and LobbyStateSyncPayload to message.go"
```

---

### Task 4: Fix `action_mapper.go` — map `Interact` and `Move.From`

**Files:**
- Modify: `internal/app/game/action_mapper.go`

- [ ] **Step 1: Update `buildAction`**

Replace the entire `buildAction` function:

```go
func buildAction(actorID uuid.UUID, p ActionPayload) *action.Action {
	var dodge *action.Dodge
	if p.Dodge != nil {
		var rc action.RollCheck
		if p.Dodge.RollCheck != nil {
			rc = action.RollCheck{SkillName: p.Dodge.RollCheck.SkillName}
		}
		dodge = &action.Dodge{
			Category:  enum.DodgeCategory(p.Dodge.Category),
			RollCheck: rc,
		}
	}
	var move *action.Move
	if p.Move != nil {
		move = &action.Move{
			Category: enum.MoveCategory(p.Move.Category),
			From:     p.Move.From,
			Position: p.Move.Position,
		}
		// TODO: map Speed, Charge, FinalSpeed once frontend contract is finalized
	}
	var interact *action.Interact
	if p.Interact != nil {
		interact = &action.Interact{Kind: action.InteractKind(p.Interact.Kind)}
	}
	// TODO: map Attack, Defense, Feint, Skills, Speed once frontend payload contract is finalized
	return action.NewAction(
		actorID, p.TargetID, p.ReactToID,
		nil, action.ActionSpeed{},
		nil, move, nil, nil, dodge, nil, interact,
	)
}
```

- [ ] **Step 2: Update `buildMasterAction`**

Add Interact mapping (after the existing Attack TODO block):

```go
	if p.Interact != nil {
		ma.Interact = &action.Interact{Kind: action.InteractKind(p.Interact.Kind)}
	}
```

Full function:

```go
func buildMasterAction(masterUUID uuid.UUID, p MasterActionPayload) *action.MasterAction {
	_ = masterUUID
	ma := action.NewMasterAction()
	ma.TargetID = p.TargetIDs
	if p.ActionSpeed != nil {
		ma.ActionSpeed = &action.RollCheck{SkillName: p.ActionSpeed.SkillName}
	}
	for _, s := range p.Skills {
		ma.Skills = append(ma.Skills, action.Skill{SkillName: s.SkillName})
	}
	if p.Move != nil {
		// TODO: map Move fully once frontend contract is finalized
		_ = p.Move
	}
	if p.Attack != nil {
		// TODO: map Attack once frontend contract is finalized
		_ = p.Attack
	}
	if p.Interact != nil {
		ma.Interact = &action.Interact{Kind: action.InteractKind(p.Interact.Kind)}
	}
	return ma
}
```

- [ ] **Step 3: Build clean**

```bash
go build ./...
```

Expected: success.

- [ ] **Step 4: Run tests**

```bash
go test ./internal/...
```

Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add internal/app/game/action_mapper.go
git commit -m "feat(game): map Interact and Move.From in buildAction/buildMasterAction"
```

---

### Task 5: `IsPathBlocked` — TDD

**Files:**
- Create: `internal/domain/map/service/wall_geometry_test.go`
- Create: `internal/domain/map/service/wall_geometry.go`

- [ ] **Step 1: Write failing tests**

Create `internal/domain/map/service/wall_geometry_test.go`:

```go
package service_test

import (
	"testing"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/map/service"
)

func wall(p1, p2 [2]float64) entity.WallSegment {
	return entity.WallSegment{
		ID:        "w",
		P1:        p1,
		P2:        p2,
		WallType:  entity.WallTypeWall,
		Move:      true,
		Direction: entity.WallDirectionBoth,
	}
}

func TestIsPathBlocked_NoWalls(t *testing.T) {
	if service.IsPathBlocked([2]float64{0, 0}, [2]float64{100, 0}, nil) {
		t.Error("expected not blocked with no walls")
	}
}

func TestIsPathBlocked_CrossingWall(t *testing.T) {
	// Vertical wall at x=50, y=0..100; path goes horizontally through it.
	w := wall([2]float64{50, 0}, [2]float64{50, 100})
	if !service.IsPathBlocked([2]float64{0, 50}, [2]float64{100, 50}, []entity.WallSegment{w}) {
		t.Error("expected path to be blocked by crossing wall")
	}
}

func TestIsPathBlocked_ParallelWall(t *testing.T) {
	w := wall([2]float64{0, 20}, [2]float64{100, 20})
	if service.IsPathBlocked([2]float64{0, 50}, [2]float64{100, 50}, []entity.WallSegment{w}) {
		t.Error("expected parallel wall not to block")
	}
}

func TestIsPathBlocked_OpenDoor(t *testing.T) {
	w := wall([2]float64{50, 0}, [2]float64{50, 100})
	w.WallType = entity.WallTypeDoor
	w.Open = true
	if service.IsPathBlocked([2]float64{0, 50}, [2]float64{100, 50}, []entity.WallSegment{w}) {
		t.Error("expected open door not to block movement")
	}
}

func TestIsPathBlocked_MoveFalse(t *testing.T) {
	w := wall([2]float64{50, 0}, [2]float64{50, 100})
	w.Move = false
	if service.IsPathBlocked([2]float64{0, 50}, [2]float64{100, 50}, []entity.WallSegment{w}) {
		t.Error("expected wall with move=false not to block")
	}
}

// Wall vector p1→p2 is (50,0)→(50,100), i.e. pointing downward.
// Cross product of wall vector with (from - p1): positive → from is to the LEFT of p1→p2.
// direction=left means it only blocks from the left side (cross > 0).

func TestIsPathBlocked_DirectionLeft_FromRight(t *testing.T) {
	// from=(100,50) is to the RIGHT of the wall vector → NOT blocked
	w := wall([2]float64{50, 0}, [2]float64{50, 100})
	w.Direction = entity.WallDirectionLeft
	if service.IsPathBlocked([2]float64{100, 50}, [2]float64{0, 50}, []entity.WallSegment{w}) {
		t.Error("direction=left wall should not block from the right side")
	}
}

func TestIsPathBlocked_DirectionLeft_FromLeft(t *testing.T) {
	// from=(0,50) is to the LEFT of the wall vector → blocked
	w := wall([2]float64{50, 0}, [2]float64{50, 100})
	w.Direction = entity.WallDirectionLeft
	if !service.IsPathBlocked([2]float64{0, 50}, [2]float64{100, 50}, []entity.WallSegment{w}) {
		t.Error("direction=left wall should block from the left side")
	}
}
```

- [ ] **Step 2: Run tests to confirm compilation failure**

```bash
go test ./internal/domain/map/service/...
```

Expected: compilation error — `service.IsPathBlocked undefined`.

- [ ] **Step 3: Implement `IsPathBlocked`**

Create `internal/domain/map/service/wall_geometry.go`:

```go
package service

import (
	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
)

// IsPathBlocked reports whether the straight path from→to is blocked by any wall
// that has move=true and open=false. WallDirection is respected: "left" blocks
// only movement originating from the left side of the wall vector p1→p2; "right"
// from the right; "both" from either side.
func IsPathBlocked(from, to [2]float64, walls []entity.WallSegment) bool {
	for _, w := range walls {
		if !w.Move || w.Open {
			continue
		}
		if !segmentsIntersect(from, to, w.P1, w.P2) {
			continue
		}
		if w.Direction == entity.WallDirectionBoth {
			return true
		}
		// Cross product of wall vector (p2-p1) with (from-p1).
		// Positive → from is to the LEFT of the wall direction.
		wx := w.P2[0] - w.P1[0]
		wy := w.P2[1] - w.P1[1]
		fx := from[0] - w.P1[0]
		fy := from[1] - w.P1[1]
		cross := wx*fy - wy*fx
		if w.Direction == entity.WallDirectionLeft && cross > 0 {
			return true
		}
		if w.Direction == entity.WallDirectionRight && cross < 0 {
			return true
		}
	}
	return false
}

// segmentsIntersect reports whether segment AB intersects segment CD.
// Uses the cross-product sign test (standard computational geometry).
func segmentsIntersect(a, b, c, d [2]float64) bool {
	d1 := cross2(c, d, a)
	d2 := cross2(c, d, b)
	d3 := cross2(a, b, c)
	d4 := cross2(a, b, d)

	if ((d1 > 0 && d2 < 0) || (d1 < 0 && d2 > 0)) &&
		((d3 > 0 && d4 < 0) || (d3 < 0 && d4 > 0)) {
		return true
	}
	const eps = 1e-9
	if absF(d1) < eps && onSegment(c, d, a) {
		return true
	}
	if absF(d2) < eps && onSegment(c, d, b) {
		return true
	}
	if absF(d3) < eps && onSegment(a, b, c) {
		return true
	}
	if absF(d4) < eps && onSegment(a, b, d) {
		return true
	}
	return false
}

// cross2 returns the z-component of (b-a) × (p-a).
func cross2(a, b, p [2]float64) float64 {
	return (b[0]-a[0])*(p[1]-a[1]) - (b[1]-a[1])*(p[0]-a[0])
}

func onSegment(a, b, p [2]float64) bool {
	const eps = 1e-9
	minX, maxX := a[0], b[0]
	if minX > maxX {
		minX, maxX = maxX, minX
	}
	minY, maxY := a[1], b[1]
	if minY > maxY {
		minY, maxY = maxY, minY
	}
	return p[0] >= minX-eps && p[0] <= maxX+eps &&
		p[1] >= minY-eps && p[1] <= maxY+eps
}

func absF(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/domain/map/service/...
```

Expected: all 7 new tests PASS + existing validator tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/map/service/wall_geometry.go \
        internal/domain/map/service/wall_geometry_test.go
git commit -m "feat(map): IsPathBlocked with direction-aware wall blocking (TDD)"
```

---

### Task 6: Add wall state to `Room`, extend `lobby_state_sync` handler

**Files:**
- Modify: `internal/app/game/room.go`

- [ ] **Step 1: Add imports**

At the top of `room.go`, add to the import block:

```go
mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
mapservice "github.com/422UR4H/HxH_RPG_System/internal/domain/map/service"
```

- [ ] **Step 2: Add fields to `Room` struct**

After `lobbyPieces map[string]LobbyPieceMovedPayload`:

```go
lobbyWalls    map[string]mapentity.WallSegment // keyed by wall ID; in-memory runtime wall state
lobbyGridSize float64                          // cell size in world coords; used for movement blocking
```

- [ ] **Step 3: Initialize in `NewRoom`**

After `lobbyPieces: make(map[string]LobbyPieceMovedPayload)`:

```go
lobbyWalls:    make(map[string]mapentity.WallSegment),
lobbyGridSize: 64, // default; overridden by lobby_state_sync
```

- [ ] **Step 4: Update `lobby_state_sync` handler**

Find `case MsgTypeLobbyStateSync:` in room.go. Replace the `LobbyPiecesPayload` unmarshal with `LobbyStateSyncPayload` and add wall seeding:

```go
case MsgTypeLobbyStateSync:
	if !r.IsMaster(client.userUUID) {
		client.SendMessage(NewErrorMessage("forbidden", ErrNotMaster.Error()))
		return
	}
	var payload LobbyStateSyncPayload
	if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
		client.SendMessage(NewErrorMessage("invalid_payload", "invalid lobby_state_sync payload"))
		return
	}
	r.mu.Lock()
	r.lobbyPieces = make(map[string]LobbyPieceMovedPayload, len(payload.Pieces))
	for _, p := range payload.Pieces {
		r.lobbyPieces[p.PieceID] = p
	}
	r.lobbyWalls = make(map[string]mapentity.WallSegment, len(payload.Walls))
	for _, w := range payload.Walls {
		r.lobbyWalls[w.ID] = w
	}
	if payload.Grid != nil && payload.Grid.CellSize > 0 {
		r.lobbyGridSize = payload.Grid.CellSize
	}
	r.mu.Unlock()
	// No relay — only seeds the server's in-memory state.
```

- [ ] **Step 5: Build**

```bash
go build ./...
```

Expected: success. If `mapservice` import is unused, the next task will use it.

- [ ] **Step 6: Commit**

```bash
git add internal/app/game/room.go
git commit -m "feat(game): add lobbyWalls to Room; extend lobby_state_sync to seed wall state"
```

---

### Task 7: Handle `Interact` in `enqueue_master_action`

**Files:**
- Modify: `internal/app/game/room.go`

- [ ] **Step 1: Add `applyWallInteract` helper method**

Append at the bottom of `room.go`, before or after `sendLobbyFullState`:

```go
// applyWallInteract updates in-memory wall state for open/close/toggle.
// Returns (newOpen, newLocked, ok). ok=false means wall not found or interaction
// not applicable (e.g. lockpick/examine are player-only actions requiring rolls).
func (r *Room) applyWallInteract(wallID string, interact *action.Interact) (open, locked bool, ok bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	w, exists := r.lobbyWalls[wallID]
	if !exists {
		return false, false, false
	}
	switch interact.Kind {
	case action.InteractOpen:
		w.Open = true
	case action.InteractClose:
		w.Open = false
	case action.InteractToggle:
		w.Open = !w.Open
	default:
		return false, false, false
	}
	r.lobbyWalls[wallID] = w
	return w.Open, w.Locked, true
}
```

Add the action import if not present:

```go
"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
```

- [ ] **Step 2: Wire Interact in `enqueue_master_action` case**

In `case MsgTypeEnqueueMasterAction:`, immediately after `ma := buildMasterAction(...)`, add:

```go
// Wall interaction: handled in-memory + broadcast; does not go through the use case queue.
if ma.Interact != nil && len(ma.TargetID) > 0 {
	for _, targetID := range ma.TargetID {
		newOpen, newLocked, ok := r.applyWallInteract(targetID.String(), ma.Interact)
		if !ok {
			// Wall not in in-memory state — skip.
			// TODO: when the match turn system is finalized, consider loading from DB if missing.
			continue
		}
		evt := NewServerMessage(MsgTypeWallStateChanged, WallStateChangedPayload{
			WallID: targetID.String(),
			Open:   newOpen,
			Locked: newLocked,
		})
		data, _ := json.Marshal(evt)
		go func() { r.broadcast <- data }()
	}
	return
}
```

- [ ] **Step 3: Build**

```bash
go build ./...
```

Expected: success.

- [ ] **Step 4: Run tests**

```bash
go test ./internal/...
```

Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add internal/app/game/room.go
git commit -m "feat(game): MasterAction.Interact opens/closes walls in-memory and broadcasts wall_state_changed"
```

---

### Task 8: Movement blocking in `enqueue_action`

**Files:**
- Modify: `internal/app/game/room.go`

- [ ] **Step 1: Add movement blocking check**

In `case MsgTypeEnqueueAction:`, after building `a` via `buildAction(...)` and before calling `enqueueActionUC.Execute`, add:

```go
// Movement blocking: validate path against walls with move=true and !open.
// TODO: Revisit when the full turn system is finalized — this is a best-effort check.
if a.Move != nil {
	from := a.Move.From
	to := a.Move.Position
	// Only validate when the client provided a non-zero From (zero means "not provided").
	if from != ([3]int{}) {
		fromWorld := [2]float64{float64(from[0]) * r.lobbyGridSize, float64(from[1]) * r.lobbyGridSize}
		toWorld := [2]float64{float64(to[0]) * r.lobbyGridSize, float64(to[1]) * r.lobbyGridSize}
		r.mu.RLock()
		walls := make([]mapentity.WallSegment, 0, len(r.lobbyWalls))
		for _, w := range r.lobbyWalls {
			walls = append(walls, w)
		}
		r.mu.RUnlock()
		if mapservice.IsPathBlocked(fromWorld, toWorld, walls) {
			client.SendMessage(NewErrorMessage("move_blocked", "movement blocked by a wall"))
			return
		}
	}
}
```

- [ ] **Step 2: Build**

```bash
go build ./...
```

Expected: success. If `mapservice` import was already added in Task 6, no change needed. Otherwise add:

```go
mapservice "github.com/422UR4H/HxH_RPG_System/internal/domain/map/service"
```

- [ ] **Step 3: Run all tests + vet**

```bash
go test ./internal/...
go vet -tags=integration ./internal/gateway/pg/...
```

Expected: all pass, no vet errors.

- [ ] **Step 4: Commit**

```bash
git add internal/app/game/room.go
git commit -m "feat(game): validate movement path against blocking walls in enqueue_action"
```

---

## Self-Review

**Spec coverage:**

| Spec requirement | Task |
|---|---|
| `interact.go` — `Interact` + `InteractKind` enum | Task 1 |
| `action.go` — `Interact *Interact` field | Task 2 |
| `master_action.go` — `Interact *Interact` field | Task 2 |
| `message.go` — `InteractPayload`, `MsgTypeWallStateChanged` | Task 3 |
| `action_mapper.go` — map `InteractPayload` → `action.Interact` | Task 4 |
| `enqueue_master_action` — Interact → update in-memory + broadcast | Task 7 |
| Validação de bloqueio de movimento em `enqueue_action` | Task 8 |
| `IsPathBlocked` pure function | Task 5 |
| In-memory wall state in Room | Task 6 |

**Type consistency check:**
- `action.Interact.Kind` is `InteractKind`; `InteractPayload.Kind` is `string` — conversion via `action.InteractKind(p.Interact.Kind)` in mapper ✅
- `lobbyWalls map[string]mapentity.WallSegment` keyed by `w.ID` (string) — same key type used in `applyWallInteract(targetID.String(), ...)` ✅
- `mapservice.IsPathBlocked` takes `[]mapentity.WallSegment` — matches what we collect from `lobbyWalls` ✅
- `Move.From [3]int` added to both `move.go` and `MovePayload`; mapped in `buildAction` via `p.Move.From` ✅

**Placeholder scan:** none. All steps contain actual code.
