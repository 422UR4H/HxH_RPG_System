---
applyTo: "internal/app/game/**"
---

# Game Server

## MatchSession

Stateful in-memory match state (`matchsession/`). Lives in `Room`, initialized by `InitMatchSessionUC` on `StartMatch`.

Holds: active `Round`, action priority queue, char-sheet cache keyed by `playerUUID`.

Use cases receive `*MatchSession` directly — they do not own or store it.
