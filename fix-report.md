# Fix Report: Lobby Hide Walls

## What Was Implemented

Added a conditional block in `internal/app/game/room.go`, function `buildMapFullState()` (lines 1140-1145), that disables wall data in the `map_full_state` WebSocket payload for non-master players in the lobby phase.

### Implementation Details

- **Location**: After `payload := MapFullStatePayload{...}` assignment (line 1139)
- **Condition**: `if !isMaster && isLobby`
- **Action**: Sets `payload.Walls = []mapentity.WallSegment{}` to send an empty walls array
- **Wall-masking computation**: Remains fully intact in the `case isLobby:` switch branch (lines 1110-1125); the code is not deleted and can be re-enabled with a one-line change to the conditional

### Test Update

Updated `TestBuildMapFullState_LobbyMasksSecretsWithoutLOS()` in `fog_dispatch_test.go` to reflect the new expected behavior:
- Now verifies that walls are NOT sent to non-master lobby players (`len(view.Walls) == 0`)
- Retained piece filtering verification (visible pieces shown, hidden pieces not shown)
- Retained VisiblePolygons check (must be empty in lobby)

## Build & Test Results

```
Build: Success
Vet: No issues found
Tests: 43 passed in 1 packages (all tests in internal/app/game/)
```

## Files Changed

1. `/internal/app/game/room.go` — Added conditional block (5 lines with comments)
2. `/internal/app/game/fog_dispatch_test.go` — Updated test assertions (simplified to check new behavior)

## Self-Review Findings

✅ **Completeness**: New conditional added exactly as specified, in the correct location

✅ **Correctness**:
- Only affects non-master + lobby players
- Master players unaffected (isMaster=true fails the condition)
- In-match players unaffected (isLobby=false when session is active)

✅ **Discipline**:
- Wall-masking computation left untouched in `case isLobby:` branch
- Only two files touched (room.go + test file)
- No other code modified

## Commits Created

- `b83e3f0` - fix(game): não enviar paredes no map_full_state do lobby (temporário)

## Notes

The wall-masking computation is still being performed and available for re-enablement later when the frontend is ready to consume it. To re-enable, change line 1144 from `payload.Walls = []mapentity.WallSegment{}` back to `payload.Walls = walls`, a single-line change as intended.
