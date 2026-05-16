# Match Domain Phase 3 — Implementation Design

> **For agentic workers:** implement this spec using `superpowers:subagent-driven-development` or `superpowers:executing-plans`.

**Goal:** Complete the match bounded context with Scene Management, Turn/Round/Scene/Action persistence, and EnqueueMasterAction end-to-end flow.

**Architecture:** Three independent concerns wired together: (1) DB schema for the full game event chain (actions → turns → rounds → scenes → matches → campaigns); (2) Scene Management as the in-session grouping mechanism for rounds; (3) MasterAction as a parallel action channel for the master within a turn.

**Phase context:** Final PR in the post-refactor match domain feature series. Reaction visibility and full buildAction payload mapping are deferred (still being designed).

---

## Scope

| Feature | Status |
|---------|--------|
| DB schema: `scenes`, `rounds`, `turns`, `actions` | ✅ In scope |
| Scene Management (`ChangeScene` WS flow) | ✅ In scope |
| Turn/Round/Scene/Action persistence (atomic, on turn close) | ✅ In scope |
| `EnqueueMasterAction` end-to-end flow | ✅ In scope |
| Reaction visibility / reveal mechanism | ❌ Deferred (still being designed) |
| Full `buildAction` payload mapping | ❌ Deferred (frontend contract pending) |
| `Scene.FinishScene` triggered by ChangeScene | ✅ In scope (brief_final_description nullable) |

---

## Domain Changes

### `Round` — add ID
File: `internal/domain/match/entity/round/round.go`

Add `id uuid.UUID` field. `NewRound(mode)` initializes `id = uuid.New()`. Add `GetID() uuid.UUID`.

### `Scene` — add ID
File: `internal/domain/match/entity/scene/scene.go`

Add `id uuid.UUID` field. `NewScene(category, briefInitialDescription)` initializes `id = uuid.New()`. Add `GetID() uuid.UUID`.

Also add `Close(at time.Time)` method (analogous to `Round.Close`) — sets `finishedAt`. `FinishScene` keeps its existing signature for the full flow (with `briefFinalDescription`); `Close` is used internally by ChangeScene when a brief final description is not required.

### `Turn` — activate MasterAction
File: `internal/domain/match/entity/turn/turn.go`

Add `AddMasterAction(ma *action.MasterAction)` method. Remove `//nolint:unused` from `masterActions` field.

### `MasterAction` — expose `happenedAt`
File: `internal/domain/match/entity/action/master_action.go`

Add `SetHappenedAt(t time.Time)` or set it in a constructor. The field is set when the master enqueues the action.

---

## MatchSession Changes
File: `internal/domain/match/matchsession/match_session.go`

### Add `activeScene *scene.Scene`

`NewMatchSession` creates the initial scene:
```go
activeScene: scene.NewScene(enum.Roleplay, ""),
```
Default category Roleplay, empty initial description — master can `ChangeScene` immediately if they want a different type.

Add `GetActiveScene() *scene.Scene`.

### `ChangeScene(category enum.SceneCategory, briefDesc string) (*scene.Scene, *round.Round, error)`

```
1. If activeRound.HasOpenTurn() → return ErrRoundHasOpenTurn
2. closedRound = roundOrch.CloseRound(activeRound, now)
3. activeScene.Close(now)              // mark old scene finished
4. oldScene = activeScene
5. activeScene = scene.NewScene(category, briefDesc)
6. activeRound = round.NewRound(activeRound.GetMode())
7. return oldScene, closedRound, nil
```

Returns the closed scene and round so the caller (Room handler) can persist them if they were already in DB.

### `EnqueueMasterAction(masterUUID uuid.UUID, ma *action.MasterAction) error`

```
1. If activeRound.CurrentTurn() == nil → return ErrNoActiveTurn
2. ma.happenedAt = time.Now()
3. activeRound.CurrentTurn().AddMasterAction(ma)
4. return nil
```

---

## DB Schema — Migrations

### Migration 1: `add_scenes_table`
```sql
CREATE TABLE IF NOT EXISTS scenes (
  id SERIAL PRIMARY KEY,
  uuid UUID NOT NULL DEFAULT gen_random_uuid(),
  match_uuid UUID NOT NULL,
  category VARCHAR(32) NOT NULL,
  brief_initial_description VARCHAR(255) NOT NULL DEFAULT '',
  brief_final_description VARCHAR(255),
  created_at TIMESTAMP NOT NULL,
  finished_at TIMESTAMP,
  UNIQUE (uuid),
  FOREIGN KEY (match_uuid) REFERENCES matches(uuid)
);
```

### Migration 2: `add_rounds_table`
```sql
CREATE TABLE IF NOT EXISTS rounds (
  id SERIAL PRIMARY KEY,
  uuid UUID NOT NULL DEFAULT gen_random_uuid(),
  scene_uuid UUID NOT NULL,
  mode VARCHAR(16) NOT NULL,
  created_at TIMESTAMP NOT NULL,
  finished_at TIMESTAMP,
  UNIQUE (uuid),
  FOREIGN KEY (scene_uuid) REFERENCES scenes(uuid)
);
```

No direct `match_uuid` on rounds — reach match through scene. Recovery queries use the scene join.

### Migration 3: `add_turns_table`
```sql
CREATE TABLE IF NOT EXISTS turns (
  id SERIAL PRIMARY KEY,
  uuid UUID NOT NULL DEFAULT gen_random_uuid(),
  round_uuid UUID NOT NULL,
  created_at TIMESTAMP NOT NULL,
  finished_at TIMESTAMP NOT NULL,
  UNIQUE (uuid),
  FOREIGN KEY (round_uuid) REFERENCES rounds(uuid)
);
```

`finished_at` is NOT NULL — turns are only inserted on close.

### Migration 4: `add_actions_table`
```sql
CREATE TABLE IF NOT EXISTS actions (
  id SERIAL PRIMARY KEY,
  uuid UUID NOT NULL DEFAULT gen_random_uuid(),
  turn_uuid UUID NOT NULL,
  actor_uuid UUID NOT NULL,
  react_to_uuid UUID,          -- NULL = main action; set = reaction (self-referential)
  target_ids UUID[] NOT NULL DEFAULT '{}',
  type VARCHAR(32) NOT NULL,   -- 'move', 'attack', 'defense', 'dodge', 'feint', 'trigger'
  speed JSONB,                 -- ActionSpeed { bar, roll_check }
  skills JSONB,                -- []Skill
  move JSONB,                  -- *Move { category, position, speed, charge, final_speed }
  attack JSONB,                -- *Attack { weapon, hit, damage, charge, relative_velocity }
  defense JSONB,               -- *Defense { weapon, roll_check }
  dodge JSONB,                 -- *Dodge { category, roll_check }
  feint JSONB,                 -- *RollCheck
  trigger JSONB,               -- *Trigger (WIP)
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (uuid),
  FOREIGN KEY (turn_uuid) REFERENCES turns(uuid),
  FOREIGN KEY (actor_uuid) REFERENCES users(uuid),
  FOREIGN KEY (react_to_uuid) REFERENCES actions(uuid)
);
```

Sub-type data stored as nullable JSONB columns (one per sub-type, not one big blob). When any sub-type schema stabilizes, its column can be expanded to structured columns independently. Reactions use the same table with `react_to_uuid` set. Reactions are not inserted in Phase 3 — the column and FK exist for future use.

---

## Persistence Strategy

### Trigger
All persistence for a scene's data happens **atomically on the first turn close** of that scene. Subsequent turn closes in the same scene/round only insert the turn and action.

### Gateway method: `PersistTurnClose`
New package: `internal/gateway/pg/round/` (handles round + scene + turn + action in one transaction).

```go
func (r *Repository) PersistTurnClose(
    ctx context.Context,
    scene *scene.Scene,
    round *round.Round,
    turn *turn.Turn,
    action *action.Action,
    matchUUID uuid.UUID,
) error
```

Single transaction:
```sql
-- Scene and round: idempotent (only first turn of each inserts them)
INSERT INTO scenes (uuid, match_uuid, category, ..., created_at)
VALUES ($1, $2, $3, ..., $N)
ON CONFLICT (uuid) DO NOTHING;

INSERT INTO rounds (uuid, scene_uuid, mode, created_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (uuid) DO NOTHING;

-- Turn and action: always insert (called once per turn close)
INSERT INTO turns (uuid, round_uuid, created_at, finished_at) VALUES (...);
INSERT INTO actions (uuid, turn_uuid, actor_uuid, target_ids, type, ...) VALUES (...);
```

Uses Go timestamps (`time.Now()` in caller, passed as parameters) — never SQL `NOW()`.
Uses unconditional rollback defer per gateway conventions.

### Recovery: `FindActiveSession`
```go
func (r *Repository) FindActiveSession(ctx context.Context, matchUUID uuid.UUID) (*ActiveSessionData, error)
```

```sql
SELECT s.uuid, s.category, s.brief_initial_description, s.created_at,
       r.uuid, r.mode, r.created_at
FROM scenes s
JOIN rounds r ON r.scene_uuid = s.uuid
WHERE s.match_uuid = $1
  AND s.finished_at IS NULL
  AND r.finished_at IS NULL
LIMIT 1
```

If no row found: no persisted active session (normal on first turn, or after server restart before first turn close). `InitMatchSessionUC` creates a fresh session.

### `InitMatchSessionUC` — recovery integration
```go
data, err := uc.roundRepo.FindActiveSession(ctx, matchUUID)
if err != nil {
    return nil, err
}
if data != nil {
    // reconstruct scene and round with existing IDs
    activeScene := scene.ReconstructScene(data.SceneID, data.Category, data.BriefInitialDesc, data.SceneCreatedAt)
    activeRound := round.ReconstructRound(data.RoundID, data.Mode, data.RoundCreatedAt)
    return matchsession.NewMatchSessionWithState(matchUUID, charSheets, participants, activeScene, activeRound), nil
}
return matchsession.NewMatchSession(matchUUID, charSheets, participants), nil
```

Add `ReconstructScene` and `ReconstructRound` constructors (accept existing UUID, skip `uuid.New()`).
Add `NewMatchSessionWithState` constructor (accepts pre-built scene and round).

### `ChangeScene` persistence
When ChangeScene closes an old scene/round that are already in DB (i.e., at least one turn closed in them), their `finished_at` must be updated.

Gateway method:
```go
func (r *Repository) CloseSceneAndRound(ctx context.Context, sceneUUID, roundUUID uuid.UUID, at time.Time) error
```

The Room handler checks whether the old scene was persisted before calling `CloseSceneAndRound`. Simplest heuristic: track `session.IsScenePersisted() bool` (a flag set when `PersistTurnClose` succeeds for the first time in the current scene).

Add to `MatchSession`:
- `GetMatchUUID() uuid.UUID`
- `MarkRoundPersisted()` — sets `roundPersisted = true` AND `scenePersisted = true` (both are persisted on first turn close)
- `IsRoundPersisted() bool`
- `IsScenePersisted() bool`

`CloseRound` (inside `session.CloseRound()`) resets `roundPersisted = false` (scene remains persisted).
`ChangeScene` resets both `roundPersisted = false` and `scenePersisted = false`.

---

## WS Layer — Scene Management

### New message types
```go
MsgTypeChangeScene    MessageType = "change_scene"      // client → server (master only)
MsgTypeSceneChanged   MessageType = "scene_changed"     // server → all clients
```

### Payload
```go
type ChangeScenePayload struct {
    Category             string `json:"category"`               // "roleplay" | "battle"
    BriefInitialDescription string `json:"brief_initial_description"`
}

type SceneChangedPayload struct {
    SceneID  uuid.UUID `json:"scene_id"`
    Category string    `json:"category"`
    BriefInitialDescription string `json:"brief_initial_description"`
}
```

### Room handler
```go
case MsgTypeChangeScene:
    // master-only guard
    var payload ChangeScenePayload
    // parse + validate
    oldScene, oldRound, err := session.ChangeScene(category, desc)
    if err != nil { /* send error */ return }
    if session.IsScenePersisted() { // old scene had at least one turn
        _ = r.roundRepo.CloseSceneAndRound(ctx, oldScene.GetID(), oldRound.GetID(), time.Now())
    }
    // ChangeScene resets both roundPersisted and scenePersisted to false (new scene + new round)
    r.broadcastAll(NewServerMessage(MsgTypeSceneChanged, SceneChangedPayload{...}))
```

New interface on Room: `IRoundRepository` (for `CloseSceneAndRound` and `FindActiveSession`).

---

## WS Layer — EnqueueMasterAction

### New message types
```go
MsgTypeEnqueueMasterAction  MessageType = "enqueue_master_action"   // client → server (master only)
MsgTypeMasterActionEnqueued MessageType = "master_action_enqueued"   // server → all clients
```

### Payload
```go
type MasterActionPayload struct {
    TargetIDs   []uuid.UUID          `json:"target_ids"`
    Skills      []ActionSkillPayload `json:"skills,omitempty"`
    Move        *MovePayload         `json:"move,omitempty"`
    Attack      *AttackPayload       `json:"attack,omitempty"`
    ActionSpeed *RollCheckPayload    `json:"action_speed,omitempty"`
}

type MasterActionEnqueuedPayload struct {
    TargetIDs   []uuid.UUID          `json:"target_ids"`
    Skills      []ActionSkillPayload `json:"skills,omitempty"`
    Move        *MovePayload         `json:"move,omitempty"`
    Attack      *AttackPayload       `json:"attack,omitempty"`
    ActionSpeed *RollCheckPayload    `json:"action_speed,omitempty"`
}
```

### New use case: `EnqueueMasterActionUC`
File: `internal/application/match/enqueue_master_action.go`

```go
type EnqueueMasterActionResult struct{}

func (uc *EnqueueMasterActionUC) Execute(
    ctx context.Context,
    session *matchsession.MatchSession,
    masterUUID uuid.UUID,
    ma *action.MasterAction,
) (*EnqueueMasterActionResult, error)
```

Validates caller is master, delegates to `session.EnqueueMasterAction(masterUUID, ma)`.

### Room handler
```go
case MsgTypeEnqueueMasterAction:
    // master-only guard
    var payload MasterActionPayload
    // parse
    ma := buildMasterAction(payload)
    _, err := r.enqueueMasterActionUC.Execute(ctx, session, client.userUUID, ma)
    if err != nil { /* send error */ return }
    r.broadcastAll(NewServerMessage(MsgTypeMasterActionEnqueued, MasterActionEnqueuedPayload{...}))
```

Add `buildMasterAction(payload MasterActionPayload) *action.MasterAction` to `action_mapper.go`.

---

## New Gateway Packages

### `internal/gateway/pg/round/` (new)
- `repository.go` — `Repository` struct with `*pgxpool.Pool`
- `persist_turn_close.go` — `PersistTurnClose(ctx, scene, round, turn, action, matchUUID)`
- `find_active_session.go` — `FindActiveSession(ctx, matchUUID)`
- `close_scene_and_round.go` — `CloseSceneAndRound(ctx, sceneUUID, roundUUID, at)`
- `round_integration_test.go`

### pgtest helpers (add to `internal/gateway/pg/pgtest/setup.go`)
- `InsertTestScene(t, pool, matchUUID, category)` → scene UUID string
- `InsertTestRound(t, pool, sceneUUID, mode)` → round UUID string
- `InsertTestTurn(t, pool, roundUUID)` → turn UUID string

---

## Application Layer Changes

### `InitMatchSessionUC`
Add `roundRepo IRoundRepository`. On `Init`:
1. Load participants + char sheets (unchanged)
2. Call `FindActiveSession` — if found, reconstruct; else create fresh session

### `CloseRoundUC`
Add `roundRepo IRoundRepository`. On `Execute`:
1. Call `session.CloseRound()` → returns closed round (session now has new active round)
2. If `session.IsRoundPersisted()`: call `roundRepo.CloseRound(ctx, closedRound.GetID(), closedRound.GetFinishedAt())`
   - Gateway: `UPDATE rounds SET finished_at = $1 WHERE uuid = $2`
3. Session resets `roundPersisted = false` automatically (new round not yet in DB)

The new active round created by `session.CloseRound()` remains in memory only — it will be persisted on its first turn close (same trigger as always).

### `i_repository.go`
Add `IRoundRepository` interface with `PersistTurnClose`, `FindActiveSession`, `CloseSceneAndRound`, `CloseRound`.

---

## Room Handler Integration

Room gains one new dependency: `IRoundRepository`.

On `handleOpenNextAction` and `handlePullAction`: after getting `result.ClosedTurn`, call `PersistTurnClose(ctx, session.GetActiveScene(), session.GetActiveRound(), result.ClosedTurn, result.ClosedTurn.GetAction(), session.GetMatchUUID())`. On success, call `session.MarkRoundPersisted()`.

On `handleChangeScene`: call `CloseSceneAndRound` only if `session.IsScenePersisted()` (at least one turn closed in the old scene).
On `handleCloseRound`: call `roundRepo.CloseRound` only if `session.IsRoundPersisted()`.

---

## Testing

### Unit tests
- `MatchSession.ChangeScene` — happy path, error on open turn
- `MatchSession.EnqueueMasterAction` — happy path, error on no active turn
- `Round.GetID` — trivial
- `Turn.AddMasterAction` — appends correctly
- `EnqueueMasterActionUC` — master guard, delegates to session

### Integration tests (`round_integration_test.go`)
- `TestPersistTurnClose` — first call inserts scene+round+turn+action; second call inserts only turn+action (scene and round ON CONFLICT DO NOTHING)
- `TestFindActiveSession` — finds open scene+round; returns nil if all closed
- `TestCloseSceneAndRound` — sets finished_at on both

### Handler tests (mock-based)
- `handleChangeScene` — master guard, broadcasts `scene_changed`, calls `CloseSceneAndRound` only when `IsScenePersisted`
- `handleEnqueueMasterAction` — master guard, broadcasts `master_action_enqueued`

---

## Documentation Impact

- `domain-map.instructions.md` — update Scene and Round entries (now have IDs, Phase 3 complete)
- `AGENTS.md` Known Issues — update to "(Phase 3 complete)" after implementation
