# Match Maps API

## Overview

Endpoints to attach, retrieve, and detach a tactical map from a match. One map per match. Operations blocked after match starts.

## REST Endpoints

### POST /matches/{match_uuid}/map

Attach a map to a match. Replaces any previously attached map (upsert).

**Auth:** required (JWT Bearer)  
**Role:** match master only

**Request body:**
```json
{
  "map_uuid": "uuid"
}
```

**Responses:**
| Status | Description |
|--------|-------------|
| 200 | Map attached. Body: `{"match_map": {"match_uuid": "...", "map_uuid": "...", "attached_at": "ISO8601"}}` |
| 400 | Bad request (invalid UUID) |
| 401 | Unauthenticated |
| 403 | Not the match master |
| 404 | Match or map not found |
| 422 | Match already started |
| 500 | Internal server error |

---

### GET /matches/{match_uuid}/map

Get the map currently attached to a match.

**Auth:** required (JWT Bearer)

**Responses:**
| Status | Description |
|--------|-------------|
| 200 | Map attached. Body: `{"match_map": {"match_uuid": "...", "map_uuid": "...", "attached_at": "ISO8601"}}` |
| 204 | No map attached |
| 400 | Bad request |
| 401 | Unauthenticated |
| 500 | Internal server error |

---

### DELETE /matches/{match_uuid}/map

Detach the map from a match.

**Auth:** required (JWT Bearer)  
**Role:** match master only

**Responses:**
| Status | Description |
|--------|-------------|
| 204 | Map detached |
| 400 | Bad request |
| 401 | Unauthenticated |
| 403 | Not the match master |
| 404 | Match or map not found |
| 422 | Match already started |
| 500 | Internal server error |

---

## WebSocket: lobby_piece_moved

**Direction:** Client → Server (broadcast to all other participants)  
**When:** During lobby phase, when a participant moves a piece on the tactical map.

**Send payload:**
```json
{
  "type": "lobby_piece_moved",
  "payload": {
    "piece_id": "uuid-string",
    "slot": {
      "kind": "square",
      "col": 3,
      "row": 5
    }
  }
}
```

Hex slot:
```json
{
  "slot": { "kind": "hex", "q": 2, "r": -1 }
}
```

**Broadcast to other clients:**
```json
{
  "type": "lobby_piece_moved",
  "sender_id": "user-uuid",
  "payload": {
    "piece_id": "...",
    "slot": { "kind": "square", "col": 3, "row": 5 }
  }
}
```

**Notes:**
- Server broadcasts to all lobby participants EXCEPT the sender.
- No server-side piece ownership validation in Phase 6. Client restricts drag to allowed pieces.
- TODO: validate piece ownership per user (Phase 7+).
