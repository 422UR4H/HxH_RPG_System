# Master piece perspective + player framing — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** (1) In a running match, when the master selects a character, the master's board shows only what THAT piece sees (fog, walls, pieces); with nothing selected, the full board as today. (2) The player's "Enquadrar" (and the opening framing) zooms in centered on the player's own piece(s); the master keeps the whole-map framing.

**Architecture:** The server already computes line of sight (LOS) per player. We add a master-only subscription: the front says which piece the master is looking through (`watch_piece_view`), and the server answers with a `piece_view` (that piece's polygon, the wall ids and piece ids it sees) and re-sends it every time LOS can change. The front draws the master's board through that view. Framing is front-only: `ViewportInner` learns an optional focus (slots) to frame instead of the whole grid.

**Tech Stack:** Go 1.23 (`internal/app/game`, `internal/domain/match`), React/TS + Pixi (`@pixi/react`, pixi-viewport), vitest.

**Spec:** this document. The design was agreed in chat with the product owner on 2026-11-04; its decisions are in "Design decisions" below.

## Design decisions (product owner, 2026-11-04)

- **What the master sees:** only what the selected piece sees right now from where it stands: its live LOS polygon, the walls in that LOS, and the pieces in that LOS. This is not the owner player's screen: no explored-wall memory and no union of the player's other pieces. The same rule applies to NPCs and player characters.
- **Unrevealed secret doors** inside that view are drawn as a plain wall, exactly as a player receives them (`maps.md` → "Máscara de porta secreta não revelada").
- **What turns it on:** in play mode (`boardMode === "play"` and not arranging), the actor NPC the master is acting through (`actorId`), or else the character the master tapped to inspect (`inspectedId`). With neither, there is no fog and the full board shows. Arranging mode always shows the full board. While a fall/reaction-escape square pick is open, keep whichever piece was already active (the actor).
- **No UI text** announces the mode. It is documented for the master in `docs/game/combate/o-mestre.md`.
- **Player framing:** the button and the opening framing center on the player's own piece(s), zoomed so about **9 cells** fit across the shorter side of the canvas, never zoomed out further than the whole-map framing. With several own pieces, frame their bounding box (at least 9 cells). With no own piece on the board, frame the whole map (today's behavior). The camera does not follow the piece on every move. It frames on open, on "Enquadrar", and once when an own piece first becomes available, as long as the user has not moved the camera by hand. The master's framing is unchanged.

## Global Constraints

- Code identifiers and test titles in English; code comments, docs and UI copy in Portuguese (pt-BR).
- Wire format camelCase; no conversion layer on the front.
- Backend: `r.mu` guards session/room state; never call `persistBoard` or a dispatch while holding `r.mu`. Callers of `MatchSession` hold `r.mu`.
- Backend: `go vet ./...` after each task; `golangci-lint run` before the PR.
- Front: `verbatimModuleSyntax` (type-only imports use `import type`); no raw colors.
- Branches: back `feat/master-piece-perspective` (from `main`), front `feat/master-perspective-player-framing` (from `main`). Edit the main checkouts directly (no worktree).
- Commits end with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.

## Review Focus

- A watched piece that is `visible: false` (hidden by the master) must still be its own viewpoint: it is always in `pieceIds`. Pinned by a test in Task 1.
- The watched piece is removed from the board: the server drops the watch and sends nothing more; the front falls back to the full board because the selection disappears with the piece. Pinned in Task 2.
- WS reconnect while watching: the front re-sends `watch_piece_view` when the status returns to `connected`. Pinned in Task 4.
- Another piece walks into or out of the watched piece's LOS: the master gets a fresh `piece_view` with the new `pieceIds`. Pinned in Task 2.
- A late `piece_view` for the previous selection must never be drawn for the current one (the front ignores a `pieceId` mismatch). Pinned in Task 4.

---

## Backend (`System_X_System/`, branch `feat/master-piece-perspective`)

### Task 1: Domain — LOS from a single point, and the perspective filter

**Files:**
- Modify: `internal/domain/match/matchsession/match_session.go` (around `RecomputeVisibility`, ~L1946)
- Modify: `internal/domain/match/service/filter_map_state.go`
- Test: `internal/domain/match/matchsession/` (existing visibility tests' file, or a new `visibility_from_test.go`), `internal/domain/match/service/filter_map_state_test.go`

**Interfaces:**
- Produces: `func (s *MatchSession) VisibilityFrom(pos service.Point2D) service.VisibilityPolygon`. It is pure: no cache write and no memory write. Caller holds `r.mu` (read is enough).
- Produces: `func FilterPerspective(allWalls []mapentity.WallSegment, pieces []PieceVisibility, poly VisibilityPolygon, viewerPieceID string) (walls []mapentity.WallSegment, visiblePieceIDs map[string]bool)` in package `service`.

- [ ] **Step 1: Failing tests.**
  - `TestVisibilityFrom_MatchesRecomputeForSinglePiece`: one player with one piece; `VisibilityFrom(thatPos)` equals `RecomputeVisibility(player)[0]`. Also assert that `GetVisibility(player)` and the player's memory are unchanged by `VisibilityFrom`, using a fresh session where `RecomputeVisibility` was never called, so the cache stays nil.
  - `TestFilterPerspective` (table-driven):
    - (a) a wall in the polygon is returned; a wall outside is not.
    - (b) an unrevealed `secret_door` in sight is returned masked (`WallType == "wall"`, via `MaskSecretDoorForPlayer`).
    - (c) a piece inside the polygon and `Visible` is in the set; a piece outside is not.
    - (d) a piece with `Visible: false` inside the polygon is NOT in the set.
    - (e) `viewerPieceID` is in the set even when `Visible: false`.
- [ ] **Step 2: Run them and see them fail.** `go test ./internal/domain/match/...` fails to compile (undefined).
- [ ] **Step 3: Implement.**
  - Extract the LOS setup that `RecomputeVisibility` does inline (`losWalls` with the boundary walls, and `maxRadius` from `GridLocalBounds`) into one unexported helper. Both `RecomputeVisibility` and `VisibilityFrom` use it, so there is no copy.
  - `FilterPerspective` reuses `wallInLOS`/`IsVisible`/`MaskSecretDoorForPlayer`. It must not route through `FilterMapState`: that function's "own piece" rule is keyed by player, and here it is keyed by piece.
- [ ] **Step 4: Run the tests and see them pass.** Then run `go vet ./...`.
- [ ] **Step 5: Commit.** `feat(match): LOS from a single point and the piece perspective filter`

### Task 2: Game server — `watch_piece_view` / `piece_view`, the contract and the master doc

**Files:**
- Modify: `internal/app/game/message.go` (new types + payloads)
- Modify: `internal/app/game/room.go` (Room field, `handleIncoming` arm, push hooks)
- Test: new `internal/app/game/piece_view_e2e_test.go`. Follow the harness in `visibility_e2e_test.go` / `fog_e2e_test.go`, and reuse `liveMatchRoom` from `fog_regression_test.go` where it fits.
- Docs: `docs/dev/api/maps.md` (section "Fog of War / Visibilidade": add a "Ponto de vista do mestre" subsection with both messages), `docs/documentation-map.yaml` (add `room.go`/`message.go` → `maps.md` if they are not mapped there already), `docs/game/combate/o-mestre.md` (new section, see Step 6).

**Interfaces:**
- Consumes: Task 1's `VisibilityFrom`, `FilterPerspective`.
- Produces (wire, the front relies on it exactly):
  - c→s `watch_piece_view`, payload `{ "pieceId": string }`. An empty string stops watching.
  - s→c `piece_view`, payload `{ "pieceId": string, "visiblePolygons": [{x,y}][], "wallIds": string[], "pieceIds": string[] }`. Polygons are LOCAL grid coords, like `visibility_updated` (reuse `polysToPayload` with a single polygon). `wallIds`/`pieceIds` are never `null` (always encode `[]`).
  - Errors (via `NewErrorMessage`):
    - `forbidden` + `ErrNotMaster` when a non-master sends it.
    - `invalid_payload` when the JSON is bad.
    - `no_session` when there is no live match (reuse the code the room already uses for "no session" if one exists; grep before adding).
    - `unknown_piece` when the piece is not on the board. The watch is cleared.

- [ ] **Step 1: Failing e2e tests** (master and player connected to a live room):
  - `TestWatchPieceView_MasterGetsPieceView`: the master sends `watch_piece_view{p1}` and receives `piece_view` with `pieceId == "p1"`, a non-empty polygon, `wallIds` containing the in-sight wall, and `pieceIds` containing `p1`.
  - `TestWatchPieceView_PlayerForbidden`: a player gets `error` `forbidden`.
  - `TestWatchPieceView_UnknownPiece`: the master gets `unknown_piece`, and no `piece_view` follows later moves.
  - `TestWatchPieceView_UpdatesWhenAnotherPieceMoves`: watching `p1`, a second piece moves from outside to inside `p1`'s LOS. The master receives a `piece_view` whose `pieceIds` now includes it, and it arrives AFTER that move's `piece_moved` on the master's socket.
  - `TestWatchPieceView_UpdatesOnWallChange`: watching, the master opens/closes a door through `enqueue_master_action` interact. The master receives a fresh `piece_view`.
  - `TestWatchPieceView_StopsOnEmptyAndOnRemoval`: after `watch_piece_view{""}`, a move produces no `piece_view`. In a second subtest, removing the watched piece produces no `piece_view` afterwards.
- [ ] **Step 2: Run them and see them fail.** `go test ./internal/app/game/ -run WatchPieceView`.
- [ ] **Step 3: Implement.**
  - **Room state:** a `masterViewPieceID string` field on `Room`, guarded by `r.mu`. Clear it when the master's connection registers, because a new socket starts with no watch and the front re-sends.
  - **Handler arm `MsgTypeWatchPieceView`:** master-only. Set or clear the field under `r.mu`, then send the view right away (when non-empty).
  - **`func (r *Room) buildPieceView() *Message`:** takes `r.mu.RLock`. Returns nil when there is no watch, no session, or the watched piece is gone; in the gone case, clear the field (take the write lock separately, never upgrade a held lock). Otherwise it builds the piece projection the same way `buildMapFullState` does, then `VisibilityFrom(piecePos)` + `FilterPerspective`.
  - **`func (r *Room) pushMasterPieceView()`:** sends `buildPieceView()` to the master's client on the direct lane (`client.SendMessage`), the same lane as `piece_moved`, so the order is guaranteed. Call it, without holding `r.mu`:
    1. at the end of `pushVisibilityUpdates`;
    2. at every exit of the piece-move relay after its dispatch (`relayPieceMove` or whatever the function containing `pieceMoveView` is named, ~L3150), including the `owner == uuid.Nil` early return. NPC moves are exactly the ones that take that return;
    3. at every exit of the piece-removal relay (~L3700);
    4. after each `r.dispatchPerPlayer(... buildMapFullState ...)` that re-pushes the board to everyone (~L369 and ~L724).
  - Keep the comment that explains why the master is not an LOS "owner" (~L3157) accurate. The perspective is computed on demand, not cached, so that reasoning still holds.
- [ ] **Step 4: Run the tests and see them pass.** Then `go test ./internal/app/game/...` and `go vet ./...`.
- [ ] **Step 5: Contract.** In `maps.md`, document `watch_piece_view` (c→s) and `piece_view` (s→c): payload tables, the errors above, master-only, when the server re-sends it, that `wallIds` follow `IsWallInSight` and that unrevealed secret doors in sight are listed by id (the front masks them, per the same table), that `pieceIds` exclude `visible:false` pieces except the watched one, and that the watch dies with the master's connection. Update the "Nunca enviado ao master" line of `visibility_updated` only if it reads as contradicting this (it does not; `piece_view` is a different message).
- [ ] **Step 6: Master rules doc.** In `docs/game/combate/o-mestre.md`, add the section `## Ver pelos olhos de um personagem`, before "O que os jogadores enxergam". In plain pt-BR, for a game master, not a developer, it says:
  - when the master picks an NPC to act through, or taps a character, the table shows only what that character sees from where it stands: the fog, the walls in view, the pieces in view;
  - it is the character's view right now, not the player's screen (no memory of explored walls);
  - with no one selected, the master sees the whole board again;
  - in arranging mode, the master always sees everything.
- [ ] **Step 7: Commit.** `feat(game): master watches the board through a piece (watch_piece_view/piece_view)`

---

## Front (`System_X_System_React/`, branch `feat/master-perspective-player-framing`)

### Task 3: WS plumbing for `watch_piece_view` / `piece_view`

**Files:**
- Modify: `src/hooks/useMatchWs.ts`, `src/features/match/combat/useMatchCombat.ts`, `src/features/match/combat/useGameTable.ts`
- Modify: `src/types/tacticalMap.ts` (type)
- Test: `src/hooks/__tests__/useMatchWs.test.ts`

**Interfaces:**
- Consumes: Task 2's wire.
- Produces:
  - `export type PieceView = { pieceId: string; visiblePolygons: Array<Array<[number, number]>>; wallIds: string[]; pieceIds: string[] }` in `src/types/tacticalMap.ts`. The polygons use the same parsed shape `visibility_updated` produces (`parsePolys`).
  - `useMatchWs` option `onPieceView?: (v: PieceView) => void` and returned `sendWatchPieceView(pieceId: string | null): boolean` (null → `""` on the wire).
  - `useMatchCombat` forwards both: option `onPieceView`, and `send.watchPieceView`.
  - `useGameTable` returns `pieceView: PieceView | undefined` (latest received; cleared to `undefined` on every `match_full_state`/reconnect via the existing `onFullState`) and passes `onPieceView`.

- [ ] **Step 1: Failing tests** in `useMatchWs.test.ts`, following the file's existing message tests:
  - `parses piece_view into PieceView`: `{x,y}` points become tuples, and missing `wallIds`/`pieceIds` become `[]`;
  - `sendWatchPieceView(null) sends an empty pieceId`.
- [ ] **Step 2: Run them and see them fail.** `npx vitest run src/hooks/__tests__/useMatchWs.test.ts`.
- [ ] **Step 3: Implement**, following the existing `visibility_updated` / `sendRaw` patterns (callback kept in a ref).
- [ ] **Step 4: Run the tests and see them pass.** Then run `npx tsc -b`.
- [ ] **Step 5: Commit.** `feat(match): piece_view plumbing in the match socket`

### Task 4: Master board through the selected piece

**Files:**
- Create: `src/features/match/combat/piecePerspective.ts` (pure)
- Create: `src/features/match/combat/usePiecePerspective.ts` (hook)
- Modify: `src/pages/GameMasterPage.tsx` (the `TacticalMapViewer` block ~L725)
- Modify: `src/features/tactical-map/TacticalMapViewer.tsx` (fog switch)
- Test: `src/features/match/combat/__tests__/piecePerspective.test.ts`, `src/pages/__tests__/GameMasterPage.test.tsx`

**Interfaces:**
- Consumes: Task 3 (`PieceView`, `combat.send.watchPieceView`, `game.pieceView`, `combat.status`).
- Produces:
  - `export function maskSecretDoor(w: WallSegment): WallSegment`. It mirrors the backend mask table in `maps.md`: `wallType "wall"`, `doorSubtype`/`windowSubtype` removed, `open false`, `locked false`, everything else kept. It applies only when `wallType === "secret_door" && !revealed`; otherwise it returns `w`. The comment points to the contract table.
  - `export function boardThroughPiece(walls: WallSegment[], pieces: Piece[], view: PieceView, fogMode: FogState["fogMode"]): { walls: WallSegment[]; pieces: Piece[]; fog: FogState }`. It keeps the walls whose id is in `view.wallIds` (masked) and the pieces whose id is in `view.pieceIds`, and builds `fog = { fogMode, visiblePolygons: view.visiblePolygons }`.
  - `export function usePiecePerspective(pieceId: string | undefined, view: PieceView | undefined, status: MatchWsStatus, watch: (id: string | null) => boolean): PieceView | undefined`. It sends `watch(pieceId ?? null)` whenever `pieceId` changes and again whenever `status` becomes `"connected"`. It returns `view` only when `view?.pieceId === pieceId`.
  - `TacticalMapViewer`: replace `fogDisabled={!!isMaster || !fog}` with `fogDisabled={!fog}`. Callers that must not fog pass `fog={undefined}`. Check every `TacticalMapViewer` caller with `grep -rn "<TacticalMapViewer" src` and keep their current behavior. If `isMaster` ends up unused, remove the prop.

- [ ] **Step 1: Failing tests.**
  - `piecePerspective.test.ts`:
    - only listed walls and pieces are kept;
    - an unrevealed secret door comes out masked, and a revealed one comes out untouched;
    - the fog carries the view's polygons and the given mode.
  - `GameMasterPage.test.tsx`, using the file's existing helpers for a live table and its socket stub:
    - `watches the actor NPC piece when the master picks it`: `watch_piece_view` is sent with that piece id;
    - `watches the inspected character piece`;
    - `stops watching when the actor is cleared` (sends `""`);
    - `does not watch while arranging`;
    - `ignores a piece_view for a different piece`: the viewer still gets the full walls/pieces and no fog. Assert through whatever the file's Pixi mock exposes. If the mock exposes nothing usable, assert on the hook's return value in a `usePiecePerspective` test with `renderHook` instead.
    - `re-sends the watch after a reconnect`.
- [ ] **Step 2: Run them and see them fail.**
- [ ] **Step 3: Implement.**
  - In `GameMasterPage`: `perspectivePieceId = boardMode === "play" && !arranging ? (composer.actorPiece?.id ?? inspectedPieceId) : undefined`. Use the page's real names for these.
  - `const view = usePiecePerspective(...)`.
  - When `view` is set, the viewer gets `boardThroughPiece(live.liveWalls, game.boardPieces, view, live.fog.fogMode)`'s walls, pieces and fog. Otherwise it gets the full walls/pieces and `fog={undefined}`.
  - Memoize the derived board.
- [ ] **Step 4: Run the tests and see them pass.** `npm run test`, `npm run lint`, `npx tsc -b`.
- [ ] **Step 5: Commit.** `feat(match): master sees the board through the selected piece`

### Task 5: Player framing on their own piece

**Files:**
- Create: `src/features/tactical-map/utils/focusFrame.ts` (pure)
- Modify: `src/features/tactical-map/stage/ViewportInner.tsx` (fit effect ~L188-214), `src/features/tactical-map/stage/stageProps.ts`, `src/features/tactical-map/TacticalMapStage.tsx`, `src/features/tactical-map/TacticalMapViewer.tsx` (pass-through)
- Modify: `src/pages/GamePlayerPage.tsx` (~L345)
- Test: `src/features/tactical-map/utils/__tests__/focusFrame.test.ts`

**Interfaces:**
- Produces:
  - `export const FOCUS_CELLS = 9;`
  - `export function focusFrame(focus: XY[], cellSize: number, view: { width: number; height: number }, wholeMapScale: number): { center: XY; scale: number }`. `focus` holds world points (piece centers via `slotToWorld`). The frame box is the focus bounding box grown to at least `FOCUS_CELLS * cellSize` on each axis, around the box center. `scale = min(view.width / boxW, view.height / boxH)`, then `max(scale, wholeMapScale)`, then clamped to `[0.25, 1.75]` (the same clamp the whole-map fit uses).
  - Stage prop `fitFocusSlots?: SlotCoord[]`. An empty array or `undefined` means whole-map framing.

- [ ] **Step 1: Failing tests** in `focusFrame.test.ts`:
  - `centers on a single piece`;
  - `fits about 9 cells across the shorter side`: cellSize 64, view 576×1000, one point → scale 1;
  - `never zooms out past the whole-map fit` (wholeMapScale 1.2 → 1.2);
  - `frames the bounding box of several pieces`: two points 20 cells apart horizontally → the box is ≥ 20 cells wide;
  - `clamps to 1.75`.
- [ ] **Step 2: Run them and see them fail.**
- [ ] **Step 3: Implement.**
  - Write `focusFrame`.
  - In `ViewportInner`:
    - read `fitFocusSlots` through a ref (like `gridRef`);
    - when it is non-empty, the fit effect uses `focusFrame` (computing `wholeMapScale` exactly as today, without the 0.25/1.75 clamp's effect lost), then `setZoom` + `moveCenter(center)`;
    - add `hasFocus = !!fitFocusSlots?.length` to the effect's deps, so the first time an own piece shows up it frames once (subject to the existing `userMovedRef` rule), while later moves do not reframe.
  - In `GamePlayerPage`: `fitFocusSlots` = the slots of the board pieces whose `characterId` belongs to `myCharacters`, memoized on the ids and slots. Pass it only on the player page. `GameMasterPage`, the editor and the placer stay untouched.
- [ ] **Step 4: Run the tests and see them pass.** `npm run test`, `npm run lint`, `npx tsc -b`.
- [ ] **Step 5: Commit.** `feat(match): player framing centers on their own piece`

---

## Delivery (orchestrator, not a subagent task)

- **Browser check, per the root `CLAUDE.md` "Entrega":**
  - Run `go run ./cmd/devseed`; see `docs/dev/seed.md` and the memory notes on browser verification.
  - Master:
    - pick an NPC → fog plus only the walls and pieces in view;
    - tap a player character → its view;
    - clear the selection → full board;
    - arranging → full board;
    - move a piece in view → the fog follows.
  - Player: open the table → framed on their own piece; pan, then press Enquadrar → reframed on the piece.
  - Master's Enquadrar → whole map.
- **PRs:** one per repo, cross-linked. Each says what was verified and what was not.
- **Handoff:** if anything was left unverified, run `./dev-checkout.sh <branch>` from the project root.
