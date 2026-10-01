---
applyTo: "internal/app/game/**"
---

# Game Server

## MatchSession

Stateful in-memory match state (`matchsession/`). Lives in `Room`, initialized by `InitMatchSessionUC` on `StartMatch`.

Holds: active `Round`, action priority queue, char-sheet cache and `statuses` (live combat state) keyed by **sheet UUID** (`CharacterID`, the same ID board pieces carry), map walls, pieces, grid cell size, the `MatchRules`, the weapon catalogue and the `RollSource`. `participants` stays keyed by **player UUID** for authorization; `charToPlayer` (sheet-UUID-string → player UUID) bridges the two axes.

**The session is where the dice fall.** `EnqueueAction` and `AttachReaction` call `rollActionDice`, which fills `action.RollCheck.Attempts` once, on arrival. Everything downstream derives and never rolls — that is what lets the master edit and the reactions collide without re-rolling a player's die.

**`Action.actorID` is the sheet UUID**, not the player's; `ActionPayload.actorId` carries it and is required. Authorization is still per player: `charToPlayer[actorCharID] == playerUUID`.

`OpenNextAction`/`PullAction` return a `TurnTransition` (closed turn, opened turn, a resolution for each, and the `DamagedCharacter`s the close applied). Damage is a dry-run in every resolution and is written to the sheet exactly once, when the turn closes.

Use cases receive `*MatchSession` directly — they do not own or store it.

`room.go` owns the concurrency lock (`r.mu sync.RWMutex`). `MatchSession` exposes data without its own internal lock — callers must hold `r.mu` before accessing session state.

## RoomDeps

`RoomDeps` (`room_deps.go`) is every external dependency a `Room` can have, in one struct — the handler, the hub and the room used to take the same growing list of use cases positionally, and every new dependency meant editing all three plus every test that builds a handler. A field left `nil` is a capability the room does not have; the arm that needs it checks (`r.deps.XxxUC == nil` → no-op or skip), the same convention every field's own doc comment states. `LoadBoardUC`/`SaveBoardUC`/`MemoryLoader` (board + fog persistence, B3/B14), `SheetOwnership` (lobby piece ownership, B14 — fails CLOSED when nil, the one field that is the exception to the rule above), `MasterActionRepo` (B14/§4.8) and `EventRepo` (B15) were added for Phase 6 closure.

## The board — the server owns it (B3, B14)

Since Phase 6 closure, the `Room` is the board's owner, not the client. `loadBoard` (`board.go`) reads `match_boards` (or a fresh snapshot of the attached map, if the match has no board row yet) and replaces `r.pieces`/`r.walls`/`r.grid`/`r.bg`. It runs on `Run`'s own goroutine once when the `Room` is born and — while the match is still a lobby — again on every master connect (that is the moment `map_state_sync` used to seed the board by hand); `StartMatch` (on the master's read pump) runs it once more, right before the start's own save — the attached map can change over REST while the lobby socket is open. `LoadMatchBoardUC` only accepts a saved row whose `map_uuid` is the map attached now; `boardLoaded` only counts a load that installed a board. A lobby load that finds no map attached (detached over REST) clears the room's board and `mapUUID`, so `persistBoard` is a no-op until a map is attached again. `StartMatch` holds `persistMu` from that reload's read through its own save (`persistBoardLocked`; lock order `persistMu` → `r.mu`), so an in-flight lobby save cannot be read back stale. After the match starts, the board loads only at birth; a reconnect no longer reloads it. `map_state_sync` itself is now accepted and ignored: it writes nothing, and answers the sender with a `map_full_state` — kept only so the front already in the wild does not break before Phase 13 removes the send.

`persistBoard(reason string)` (`board.go`) is the single write point for every definitive board change: a lobby move/remove, `start_match`, the three turn-closing verbs (after `applyClosedEscapes`, before `persistClosedTurn`'s own round trip — so a crash between the two leaves disk agreeing with the turn that is about to be lost, not one that half-committed), a master's piece action (move/place/remove via `enqueue_master_action`), and wall interact/reveal. It snapshots under `r.mu.RLock()`, releases it, and writes outside the lock; `persistMu` (a dedicated `sync.Mutex`, separate from `r.mu`) wraps the snapshot **and** the write together, so two saves racing from two read pumps land in the order their snapshots were taken, not the order their writes happen to finish. A save failure is logged with what was lost and swallowed — the table goes on, the same policy `persistClosedTurn` already uses. The caller must never hold `r.mu` when calling it.

## Two send lanes, and the order between them

Two lanes carry server → client messages, and they do **not** have a guaranteed order relative to each other: `r.broadcast` (a channel, delivered by `Run`'s own goroutine) and the direct per-client lane (`client.SendMessage`, used by `dispatchPerPlayer`). Since B2, `turn_opened` is projected per recipient (master gets the full action, everyone else — including the action's own owner — gets it through `ProjectAction` then cut to the `Opened` level), which means it cannot go out as a single broadcast anymore. Moving it to the direct lane risked it arriving BEFORE the previous turn's `turn_closed`, which still went out on `r.broadcast`. The fix: both `turn_closed` and `turn_opened` moved to the direct lane, keeping today's order — `piece_moved` → `turn_closed` → `resolution_updated` → `turn_opened` — because within one lane, on one goroutine, send order is arrival order. The contract now promises that order explicitly (see `match-combat-ws.md`).

## WS Event Design

**Prefer fewer event types with richer payloads** over many event types with minimal data. The client derives state transitions (e.g., "damaged" vs "destroyed") from payload fields (`hp`, `destroyed`, etc.), not from the event type name. Add a new event type only when the domain transition is meaningfully distinct and cannot be inferred from existing payload fields.
