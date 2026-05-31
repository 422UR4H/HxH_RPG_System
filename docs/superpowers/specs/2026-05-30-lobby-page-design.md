# Lobby Page — Design Spec

**Date:** 2026-05-30  
**Status:** Approved  
**Scope:** Cross-stack — game server (Go) + frontend (React/TS). Last step before StartMatch.

---

## Problem Statement

The WebSocket game server is fully implemented (Hub/Room/Client, StartMatch, KickPlayer). The
`MatchPage` already navigates to `/lobby` on master confirmation. What is missing is the lobby
page itself, the WebSocket hook that drives it, and the small backend additions that make the
lobby lifecycle complete (lobby cancellation broadcast, player connection guard).

---

## Decisions Summary

| Decision | Choice | Rationale |
|---|---|---|
| Cancel lobby notification | `lobby_closed` WS message broadcast | Players redirect automatically; no polling needed |
| Chat in lobby | None | Out of scope for MVP |
| Reconnection | Auto on mount + exponential backoff + throttle | Natural for realtime apps |
| Reconnection throttle | 5 attempts / 60 s window | Prevents abuse without blocking unstable connections |
| Sidebar component | Extend `CharacterSidebarItem` with `isOnline?` | Reuse, no duplication |
| Player entry point | "Entrar no Lobby" button on MatchPage | No polling; simple, no new backend endpoint |
| Player connect guard | Upgrade WS → send `lobby_not_open` → close 4001 | Only way to pass readable error through WS handshake |
| WS hook architecture | Feature-specific `useLobbyWs` | Cohesive, YAGNI — game phase gets its own hook |
| Redirect on `match_started` | Automatic via `onMatchStarted` callback | Players need zero clicks after lobby |
| Back button | Hidden on LobbyPage | Exits only via cancel (master) or `lobby_closed` / `kicked` |
| Kick confirmation | `ConfirmDialog` on kick button | Prevents accidental kicks; matches project patterns |
| Kick availability | Only when participant `isOnline` | Cannot kick someone who hasn't connected |

---

## Architecture Overview

```
MatchPage (master)
  │  clicks "Abrir Lobby" → confirm → navigate /lobby
  ▼
LobbyPage
  ├── useMatchDetails (REST)          — title, masterUuid
  ├── useMatchEnrollments (REST)      — accepted enrollments + sheet data
  └── useLobbyWs (WebSocket)
        │  ws://localhost:8081/ws?match_uuid=<uuid>&token=<jwt>&nickname=<name>
        │
        ▼
   Game Server
     Hub.GetOrCreateRoom  (master)
     Hub.GetRoom          (participant — nil → lobby_not_open)
     Room (state: lobby → playing)

MatchPage (participant with accepted enrollment)
  │  clicks "Entrar no Lobby" → navigate /lobby
  ▼
LobbyPage
  └── useLobbyWs connects → room_state received → shows lobby
```

---

## Backend Changes

### `internal/app/game/message.go`

Two new server→client message types:

```go
MsgTypeLobbyClosed  MessageType = "lobby_closed"   // master cancelled lobby
MsgTypeLobbyNotOpen MessageType = "lobby_not_open" // room does not exist for participant
```

`lobby_not_open` is sent as a WS message (after upgrade) rather than an HTTP error because
browsers cannot read the body of a rejected WS handshake response.

### `internal/app/game/hub.go`

New method alongside `GetOrCreateRoom`:

```go
func (h *Hub) GetRoom(matchUUID uuid.UUID) *Room
```

Returns existing room or `nil`. Never creates. Used by `handler.go` for non-master connections.

### `internal/app/game/handler.go`

After auth and match validation, split by role:

- **Master** → `GetOrCreateRoom` (existing behavior)
- **Participant** → `GetRoom`; if `nil`: upgrade WS, send `{type:"lobby_not_open"}`, close with
  code `4001`, return. If room exists: proceed normally.

### `internal/app/game/room.go`

New method:

```go
func (r *Room) CloseLobby(masterUUID uuid.UUID) error
```

1. Validate `masterUUID == r.masterUUID`
2. Broadcast `{type: "lobby_closed"}` to all clients
3. Call `r.Stop()`

---

## Frontend Changes

### `src/components/atoms/PageHeader.tsx`

Add `showBack?: boolean` (default `true`). When `false`, `<BackButton />` is not rendered.

### `src/components/templates/DetailPageTemplate.tsx`

Add `hideBack?: boolean`. Passes `showBack={!hideBack}` to `<PageHeader />`. No existing page
is affected (prop omitted = `false` = current behaviour).

### `src/components/molecules/CharacterSidebarItem.tsx`

Add `isOnline?: boolean` prop:

- `isOnline === true` → green badge `"ONLINE"` + `colors.statusOngoing` left border
- `isOnline === false` → gray badge `"AGUARDANDO"` + `colors.statusLeft` left border
- `isOnline === undefined` → no badge (existing behaviour unchanged)

Border-left precedence: `isDead` > `isPending` > `isOnline===false` > `isOnline===true` >
`isNpc` > default orange.

### `src/pages/MatchPage.tsx`

New condition for participants with accepted enrollment:

```ts
const canEnterLobby =
  !isMaster &&
  !match.gameStartAt &&
  enrollments.some(e => e.characterSheet.uuid === sheetId && e.status === "accepted");
```

Renders "Entrar no Lobby" button in `BottomActions`. Mutually exclusive with `canEnroll`.
Navigates to `/campaigns/:campaignId/matches/:matchId/lobby`.

---

## `useLobbyWs` Hook

**File:** `src/hooks/useLobbyWs.ts`

### Types

```ts
type WsStatus =
  | "connecting"
  | "connected"
  | "disconnected"
  | "lobby_not_open"  // server: room does not exist
  | "kicked"          // this user was kicked by master
  | "lobby_closed"    // master cancelled the lobby
  | "throttled"       // exceeded 5 reconnects in 60 s
  | "error";

type LobbyParticipant = {
  uuid: string;
  nickname: string;
  isMaster: boolean;
  isOnline: boolean; // always true — only connected participants are in this list
};
```

### Interface

```ts
useLobbyWs(params: {
  matchUuid: string;
  token: string;
  nickname: string;
  onMatchStarted: () => void;
}): {
  status: WsStatus;
  participants: LobbyParticipant[];
  sendStartMatch: () => void;
  sendKick: (userUuid: string) => void;
  sendCancelLobby: () => void;
}
```

### Connection URL

```
${import.meta.env.VITE_WS_URL}/ws?match_uuid=<uuid>&token=<jwt>&nickname=<name>
```

`VITE_WS_URL` defaults to `ws://localhost:8081` in `.env`.

### Message Handling (`onmessage`)

| Received type | Effect |
|---|---|
| `room_state` | Replace `participants` with full list from payload |
| `player_joined` | Append participant |
| `player_left` | Remove participant by uuid |
| `player_kicked` (self) | `status → "kicked"` — no reconnect |
| `match_started` | Call `onMatchStarted()` — triggers navigate in page |
| `lobby_closed` | `status → "lobby_closed"` — no reconnect |
| `error {code: "lobby_not_open"}` | `status → "lobby_not_open"` — no reconnect |

### Reconnection Logic

On unexpected disconnect (`onclose` with non-terminal status):

1. Increment attempt counter and record timestamp
2. If 5+ attempts in last 60 s → `status = "throttled"`, stop
3. Otherwise: wait with exponential backoff (`500ms * 2^attempt`, capped at 30 s) then reconnect

Terminal statuses (`lobby_not_open`, `kicked`, `lobby_closed`, `throttled`) never trigger
reconnection. `useEffect` cleanup calls `ws.close()` to avoid reconnect on unmount.

---

## `LobbyConnectionSidebarItem`

**File:** `src/features/match/LobbyConnectionSidebarItem.tsx`

Thin wrapper around `CharacterSidebarItem`:

```ts
interface LobbyConnectionSidebarItemProps {
  enrollment: Enrollment;
  isOnline: boolean;
  isMaster: boolean;
  onKick?: (enrollmentPlayerUuid: string) => void;
}
```

- Passes `isOnline` to `CharacterSidebarItem`
- Renders "Expulsar" button only when `isMaster && isOnline`
- On click: opens `ConfirmDialog` ("Tem certeza que deseja expulsar este jogador?")
- On confirm: calls `onKick(playerUuid)`
- Button disabled when `!isOnline` (participant hasn't connected yet)

---

## `LobbyPage`

**File:** `src/pages/LobbyPage.tsx`  
**Route:** `/campaigns/:campaignId/matches/:matchId/lobby`

### Data Sources

```ts
const { data: match }       = useMatchDetails(token, matchId);
const { data: enrollments } = useMatchEnrollments(token, matchId, true); // always enabled
const { status, participants, sendStartMatch, sendKick, sendCancelLobby } = useLobbyWs({
  matchUuid: matchId,
  token,
  nickname: user?.username ?? "",
  onMatchStarted: () => navigate(`/campaigns/${campaignId}/matches/${matchId}/game`),
});
```

### Combined List

```ts
const acceptedEnrollments = enrollments.filter(e => e.status === "accepted");

const lobbyEntries = acceptedEnrollments.map(enrollment => ({
  enrollment,
  isOnline: participants.some(p => p.uuid === enrollment.characterSheet.playerUuid),
}));
```

### Layout

Uses `<DetailPageTemplate hideBack>`:

- **Left sidebar:** `lobbyEntries` rendered with `LobbyConnectionSidebarItem`. Master also
  appears in sidebar via `participants` (using a local `MasterLobbyItem` styled component
  inside `LobbyPage.tsx` — no character sheet, just name + "MESTRE" badge).
- **Right sidebar:** same `<RulesSidebar>` as `MatchPage`.
- **Main content:**
  - Match title + `"LOBBY ABERTO"` pill + `"X/Y conectados"` counter
  - WS status panel (pulsing green dot when connected; amber when reconnecting; red on error)
  - Status-specific messages (see below)
  - Action buttons

### WS Status Messages

| Status | Message displayed |
|---|---|
| `connecting` | "Conectando ao lobby..." |
| `connected` | — (status panel shows green dot) |
| `disconnected` | "Reconectando..." |
| `lobby_not_open` | "O lobby ainda não foi aberto pelo mestre." |
| `throttled` | "Muitas tentativas de conexão. Recarregue a página para tentar novamente." |
| `kicked` | "Você foi removido do lobby pelo mestre." |
| `lobby_closed` | "O lobby foi encerrado pelo mestre." (auto-navigate to MatchPage after 2 s) |
| `error` | "Erro de conexão. Verifique sua internet." |

### Actions

**Master:**
- `"Iniciar Partida"` → `sendStartMatch()` (no confirmation — master sees who is online)
- `"Cancelar Lobby"` → `ConfirmDialog` → `sendCancelLobby()`

**Participant:**
- No action buttons
- Text: "Aguardando o mestre iniciar a partida..."

### Guards

- No token → `<Navigate to="/" replace />`
- Match loading / error → `LoadingContainer` / `ErrorContainer` (existing atoms)

---

## `GamePage` Placeholder

**File:** `src/pages/GamePage.tsx`

Minimal placeholder so `navigate('/game')` doesn't 404. Replaced entirely in the next task.

```tsx
export default function GamePage() {
  return <div>Partida em andamento — em breve.</div>;
}
```

---

## Routes (`App.tsx`)

```tsx
<Route path="/campaigns/:campaignId/matches/:matchId/lobby" element={<LobbyPage />} />
<Route path="/campaigns/:campaignId/matches/:matchId/game"  element={<GamePage />} />
```

---

## Environment

**`System_X_System_React/.env`** (add):
```
VITE_WS_URL=ws://localhost:8081
```

---

## File Map

| Action | File | What changes |
|---|---|---|
| new | `src/pages/LobbyPage.tsx` | Full lobby page |
| new | `src/pages/GamePage.tsx` | Empty placeholder |
| new | `src/hooks/useLobbyWs.ts` | WS lifecycle + reconnect |
| new | `src/features/match/LobbyConnectionSidebarItem.tsx` | Sidebar item with kick |
| edit | `src/App.tsx` | 2 new routes |
| edit | `src/components/atoms/PageHeader.tsx` | `showBack?` prop |
| edit | `src/components/templates/DetailPageTemplate.tsx` | `hideBack?` prop |
| edit | `src/components/molecules/CharacterSidebarItem.tsx` | `isOnline?` + badges |
| edit | `src/pages/MatchPage.tsx` | "Entrar no Lobby" button |
| edit | `internal/app/game/message.go` | `lobby_closed`, `lobby_not_open` |
| edit | `internal/app/game/hub.go` | `GetRoom` method |
| edit | `internal/app/game/handler.go` | Master/participant split |
| edit | `internal/app/game/room.go` | `CloseLobby` method |

---

## Out of Scope

- Chat during lobby
- Lobby discovery (listing open lobbies)
- Spectator mode
- Mid-game kick
- Real-time sync between REST API server and game server via PostgreSQL LISTEN/NOTIFY
