package game_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/422UR4H/HxH_RPG_System/internal/app/wire/actionwire"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// This file is the Phase 7 back package's reaction_opened (spec §4.1, decision D1): opening a
// reaction tells the whole table WHO narrates next AND WITH WHAT — the reaction itself, cut per
// recipient by the very rule turn_opened's action is cut by. The master gets it whole; everyone
// else, its owner included, gets it at Opened (dice cut, both speeds kept); a third party also
// gets the closed label demoted and the Evasion entry and consumedActionIds stripped. And the
// escape's destination passes the reactor's piece's fog gate, like an opened move's does.

// reactionOpenedTo reads the first reaction_opened a collector holds: its ids, the projected
// reaction decoded, and the raw reaction — "is this key on the wire at all" is asked of the raw.
func reactionOpenedTo(t *testing.T, who string, c *collector) (turnID, reactionID uuid.UUID, react actionwire.Action, raw json.RawMessage) {
	t.Helper()
	msg := findMessage(t, c.snapshotMessages(), game.MsgTypeReactionOpened)
	var p struct {
		TurnID     uuid.UUID       `json:"turnId"`
		ReactionID uuid.UUID       `json:"reactionId"`
		Reaction   json.RawMessage `json:"reaction"`
	}
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		t.Fatalf("%s: unmarshal reaction_opened: %v", who, err)
	}
	if len(p.Reaction) == 0 || string(p.Reaction) == "null" {
		t.Fatalf("%s: reaction_opened carries no reaction: %s", who, msg.Payload)
	}
	if err := json.Unmarshal(p.Reaction, &react); err != nil {
		t.Fatalf("%s: unmarshal reaction_opened.reaction: %v", who, err)
	}
	return p.TurnID, p.ReactionID, react, p.Reaction
}

func hasSkill(a actionwire.Action, name string) bool {
	for _, s := range a.Skills {
		if s.SkillName == name {
			return true
		}
	}
	return false
}

func TestE2E_ReactionOpenedCarriesTheReactionCutPerRecipient(t *testing.T) {
	f := newCombatFixture(t, withBystander, withVictimPiece)
	f.seedBoard(t)
	master, player, blind, mc, pc, bc := f.connectTable(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	defer blind.Close()  //nolint:errcheck

	reactionID := f.escapeStage(t, master, player, mc, pc, true, true, true)
	for who, c := range map[string]*collector{"owner": pc, "bystander": bc} {
		if !c.await(game.MsgTypeReactionOpened, 2*time.Second) {
			t.Fatalf("the %s never saw the reaction open", who)
		}
	}
	evasion := enum.Evasion.String()

	// Every copy names the same turn and reaction, and the reaction it carries IS that one.
	read := func(who string, c *collector) (actionwire.Action, json.RawMessage) {
		turnID, id, react, raw := reactionOpenedTo(t, who, c)
		if turnID == uuid.Nil || id != reactionID || react.UUID != reactionID || react.ActorID != f.victimID {
			t.Fatalf("%s: reaction_opened turnId=%s reactionId=%s reaction.uuid=%s actorId=%s, want reaction %s by %s",
				who, turnID, id, react.UUID, react.ActorID, reactionID, f.victimID)
		}
		return react, raw
	}

	// The master: whole (Full) — the closed label, the Evasion entry, the dodge's number.
	toMaster, _ := read("master", mc)
	if toMaster.ReactionKind != "closedEscape" {
		t.Errorf("master: reactionKind = %q, want closedEscape", toMaster.ReactionKind)
	}
	if !hasSkill(toMaster, evasion) {
		t.Errorf("master: no %s entry in skills %+v", evasion, toMaster.Skills)
	}
	if toMaster.Dodge == nil || toMaster.Dodge.RollCheck.Result == nil {
		t.Errorf("master: dodge.rollCheck.result is missing — the master's copy is Full: %+v", toMaster.Dodge)
	}

	// The owner: the closed label and its destination, but at Opened — the dice cut.
	toOwner, _ := read("owner", pc)
	if toOwner.ReactionKind != "closedEscape" {
		t.Errorf("owner: reactionKind = %q, want closedEscape", toOwner.ReactionKind)
	}
	if toOwner.Dodge == nil || toOwner.Dodge.RollCheck.Result != nil {
		t.Errorf("owner: dodge = %+v, want a dodge with no rollCheck.result (Opened)", toOwner.Dodge)
	}
	if toOwner.Move == nil || toOwner.Move.Position == nil || *toOwner.Move.Position != escapeTo {
		t.Errorf("owner: move = %+v, want position %v", toOwner.Move, escapeTo)
	}

	// The third party: the label demoted, the Evasion entry and consumedActionIds stripped, the
	// dice cut.
	toBystander, rawBystander := read("bystander", bc)
	if toBystander.ReactionKind != "escape" {
		t.Errorf("bystander: reactionKind = %q, want escape (the closed label demoted)", toBystander.ReactionKind)
	}
	if hasSkill(toBystander, evasion) {
		t.Errorf("bystander: skills %+v carry the %s entry", toBystander.Skills, evasion)
	}
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(rawBystander, &keys); err != nil {
		t.Fatal(err)
	}
	if _, ok := keys["consumedActionIds"]; ok {
		t.Errorf("bystander: consumedActionIds is on the wire: %s", keys["consumedActionIds"])
	}
	if toBystander.Dodge == nil || toBystander.Dodge.RollCheck.Result != nil {
		t.Errorf("bystander: dodge = %+v, want a dodge with no rollCheck.result (Opened)", toBystander.Dodge)
	}
}

// seedReactorBoard seeds a board with the VICTIM — the reactor — at `at` (visible or not) and
// the bystander at hiddenMoveBystanderAt, split by the given walls: seedMoverBoard's shape, with
// the piece that moves being the one that reacts instead of the one that acts.
func (f *combatFixture) seedReactorBoard(t *testing.T, at [2]int, visible bool, walls ...mapentity.WallSegment) {
	t.Helper()
	f.seedHiddenMoveBoard(t, []mapentity.Piece{{
		ID:          victimPieceID,
		CharacterID: f.victimID.String(),
		Coord:       mapentity.PieceCoord{Slot: mapentity.SquareCoord{Kind: "square", Col: at[0], Row: at[1]}},
		Visible:     visible,
	}}, walls...)
}

// openEscapeOnTable opens the attack on the victim, attaches the victim's Dash escape from
// `from` to `to` and opens it, waiting for every recipient's reaction_opened. It returns the
// reaction's id.
func (f *combatFixture) openEscapeOnTable(
	t *testing.T, master, player *websocket.Conn, mc, pc, bc *collector, from, to [3]int,
) uuid.UUID {
	t.Helper()
	actionID := f.openAttackOn(t, player, master, mc)
	reactionID := f.attachDashEscape(t, player, mc, actionID, from, to)
	sendWS(t, master, string(game.MsgTypeOpenReaction), game.OpenReactionPayload{ReactionID: reactionID})
	for who, c := range map[string]*collector{"master": mc, "owner": pc, "bystander": bc} {
		if !c.await(game.MsgTypeReactionOpened, 2*time.Second) {
			t.Fatalf("the reaction never opened for the %s; they received: %v", who, messageTypes(c.snapshotMessages()))
		}
	}
	return reactionID
}

// The escape's destination is the opened move's WHERE: it reaches a third party only if they
// see where the reactor's piece would go, judged from where it stands now (it does not move at
// the opening). A reaction never carries move.from, so an "origin only" verdict leaves the
// category alone, exactly like "sees neither end".
func TestE2E_ReactionOpenedGatesTheEscapeDestinationByTheBystandersSight(t *testing.T) {
	tests := []struct {
		name    string
		at      [2]int
		to      [3]int
		visible bool
		walls   []mapentity.WallSegment
		// sees is whether the bystander gets move.position; master and owner always do.
		sees bool
	}{
		{
			name: "sees the destination: position",
			at:   [2]int{14, 4}, to: [3]int{16, 4, 0}, visible: true,
			walls: []mapentity.WallSegment{moveBoardWall},
			sees:  true,
		},
		{
			name: "sees neither end: no position",
			at:   [2]int{4, 4}, to: [3]int{6, 4, 0}, visible: true,
			walls: []mapentity.WallSegment{moveBoardWall},
		},
		{
			name: "sees only the origin: no position, and no from either",
			at:   [2]int{14, 4}, to: [3]int{14, 8, 0}, visible: true,
			walls: []mapentity.WallSegment{originOnlyWall},
		},
		{
			name: "a visible:false piece in plain sight: no position",
			at:   [2]int{14, 4}, to: [3]int{16, 4, 0}, visible: false,
			walls: []mapentity.WallSegment{moveBoardWall},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newCombatFixture(t, withBystander)
			f.seedReactorBoard(t, tt.at, tt.visible, tt.walls...)
			from := [3]int{tt.at[0], tt.at[1], 0}
			master, player, blind, mc, pc, bc := f.connectTable(t)
			defer master.Close() //nolint:errcheck
			defer player.Close() //nolint:errcheck
			defer blind.Close()  //nolint:errcheck

			f.openEscapeOnTable(t, master, player, mc, pc, bc, from, tt.to)

			move := func(c *collector) map[string]json.RawMessage {
				return rawMoveAt(t, findMessage(t, c.snapshotMessages(), game.MsgTypeReactionOpened).Payload,
					"reaction", "move")
			}
			// from is never on the wire: buildAction does not derive one for a reaction.
			assertMoveShape(t, "master", move(mc), moveShape{position: true}, from, tt.to)
			assertMoveShape(t, "owner", move(pc), moveShape{position: true}, from, tt.to)
			assertMoveShape(t, "bystander", move(bc), moveShape{position: tt.sees}, from, tt.to)
		})
	}
}

// reaction_opened and the resolution the opening recomputes leave the room on the same lane,
// from the same goroutine, in that order: the master's client can rely on knowing the reaction
// before the resolution that reads it.
func TestE2E_ReactionOpenedReachesTheMasterBeforeTheRecomputedResolution(t *testing.T) {
	f := newCombatFixture(t, withVictimPiece)
	f.seedBoard(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc, pc := collectFrom(master), collectFrom(player)
	if !mc.await(game.MsgTypeMapFullState, 2*time.Second) || !pc.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the table never got the board")
	}

	actionID := f.openAttackOn(t, player, master, mc)
	reactionID := f.attachDashEscape(t, player, mc, actionID, escapeFrom, escapeTo)

	resolved := mc.count(game.MsgTypeResolutionUpdate)
	sendWS(t, master, string(game.MsgTypeOpenReaction), game.OpenReactionPayload{ReactionID: reactionID})
	if !mc.await(game.MsgTypeReactionOpened, 2*time.Second) {
		t.Fatalf("the reaction never opened; the master received: %v", messageTypes(mc.snapshotMessages()))
	}
	if !awaitCount(mc, game.MsgTypeResolutionUpdate, resolved+1, 2*time.Second) {
		t.Fatal("no resolution_updated followed the opening")
	}

	openedAt, recomputedAt, seen := -1, -1, 0
	for i, m := range mc.snapshotMessages() {
		switch m.Type {
		case game.MsgTypeReactionOpened:
			if openedAt < 0 {
				openedAt = i
			}
		case game.MsgTypeResolutionUpdate:
			seen++
			if seen == resolved+1 {
				recomputedAt = i
			}
		}
	}
	if openedAt < 0 || recomputedAt < 0 || openedAt > recomputedAt {
		t.Fatalf("reaction_opened at %d, the recomputed resolution_updated at %d — the reaction must come first; got %v",
			openedAt, recomputedAt, messageTypes(mc.snapshotMessages()))
	}
}

// What each player saw of the escape's destination at the opening is held with the turn and
// written with it, so GET /history can show each reader the reaction's move as they saw it then.
func TestE2E_OpeningAReactionRecordsWhatEachPlayerSawOfItsDestination(t *testing.T) {
	f := newCombatFixture(t, withBystander)
	// The bystander sees the destination: the reactor and its destination are east of the wall.
	f.seedReactorBoard(t, [2]int{14, 4}, true, moveBoardWall)
	master, player, blind, mc, pc, bc := f.connectTable(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	defer blind.Close()  //nolint:errcheck

	reactionID := f.openEscapeOnTable(t, master, player, mc, pc, bc, [3]int{14, 4, 0}, [3]int{16, 4, 0})
	turnID, _, _, _ := reactionOpenedTo(t, "master", mc)

	sendWS(t, master, string(game.MsgTypeCloseTurn), game.CloseTurnPayload{Confirm: true})
	f.awaitPersistedTurn(t, turnID)

	var views map[uuid.UUID]map[uuid.UUID]masteraction.View
	found := false
	for _, d := range f.roundRepo.closeData() {
		if d.Turn.GetID() != turnID {
			continue
		}
		views, found = d.ReactionMoveViews, true
	}
	if !found {
		t.Fatalf("turn %s never reached PersistTurnClose", turnID)
	}
	got := views[reactionID]
	if got[f.bystanderUUID] != masteraction.ViewFull {
		t.Errorf("the bystander's view of the escape = %q, want %q (they saw the destination)", got[f.bystanderUUID], masteraction.ViewFull)
	}
	for who, id := range map[string]uuid.UUID{"master": f.masterUUID, "owner": f.playerUUID} {
		if v, ok := got[id]; ok {
			t.Errorf("the %s has an entry (%q): they always see all of it, and are never recorded", who, v)
		}
	}
}
