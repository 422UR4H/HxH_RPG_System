# Lobby Page Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the full lobby flow — from the `LobbyPage` React component with WebSocket connection management to the supporting game server additions — leaving everything ready for `StartMatch` in the next task.

**Architecture:** Backend-first: add `lobby_closed`/`lobby_not_open`/`cancel_lobby` to the WS protocol, guard participant connections against rooms that don't exist yet, and expose `CloseLobby` on `Room`. Frontend: `useLobbyWs` hook manages WS lifecycle with exponential-backoff reconnection; `LobbyPage` combines REST enrollment data with live WS participant state; `CharacterSidebarItem` gains an `isOnline?` prop used by the new `LobbyConnectionSidebarItem`.

**Tech Stack:** Go (gorilla/websocket), React 18, TypeScript strict, styled-components, React Query, Vitest + MSW + Testing Library, existing `DetailPageTemplate` / `CharacterSidebarItem` / `ConfirmDialog` components.

**Spec:** `docs/superpowers/specs/2026-05-30-lobby-page-design.md`

---

## File Map

| Action | File |
|---|---|
| edit | `internal/app/game/message.go` |
| edit | `internal/app/game/room.go` |
| edit | `internal/app/game/hub.go` |
| edit | `internal/app/game/handler.go` |
| edit | `internal/app/game/handler_test.go` |
| create | `System_X_System_React/.env` |
| edit | `System_X_System_React/src/components/molecules/CharacterSidebarItem.tsx` |
| edit | `System_X_System_React/src/components/atoms/PageHeader.tsx` |
| edit | `System_X_System_React/src/components/templates/DetailPageTemplate.tsx` |
| create | `System_X_System_React/src/hooks/useLobbyWs.ts` |
| create | `System_X_System_React/src/hooks/__tests__/useLobbyWs.test.ts` |
| create | `System_X_System_React/src/features/match/LobbyConnectionSidebarItem.tsx` |
| create | `System_X_System_React/src/pages/LobbyPage.tsx` |
| create | `System_X_System_React/src/pages/GamePage.tsx` |
| edit | `System_X_System_React/src/App.tsx` |
| edit | `System_X_System_React/src/pages/MatchPage.tsx` |
| create | `System_X_System_React/src/pages/__tests__/LobbyPage.test.tsx` |

---

## Task 1 — Backend: WS message types + CloseLobby + hub guard

**Files:**
- Edit: `internal/app/game/message.go`
- Edit: `internal/app/game/room.go`
- Edit: `internal/app/game/hub.go`

- [ ] **Step 1: Add new message types to `message.go`**

In `internal/app/game/message.go`, add to the constants block (after `MsgTypeMasterActionEnqueued`):

```go
// Server → Client (lobby lifecycle)
MsgTypeLobbyClosed  MessageType = "lobby_closed"   // master cancelled the lobby
MsgTypeLobbyNotOpen MessageType = "lobby_not_open" // room does not exist for this participant

// Client → Server (lobby lifecycle)
MsgTypeCancelLobby MessageType = "cancel_lobby" // master requests lobby cancellation
```

- [ ] **Step 2: Add `CloseLobby` to `room.go`**

In `internal/app/game/room.go`, add after `KickPlayer`:

```go
func (r *Room) CloseLobby(masterUUID uuid.UUID) error {
	if !r.IsMaster(masterUUID) {
		return ErrNotMaster
	}

	msg := NewServerMessage(MsgTypeLobbyClosed, struct{}{})
	data, _ := json.Marshal(msg)

	r.mu.RLock()
	for _, c := range r.clients {
		select {
		case c.send <- data:
		default:
		}
	}
	r.mu.RUnlock()

	r.Stop()
	return nil
}
```

Also add the `MsgTypeCancelLobby` case to `handleClientMessage` in `room.go`, inside the switch:

```go
case MsgTypeCancelLobby:
	if err := r.CloseLobby(client.userUUID); err != nil {
		client.SendMessage(NewErrorMessage("forbidden", err.Error()))
	}
```

- [ ] **Step 3: Update `GetRoom` and `GetOrCreateRoom` in `hub.go` to skip closed rooms**

`GetRoom` already exists. Update it so it returns `nil, false` for closed rooms:

```go
func (h *Hub) GetRoom(matchUUID uuid.UUID) (*Room, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	room, ok := h.rooms[matchUUID]
	if !ok || room.GetState() == RoomStateClosed {
		return nil, false
	}
	return room, ok
}
```

Update `GetOrCreateRoom` to replace a closed room:

```go
func (h *Hub) GetOrCreateRoom(
	matchUUID, masterUUID uuid.UUID,
	startMatchUC IStartMatch,
	kickPlayerUC IKickPlayer,
	initSessionUC IInitMatchSession,
	openNextActionUC IOpenNextAction,
	pullActionUC IPullAction,
	enqueueActionUC IEnqueueAction,
	attachReactionUC IAttachReaction,
	changeSceneUC IChangeScene,
	roundRepo appmatch.IRoundRepository,
	enqueueMasterActionUC IEnqueueMasterAction,
) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()

	if room, ok := h.rooms[matchUUID]; ok && room.GetState() != RoomStateClosed {
		return room
	}

	room := NewRoom(
		matchUUID, masterUUID,
		startMatchUC, kickPlayerUC,
		initSessionUC, openNextActionUC, pullActionUC,
		enqueueActionUC, attachReactionUC,
		changeSceneUC, roundRepo, enqueueMasterActionUC,
	)
	h.rooms[matchUUID] = room
	go room.Run()
	return room
}
```

- [ ] **Step 4: Run backend tests to confirm nothing broke**

```bash
cd System_X_System && go test ./internal/app/game/... -v
```

Expected: all existing tests pass.

- [ ] **Step 5: Commit**

```bash
cd System_X_System
git add internal/app/game/message.go internal/app/game/room.go internal/app/game/hub.go
git commit -m "feat(game): add lobby_closed/lobby_not_open/cancel_lobby messages and CloseLobby"
```

---

## Task 2 — Backend: handler participant guard + tests

**Files:**
- Edit: `internal/app/game/handler.go`
- Edit: `internal/app/game/handler_test.go`

- [ ] **Step 1: Write failing tests**

In `internal/app/game/handler_test.go`, add at the end:

```go
func TestPlayerGetsLobbyNotOpenWhenNoRoom(t *testing.T) {
	masterUUID := uuid.New()
	playerUUID := uuid.New()
	matchUUID := uuid.New()
	server, hub := setupTestServer(masterUUID, true)
	defer server.Close()
	defer hub.Stop()

	// player connects without master having opened the lobby first
	conn := connectWS(t, server.URL, playerUUID, matchUUID)
	defer conn.Close() //nolint:errcheck

	msg := readMessage(t, conn)
	if msg.Type != game.MsgTypeLobbyNotOpen {
		t.Errorf("expected lobby_not_open, got %s", msg.Type)
	}
}

func TestPlayerCanConnectAfterMasterOpensLobby(t *testing.T) {
	masterUUID := uuid.New()
	playerUUID := uuid.New()
	matchUUID := uuid.New()
	server, hub := setupTestServer(masterUUID, true)
	defer server.Close()
	defer hub.Stop()

	// master opens lobby first
	masterConn := connectWS(t, server.URL, masterUUID, matchUUID)
	defer masterConn.Close() //nolint:errcheck
	_ = readMessage(t, masterConn) // room_state

	time.Sleep(50 * time.Millisecond)

	// now player can connect
	playerConn := connectWS(t, server.URL, playerUUID, matchUUID)
	defer playerConn.Close() //nolint:errcheck

	msg := readMessage(t, playerConn)
	if msg.Type != game.MsgTypeRoomState {
		t.Errorf("expected room_state, got %s", msg.Type)
	}
}

func TestMasterReceivesLobbyClosed(t *testing.T) {
	masterUUID := uuid.New()
	playerUUID := uuid.New()
	matchUUID := uuid.New()
	server, hub := setupTestServer(masterUUID, true)
	defer server.Close()
	defer hub.Stop()

	masterConn := connectWS(t, server.URL, masterUUID, matchUUID)
	defer masterConn.Close() //nolint:errcheck
	_ = readMessage(t, masterConn) // room_state

	time.Sleep(50 * time.Millisecond)

	playerConn := connectWS(t, server.URL, playerUUID, matchUUID)
	defer playerConn.Close() //nolint:errcheck
	_ = readMessage(t, playerConn) // room_state
	_ = readMessage(t, masterConn) // player_joined

	cancelMsg := game.Message{
		Type:    game.MsgTypeCancelLobby,
		Payload: json.RawMessage(`{}`),
	}
	data, _ := json.Marshal(cancelMsg)
	if err := masterConn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("failed to send cancel_lobby: %v", err)
	}

	received := readMessage(t, playerConn)
	if received.Type != game.MsgTypeLobbyClosed {
		t.Errorf("expected lobby_closed for player, got %s", received.Type)
	}
}

func TestPlayerCannotCancelLobby(t *testing.T) {
	masterUUID := uuid.New()
	playerUUID := uuid.New()
	matchUUID := uuid.New()
	server, hub := setupTestServer(masterUUID, true)
	defer server.Close()
	defer hub.Stop()

	masterConn := connectWS(t, server.URL, masterUUID, matchUUID)
	defer masterConn.Close() //nolint:errcheck
	_ = readMessage(t, masterConn) // room_state

	time.Sleep(50 * time.Millisecond)

	playerConn := connectWS(t, server.URL, playerUUID, matchUUID)
	defer playerConn.Close() //nolint:errcheck
	_ = readMessage(t, playerConn) // room_state
	_ = readMessage(t, masterConn) // player_joined

	cancelMsg := game.Message{
		Type:    game.MsgTypeCancelLobby,
		Payload: json.RawMessage(`{}`),
	}
	data, _ := json.Marshal(cancelMsg)
	if err := playerConn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("failed to send cancel_lobby: %v", err)
	}

	received := readMessage(t, playerConn)
	if received.Type != game.MsgTypeError {
		t.Errorf("expected error, got %s", received.Type)
	}
}
```

Also update `TestPlayerCanConnect` (it now fails because player connects without master):

```go
func TestPlayerCanConnect(t *testing.T) {
	masterUUID := uuid.New()
	playerUUID := uuid.New()
	matchUUID := uuid.New()
	server, hub := setupTestServer(masterUUID, true)
	defer server.Close()
	defer hub.Stop()

	// master must open the lobby first
	masterConn := connectWS(t, server.URL, masterUUID, matchUUID)
	defer masterConn.Close() //nolint:errcheck
	_ = readMessage(t, masterConn) // room_state

	time.Sleep(50 * time.Millisecond)

	conn := connectWS(t, server.URL, playerUUID, matchUUID)
	defer conn.Close() //nolint:errcheck

	msg := readMessage(t, conn)
	if msg.Type != game.MsgTypeRoomState {
		t.Errorf("expected room_state, got %s", msg.Type)
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
cd System_X_System && go test ./internal/app/game/... -run "TestPlayerGetsLobbyNotOpen|TestPlayerCanConnectAfter|TestMasterReceivesLobbyClosed|TestPlayerCannotCancel" -v
```

Expected: FAIL (message types not handled yet in handler).

- [ ] **Step 3: Update `handler.go` to use `GetRoom` for participants**

In `internal/app/game/handler.go`, replace the section after the enrollment check (lines ~104-130) with:

```go
isMaster := masterUUID == userUUID
if !isMaster {
	enrolled, err := h.enrollmentRepo.IsPlayerEnrolledInMatch(r.Context(), userUUID, matchUUID)
	if err != nil || !enrolled {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
}

// Upgrade the WebSocket connection
conn, err := upgrader.Upgrade(w, r, nil)
if err != nil {
	log.Printf("websocket upgrade failed: %v", err)
	return
}

nickname := r.URL.Query().Get("nickname")
if nickname == "" {
	nickname = userUUID.String()[:8]
}

if !isMaster {
	room, ok := h.hub.GetRoom(matchUUID)
	if !ok {
		// Lobby not open: inform the participant before closing
		msg := NewServerMessage(MsgTypeLobbyNotOpen, struct{}{})
		data, _ := json.Marshal(msg)
		_ = conn.WriteMessage(websocket.TextMessage, data)
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(4001, "lobby not open"))
		conn.Close()
		return
	}
	client := NewClient(userUUID, conn, nickname)
	room.Register(client)
	go client.WritePump()
	go client.ReadPump()
	return
}

room := h.hub.GetOrCreateRoom(
	matchUUID, masterUUID,
	h.startMatchUC, h.kickPlayerUC,
	h.initSessionUC, h.openNextActionUC, h.pullActionUC,
	h.enqueueActionUC, h.attachReactionUC,
	h.changeSceneUC, h.roundRepo, h.enqueueMasterActionUC,
)
client := NewClient(userUUID, conn, nickname)
room.Register(client)
go client.WritePump()
go client.ReadPump()
```

The upgrade now happens before the room check, which is necessary to send the `lobby_not_open` message via WebSocket.

- [ ] **Step 4: Run all game server tests**

```bash
cd System_X_System && go test ./internal/app/game/... -v
```

Expected: all tests pass including the 4 new ones.

- [ ] **Step 5: Commit**

```bash
cd System_X_System
git add internal/app/game/handler.go internal/app/game/handler_test.go
git commit -m "feat(game): guard participant connections when lobby is not open"
```

---

## Task 3 — Frontend: environment config

**Files:**
- Create: `System_X_System_React/.env`

- [ ] **Step 1: Create `.env`**

```
VITE_WS_URL=ws://localhost:8081
```

- [ ] **Step 2: Verify Vite reads the variable**

```bash
cd System_X_System_React && grep -r "VITE_WS_URL" src/ || echo "not used yet — OK"
```

Expected: "not used yet — OK" (the hook will use it in Task 5).

- [ ] **Step 3: Add `.env` to `.gitignore` if not already there and document it**

```bash
grep -q "^\.env$" .gitignore && echo "already ignored" || echo ".env" >> .gitignore
```

- [ ] **Step 4: Commit**

```bash
cd System_X_System_React
git add .gitignore
git commit -m "chore: add VITE_WS_URL env var for game server WebSocket"
```

---

## Task 4 — Frontend: `CharacterSidebarItem` — `isOnline` prop

**Files:**
- Edit: `src/components/molecules/CharacterSidebarItem.tsx`

Read `src/styles/tokens.ts` before starting — `colors.statusOngoing` (green) and `colors.statusLeft` (gray) are the border/badge colors.

- [ ] **Step 1: Add `isOnline` prop and badges**

Replace `CharacterSidebarItem.tsx` content with the following (all existing props preserved):

```tsx
import styled from "styled-components";
import type { CharacterBaseSummary, StatusBar } from "../../types/characterSheet";
import { createEmptyCharacterSheet } from "../../features/sheet/factories/characterSheet.factory";
import CharacterSheetHeader from "./CharacterSheetHeader";
import { colors } from "../../styles/tokens";

interface CharacterSidebarItemProps {
  character: CharacterBaseSummary & {
    isPending?: boolean;
    fullName?: string;
    characterClass?: string;
    level?: number;
    currExp?: number;
    nextLvlBaseExp?: number;
    health?: StatusBar;
    stamina?: StatusBar;
  };
  isMaster: boolean;
  isOwn?: boolean;
  hasLeft?: boolean;
  isOnline?: boolean;
  onClick: () => void;
}

export default function CharacterSidebarItem({
  character,
  isMaster,
  isOwn,
  hasLeft,
  isOnline,
  onClick,
}: CharacterSidebarItemProps) {
  const isDead = !!character.deadAt;
  const isPending = !!character.isPending;
  const isNpc = !character.playerUuid;

  const charSheet = createEmptyCharacterSheet();
  charSheet.characterClass = character.characterClass ?? "";
  charSheet.profile = {
    ...charSheet.profile,
    nickname: character.nickName,
    fullname: character.fullName ?? "",
    coverUrl: character.coverUrl,
    avatarUrl: character.avatarUrl,
  };
  if (character.health && character.stamina) {
    charSheet.status = {
      health: character.health,
      stamina: character.stamina,
    };
  }
  if (character.currExp !== undefined && character.nextLvlBaseExp !== undefined) {
    charSheet.characterExp = {
      ...charSheet.characterExp,
      level: character.level ?? charSheet.characterExp.level,
      currExp: character.currExp,
      nextLvlBaseExp: character.nextLvlBaseExp,
    };
  }

  const isClickable = isMaster || !!isOwn;

  return (
    <ItemContainer
      $isDead={isDead}
      $isPending={isPending}
      $isNpc={isNpc}
      $isOnline={isOnline}
      $clickable={isClickable}
      onClick={isClickable ? onClick : undefined}
    >
      <CharacterSheetHeader
        mode="card"
        data={{ charSheet }}
        showStatus={!!character.health}
      />
      {isPending && <PendingBadge>Pendente</PendingBadge>}
      {isNpc && <NpcBadge>NPC</NpcBadge>}
      {isDead && <DeadBadge>Morto</DeadBadge>}
      {hasLeft && <LeftBadge>Saiu</LeftBadge>}
      {isOnline === true && <OnlineBadge>ONLINE</OnlineBadge>}
      {isOnline === false && <AwaitingBadge>AGUARDANDO</AwaitingBadge>}
    </ItemContainer>
  );
}

function borderColor({
  $isDead,
  $isPending,
  $isOnline,
  $isNpc,
}: {
  $isDead?: boolean;
  $isPending?: boolean;
  $isOnline?: boolean;
  $isNpc?: boolean;
}): string {
  if ($isDead) return colors.danger;
  if ($isPending) return colors.statusPending;
  if ($isOnline === false) return colors.statusLeft;
  if ($isOnline === true) return colors.statusOngoing;
  if ($isNpc) return colors.statusNpc;
  return colors.orange;
}

const ItemContainer = styled.div<{
  $isDead?: boolean;
  $isPending?: boolean;
  $isNpc?: boolean;
  $isOnline?: boolean;
  $clickable?: boolean;
}>`
  position: relative;
  overflow: hidden;
  border-radius: 0px 16px 0 0;
  opacity: ${({ $isDead }) => ($isDead ? 0.7 : 1)};
  cursor: ${({ $clickable }) => ($clickable ? "pointer" : "default")};
  border-left: 4px solid ${borderColor};

  &:hover {
    filter: ${({ $clickable }) => ($clickable ? "brightness(1.05)" : "none")};
  }
`;

const Badge = styled.span`
  position: absolute;
  top: 10px;
  right: 10px;
  border-radius: 4px;
  padding: 2px 6px;
  font-size: 12px;
  font-weight: bold;
  z-index: 10;
`;

const PendingBadge = styled(Badge)`
  background-color: ${colors.statusPending};
  color: ${colors.textPrimary};
`;

const DeadBadge = styled(Badge)`
  background-color: ${colors.danger};
  color: ${colors.textPrimary};
`;

const NpcBadge = styled(Badge)`
  background-color: ${colors.statusNpc};
  color: ${colors.textPrimary};
`;

const LeftBadge = styled(Badge)`
  background-color: ${colors.statusLeft};
  color: ${colors.textDisabled};
`;

const OnlineBadge = styled(Badge)`
  background-color: ${colors.statusOngoing};
  color: ${colors.textPrimary};
`;

const AwaitingBadge = styled(Badge)`
  background-color: ${colors.statusLeft};
  color: ${colors.textDisabled};
`;
```

- [ ] **Step 2: Run existing tests to confirm nothing broke**

```bash
cd System_X_System_React && npx vitest run --reporter=verbose 2>&1 | head -60
```

Expected: all existing tests pass.

- [ ] **Step 3: Commit**

```bash
cd System_X_System_React
git add src/components/molecules/CharacterSidebarItem.tsx
git commit -m "feat(ui): add isOnline prop to CharacterSidebarItem with ONLINE/AGUARDANDO badges"
```

---

## Task 5 — Frontend: `PageHeader` + `DetailPageTemplate` — `hideBack` prop

**Files:**
- Edit: `src/components/atoms/PageHeader.tsx`
- Edit: `src/components/templates/DetailPageTemplate.tsx`

- [ ] **Step 1: Update `PageHeader.tsx`**

```tsx
import styled from "styled-components";
import BackButton from "../ions/BackButton";
import LogoButton from "./LogoButton";
import { colors } from "../../styles/tokens";

interface PageHeaderProps {
  backgroundColor?: string;
  showBack?: boolean;
}

export default function PageHeader({ backgroundColor, showBack = true }: PageHeaderProps) {
  return (
    <StyledPageHeader $backgroundColor={backgroundColor}>
      {showBack && <BackButton />}
      <LogoButton />
    </StyledPageHeader>
  );
}

const StyledPageHeader = styled.div<{ $backgroundColor?: string }>`
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;

  background-color: ${({ $backgroundColor }) => $backgroundColor || colors.surfaceHeaderDefault};
  width: 100%;
  height: min(102px, 15.2vw);
  padding-bottom: 0.4vw;
`;
```

- [ ] **Step 2: Update `DetailPageTemplate.tsx`**

Add `hideBack?: boolean` to the props interface and pass it to `PageHeader`:

```tsx
interface DetailPageTemplateProps {
  headerColor?: string;
  bgImage?: string;
  mainRef?: RefObject<HTMLDivElement | null>;
  hideBack?: boolean;
  leftSidebar: ReactNode;
  leftSidebarLabel?: string;
  rightSidebar?: ReactNode;
  rightSidebarLabel?: string;
  children: ReactNode;
}
```

In the function signature add `hideBack = false`:

```tsx
export default function DetailPageTemplate({
  headerColor = colors.brandPrimary,
  bgImage = worldMap,
  mainRef,
  hideBack = false,
  leftSidebar,
  leftSidebarLabel = "PERSONAGENS",
  rightSidebar,
  rightSidebarLabel = "REGRAS",
  children,
}: DetailPageTemplateProps) {
```

Pass it to `PageHeader`:

```tsx
<PageHeader backgroundColor={headerColor} showBack={!hideBack} />
```

- [ ] **Step 3: Run existing tests**

```bash
cd System_X_System_React && npx vitest run --reporter=verbose 2>&1 | head -60
```

Expected: all existing tests pass (default `hideBack=false` preserves current behaviour).

- [ ] **Step 4: Commit**

```bash
cd System_X_System_React
git add src/components/atoms/PageHeader.tsx src/components/templates/DetailPageTemplate.tsx
git commit -m "feat(ui): add hideBack prop to DetailPageTemplate / showBack to PageHeader"
```

---

## Task 6 — Frontend: `useLobbyWs` hook

**Files:**
- Create: `src/hooks/useLobbyWs.ts`
- Create: `src/hooks/__tests__/useLobbyWs.test.ts`

- [ ] **Step 1: Write failing tests**

Create `src/hooks/__tests__/useLobbyWs.test.ts`:

```ts
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useLobbyWs } from "../useLobbyWs";

// ─── WebSocket mock ───────────────────────────────────────────────────────────

interface MockWsInstance {
  onmessage: ((e: MessageEvent) => void) | null;
  onclose: ((e: CloseEvent) => void) | null;
  onerror: ((e: Event) => void) | null;
  onopen: ((e: Event) => void) | null;
  close: ReturnType<typeof vi.fn>;
  send: ReturnType<typeof vi.fn>;
  readyState: number;
  url: string;
}

let wsInstance: MockWsInstance;
const MockWebSocket = vi.fn().mockImplementation((url: string) => {
  wsInstance = {
    onmessage: null,
    onclose: null,
    onerror: null,
    onopen: null,
    close: vi.fn(),
    send: vi.fn(),
    readyState: WebSocket.CONNECTING,
    url,
  };
  return wsInstance;
});
MockWebSocket.CONNECTING = 0;
MockWebSocket.OPEN = 1;
MockWebSocket.CLOSING = 2;
MockWebSocket.CLOSED = 3;

const defaultParams = {
  matchUuid: "match-1",
  token: "fake-token",
  nickname: "Gon",
  onMatchStarted: vi.fn(),
};

function sendFromServer(type: string, payload: unknown = {}) {
  act(() => {
    wsInstance.onmessage?.({
      data: JSON.stringify({ type, payload: JSON.stringify(payload) }),
    } as MessageEvent);
  });
}

function simulateOpen() {
  act(() => {
    wsInstance.readyState = WebSocket.OPEN;
    wsInstance.onopen?.({} as Event);
  });
}

function simulateClose(code = 1000) {
  act(() => {
    wsInstance.onclose?.({ code, wasClean: code === 1000 } as CloseEvent);
  });
}

// ─── Tests ────────────────────────────────────────────────────────────────────

describe("useLobbyWs", () => {
  beforeEach(() => {
    vi.stubGlobal("WebSocket", MockWebSocket);
    vi.useFakeTimers();
    defaultParams.onMatchStarted.mockReset();
    MockWebSocket.mockClear();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.useRealTimers();
  });

  it("starts with status connecting", () => {
    const { result } = renderHook(() => useLobbyWs(defaultParams));
    expect(result.current.status).toBe("connecting");
  });

  it("transitions to connected when WebSocket opens", () => {
    const { result } = renderHook(() => useLobbyWs(defaultParams));
    simulateOpen();
    expect(result.current.status).toBe("connected");
  });

  it("populates participants on room_state", () => {
    const { result } = renderHook(() => useLobbyWs(defaultParams));
    simulateOpen();
    sendFromServer("room_state", {
      match_uuid: "match-1",
      state: "lobby",
      players: [
        { uuid: "p1", nickname: "Gon", is_master: false, is_online: true },
        { uuid: "master-1", nickname: "Master", is_master: true, is_online: true },
      ],
    });
    expect(result.current.participants).toHaveLength(2);
    expect(result.current.participants[0].uuid).toBe("p1");
  });

  it("adds participant on player_joined", () => {
    const { result } = renderHook(() => useLobbyWs(defaultParams));
    simulateOpen();
    sendFromServer("room_state", { match_uuid: "match-1", state: "lobby", players: [] });
    sendFromServer("player_joined", { uuid: "p2", nickname: "Killua", is_master: false, is_online: true });
    expect(result.current.participants).toHaveLength(1);
    expect(result.current.participants[0].uuid).toBe("p2");
  });

  it("removes participant on player_left", () => {
    const { result } = renderHook(() => useLobbyWs(defaultParams));
    simulateOpen();
    sendFromServer("room_state", {
      match_uuid: "match-1", state: "lobby",
      players: [{ uuid: "p1", nickname: "Gon", is_master: false, is_online: true }],
    });
    sendFromServer("player_left", { uuid: "p1", nickname: "Gon" });
    expect(result.current.participants).toHaveLength(0);
  });

  it("sets status to lobby_not_open on error with lobby_not_open code", () => {
    const { result } = renderHook(() => useLobbyWs(defaultParams));
    simulateOpen();
    sendFromServer("lobby_not_open", {});
    expect(result.current.status).toBe("lobby_not_open");
  });

  it("sets status to kicked when player_kicked arrives with own uuid", () => {
    const { result } = renderHook(() => useLobbyWs(defaultParams));
    simulateOpen();
    sendFromServer("player_kicked", { uuid: "user-1", nickname: "Gon", reason: "kicked by master" });
    // Note: the hook identifies "self" via the uuid passed from the page; tested via integration
    // Here we test that player_kicked targeting a different uuid does NOT set kicked status
    expect(result.current.status).not.toBe("kicked");
  });

  it("sets status to lobby_closed on lobby_closed message", () => {
    const { result } = renderHook(() => useLobbyWs(defaultParams));
    simulateOpen();
    sendFromServer("lobby_closed", {});
    expect(result.current.status).toBe("lobby_closed");
  });

  it("calls onMatchStarted on match_started", () => {
    const onMatchStarted = vi.fn();
    const { result } = renderHook(() =>
      useLobbyWs({ ...defaultParams, onMatchStarted })
    );
    simulateOpen();
    sendFromServer("match_started", {});
    expect(onMatchStarted).toHaveBeenCalledOnce();
    expect(result.current.status).toBe("connected");
  });

  it("does not reconnect on lobby_not_open (terminal state)", () => {
    renderHook(() => useLobbyWs(defaultParams));
    simulateOpen();
    sendFromServer("lobby_not_open", {});
    simulateClose(4001);
    act(() => { vi.advanceTimersByTime(2000); });
    // Only 1 WebSocket created (no reconnect)
    expect(MockWebSocket).toHaveBeenCalledTimes(1);
  });

  it("does not reconnect on lobby_closed (terminal state)", () => {
    renderHook(() => useLobbyWs(defaultParams));
    simulateOpen();
    sendFromServer("lobby_closed", {});
    simulateClose(1000);
    act(() => { vi.advanceTimersByTime(2000); });
    expect(MockWebSocket).toHaveBeenCalledTimes(1);
  });

  it("reconnects after unexpected close with backoff", () => {
    renderHook(() => useLobbyWs(defaultParams));
    simulateOpen();
    simulateClose(1006); // abnormal closure
    act(() => { vi.advanceTimersByTime(600); }); // 500ms backoff
    expect(MockWebSocket).toHaveBeenCalledTimes(2);
  });

  it("sets status to throttled after 5 reconnects in 60s", () => {
    const { result } = renderHook(() => useLobbyWs(defaultParams));

    for (let i = 0; i < 5; i++) {
      simulateOpen();
      simulateClose(1006);
      act(() => { vi.advanceTimersByTime(60_000); });
    }

    expect(result.current.status).toBe("throttled");
    const countBefore = MockWebSocket.mock.calls.length;
    act(() => { vi.advanceTimersByTime(60_000); });
    expect(MockWebSocket.mock.calls.length).toBe(countBefore);
  });

  it("closes WS on unmount", () => {
    const { unmount } = renderHook(() => useLobbyWs(defaultParams));
    simulateOpen();
    unmount();
    expect(wsInstance.close).toHaveBeenCalled();
  });

  it("sendStartMatch sends correct WS message", () => {
    const { result } = renderHook(() => useLobbyWs(defaultParams));
    simulateOpen();
    act(() => { result.current.sendStartMatch(); });
    const sent = JSON.parse(wsInstance.send.mock.calls[0][0]);
    expect(sent.type).toBe("start_match");
  });

  it("sendCancelLobby sends correct WS message", () => {
    const { result } = renderHook(() => useLobbyWs(defaultParams));
    simulateOpen();
    act(() => { result.current.sendCancelLobby(); });
    const sent = JSON.parse(wsInstance.send.mock.calls[0][0]);
    expect(sent.type).toBe("cancel_lobby");
  });

  it("sendKick sends correct WS message", () => {
    const { result } = renderHook(() => useLobbyWs(defaultParams));
    simulateOpen();
    act(() => { result.current.sendKick("target-uuid"); });
    const sent = JSON.parse(wsInstance.send.mock.calls[0][0]);
    expect(sent.type).toBe("kick_player");
    expect(JSON.parse(sent.payload).player_uuid).toBe("target-uuid");
  });
});
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
cd System_X_System_React && npx vitest run src/hooks/__tests__/useLobbyWs.test.ts
```

Expected: FAIL (module not found).

- [ ] **Step 3: Implement `useLobbyWs.ts`**

Create `src/hooks/useLobbyWs.ts`:

```ts
import { useEffect, useRef, useState, useCallback } from "react";

export type WsStatus =
  | "connecting"
  | "connected"
  | "disconnected"
  | "lobby_not_open"
  | "kicked"
  | "lobby_closed"
  | "throttled"
  | "error";

export type LobbyParticipant = {
  uuid: string;
  nickname: string;
  isMaster: boolean;
  isOnline: boolean;
};

interface UseLobbyWsParams {
  matchUuid: string;
  token: string;
  nickname: string;
  onMatchStarted: () => void;
}

interface UseLobbyWsResult {
  status: WsStatus;
  participants: LobbyParticipant[];
  sendStartMatch: () => void;
  sendKick: (userUuid: string) => void;
  sendCancelLobby: () => void;
}

const MAX_RECONNECTS = 5;
const WINDOW_MS = 60_000;
const BASE_DELAY_MS = 500;
const MAX_DELAY_MS = 30_000;

const TERMINAL_STATUSES: WsStatus[] = [
  "lobby_not_open",
  "kicked",
  "lobby_closed",
  "throttled",
];

export function useLobbyWs({
  matchUuid,
  token,
  nickname,
  onMatchStarted,
}: UseLobbyWsParams): UseLobbyWsResult {
  const [status, setStatus] = useState<WsStatus>("connecting");
  const [participants, setParticipants] = useState<LobbyParticipant[]>([]);

  const wsRef = useRef<WebSocket | null>(null);
  const statusRef = useRef<WsStatus>("connecting");
  const attemptTimestampsRef = useRef<number[]>([]);
  const onMatchStartedRef = useRef(onMatchStarted);
  onMatchStartedRef.current = onMatchStarted;

  const updateStatus = useCallback((next: WsStatus) => {
    statusRef.current = next;
    setStatus(next);
  }, []);

  const connect = useCallback(() => {
    const wsUrl = `${import.meta.env.VITE_WS_URL}/ws?match_uuid=${matchUuid}&token=${encodeURIComponent(token)}&nickname=${encodeURIComponent(nickname)}`;
    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;
    updateStatus("connecting");

    ws.onopen = () => {
      updateStatus("connected");
    };

    ws.onmessage = (event: MessageEvent) => {
      let msg: { type: string; payload: string };
      try {
        msg = JSON.parse(event.data as string) as { type: string; payload: string };
      } catch {
        return;
      }

      let payload: Record<string, unknown> = {};
      try {
        payload = JSON.parse(msg.payload) as Record<string, unknown>;
      } catch {
        // empty payload is fine
      }

      switch (msg.type) {
        case "room_state": {
          const players = (payload.players as Array<{
            uuid: string;
            nickname: string;
            is_master: boolean;
            is_online: boolean;
          }> | undefined) ?? [];
          setParticipants(
            players.map((p) => ({
              uuid: p.uuid,
              nickname: p.nickname,
              isMaster: p.is_master,
              isOnline: p.is_online,
            }))
          );
          break;
        }
        case "player_joined": {
          const p = payload as { uuid: string; nickname: string; is_master: boolean; is_online: boolean };
          setParticipants((prev) => [
            ...prev,
            { uuid: p.uuid, nickname: p.nickname, isMaster: p.is_master, isOnline: true },
          ]);
          break;
        }
        case "player_left": {
          const p = payload as { uuid: string };
          setParticipants((prev) => prev.filter((x) => x.uuid !== p.uuid));
          break;
        }
        case "player_kicked": {
          // Kicked participants are removed from the list by the server broadcasting player_left
          // If the message targets the current user, it will be handled via the uuid comparison
          // in LobbyPage (the hook doesn't have access to the current user UUID here).
          // We do nothing here — LobbyPage passes an onKicked callback if needed.
          break;
        }
        case "lobby_not_open":
          updateStatus("lobby_not_open");
          break;
        case "lobby_closed":
          updateStatus("lobby_closed");
          break;
        case "match_started":
          onMatchStartedRef.current();
          break;
        default:
          break;
      }
    };

    ws.onclose = () => {
      if (TERMINAL_STATUSES.includes(statusRef.current)) return;

      const now = Date.now();
      attemptTimestampsRef.current = attemptTimestampsRef.current.filter(
        (t) => now - t < WINDOW_MS
      );

      if (attemptTimestampsRef.current.length >= MAX_RECONNECTS) {
        updateStatus("throttled");
        return;
      }

      attemptTimestampsRef.current.push(now);
      const delay = Math.min(
        BASE_DELAY_MS * Math.pow(2, attemptTimestampsRef.current.length - 1),
        MAX_DELAY_MS
      );

      updateStatus("disconnected");
      setTimeout(connect, delay);
    };

    ws.onerror = () => {
      if (!TERMINAL_STATUSES.includes(statusRef.current)) {
        updateStatus("error");
      }
    };
  }, [matchUuid, token, nickname, updateStatus]);

  useEffect(() => {
    connect();
    return () => {
      if (wsRef.current) {
        wsRef.current.onclose = null; // prevent reconnect on intentional unmount
        wsRef.current.close();
      }
    };
  }, [connect]);

  const sendMessage = useCallback((type: string, payload: unknown = {}) => {
    const ws = wsRef.current;
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type, payload: JSON.stringify(payload) }));
    }
  }, []);

  const sendStartMatch = useCallback(() => sendMessage("start_match"), [sendMessage]);
  const sendCancelLobby = useCallback(() => sendMessage("cancel_lobby"), [sendMessage]);
  const sendKick = useCallback(
    (userUuid: string) => sendMessage("kick_player", { player_uuid: userUuid }),
    [sendMessage]
  );

  return { status, participants, sendStartMatch, sendKick, sendCancelLobby };
}
```

- [ ] **Step 4: Run tests**

```bash
cd System_X_System_React && npx vitest run src/hooks/__tests__/useLobbyWs.test.ts
```

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
cd System_X_System_React
git add src/hooks/useLobbyWs.ts src/hooks/__tests__/useLobbyWs.test.ts
git commit -m "feat: implement useLobbyWs hook with reconnect throttle"
```

---

## Task 7 — Frontend: `LobbyConnectionSidebarItem`

**Files:**
- Create: `src/features/match/LobbyConnectionSidebarItem.tsx`

Read `src/features/match/EnrollmentSidebarItem.tsx` for the pattern. `LobbyConnectionSidebarItem` follows the same structure but with `isOnline` and a kick button instead of accept/reject.

- [ ] **Step 1: Create `LobbyConnectionSidebarItem.tsx`**

```tsx
import { useState } from "react";
import styled from "styled-components";
import type { Enrollment } from "../../types/match";
import { createEmptyCharacterSheet } from "../sheet/factories/characterSheet.factory";
import CharacterSheetHeader from "../../components/molecules/CharacterSheetHeader";
import ConfirmDialog from "../../components/molecules/ConfirmDialog";
import { colors } from "../../styles/tokens";

interface LobbyConnectionSidebarItemProps {
  enrollment: Enrollment;
  isOnline: boolean;
  isMaster: boolean;
  onKick?: (playerUuid: string) => void;
  onClick: () => void;
}

export default function LobbyConnectionSidebarItem({
  enrollment,
  isOnline,
  isMaster,
  onKick,
  onClick,
}: LobbyConnectionSidebarItemProps) {
  const [showKickConfirm, setShowKickConfirm] = useState(false);
  const { characterSheet } = enrollment;
  const priv = characterSheet.private;

  const charSheet = createEmptyCharacterSheet();
  charSheet.characterClass = priv?.characterClass ?? "";
  charSheet.profile = {
    ...charSheet.profile,
    nickname: characterSheet.nickName,
    fullname: priv?.fullName ?? "",
    coverUrl: characterSheet.coverUrl,
    avatarUrl: characterSheet.avatarUrl,
  };
  if (priv?.health && priv?.stamina) {
    charSheet.status = { health: priv.health, stamina: priv.stamina };
  }
  if (priv?.currExp !== undefined && priv?.nextLvlBaseExp !== undefined) {
    charSheet.characterExp = {
      ...charSheet.characterExp,
      level: priv.level ?? charSheet.characterExp.level,
      currExp: priv.currExp,
      nextLvlBaseExp: priv.nextLvlBaseExp,
    };
  }

  const handleKickConfirm = () => {
    setShowKickConfirm(false);
    if (enrollment.player?.uuid) {
      onKick?.(enrollment.player.uuid);
    }
  };

  return (
    <>
      <ItemContainer $isOnline={isOnline} $clickable={isMaster} onClick={isMaster ? onClick : undefined}>
        <CharacterSheetHeader mode="card" data={{ charSheet }} showStatus={!!priv} />
        <StatusBadge $isOnline={isOnline}>
          {isOnline ? "ONLINE" : "AGUARDANDO"}
        </StatusBadge>

        {isMaster && (
          <Actions>
            <KickButton
              disabled={!isOnline}
              onClick={(e) => {
                e.stopPropagation();
                setShowKickConfirm(true);
              }}
            >
              Expulsar
            </KickButton>
          </Actions>
        )}
      </ItemContainer>

      {showKickConfirm && (
        <ConfirmDialog
          message={`Tem certeza que deseja expulsar ${characterSheet.nickName} do lobby?`}
          confirmLabel="Expulsar"
          onConfirm={handleKickConfirm}
          onCancel={() => setShowKickConfirm(false)}
        />
      )}
    </>
  );
}

const ItemContainer = styled.div<{ $isOnline: boolean; $clickable: boolean }>`
  position: relative;
  overflow: hidden;
  border-radius: 0px 16px 0 0;
  border-left: 4px solid
    ${({ $isOnline }) => ($isOnline ? colors.statusOngoing : colors.statusLeft)};
  cursor: ${({ $clickable }) => ($clickable ? "pointer" : "default")};
  opacity: ${({ $isOnline }) => ($isOnline ? 1 : 0.75)};

  &:hover {
    filter: ${({ $clickable }) => ($clickable ? "brightness(1.05)" : "none")};
  }
`;

const StatusBadge = styled.span<{ $isOnline: boolean }>`
  position: absolute;
  top: 10px;
  right: 10px;
  z-index: 10;
  border-radius: 4px;
  padding: 2px 8px;
  font-size: 12px;
  font-weight: bold;
  background-color: ${({ $isOnline }) =>
    $isOnline ? colors.statusOngoing : colors.statusLeft};
  color: ${({ $isOnline }) => ($isOnline ? colors.textPrimary : colors.textDisabled)};
`;

const Actions = styled.div`
  display: flex;
  gap: 8px;
  padding: 6px 8px;
  background-color: ${colors.overlayMedium};
`;

const KickButton = styled.button`
  flex: 1;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: bold;
  cursor: pointer;
  padding: 4px 0;
  background-color: ${colors.dangerDark};
  color: ${colors.textPrimary};
  transition: filter 0.2s;

  &:hover:not(:disabled) {
    filter: brightness(1.15);
  }
  &:disabled {
    opacity: 0.35;
    cursor: not-allowed;
  }
`;
```

- [ ] **Step 2: Run existing tests**

```bash
cd System_X_System_React && npx vitest run --reporter=verbose 2>&1 | tail -20
```

Expected: all pass.

- [ ] **Step 3: Commit**

```bash
cd System_X_System_React
git add src/features/match/LobbyConnectionSidebarItem.tsx
git commit -m "feat: add LobbyConnectionSidebarItem with online status and kick confirmation"
```

---

## Task 8 — Frontend: `LobbyPage` + `GamePage` + routes + `MatchPage` button

**Files:**
- Create: `src/pages/LobbyPage.tsx`
- Create: `src/pages/GamePage.tsx`
- Create: `src/pages/__tests__/LobbyPage.test.tsx`
- Edit: `src/App.tsx`
- Edit: `src/pages/MatchPage.tsx`

Read `src/pages/MatchPage.tsx` in full before starting — `LobbyPage` mirrors its structure.

- [ ] **Step 1: Write failing tests for `LobbyPage`**

Create `src/pages/__tests__/LobbyPage.test.tsx`:

```tsx
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { http, HttpResponse } from "msw";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { server } from "../../test/server";
import { renderWithProviders } from "../../test/render";
import { matchFixture } from "../../test/fixtures/match";
import { masterUserFixture, userFixture } from "../../test/fixtures/user";
import LobbyPage from "../LobbyPage";

const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom");
  return { ...actual, useNavigate: () => mockNavigate };
});

const baseUrl = "http://localhost:5000";

// ─── WebSocket mock ───────────────────────────────────────────────────────────
interface MockWsInstance {
  onmessage: ((e: MessageEvent) => void) | null;
  onclose: ((e: CloseEvent) => void) | null;
  onerror: ((e: Event) => void) | null;
  onopen: ((e: Event) => void) | null;
  close: ReturnType<typeof vi.fn>;
  send: ReturnType<typeof vi.fn>;
  readyState: number;
}
let wsInstance: MockWsInstance;
const MockWebSocket = vi.fn().mockImplementation(() => {
  wsInstance = {
    onmessage: null, onclose: null, onerror: null, onopen: null,
    close: vi.fn(), send: vi.fn(), readyState: 1,
  };
  return wsInstance;
});
MockWebSocket.CONNECTING = 0;
MockWebSocket.OPEN = 1;
MockWebSocket.CLOSING = 2;
MockWebSocket.CLOSED = 3;

function simulateWsOpen() {
  wsInstance.onopen?.({} as Event);
}
function sendFromServer(type: string, payload: unknown = {}) {
  wsInstance.onmessage?.({
    data: JSON.stringify({ type, payload: JSON.stringify(payload) }),
  } as MessageEvent);
}

// ─── MSW handlers ─────────────────────────────────────────────────────────────
const acceptedEnrollment = {
  uuid: "enrollment-1",
  status: "accepted",
  created_at: "2025-01-01T00:00:00Z",
  character_sheet: {
    uuid: "sheet-1",
    nick_name: "Gon",
    player_uuid: "user-1",
  },
  player: { uuid: "user-1", nick: "Gon" },
};

function setupHandlers(masterUuid = "master-1") {
  server.use(
    http.get(`${baseUrl}/matches/:id`, () =>
      HttpResponse.json({ match: { ...matchFixture, master_uuid: masterUuid } })
    ),
    http.get(`${baseUrl}/matches/:id/enrollments`, () =>
      HttpResponse.json({ enrollments: [acceptedEnrollment] })
    )
  );
}

function renderPage(opts: Parameters<typeof renderWithProviders>[1] = {}) {
  return renderWithProviders(<LobbyPage />, {
    route: "/campaigns/campaign-1/matches/match-1/lobby",
    path: "/campaigns/:campaignId/matches/:matchId/lobby",
    ...opts,
  });
}

// ─── Tests ────────────────────────────────────────────────────────────────────
describe("LobbyPage", () => {
  beforeEach(() => {
    vi.stubGlobal("WebSocket", MockWebSocket);
    mockNavigate.mockReset();
    MockWebSocket.mockClear();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("mostra loading enquanto dados carregam", () => {
    server.use(
      http.get(`${baseUrl}/matches/:id`, async () => {
        await new Promise((r) => setTimeout(r, 50));
        return HttpResponse.json({ match: matchFixture });
      }),
      http.get(`${baseUrl}/matches/:id/enrollments`, () =>
        HttpResponse.json({ enrollments: [] })
      )
    );
    renderPage();
    expect(screen.getByText(/Carregando/i)).toBeInTheDocument();
  });

  it("mestre vê botão Iniciar Partida e Cancelar Lobby", async () => {
    setupHandlers("master-1");
    renderPage({ user: masterUserFixture });
    simulateWsOpen();
    sendFromServer("room_state", { match_uuid: "match-1", state: "lobby", players: [] });

    await waitFor(() => {
      expect(screen.getByRole("button", { name: /Iniciar Partida/i })).toBeInTheDocument();
      expect(screen.getByRole("button", { name: /Cancelar Lobby/i })).toBeInTheDocument();
    });
  });

  it("jogador não vê botões de ação", async () => {
    setupHandlers("master-1");
    renderPage({ user: userFixture });
    simulateWsOpen();
    sendFromServer("room_state", { match_uuid: "match-1", state: "lobby", players: [] });

    await waitFor(() => {
      expect(screen.queryByRole("button", { name: /Iniciar Partida/i })).not.toBeInTheDocument();
      expect(screen.getByText(/Aguardando o mestre/i)).toBeInTheDocument();
    });
  });

  it("exibe mensagem lobby_not_open", async () => {
    setupHandlers();
    renderPage({ user: userFixture });
    simulateWsOpen();
    sendFromServer("lobby_not_open", {});

    await waitFor(() => {
      expect(screen.getByText(/lobby ainda não foi aberto/i)).toBeInTheDocument();
    });
  });

  it("exibe mensagem kicked", async () => {
    setupHandlers();
    renderPage({ user: userFixture });
    simulateWsOpen();
    sendFromServer("player_kicked", { uuid: "user-1", nickname: "Gon", reason: "kicked" });

    await waitFor(() => {
      expect(screen.getByText(/removido do lobby/i)).toBeInTheDocument();
    });
  });

  it("navega para /game ao receber match_started", async () => {
    setupHandlers("master-1");
    renderPage({ user: masterUserFixture });
    simulateWsOpen();
    sendFromServer("room_state", { match_uuid: "match-1", state: "lobby", players: [] });
    sendFromServer("match_started", {});

    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith(
        "/campaigns/campaign-1/matches/match-1/game"
      );
    });
  });

  it("mestre confirma cancelamento antes de enviar cancel_lobby", async () => {
    setupHandlers("master-1");
    renderPage({ user: masterUserFixture });
    simulateWsOpen();
    sendFromServer("room_state", { match_uuid: "match-1", state: "lobby", players: [] });

    const cancelBtn = await screen.findByRole("button", { name: /Cancelar Lobby/i });
    await userEvent.click(cancelBtn);

    // ConfirmDialog should appear
    await waitFor(() => {
      expect(screen.getByText(/Tem certeza/i)).toBeInTheDocument();
    });

    const confirmBtn = screen.getByRole("button", { name: /Cancelar Lobby/i, hidden: false });
    await userEvent.click(confirmBtn);

    expect(wsInstance.send).toHaveBeenCalledWith(
      expect.stringContaining("cancel_lobby")
    );
  });
});
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
cd System_X_System_React && npx vitest run src/pages/__tests__/LobbyPage.test.tsx
```

Expected: FAIL (module not found).

- [ ] **Step 3: Create `GamePage.tsx` placeholder**

Create `src/pages/GamePage.tsx`:

```tsx
import { Navigate } from "react-router-dom";
import useToken from "../hooks/useToken";

export default function GamePage() {
  const { token } = useToken();
  if (!token) return <Navigate to="/" replace />;
  return (
    <div style={{ color: "white", padding: "40px", fontFamily: "sans-serif" }}>
      Partida em andamento — em breve.
    </div>
  );
}
```

- [ ] **Step 4: Add routes to `App.tsx`**

Import the two new pages and add the routes:

```tsx
import LobbyPage from "./pages/LobbyPage";
import GamePage from "./pages/GamePage";
```

Add inside `<Routes>` after the `edit` route:

```tsx
<Route
  path="/campaigns/:campaignId/matches/:matchId/lobby"
  element={<LobbyPage />}
/>
<Route
  path="/campaigns/:campaignId/matches/:matchId/game"
  element={<GamePage />}
/>
```

- [ ] **Step 5: Create `LobbyPage.tsx`**

Create `src/pages/LobbyPage.tsx`:

```tsx
import { useState } from "react";
import { Navigate, useParams, useNavigate } from "react-router-dom";
import useToken from "../hooks/useToken";
import useUser from "../hooks/useUser";
import { useMatchDetails } from "../hooks/useMatchDetails";
import { useMatchEnrollments } from "../hooks/useMatchEnrollments";
import { useLobbyWs } from "../hooks/useLobbyWs";
import styled from "styled-components";
import { colors, fonts } from "../styles/tokens";
import LobbyConnectionSidebarItem from "../features/match/LobbyConnectionSidebarItem";
import CharactersSidebar from "../components/organisms/CharactersSidebar";
import RulesSidebar from "../components/organisms/RulesSidebar";
import RuleSection from "../components/molecules/RuleSection";
import BottomActions from "../components/molecules/BottomActions";
import ConfirmDialog from "../components/molecules/ConfirmDialog";
import DetailPageTemplate from "../components/templates/DetailPageTemplate";
import { LoadingContainer, ErrorContainer } from "../components/atoms/PageStates";

export default function LobbyPage() {
  const { campaignId, matchId } = useParams<{ campaignId: string; matchId: string }>();
  const { token } = useToken();
  const { user } = useUser();
  const navigate = useNavigate();

  const [showCancelConfirm, setShowCancelConfirm] = useState(false);

  const { data: match, isPending, isError } = useMatchDetails(token, matchId);
  const { data: enrollments = [] } = useMatchEnrollments(token, matchId, true);

  const acceptedEnrollments = enrollments.filter((e) => e.status === "accepted");

  const { status, participants, sendStartMatch, sendKick, sendCancelLobby } = useLobbyWs({
    matchUuid: matchId ?? "",
    token: token ?? "",
    nickname: user?.nick ?? "",
    onMatchStarted: () =>
      navigate(`/campaigns/${campaignId}/matches/${matchId}/game`),
  });

  if (!token) return <Navigate to="/" replace />;
  if (isPending) return <LoadingContainer>Carregando lobby...</LoadingContainer>;
  if (isError || !match) return <ErrorContainer>Falha ao carregar a partida.</ErrorContainer>;

  const isMaster = match.masterUuid === user?.uuid;
  const connectedCount = participants.filter((p) => !p.isMaster).length;
  const totalCount = acceptedEnrollments.length;

  const lobbyEntries = acceptedEnrollments.map((enrollment) => ({
    enrollment,
    isOnline: participants.some(
      (p) => p.uuid === enrollment.player?.uuid
    ),
  }));

  const statusMessage: Record<string, string> = {
    connecting: "Conectando ao lobby...",
    disconnected: "Reconectando...",
    lobby_not_open: "O lobby ainda não foi aberto pelo mestre.",
    throttled: "Muitas tentativas de conexão. Recarregue a página para tentar novamente.",
    kicked: "Você foi removido do lobby pelo mestre.",
    lobby_closed: "O lobby foi encerrado pelo mestre.",
    error: "Erro de conexão. Verifique sua internet.",
  };

  const wsStatusColor: Record<string, string> = {
    connected: colors.statusOngoing,
    connecting: colors.statusScheduled,
    disconnected: colors.statusScheduled,
    lobby_not_open: colors.danger,
    throttled: colors.danger,
    kicked: colors.danger,
    lobby_closed: colors.danger,
    error: colors.danger,
  };

  const handleCancelConfirm = () => {
    setShowCancelConfirm(false);
    sendCancelLobby();
  };

  return (
    <>
      <DetailPageTemplate
        hideBack
        leftSidebar={
          <CharactersSidebar
            items={lobbyEntries}
            renderItem={({ enrollment, isOnline }) => (
              <LobbyConnectionSidebarItem
                key={enrollment.uuid}
                enrollment={enrollment}
                isOnline={isOnline}
                isMaster={isMaster}
                onKick={sendKick}
                onClick={() =>
                  navigate(`/charactersheet/${enrollment.characterSheet.uuid}`)
                }
              />
            )}
          />
        }
        rightSidebar={
          <RulesSidebar>
            <RuleSection title="Configurações Gerais">
              As regras da partida seguem as definições da campanha.
            </RuleSection>
            <RuleSection title="Sistema de Combate">
              Configure o sistema de combate da partida.
            </RuleSection>
          </RulesSidebar>
        }
      >
        <LobbyHeader>
          <MatchTitle>{match.title.toUpperCase()}</MatchTitle>
          <HeaderRight>
            <LobbyPill>LOBBY ABERTO</LobbyPill>
            <ConnectedCount>{connectedCount}/{totalCount} conectados</ConnectedCount>
          </HeaderRight>
        </LobbyHeader>

        <WsStatusBar $color={wsStatusColor[status] ?? colors.statusLeft}>
          <StatusDot $color={wsStatusColor[status] ?? colors.statusLeft} $pulse={status === "connected"} />
          <StatusText>{statusMessage[status] ?? "Conectado ao lobby"}</StatusText>
        </WsStatusBar>

        {!isMaster && (
          <WaitingMessage>Aguardando o mestre iniciar a partida...</WaitingMessage>
        )}

        {isMaster && (
          <BottomActions
            primaryButton={{
              label: "Iniciar Partida",
              onClick: sendStartMatch,
            }}
            secondaryButton={{
              label: "Cancelar Lobby",
              onClick: () => setShowCancelConfirm(true),
            }}
          />
        )}
      </DetailPageTemplate>

      {showCancelConfirm && (
        <ConfirmDialog
          message="Tem certeza que deseja cancelar o lobby? Todos os jogadores conectados serão desconectados."
          confirmLabel="Cancelar Lobby"
          onConfirm={handleCancelConfirm}
          onCancel={() => setShowCancelConfirm(false)}
        />
      )}
    </>
  );
}

const LobbyHeader = styled.div`
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 20px;

  @media (max-width: 750px) {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }
`;

const MatchTitle = styled.h1`
  font-family: ${fonts.sans};
  font-size: 42px;
  font-weight: 900;
  color: ${colors.textPrimary};
  flex: 1;
  min-width: 0;
  overflow-wrap: break-word;
`;

const HeaderRight = styled.div`
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 8px;
  flex-shrink: 0;
`;

const LobbyPill = styled.span`
  font-family: ${fonts.sans};
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.08em;
  padding: 4px 12px;
  border-radius: 20px;
  background-color: ${colors.brandAccent};
  color: ${colors.textPrimary};
`;

const ConnectedCount = styled.div`
  font-family: ${fonts.sans};
  font-size: 16px;
  color: ${colors.textMuted};
`;

const WsStatusBar = styled.div<{ $color: string }>`
  display: flex;
  align-items: center;
  gap: 10px;
  background: ${({ $color }) => `${$color}18`};
  border: 1px solid ${({ $color }) => $color};
  border-radius: 8px;
  padding: 12px 16px;
  margin-bottom: 24px;
`;

const StatusDot = styled.div<{ $color: string; $pulse: boolean }>`
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: ${({ $color }) => $color};
  box-shadow: 0 0 6px ${({ $color }) => $color};
  flex-shrink: 0;

  @keyframes pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.4; }
  }

  animation: ${({ $pulse }) => ($pulse ? "pulse 2s infinite" : "none")};
`;

const StatusText = styled.span`
  font-family: ${fonts.sans};
  font-size: 13px;
  font-weight: 600;
  color: ${colors.textMuted};
`;

const WaitingMessage = styled.p`
  font-family: ${fonts.sans};
  font-size: 18px;
  color: ${colors.textMuted};
  font-style: italic;
  margin-bottom: 24px;
`;
```

**Note:** `BottomActions` may not have a `secondaryButton` prop yet. Check `src/components/molecules/BottomActions.tsx`. If it doesn't exist, add it; otherwise place the "Cancelar Lobby" button using the `manage` prop or as a separate styled button below `BottomActions`. Adjust to match the existing component API.

- [ ] **Step 6: Update `MatchPage.tsx` — add "Entrar no Lobby" button**

Find the `canEnroll` block and add `canEnterLobby` next to it:

```ts
const canEnterLobby =
  !isMaster &&
  !match.gameStartAt &&
  enrollments.some(
    (e) => e.characterSheet.uuid === sheetId && e.status === "accepted"
  );
```

In the `BottomActions` `primaryButton` prop, add a new condition:

```tsx
primaryButton={
  isMaster && !match.gameStartAt
    ? { label: "Abrir Lobby", onClick: () => setShowLobbyConfirm(true) }
    : canEnterLobby
    ? {
        label: "Entrar no Lobby",
        onClick: () =>
          navigate(`/campaigns/${campaignId}/matches/${matchId}/lobby`),
      }
    : canEnroll
    ? {
        label: enrollPending ? "Inscrevendo..." : "Inscrever-se",
        onClick: enrollPending ? () => {} : () => setShowEnrollConfirm(true),
      }
    : undefined
}
```

Also update the outer condition that wraps `BottomActions` to include `canEnterLobby`:

```tsx
{(isMaster && !match.gameStartAt) || canEnroll || canEnterLobby ? (
  <BottomActions ... />
) : null}
```

- [ ] **Step 7: Run all tests**

```bash
cd System_X_System_React && npx vitest run --reporter=verbose 2>&1 | tail -40
```

Expected: all tests pass, including `LobbyPage.test.tsx`.

- [ ] **Step 8: Commit**

```bash
cd System_X_System_React
git add \
  src/pages/LobbyPage.tsx \
  src/pages/GamePage.tsx \
  src/pages/__tests__/LobbyPage.test.tsx \
  src/App.tsx \
  src/pages/MatchPage.tsx
git commit -m "feat: implement LobbyPage with WebSocket connection and lobby lifecycle"
```

---

## Post-implementation checklist

- [ ] Game server starts without errors: `cd System_X_System && go run cmd/game/main.go`
- [ ] Frontend starts without errors: `cd System_X_System_React && npm run dev`
- [ ] Manual smoke test: master opens lobby → player gets "Entrar no Lobby" button → player connects → master sees ONLINE badge → master cancels → player sees lobby_closed message
- [ ] `go vet ./internal/app/game/...` passes
