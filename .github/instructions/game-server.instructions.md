---
applyTo: "internal/app/game/**"
---

# Game Server (WebSocket Layer)

## MatchSession

`MatchSession` is the stateful in-memory match state. It lives inside `Room` and is initialized by `InitMatchSessionUC` when `StartMatch` is called.

It holds:
- The active `Round` (current combat round)
- Action priority queue (ordered list of pending actions)
- Char-sheet cache keyed by `playerUUID`

`Room` accesses it via `r.session` (guarded by `r.mu`). Use cases receive `*MatchSession` directly — they do not own or store it.

## Room / Client / Hub Pattern

- `Hub` — manages the set of active rooms; creates `Room` on match join
- `Room` — owns one `MatchSession` after `StartMatch`; routes WebSocket messages to use cases
- `Client` — one per connected user; wraps the WebSocket connection

## Message Routing (room.go)

`handleClientMessage` dispatches on `MessageType`. Key invariants:
- `actorID` is always `client.userUUID` — never from the payload
- `ReactToID != uuid.Nil` in an `enqueue_action` payload → silently rerouted to `handleReaction`
- `Dodge != nil && ReactToID == uuid.Nil` → validation error (Dodge must always be a reaction)
- Both `enqueue_action` and `attach_reaction` use the unified `ActionPayload` shape

## Action Construction

`buildAction` in `action_mapper.go` is the single place in the delivery layer that calls `action.NewAction`. Keep all payload→domain mapping there.
