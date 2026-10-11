package game_test

// A character's hand on a wall (enqueue_action with interact) needs the wall within reach, and
// only the master locks, unlocks or reveals. The move check measures in LOCAL grid space, the
// space walls live in — on a rotated grid the transformed centers it used to read sat elsewhere.

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	mapservice "github.com/422UR4H/HxH_RPG_System/internal/domain/map/service"
)

// reachDoorID is a UUID: targetId travels as one.
const reachDoorID = "7d0c0a3e-6f0b-4c1e-9a43-1f2d3c4b5a69"

// reachDoor is a one-cell door on the horizontal edge between (3,4) and (3,5) of the 64px grid.
func reachDoor(locked bool) mapentity.WallSegment {
	return mapentity.WallSegment{
		ID: reachDoorID, P1: [2]float64{192, 320}, P2: [2]float64{256, 320},
		WallType: mapentity.WallTypeDoor, Material: mapentity.WallMaterialWood,
		Move: true, Direction: mapentity.WallDirectionBoth, Locked: locked, HP: 10, MaxHP: 10,
	}
}

func TestE2E_WallInteract_Reach(t *testing.T) {
	cases := []struct {
		name     string
		at       [2]int
		move     *[3]int
		kind     string
		wantCode string // "" = accepted
	}{
		{name: "open from afar is out of reach", at: [2]int{3, 0}, kind: "open", wantCode: "out_of_reach"},
		{name: "open while touching the door is accepted", at: [2]int{3, 4}, kind: "open"},
		{name: "open from the far side, touching it, is accepted", at: [2]int{3, 5}, kind: "open"},
		{name: "open after walking up to the door is accepted", at: [2]int{3, 0}, move: &[3]int{3, 4, 0}, kind: "open"},
		{name: "walking somewhere that does not touch the door is out of reach", at: [2]int{3, 0}, move: &[3]int{3, 2, 0}, kind: "open", wantCode: "out_of_reach"},
		{name: "lockpick from afar is out of reach", at: [2]int{3, 0}, kind: "lockpick", wantCode: "out_of_reach"},
		{name: "a character never locks", at: [2]int{3, 4}, kind: "lock", wantCode: "invalid_action"},
		{name: "a character never unlocks", at: [2]int{3, 4}, kind: "unlock", wantCode: "invalid_action"},
		{name: "an unknown kind is refused", at: [2]int{3, 4}, kind: "smash", wantCode: "invalid_action"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newCombatFixture(t)
			seedGeomBoard(t, f, squareGeomGrid,
				[]mapentity.Piece{squarePiece("p-attacker", f.attackerID, tc.at[0], tc.at[1])},
				[]mapentity.WallSegment{reachDoor(false)})
			master, player := f.connect(t)
			defer master.Close() //nolint:errcheck
			defer player.Close() //nolint:errcheck
			playerMsgs := collectFrom(player)
			if !collectFrom(master).await(game.MsgTypeMapFullState, 2*time.Second) {
				t.Fatal("the master never got the board — the fixture never started")
			}

			payload := map[string]any{
				"actorId":  f.attackerID.String(),
				"targetId": []string{reachDoorID},
				"interact": map[string]any{"kind": tc.kind},
			}
			if tc.move != nil {
				payload["move"] = map[string]any{"category": string(enum.Dash), "position": *tc.move}
			}
			sendWS(t, player, "enqueue_action", payload)
			if tc.wantCode != "" {
				awaitErrorWithCode(t, playerMsgs, tc.wantCode, 2*time.Second)
				return
			}
			refutesErrorAfter(t, playerMsgs, "out_of_reach", game.MsgTypeActionEnqueued, 2*time.Second)
		})
	}
}

func TestE2E_WallInteract_AttackNeedsNoReach(t *testing.T) {
	f := newCombatFixture(t)
	seedGeomBoard(t, f, squareGeomGrid,
		[]mapentity.Piece{squarePiece("p-attacker", f.attackerID, 3, 0)},
		[]mapentity.WallSegment{reachDoor(false)})
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	playerMsgs := collectFrom(player)
	if !collectFrom(master).await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the master never got the board — the fixture never started")
	}
	sendWS(t, player, "enqueue_action", map[string]any{
		"actorId": f.attackerID.String(), "targetId": []string{reachDoorID}, "attack": map[string]any{},
	})
	refutesErrorAfter(t, playerMsgs, "out_of_reach", game.MsgTypeActionEnqueued, 2*time.Second)
}

// awaitWallState waits for a wall_state_changed of this wall and returns it.
func awaitWallState(t *testing.T, c *collector, wallID string, d time.Duration) game.WallStateChangedPayload {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		for _, m := range c.snapshotMessages() {
			if m.Type != game.MsgTypeWallStateChanged {
				continue
			}
			var p game.WallStateChangedPayload
			if err := json.Unmarshal(m.Payload, &p); err == nil && p.WallID == wallID {
				return p
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("never saw wall_state_changed for %s", wallID)
	return game.WallStateChangedPayload{}
}

func TestE2E_MasterLocksAndUnlocksADoor(t *testing.T) {
	for _, tc := range []struct {
		kind       string
		startLock  bool
		wantLocked bool
	}{
		{kind: "lock", startLock: false, wantLocked: true},
		{kind: "unlock", startLock: true, wantLocked: false},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			f := newCombatFixture(t)
			seedGeomBoard(t, f, squareGeomGrid,
				[]mapentity.Piece{squarePiece("p-attacker", f.attackerID, 3, 4)},
				[]mapentity.WallSegment{reachDoor(tc.startLock)})
			master, player := f.connect(t)
			defer master.Close() //nolint:errcheck
			defer player.Close() //nolint:errcheck
			masterMsgs := collectFrom(master)
			if !masterMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
				t.Fatal("the master never got the board — the fixture never started")
			}
			sendWS(t, master, "enqueue_master_action", map[string]any{
				"targetIds": []string{reachDoorID}, "interact": map[string]any{"kind": tc.kind},
			})
			got := awaitWallState(t, masterMsgs, reachDoorID, 2*time.Second)
			if got.Locked != tc.wantLocked || got.Open {
				t.Errorf("got locked=%v open=%v, want locked=%v open=false", got.Locked, got.Open, tc.wantLocked)
			}
		})
	}
}

// On a grid rotated 90°, the transformed center of (4,4) sits nowhere near the local one. A
// wall between the LOCAL centers of (4,4) and (5,4) must block the Dash between them.
func TestE2E_MoveWallCheck_RotatedGridMeasuresInLocalSpace(t *testing.T) {
	f := newCombatFixture(t)
	rotated := squareGeomGrid
	rotated.Rotation = 90
	fx, fy := mapservice.SlotCenterToLocal(4, 4, rotated)
	tx, ty := mapservice.SlotCenterToLocal(5, 4, rotated)
	wall := bandWall("band", (fx+tx)/2, (fy+ty)/2, rotated.CellSize)
	seedGeomBoard(t, f, rotated,
		[]mapentity.Piece{squarePiece("p-attacker", f.attackerID, 4, 4)},
		[]mapentity.WallSegment{wall})
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	playerMsgs := collectFrom(player)
	if !collectFrom(master).await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the master never got the board — the fixture never started")
	}
	sendGeomDash(t, player, f.attackerID, [3]int{4, 4, 0}, [3]int{5, 4, 0})
	awaitErrorWithCode(t, playerMsgs, "move_blocked", 2*time.Second)
}

// The master's hand brings a wall down — the indestructible included (an earthquake is not an
// attack). The table hears it as HP news: hp 0, destroyed.
func TestE2E_MasterDestroysAWall(t *testing.T) {
	f := newCombatFixture(t)
	solid := reachDoor(false)
	solid.WallType = mapentity.WallTypeWall
	solid.HP, solid.MaxHP = 0, 0
	seedGeomBoard(t, f, squareGeomGrid,
		[]mapentity.Piece{squarePiece("p-attacker", f.attackerID, 3, 4)},
		[]mapentity.WallSegment{solid})
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	masterMsgs := collectFrom(master)
	if !masterMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the master never got the board — the fixture never started")
	}
	sendWS(t, master, "enqueue_master_action", map[string]any{
		"targetIds": []string{reachDoorID}, "interact": map[string]any{"kind": "destroy"},
	})
	got := awaitWallHp(t, masterMsgs, reachDoorID, 2*time.Second)
	if !got.Destroyed || got.HP != 0 {
		t.Errorf("got %+v, want destroyed with hp 0", got)
	}
}

func awaitWallHp(t *testing.T, c *collector, wallID string, d time.Duration) game.WallHpChangedPayload {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		for _, m := range c.snapshotMessages() {
			if m.Type != game.MsgTypeWallHpChanged {
				continue
			}
			var p game.WallHpChangedPayload
			if err := json.Unmarshal(m.Payload, &p); err == nil && p.WallID == wallID {
				return p
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("never saw wall_hp_changed for %s", wallID)
	return game.WallHpChangedPayload{}
}

// A blow on a wall lands with the blows on characters — at the close. While the turn is open the
// master reads it in the resolution (walls[]); the board only changes when the turn closes. The
// master's "no skill measures the damage" reaches the wall too.
func TestE2E_WallBlowLandsAtTheClose(t *testing.T) {
	f := newCombatFixture(t)
	door := reachDoor(false)
	door.HP, door.MaxHP, door.Resistance = 1000, 1000, 0
	seedGeomBoard(t, f, squareGeomGrid,
		[]mapentity.Piece{squarePiece("p-attacker", f.attackerID, 3, 0)},
		[]mapentity.WallSegment{door})
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc, pc := collectFrom(master), collectFrom(player)
	if !mc.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the master never got the board — the fixture never started")
	}
	sendWS(t, player, "enqueue_action", map[string]any{
		"actorId": f.attackerID.String(), "targetId": []string{reachDoorID}, "attack": map[string]any{},
	})
	if !pc.await(game.MsgTypeActionEnqueued, 2*time.Second) {
		t.Fatal("the attack was never enqueued")
	}
	sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
	if !mc.await(game.MsgTypeResolutionUpdate, 2*time.Second) {
		t.Fatal("no resolution for the opened turn")
	}
	res := lastResolutionUpdated(t, mc)
	if len(res.Walls) != 1 || res.Walls[0].WallID != reachDoorID || res.Walls[0].Kind != "attack" {
		t.Fatalf("walls = %+v, want the door's attack result", res.Walls)
	}
	if res.Walls[0].HP != 1000-res.Walls[0].EffectiveDamage {
		t.Errorf("projected hp %d, want 1000 - %d", res.Walls[0].HP, res.Walls[0].EffectiveDamage)
	}
	if mc.count(game.MsgTypeWallHpChanged) != 0 {
		t.Fatal("the wall changed when the turn opened — the blow lands at the close")
	}

	sendWS(t, master, string(game.MsgTypeEditAction), map[string]any{"damageSkill": "None"})
	if !awaitCount(mc, game.MsgTypeResolutionUpdate, 2, 2*time.Second) {
		t.Fatal("the edit never republished the resolution")
	}
	edited := lastResolutionUpdated(t, mc)
	if edited.DamageSkill != "None" {
		t.Fatalf("damageSkill = %q, want None", edited.DamageSkill)
	}
	if edited.Walls[0].EffectiveDamage > res.Walls[0].EffectiveDamage {
		t.Errorf("no damage skill raised the blow: %d > %d", edited.Walls[0].EffectiveDamage, res.Walls[0].EffectiveDamage)
	}

	sendWS(t, master, string(game.MsgTypeCloseTurn), game.CloseTurnPayload{Confirm: true})
	got := awaitWallHp(t, mc, reachDoorID, 2*time.Second)
	if got.HP != 1000-edited.Walls[0].EffectiveDamage {
		t.Errorf("hp after close = %d, want %d", got.HP, 1000-edited.Walls[0].EffectiveDamage)
	}
}
