---
applyTo: "internal/app/game/**"
---

# Game Server

## MatchSession

Stateful in-memory match state (`matchsession/`). Lives in `Room`, initialized by `InitMatchSessionUC` on `StartMatch`.

Holds: active `Round`, action priority queue, char-sheet cache and `statuses` (live combat state) keyed by **sheet UUID** (`CharacterID`, the same ID board pieces carry), map walls, pieces, and grid cell size. `participants` stays keyed by **player UUID** for authorization; `charToPlayer` (sheet-UUID-string → player UUID) bridges the two axes.

Use cases receive `*MatchSession` directly — they do not own or store it.

`room.go` owns the concurrency lock (`r.mu sync.RWMutex`). `MatchSession` exposes data without its own internal lock — callers must hold `r.mu` before accessing session state.

## WS Event Design

**Prefer fewer event types with richer payloads** over many event types with minimal data. The client derives state transitions (e.g., "damaged" vs "destroyed") from payload fields (`hp`, `destroyed`, etc.), not from the event type name. Add a new event type only when the domain transition is meaningfully distinct and cannot be inferred from existing payload fields.
