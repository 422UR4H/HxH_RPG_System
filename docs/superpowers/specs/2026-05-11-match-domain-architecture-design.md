# Match Domain Architecture Design

**Date:** 2026-05-11
**Status:** Approved
**Scope:** `internal/domain/match/`, `internal/application/`, `internal/domain/service/`, `internal/app/game/`

---

## Context

The match system (the in-game execution of a session: scenes, rounds, turns, actions, reactions) is the
most complex domain in this project. Before implementing it, we needed to establish clear architectural
boundaries so the system remains maintainable, testable, and reusable for future RPG systems built
on the same codebase.

This document records the decisions made, explains the reasoning behind each one, and serves as a
learning resource for developers unfamiliar with these patterns.

---

## Terminology (Canonical Glossary)

These names are the source of truth across code, docs, and game rules:

| Code Name | PT-BR | What it means |
|-----------|-------|---------------|
| `Match` | Partida | The entire game session (root of everything) |
| `Scene` | Cena | A continuous narrative segment: Roleplay or Battle |
| `Round` | Rodada | A cycle of actions within a scene. Has a mode (Free or Race). |
| `Turn` | Turno | One character's action + all reactions to it. The atomic unit of combat. |
| `Action` | Ação | What a character does on their Turn |
| `Reaction` | Reação | Another character's response to an Action, within the same Turn |

**Hierarchy:**
```
Match → Scene → Round → Turn → (Action + Reactions)
```

**Round modes:**
- `Free` (Rodada Livre) — no time pressure; narrative order; no priority queue
- `Race` (Rodada Disputada) — time-based; actions ordered by Speed via priority queue

Both Scene categories (Roleplay and Battle) use the same Round/Turn structure.
Mode is set per Round, not derived from Scene category.

---

## Why This Architecture?

### The core problem

The match domain has two different kinds of complexity happening simultaneously:

1. **Rich in-memory state** — while a session is running, the current Round, character sheet cache,
   and action queue all need to live in RAM. Fetching from the database on every action would be
   slow and unnecessary.

2. **Complex pure rules** — calculating combat (dice + attributes + weapon + reactions) involves
   multiple entities but belongs to none of them individually.

A naive approach would mix both concerns into a single "engine" object. That's what the old
`turn/engine.go` and `round/engine.go` did — they managed state AND applied rules, which made
them hard to test and impossible to reuse.

### The solution: four separated responsibilities

Each layer has exactly one job. This is not a new idea — it follows DDD Lite applied to a
real-time game server.

---

## Architecture Layers

```
┌──────────────────────────────────────────────────────────┐
│  app/  (Delivery Layer)                                  │
│  Translates HTTP/WebSocket messages into use case calls. │
│  Knows about gorilla/websocket, huma, JSON.              │
│  Does NOT contain game rules or business logic.          │
├──────────────────────────────────────────────────────────┤
│  application/  (Use Case Layer)                          │
│  Orchestrates: fetch from DB → call domain → persist.    │
│  Knows about repositories and MatchSession.              │
│  Does NOT know about HTTP, WebSocket, or RPG formulas.   │
├──────────────────────────────────────────────────────────┤
│  domain/match/matchsession/  (In-Memory Session State)   │
│  Holds the active runtime state of a running match:      │
│  current Round, character sheet cache, action queue.     │
│  Lives inside the Room for the duration of a WS session. │
│  Uses domain services to calculate results.              │
├──────────────────────────────────────────────────────────┤
│  domain/match/service/  (Domain Services)                │
│  Stateless structs that apply pure RPG rules.            │
│  Receive entities as parameters, return results.         │
│  Know nothing about DB, HTTP, WS, or memory management. │
├──────────────────────────────────────────────────────────┤
│  domain/match/entity/  (Entities)                        │
│  Pure state structs. Only methods about their own state. │
│  Never import any layer above.                           │
└──────────────────────────────────────────────────────────┘
       gateway/ implements repository interfaces from domain/
```

**The golden rule:** dependencies only flow downward. An entity never imports a service. A service
never imports a use case. A use case never imports a handler.

> **For junior developers:** think of it like a kitchen. The entity is the raw ingredient (it just
> exists). The domain service is the recipe (pure instructions). The use case is the chef (follows
> the recipe and manages what goes in and out of the fridge). The handler is the waiter (takes the
> order and brings the plate, knows nothing about cooking).

---

## Why `application/` is NOT the same as `app/`

A common point of confusion:

| | `app/` | `application/` |
|---|---|---|
| **Role** | Delivery layer | Use case layer |
| **Knows about** | HTTP, WebSocket, JSON serialization | Domain entities, repositories, MatchSession |
| **Framework imports** | `gorilla/websocket`, `huma` | None |
| **Testable without a server?** | No | **Yes** |
| **Example** | `room.go`, `handler.go` | `open_next_action.go`, `close_turn.go` |

Use cases must be testable in isolation with just mock repositories and a MatchSession — no
HTTP server, no WebSocket, no JSON. This is why they live separately from `app/`.

This pattern is standard in Go DDD Lite projects:
- Some projects call it `internal/usecase/` (most common in Go open-source)
- Some call it `internal/application/` (correct DDD term)
- Both are valid; we use `application/` here

---

## Entity Layer — What Changes

### What stays (pure state, no coordination)

`Round` and `Turn` become pure state structs. They record what happened — they do not decide
when things happen.

```go
// Round — records the state of a cycle of actions
type Round struct {
    mode       enum.RoundMode
    turns      []*Turn       // append-only log of Turns in this Round
    events     []GameEvent
    finishedAt *time.Time
}

// Minimal methods — only about its own state:
func (r *Round) GetMode() enum.RoundMode
func (r *Round) AppendTurn(t *Turn)
func (r *Round) CurrentTurn() *Turn   // last turn in the slice
func (r *Round) HasOpenTurn() bool    // CurrentTurn().FinishedAt == nil
func (r *Round) Close(at time.Time)

// Turn — records one character's action + all reactions
type Turn struct {
    action     action.Action
    reactions  []action.Action
    openedAt   time.Time
    finishedAt *time.Time
}
```

### What is removed

| File | Action | Reason |
|------|--------|--------|
| `entity/match/turn/engine.go` | **Deleted** | Old semantics (Turn as a cycle), replaced entirely by `round/` package |
| `entity/match/round/engine.go` | **Dissolved** | Logic moves to `RoundOrchestrator` (stateless) + `MatchSession` (stateful) |
| `entity/match/engine.go` | **Dissolved** | Orchestration moves to `MatchSession` |

### ActionPriorityQueue location

The `ActionPriorityQueue` does **not** live in `Round`. Reasons:
- Actions can be declared for a future Round before the current one ends
- Actions can carry over between Rounds within the same Scene
- The queue is operational state (alive during a session), not a historical record

**Decision:** the priority queue lives in `MatchSession` as `activeQueue`. Round records only
the Turns that result from processing the queue — it is the log, not the queue.

---

## Domain Services Layer

Located at `internal/domain/match/service/`. Three stateless structs.

> **For junior developers:** "stateless" means the struct has NO fields — it holds no data.
> You can create it once and reuse it forever. It's essentially a collection of pure functions
> grouped under a name. You test it by calling its methods with test data — no database, no
> network, no mocking anything.

### `RoundOrchestrator`

Knows everything about the Round/Turn lifecycle: when to create Turns, when to close them,
how to use the priority queue, how to attach reactions.

```go
type RoundOrchestrator struct{} // stateless — no fields

// Extracts the highest-speed Action from the queue and creates a new Turn in the Round.
// Used in Race mode (or Free mode when master picks the next action).
func (ro RoundOrchestrator) NextAction(r *round.Round, queue *action.PriorityQueue) (*turn.Turn, error)

// Extracts a specific Action by UUID — used when the master picks which action to resolve.
func (ro RoundOrchestrator) PullAction(r *round.Round, queue *action.PriorityQueue, id uuid.UUID) (*turn.Turn, error)

// Sets finishedAt on the current Turn. Called before opening the next action.
func (ro RoundOrchestrator) CloseTurn(r *round.Round, at time.Time) *turn.Turn

// Sets finishedAt on the Round. Called when the master ends a Round.
func (ro RoundOrchestrator) CloseRound(r *round.Round, at time.Time) *round.Round

// Validates that the reaction targets the current Turn's action, then adds it.
func (ro RoundOrchestrator) AttachReaction(r *round.Round, reaction *action.Action) error

// Switches Round mode between Free and Race.
func (ro RoundOrchestrator) ChangeMode(r *round.Round, initiative *action.Initiative)
```

### `CombatResolver`

Knows how to calculate the result of a Turn (action + all reactions so far).
Called every time the Turn state changes: when it opens AND each time a reaction is attached.
The master sees an updated resolution snapshot after each change.

```go
type CombatResolver struct{} // stateless

// Resolves the current state of the Turn (action + however many reactions exist).
// Called on Turn open (no reactions yet) and re-called after each AttachReaction.
// Returns a snapshot that the use case broadcasts to all participants.
func (cr CombatResolver) Resolve(
    t *turn.Turn,
    sheets map[uuid.UUID]*sheet.Sheet,
) *TurnResolution

type TurnResolution struct {
    ActionResult    RollResult
    ReactionResults []ReactionResult
    Blows           []*battle.Blow
    IsSettled       bool // false while reactions can still arrive
}
```

### `RollCalculator`

Knows how to compute the final result of a dice roll: dice values + character skill + modifiers.

```go
type RollCalculator struct{} // stateless

func (rc RollCalculator) Calculate(
    check action.RollCheck,
    sheet *sheet.Sheet,
) int
```

**Reuse for other RPG systems:** to build a different RPG (e.g., a Cyberpunk system), you import
the same entities (`Action`, `Turn`, `Round`) and write new implementations of `CombatResolver`
and `RollCalculator` specific to that system's rules. The structural entities are generic; the
rules are pluggable.

---

## MatchSession

Located at `internal/domain/match/matchsession/match_session.go`.

> **For junior developers:** `MatchSession` is the "living memory" of a match while it is
> happening. Think of it as a whiteboard that the master and players write on during the session.
> When the session ends, important parts of the whiteboard are saved to the database and the rest
> is erased. The `Room` (WebSocket server) holds this whiteboard.

### What it holds

```go
type MatchSession struct {
    matchUUID   uuid.UUID
    activeScene *scene.Scene
    activeRound *round.Round

    // Action queue — belongs to the active Scene, not to the Round.
    // Survives Round changes within the same Scene.
    activeQueue action.PriorityQueue

    // Character sheet cache. Loaded once on session start, read-only during combat.
    // Avoids DB hits on every Turn resolution.
    charSheets   map[uuid.UUID]*sheet.Sheet
    participants map[uuid.UUID]*match.Participant

    // Injected domain services
    roundOrch service.RoundOrchestrator
    combatRes service.CombatResolver
}
```

### What it does

```go
// Players enqueue their own character's action.
func (s *MatchSession) EnqueueAction(playerUUID uuid.UUID, a *action.Action) error

// Master enqueues an NPC action.
func (s *MatchSession) EnqueueMasterAction(npcUUID uuid.UUID, a action.MasterAction) error

// Master opens the highest-priority action in the queue.
// If a Turn is currently open, it is closed first.
// Returns: the closed Turn (if any) and the new Turn.
func (s *MatchSession) OpenNextAction() (closed *turn.Turn, opened *turn.Turn, err error)

// Master opens a specific action by UUID.
func (s *MatchSession) PullAction(id uuid.UUID) (closed *turn.Turn, opened *turn.Turn, err error)

// Player or master attaches a reaction to the current Turn.
// Also triggers CombatResolver.Resolve to update the resolution snapshot.
func (s *MatchSession) AttachReaction(r *action.Action) (*TurnResolution, error)

// Explicitly closes the current Turn (master decision).
func (s *MatchSession) CloseTurn() (*turn.Turn, error)

// Closes the current Round and prepares for the next.
func (s *MatchSession) CloseRound() (*round.Round, error)

// Read access
func (s *MatchSession) GetActiveRound() *round.Round
func (s *MatchSession) GetCurrentTurn() *turn.Turn
func (s *MatchSession) GetCharSheet(playerUUID uuid.UUID) (*sheet.Sheet, error)
```

### How the cyclic dependency is resolved

The old engines shared a `closeRoundTriggered *bool` flag to communicate across boundaries.
`MatchSession` makes the flow explicit instead:

```go
func (s *MatchSession) OpenNextAction() (closed *turn.Turn, opened *turn.Turn, err error) {
    // Close the previous Turn before opening the next — no flag, no shared state
    if s.activeRound != nil && s.activeRound.HasOpenTurn() {
        closed = s.roundOrch.CloseTurn(s.activeRound, time.Now())
    }
    opened, err = s.roundOrch.NextAction(s.activeRound, &s.activeQueue)
    return
}
```

The use case receives both the `closed` Turn (to persist) and the `opened` Turn (to broadcast)
from a single call.

### Room integration

`MatchSession` is created when the Room transitions to `RoomStatePlaying` and is held for the
entire session:

```go
type Room struct {
    // existing fields...
    session *matchsession.MatchSession // nil until match starts

    // new use cases injected at construction
    openNextActionUC IOpenNextAction
    pullActionUC     IPullAction
    attachReactionUC IAttachReaction
    closeTurnUC      ICloseTurn
    closeRoundUC     ICloseRound
}
```

---

## Data Flow — Complete Example

**Scenario: Master opens the next action (Race mode)**

```
1. WebSocket message arrives:
   { "type": "open_next_action" }

2. Room.handleClientMessage() dispatches to:
   r.openNextActionUC.Execute(ctx, r.session, masterUUID)

3. Use case (OpenNextActionUC.Execute):
   a. Validates caller is the master
   b. closed, opened, err := session.OpenNextAction()
      → MatchSession: closes previous Turn (if any) via RoundOrchestrator
      → MatchSession: extracts next Action from activeQueue via RoundOrchestrator
      → MatchSession: appends new Turn to activeRound
   c. if closed != nil:
      → matchRepo.SaveTurn(ctx, closed)     ← one INSERT, immutable record
   d. resolution := session.combatRes.Resolve(opened, session.charSheets)
   e. return opened, resolution

4. Room broadcasts to all participants:
   { "type": "turn_opened", "turn": opened, "resolution": resolution }
```

**Scenario: Player attaches a reaction**

```
1. { "type": "attach_reaction", "payload": { "react_to_id": "...", ... } }

2. Room → attachReactionUC.Execute(ctx, session, playerUUID, reactionData)

3. Use case:
   a. Validates player owns their character
   b. Creates action.Action with ReactToID set
   c. resolution, err := session.AttachReaction(reaction)
      → MatchSession: RoundOrchestrator.AttachReaction validates ReactToID
      → MatchSession: CombatResolver.Resolve recalculates Turn with new reaction
   d. return resolution

4. Room sends updated resolution to MASTER ONLY — not broadcast to all
   { "type": "resolution_updated", "resolution": resolution }
   Players only learn of the reaction when the master reveals it (a separate event,
   flow TBD). This is a core visibility rule of the game.
```

> **Visibility rule (to be fully designed):** reactions are submitted privately — only the
> master receives them on `attach_reaction`. All players receive reaction information only when
> the master explicitly reveals it. The full "reveal reaction" flow and its impact on `Turn`
> state (submitted vs revealed reactions) will be designed in a follow-up spec.

---

## Persistence Model

**Turns are persisted only once: when they are closed.** `openedAt` lives in the entity in
memory until then. No `UPDATE` statements — every write is an `INSERT`. Tables are immutable
after the initial write. No `updated_at` columns needed.

```
DB hierarchy (each level is an append-only log):

matches
└── scenes        (fk: match_uuid)
     └── rounds   (fk: scene_uuid, col: mode)
          └── turns (fk: round_uuid, cols: action_data, reactions_data, opened_at, finished_at)
```

A full match history is reconstructed by reading all Turns in order — this is the natural
event log the data model provides, without needing Event Sourcing infrastructure.

> **On Event Sourcing:** Full Event Sourcing (event store, projections, replay) would be
> over-engineering for this project. The append-only Turns table already provides the audit trail
> and historical record. Domain Events (simple pub/sub for broadcasting) may be added later
> without changing this model.

---

## Folder Structure

```
internal/
├── app/
│   ├── api/                        ← REST HTTP handlers (unchanged)
│   └── game/                       ← WebSocket: Hub, Room, Client
│       └── room.go                 ← adds: session *matchsession.MatchSession
│
├── application/                    ← Use Cases (moved from domain/)
│   ├── match/
│   │   ├── create_match.go
│   │   ├── start_match.go
│   │   ├── list_matches.go
│   │   ├── get_match.go
│   │   ├── open_next_action.go     ← NEW
│   │   ├── pull_action.go          ← NEW
│   │   ├── enqueue_action.go       ← NEW
│   │   ├── attach_reaction.go      ← NEW
│   │   ├── close_turn.go           ← NEW
│   │   ├── close_round.go          ← NEW
│   │   └── i_repository.go
│   ├── auth/
│   ├── campaign/
│   ├── character_sheet/
│   ├── enrollment/
│   ├── submission/
│   ├── session/
│   └── scenario/
│
├── domain/
│   ├── match/                      ← Match bounded context (NEW structure)
│   │   ├── entity/
│   │   │   ├── action/             ← Action, Attack, Defense, Dodge, etc. (unchanged)
│   │   │   ├── round/              ← Round (pure entity — engine.go REMOVED)
│   │   │   ├── turn/               ← Turn (pure entity — engine.go DELETED)
│   │   │   ├── scene/              ← Scene (unchanged)
│   │   │   ├── battle/             ← Blow (unchanged)
│   │   │   ├── match.go
│   │   │   ├── participant.go
│   │   │   ├── character_status.go
│   │   │   ├── game_event.go
│   │   │   └── summary.go
│   │   ├── service/
│   │   │   ├── round_orchestrator.go
│   │   │   ├── combat_resolver.go
│   │   │   └── roll_calculator.go
│   │   └── matchsession/
│   │       └── match_session.go
│   │
│   └── entity/                     ← Legacy location (stable — migrate separately)
│       ├── character_sheet/        ← Stable, fully tested — DO NOT refactor now
│       ├── character_class/
│       ├── campaign/
│       ├── scenario/
│       ├── user/
│       ├── enrollment/
│       ├── item/
│       ├── enum/                   ← Shared enums (used by all contexts)
│       └── die/                    ← Shared dice (used by all contexts)
│
└── gateway/                        ← PostgreSQL repositories (unchanged structure)
```

### Why bounded context for match but not for character_sheet?

`character_sheet/` is stable and fully tested. Migrating it to `domain/character_sheet/entity/`
would require updating all its imports with no immediate functional gain. It is deferred to a
dedicated refactor PR. The new `match/` code starts with the correct structure from day one.

---

## Decisions Log

| Decision | Rationale |
|----------|-----------|
| `MatchSession` in `domain/match/matchsession/` | Specific to the match context; clearly separated from auth `session/` |
| `ActionPriorityQueue` in `MatchSession`, not in `Round` | Queue is operational state that survives Round boundaries within a Scene |
| Domain services are stateless structs | Enables isolated testing with plain Go test data; enables rule replacement for future RPG systems |
| `RoundOrchestrator` (not Coordinator) | More expressive for the role of directing Round execution flow |
| Single `Resolve` method on `CombatResolver` | Called on Turn open and after each reaction; simpler than `Resolve`+`Recalculate` with equivalent behavior |
| Turn persisted only on close (one INSERT) | Append-only log — no UPDATE, no updated_at; consistent with the event-log model |
| Use cases moved to `application/` | Clear separation from delivery (`app/`) and pure domain (`domain/`); testable without any framework |
| Bounded context structure for `match/` | Groups everything about the match domain; enables future extraction as a reusable module |
| `character_sheet/` stays in `entity/` for now | Stable, tested, no disruption warranted; deferred refactor |
| No Event Sourcing | The append-only Turns table IS the event log; full ES would be over-engineering for MVP |
| Both Scene categories use Round/Turn | Roleplay = Free mode Round, Battle = Race mode Round; unified model, mode set per Round |

---

## Deferred Work

These are intentional deferrals — not forgotten items:

1. **`character_sheet/` bounded context migration** — move `domain/entity/character_sheet/` to
   `domain/character_sheet/entity/` in a dedicated PR after this architecture is stable.

2. **`enum/` and `die/` shared kernel** — move to `domain/shared/` once multiple bounded
   contexts are established.

3. **Domain Events (pub/sub)** — emit events on Turn close for future analytics or notification
   systems. Not needed for MVP.

4. **Memory optimization of `charSheets` cache** — for MVP, all character sheets are loaded at
   session start. Future: lazy load or cap cache size for large matches.

5. **`Initiative` in `RoundOrchestrator.ChangeMode`** — marked TODO in existing code; complete
   when Initiative rules are designed.
