# WebSocket Game Server — Design Spec

**Date:** 2026-04-29
**Status:** Approved
**Scope:** MVP real-time communication infrastructure for match execution

## Problem Statement

The HxH RPG platform needs real-time bidirectional communication between the
master and players during a match. The REST API handles CRUD operations
(creating matches, enrolling characters, etc.), but once a match is running,
all participants need instant event delivery: actions, reactions, turn changes,
chat messages, and game state updates.

## Context: Match Runtime Architecture

A Match subdivides into **Scenes**, which contain **Turns**, which contain
**Rounds**. This hierarchy drives the entire game flow:

```
Match (partida em execução)
├── Scene: Roleplay (interação/investigação)
│   └── Turns (modo free — sem disputa de tempo)
│       └── Rounds (ação de cada personagem)
└── Scene: Battle (combate)
    └── Turns (modo race — ordem por velocidade)
        └── Rounds (ação + reações, fila de prioridade)
```

**Scene categories** (roleplay vs battle) exist for classification,
historical ordering, and narrative readability. They do NOT determine turn
mode — a roleplay scene could theoretically use race mode and vice versa.

**Turn modes:**
- **Free** — no time pressure; players act in natural order
- **Race** — milliseconds matter; actions are resolved by speed priority queue

The Turn Engine manages turn execution without knowing the scene category.
This separation is intentional and must be preserved.

## Approach: gorilla/websocket + Hub/Room/Client

Uses the existing `gorilla/websocket` dependency (v1.5.3, already in go.mod)
with a well-structured Hub/Room/Client architecture inspired by the official
gorilla chat example.

### Why This Approach

- gorilla/websocket is already a project dependency
- The Hub/Room/Client pattern is the industry standard for this use case
- Extensive documentation and community examples
- Room abstraction can later be extracted to separate processes for scaling

### Alternative Considered

**nhooyr.io/websocket** — more idiomatic Go API (native context.Context,
concurrent-safe writes). Rejected because: the Hub/Room pattern already
solves concurrent writes via per-client write pumps, and gorilla has vastly
more reference implementations for this exact pattern. The practical
difference is minimal for this project.

## Architecture

### Component Overview

```
┌─────────────────────────────────────────────────┐
│              API Server (cmd/api)                │
│            REST — port 5000                      │
│  POST /matches, GET /campaigns, etc.            │
└──────────────────────┬──────────────────────────┘
                       │ Shared PostgreSQL
┌──────────────────────▼──────────────────────────┐
│             Game Server (cmd/game)               │
│           WebSocket — port 8080                  │
│                                                  │
│  ┌────────────────────────────────────────────┐  │
│  │                   Hub                      │  │
│  │  rooms map[matchUUID]*Room                 │  │
│  │                                            │  │
│  │  ┌──────────────┐  ┌──────────────┐       │  │
│  │  │  Room #42    │  │  Room #87    │  ...  │  │
│  │  │  (lobby)     │  │  (playing)   │       │  │
│  │  │  👑 Master   │  │  👑 Master   │       │  │
│  │  │  🎮 Player1  │  │  🎮 Player1  │       │  │
│  │  │  🎮 Player2  │  │  🎮 Player2  │       │  │
│  │  └──────────────┘  └──────────────┘       │  │
│  └────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────┘
```

### File Structure

```
internal/app/game/
├── hub.go          — Hub: manages all active Rooms
├── room.go         — Room: one match session (lobby → playing → closed)
├── client.go       — Client: one WebSocket connection (readPump/writePump)
├── message.go      — Message types and JSON protocol
├── handler.go      — HTTP upgrade handler with JWT auth
└── server.go       — chi router setup, CORS, upgrader config
```

### Hub

Manages all active rooms. Singleton per game server process.

```go
type Hub struct {
    rooms      map[uuid.UUID]*Room  // matchUUID → Room
    register   chan *Client
    unregister chan *Client
    mu         sync.RWMutex
}
```

- `Run()` — main loop processing register/unregister
- `GetOrCreateRoom(matchUUID, masterUUID)` — finds or creates a room
- `RemoveRoom(matchUUID)` — cleanup when room closes

### Room

One per match. Manages connected clients and room state.

```go
type Room struct {
    matchUUID  uuid.UUID
    masterUUID uuid.UUID
    state      RoomState              // lobby | playing | closed
    clients    map[uuid.UUID]*Client  // userUUID → Client
    broadcast  chan []byte
    register   chan *Client
    unregister chan *Client
}
```

States: `lobby` → `playing` → `closed`

- `Run()` — main loop: register/unregister clients, broadcast messages
- `IsMaster(userUUID)` — checks if user is the match master
- `Broadcast(msg)` — sends message to all clients
- `SendTo(userUUID, msg)` — unicast to specific client

### Client

One per WebSocket connection. Two goroutines per client.

```go
type Client struct {
    userUUID uuid.UUID
    conn     *websocket.Conn
    room     *Room
    send     chan []byte
}
```

- `ReadPump()` — reads messages from WS, routes to room
- `WritePump()` — writes messages from send channel to WS

### Message Protocol

All messages are JSON with a common envelope:

```json
{
  "type": "start_match",
  "payload": { ... },
  "sender_id": "user-uuid",
  "timestamp": "2026-04-29T01:00:00Z"
}
```

`sender_id` and `timestamp` are always set by the server (never trust client).

#### Server → Client Messages

| Type | Payload | Description |
|------|---------|-------------|
| `room_state` | `{match_uuid, state, players: [{uuid, nickname, is_master, is_online}]}` | Current room state (sent on join) |
| `player_joined` | `{uuid, nickname}` | A player connected |
| `player_left` | `{uuid, nickname}` | A player disconnected |
| `match_started` | `{}` | Match transitioned to playing |
| `chat_message` | `{message}` | Chat broadcast |
| `error` | `{code, message}` | Error (unicast to sender) |

#### Client → Server Messages

| Type | Payload | Required Role | Description |
|------|---------|---------------|-------------|
| `start_match` | `{}` | Master only | Transition room to playing |
| `chat` | `{message}` | Any | Send chat message |

### Connection Flow

1. Client connects: `GET /ws?match_uuid=XXX` with `Authorization: Bearer <JWT>`
2. Handler validates JWT → extracts userUUID
3. Handler validates match_uuid parameter
4. Handler queries DB: match exists? User is master or enrolled?
5. On validation failure: return HTTP error (before upgrade)
6. On success: upgrade to WebSocket
7. Hub registers client in the appropriate Room (creates if needed)
8. Room sends `room_state` to new client
9. Room broadcasts `player_joined` to other clients

### Connection Validation (pre-upgrade)

| Check | Failure | HTTP Status |
|-------|---------|-------------|
| JWT valid? | Invalid/expired token | 401 |
| match_uuid present and valid UUID? | Missing/malformed | 400 |
| Match exists in DB? | Not found | 404 |
| User is master or enrolled? | Not authorized | 403 |

### Room State Machine

```
            Master connects
                 │
                 ▼
        ┌─────────────┐
        │    LOBBY     │ ◄── players can join/leave
        │              │     chat available
        └──────┬───────┘
               │ Master sends start_match
               ▼
        ┌─────────────┐
        │   PLAYING    │ ◄── match in progress
        │              │     chat available
        └──────┬───────┘     (future: game actions)
               │ All disconnect or master ends
               ▼
        ┌─────────────┐
        │   CLOSED     │ → Room removed from Hub
        └─────────────┘
```

### Authentication

The WebSocket connection reuses the same JWT from the REST API. The token is
validated once at connection time (HTTP upgrade). Once the WebSocket is
established, the connection persists regardless of token expiry.

**Future improvement:** Token refresh over WebSocket — server sends a new JWT
before the current one expires.

### Reconnection

MVP approach: if a client disconnects and reconnects, they get a fresh Client
in the same Room. The Room sends current `room_state` on join, so the client
is immediately up to date. No message replay in MVP.

### Integration with Existing Code

**Reused packages:**
- `pkg/auth` — `ValidateToken()` for JWT authentication
- `pkg` (pgfs) — PostgreSQL connection pool
- `internal/config` — `LoadCORS()` for CORS configuration
- `go-chi/chi` — HTTP router (consistency with API server)
- `gorilla/websocket` — WebSocket implementation (already in go.mod)
- `internal/gateway/pg/match` — validate match existence
- `internal/gateway/pg/enrollment` — validate player enrollment

**Entry point rewrite:** `cmd/game/main.go` will be rewritten from the
current prototype to a proper server with dependency injection, following
the same pattern as `cmd/api/main.go`.

### Estimated Size

| File | Lines | Purpose |
|------|-------|---------|
| hub.go | ~80 | Room management |
| room.go | ~120 | Client management, state machine, broadcast |
| client.go | ~100 | readPump/writePump goroutines |
| message.go | ~50 | Message types, marshal/unmarshal |
| handler.go | ~60 | HTTP→WS upgrade, auth, validation |
| server.go | ~30 | chi router, server setup |
| **Total** | **~440** | |

## Testing Strategy

- **Unit tests:** Hub, Room, Client with mock connections
- **Integration tests:** Full WebSocket connection lifecycle
- **gorilla/websocket test helpers** for simulating client connections

## Future Improvements (not in MVP scope)

- Token refresh over WebSocket
- Add/remove players at runtime (master control)
- Scale to separate process per Room
- Game actions (turns, rounds, combat) over the same message infrastructure
- Message replay on reconnection
- Spectator mode
