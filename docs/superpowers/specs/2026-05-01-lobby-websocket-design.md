# Lobby WebSocket — Design Spec

**Date:** 2026-05-01
**Status:** Approved
**Scope:** Lobby management flow via WebSocket — from master opening the lobby to match start

## Problem Statement

The WebSocket game server infrastructure (Hub/Room/Client) is implemented, but the
lobby phase lacks several critical features needed for a complete pre-match flow:

1. No enrollment status filter — rejected/pending players can connect
2. No character sheet data in the lobby — players appear as anonymous UUIDs
3. No kick mechanism — master can't remove players from the lobby
4. Match start doesn't persist to the database — no temporal guard for enrollments
5. Pending enrollments are not auto-rejected when match starts

The goal is to implement the complete lobby flow: master opens lobby → manages
enrollments one last time → waits for players to connect → starts the match.

## Decisions Summary

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Lobby opening | WS connection creates Room (current behavior) | Simple, no persistence needed for MVP |
| Who connects | Only `status = 'accepted'` enrollments | Consistency — master accepts via REST first |
| Enrollment management | REST for accept/reject, WS for notifications | No duplication of domain logic |
| Kick mechanism | WS message → Game Server imports RejectEnrollmentUC | Atomic: reject DB + disconnect WS |
| Sheet data visibility | Same as REST API (base vs private) | Consistency with existing model |
| Player-sheet relation | 1:1 per match | Pets/companions are future, not separate enrollments |
| Ready system | None | Players communicate via Discord/external call |
| Min players to start | None | Master is sovereign authority |
| Temporal guard | Lock enrollments on match start | Prevent state changes after match begins |
| Persist match start | Yes, `game_start_at = NOW()` | Required for temporal guard + crash recovery |
| Reconnection | Current behavior (left + joined) | Simple, adequate for MVP |
| match_started payload | Empty | Frontend already has player data from lobby |
| Kick timing | Lobby only (MVP) | Mid-game kick is a separate feature |
| WS pattern | WS as notification channel, REST for mutations | Single source of truth for domain operations |
| Code structure | Domain use cases (same pattern as REST) | Follows existing architecture |
| Implementation approach | Bottom-up (domain → gateway → game server) | Consistent with project conventions |

## Architecture

### Flow Overview

```
┌──────────────────────────────────────────────────────────────┐
│                    LOBBY FLOW                                │
│                                                              │
│  1. Master opens lobby page (frontend loads match data)      │
│  2. Master connects to WS → Room created (state: lobby)     │
│  3. Master accepts/rejects enrollments via REST API          │
│  4. Accepted players connect to WS → player_joined broadcast│
│  5. Master can kick players via WS → reject + disconnect    │
│  6. Master sends start_match via WS:                        │
│     a. Persist game_start_at in DB                          │
│     b. Reject all pending enrollments                       │
│     c. Room state → playing                                 │
│     d. Broadcast match_started                              │
└──────────────────────────────────────────────────────────────┘
```

### Component Interaction

```
Frontend (Master)                     Frontend (Player)
    │                                      │
    │ REST: POST /enrollments/{uuid}/accept│
    │──────────────────────────────────────>│ API Server
    │                                      │
    │ WS: connect                          │ WS: connect
    │─────────────┐                        │────────────┐
    │             │                        │            │
    │         ┌───▼────────────────────────▼──┐         │
    │         │       Game Server (WS)        │         │
    │         │                               │         │
    │         │  Hub                          │         │
    │         │   └── Room (matchUUID)        │         │
    │         │        ├── Master Client      │         │
    │         │        └── Player Clients     │         │
    │         │                               │         │
    │         │  Dependencies:                │         │
    │         │   ├── StartMatchUC            │         │
    │         │   ├── KickPlayerUC            │         │
    │         │   └── EnrollmentReader        │         │
    │         └───────────────────────────────┘         │
    │                      │                            │
    │                      │ SQL                        │
    │                 ┌────▼────┐                       │
    │                 │ PostgreSQL│                      │
    │                 └─────────┘                       │
```

## Domain Layer

### StartMatchUC

**File:** `internal/domain/match/start_match.go`

```go
type IStartMatch interface {
    StartMatch(ctx context.Context, matchUUID, masterUUID uuid.UUID) error
}

type StartMatchUC struct {
    matchRepo      IRepository
    enrollmentRepo enrollment.IRepository
}
```

**Flow:**
1. `matchRepo.GetMatch(ctx, matchUUID)` → validate exists
2. Verify `masterUUID` matches match owner
3. Verify `game_start_at == nil` → `ErrMatchAlreadyStarted`
4. Verify `story_end_at == nil` → `ErrMatchAlreadyFinished`
5. `matchRepo.StartMatch(ctx, matchUUID)` → set `game_start_at = NOW()`
6. `enrollmentRepo.RejectPendingEnrollments(ctx, matchUUID)` → reject all pending

**Errors:**
- `ErrMatchNotFound` (existing)
- `ErrNotMatchMaster` (new, in match domain)
- `ErrMatchAlreadyStarted` (new)
- `ErrMatchAlreadyFinished` (new)

### KickPlayerUC

**File:** `internal/domain/match/kick_player.go`

```go
type IKickPlayer interface {
    KickPlayer(ctx context.Context, matchUUID, masterUUID, playerUUID uuid.UUID) error
}

type KickPlayerUC struct {
    matchRepo      IRepository
    enrollmentRepo enrollment.IRepository
}
```

**Flow:**
1. `matchRepo.GetMatch(ctx, matchUUID)` → validate exists
2. Verify `masterUUID` matches match owner
3. Verify `game_start_at == nil` → `ErrMatchAlreadyStarted` (kick only in lobby)
4. Verify `story_end_at == nil` → `ErrMatchAlreadyFinished`
5. `enrollmentRepo.RejectEnrollmentByPlayerAndMatch(ctx, playerUUID, matchUUID)`

**Errors:**
- Reuses errors from StartMatchUC
- `ErrEnrollmentNotFound` (from enrollment domain — player not enrolled)

### Temporal Guard (modifications to existing UCs)

**Files:**
- `internal/domain/enrollment/accept_enrollment.go`
- `internal/domain/enrollment/reject_enrollment.go`
- `internal/domain/enrollment/enroll_character_sheet.go`

**Change:** Add `match.IRepository` as dependency. Before processing, check:
1. Fetch match via `matchUUID` from the enrollment
2. If `game_start_at != nil` → return `ErrMatchAlreadyStarted`
3. If `story_end_at != nil` → return `ErrMatchAlreadyFinished`

## Gateway Layer

### New Methods

**`internal/gateway/pg/match/start_match.go`:**
```sql
UPDATE matches SET game_start_at = NOW(), updated_at = NOW()
WHERE uuid = $1 AND game_start_at IS NULL
```
Returns `ErrMatchAlreadyStarted` if `RowsAffected == 0`.

**`internal/gateway/pg/enrollment/reject_pending.go`:**
```sql
UPDATE enrollments SET status = 'rejected'
WHERE match_uuid = $1 AND status = 'pending'
```

**`internal/gateway/pg/enrollment/reject_by_player_match.go`:**
```sql
UPDATE enrollments SET status = 'rejected'
WHERE match_uuid = $1 AND status = 'accepted'
AND character_sheet_uuid IN (
    SELECT uuid FROM character_sheets WHERE player_uuid = $2
)
```
Returns `ErrEnrollmentNotFound` if `RowsAffected == 0`.

**`internal/gateway/pg/enrollment/read_accepted_with_sheets.go`:**
```sql
SELECT cs.uuid, cs.nick_name, cs.full_name, cs.character_class,
       cs.category_name, cs.level, cs.alignment, cs.birthday,
       cs.curr_hex_value, cs.points, cs.talent_lvl,
       cs.physicals_lvl, cs.mentals_lvl, cs.spirituals_lvl, cs.skills_lvl,
       cs.player_uuid, cs.master_uuid, cs.campaign_uuid,
       cs.story_start_at, cs.story_current_at, cs.dead_at,
       cs.created_at, cs.updated_at
FROM enrollments e
JOIN character_sheets cs ON cs.uuid = e.character_sheet_uuid
WHERE e.match_uuid = $1 AND e.status = 'accepted'
```
Returns `[]model.CharacterSheetSummary`.

### Modified Method

**`internal/gateway/pg/enrollment/is_player_enrolled.go`:**
Add `AND e.status = 'accepted'` to the existing query.

### Updated Repository Interfaces

**`internal/domain/match/i_repository.go`:**
```go
type IRepository interface {
    CreateMatch(ctx context.Context, m *match.Match) error
    GetMatch(ctx context.Context, uuid uuid.UUID) (*match.Match, error)
    GetMatchMaster(ctx context.Context, matchUUID uuid.UUID) (uuid.UUID, error)
    GetMatchCampaignUUID(ctx context.Context, matchUUID uuid.UUID) (uuid.UUID, error)
    ListMatchesByMasterUUID(ctx context.Context, masterUUID uuid.UUID) ([]*match.Summary, error)
    ListPublicUpcomingMatches(ctx context.Context, after time.Time, masterUUID uuid.UUID) ([]*match.Summary, error)
    StartMatch(ctx context.Context, matchUUID uuid.UUID) error  // NEW
}
```

**`internal/domain/enrollment/i_repository.go`:**
```go
type IRepository interface {
    EnrollCharacterSheet(ctx context.Context, matchUUID, characterSheetUUID uuid.UUID) error
    ExistsEnrolledCharacterSheet(ctx context.Context, characterSheetUUID, matchUUID uuid.UUID) (bool, error)
    GetEnrollmentByUUID(ctx context.Context, enrollmentUUID uuid.UUID) (string, uuid.UUID, error)
    AcceptEnrollment(ctx context.Context, enrollmentUUID uuid.UUID) error
    RejectEnrollment(ctx context.Context, enrollmentUUID uuid.UUID) error
    RejectPendingEnrollments(ctx context.Context, matchUUID uuid.UUID) error                        // NEW
    RejectEnrollmentByPlayerAndMatch(ctx context.Context, playerUUID, matchUUID uuid.UUID) error    // NEW
    GetAcceptedEnrollmentsWithSheets(ctx context.Context, matchUUID uuid.UUID) ([]model.CharacterSheetSummary, error) // NEW
}
```

## Game Server Layer

### New Message Types

**`internal/app/game/message.go`:**

```go
// Client → Server (new)
MsgTypeKickPlayer MessageType = "kick_player"  // master-only

// Server → Client (new)
MsgTypePlayerKicked MessageType = "player_kicked"
```

**New payloads:**
```go
type KickPayload struct {
    PlayerUUID uuid.UUID `json:"player_uuid"`
}

type PlayerKickedPayload struct {
    UUID     uuid.UUID `json:"uuid"`
    Nickname string    `json:"nickname"`
}
```

### Room Enhancements

**New dependencies injected into Room:**
```go
type Room struct {
    // ... existing fields ...
    startMatchUC   match.IStartMatch
    kickPlayerUC   match.IKickPlayer
    enrollmentRepo EnrollmentReader  // for sheet data
}
```

**New interface:**
```go
type EnrollmentReader interface {
    GetAcceptedEnrollmentsWithSheets(ctx context.Context, matchUUID uuid.UUID) ([]model.CharacterSheetSummary, error)
}
```

**Modified `StartMatch()`:**
1. Verify master + lobby state (existing)
2. Call `startMatchUC.StartMatch()` → persist + reject pending
3. Change room state to `playing`
4. Broadcast `match_started` (empty payload)

**New `KickPlayer()`:**
1. Verify master + lobby state
2. Call `kickPlayerUC.KickPlayer()` → reject enrollment in DB
3. Send `player_kicked` unicast to kicked client
4. Close kicked client connection
5. Broadcast `player_kicked` to remaining clients

**Modified `sendRoomState()`:**
1. Fetch sheets via `enrollmentRepo.GetAcceptedEnrollmentsWithSheets()`
2. Build personalized payload per recipient:
   - Master: all players with `CharacterPrivateSummaryResponse`
   - Player: own private data + others' base data

**Modified `broadcastPlayerJoined()`:**
1. On register, fetch the new player's sheet via `enrollmentRepo.GetAcceptedEnrollmentsWithSheets()`
2. Build personalized `player_joined` per recipient:
   - Master: receives `PlayerJoinedPayload` with private sheet data
   - Other players: receives `PlayerJoinedPayload` with base sheet data only
   - The joining player does NOT receive `player_joined` (they get `room_state` instead)

**`PlayerJoinedPayload` (enhanced):**
```go
type PlayerJoinedPayload struct {
    UUID           uuid.UUID   `json:"uuid"`
    Nickname       string      `json:"nickname"`
    CharacterSheet interface{} `json:"character_sheet"` // private or base, per recipient
}
```

**Modified `handleClientMessage()`:**
Add case for `MsgTypeKickPlayer`:
```go
case MsgTypeKickPlayer:
    var payload KickPayload
    // unmarshal, validate, call KickPlayer()
```

### Handler Enhancements

**New dependencies:**
```go
type Handler struct {
    hub            *Hub
    matchRepo      MatchRepository
    enrollmentRepo EnrollmentChecker
    startMatchUC   match.IStartMatch    // NEW
    kickPlayerUC   match.IKickPlayer    // NEW
    sheetReader    EnrollmentReader     // NEW
}
```

These are passed to `NewRoom()` when creating rooms via `GetOrCreateRoom()`.

### Connection Validation Fix

**`IsPlayerEnrolledInMatch`** query updated to include `AND e.status = 'accepted'`.
Only players with accepted enrollments can connect to the lobby.

## Room State Payload (Enhanced)

### For Master
```json
{
  "match_uuid": "...",
  "state": "lobby",
  "players": [
    {
      "uuid": "player-uuid",
      "nickname": "Player1",
      "is_master": false,
      "is_online": true,
      "character_sheet": {
        "uuid": "sheet-uuid",
        "nick_name": "Gon",
        "full_name": "Gon Freecss",
        "character_class": "Hunter",
        "level": 5,
        "alignment": "Chaotic Good",
        "category_name": "Enhancer",
        "stamina": { "min": 0, "current": 100, "max": 100 },
        "health": { "min": 0, "current": 80, "max": 80 }
      }
    }
  ]
}
```

### For Player (seeing others)
```json
{
  "match_uuid": "...",
  "state": "lobby",
  "players": [
    {
      "uuid": "other-player-uuid",
      "nickname": "Player2",
      "is_master": false,
      "is_online": true,
      "character_sheet": {
        "uuid": "sheet-uuid",
        "nick_name": "Killua"
      }
    },
    {
      "uuid": "own-uuid",
      "nickname": "Player1",
      "is_master": false,
      "is_online": true,
      "character_sheet": {
        "uuid": "own-sheet-uuid",
        "nick_name": "Gon",
        "full_name": "Gon Freecss",
        "character_class": "Hunter",
        "level": 5,
        ...
      }
    }
  ]
}
```

## Testing Strategy

### Domain Tests (unit, table-driven)

**`internal/domain/match/start_match_test.go`:**
| Case | Input | Expected |
|------|-------|----------|
| Valid start | valid master, lobby match | success, StartMatch + RejectPending called |
| Not master | wrong user | ErrNotMatchMaster |
| Already started | game_start_at set | ErrMatchAlreadyStarted |
| Already finished | story_end_at set | ErrMatchAlreadyFinished |
| Match not found | invalid UUID | ErrMatchNotFound |

**`internal/domain/match/kick_player_test.go`:**
| Case | Input | Expected |
|------|-------|----------|
| Valid kick | valid master, lobby, enrolled player | success, RejectByPlayerAndMatch called |
| Not master | wrong user | ErrNotMatchMaster |
| Already started | game_start_at set | ErrMatchAlreadyStarted |
| Player not enrolled | invalid player | ErrEnrollmentNotFound |

**Enrollment UCs temporal guard tests** (added to existing test files):
| Case | Expected |
|------|----------|
| Accept after match started | ErrMatchAlreadyStarted |
| Reject after match started | ErrMatchAlreadyStarted |
| Enroll after match started | ErrMatchAlreadyStarted |
| Accept after match finished | ErrMatchAlreadyFinished |

### Gateway Tests (integration, PostgreSQL)

- `start_match` — verify game_start_at persisted, idempotent behavior
- `reject_pending` — verify only pending changed, accepted untouched
- `reject_by_player_match` — verify correct enrollment rejected
- `read_accepted_with_sheets` — verify correct join, data returned
- `is_player_enrolled` — verify status filter (rejected/pending excluded)

### Game Server Tests (unit, mocked dependencies)

- Handler: rejected enrollment → 403 before upgrade
- Room: kick message flow (UC called → client disconnected → broadcast)
- Room: start_match flow (UC called → state change → broadcast)
- Room: personalized room_state (master vs player view)
- Messages: new message type marshaling

## Future Improvements (out of scope)

- Mid-game kick with character reassignment
- Lobby discovery (listing open lobbies)
- `match_participants` table for mid-game invites
- Cross-process notification (PostgreSQL LISTEN/NOTIFY) if API and Game servers need real-time sync
- Spectator mode
