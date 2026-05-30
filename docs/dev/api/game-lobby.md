# Game Server — Lobby WebSocket Protocol

**Status:** In progress
**Server port:** 8081
**URL:** `ws://localhost:8081/ws?match_uuid=<uuid>&token=<jwt>&nickname=<name>`

## New Messages (Task 1: Lobby lifecycle)

### Server → Client

#### `lobby_closed`

Broadcast to all clients when the master cancels the lobby (via `cancel_lobby`).

```json
{ "type": "lobby_closed", "payload": "{}" }
```

#### `lobby_not_open`

Sent to a participant who tries to connect before the master has opened the lobby.

```json
{ "type": "lobby_not_open", "payload": "{}" }
```

The server immediately follows with a WebSocket close frame (code 4001, reason: "lobby not open") and closes the connection.

### Client → Server

#### `cancel_lobby`

Sent by the master to cancel the open lobby. Broadcasts `lobby_closed` to all connected clients and stops the room.

```json
{ "type": "cancel_lobby", "payload": "{}" }
```

Error if sender is not the master or room is not in lobby state:

```json
{ "type": "error", "payload": "{\"code\":\"forbidden\",\"message\":\"...\"}" }
```
