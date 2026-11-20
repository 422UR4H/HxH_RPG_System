package game_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/clock"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Histórico de status (Task 14): toda mudança de efeito leva o seu evento, carimbado pela sala
// no instante — partida, cena, round, turno aberto, âncora, a lista resolvida, quem enxergava o
// personagem logo antes ou logo depois (mais o dono; o mestre nunca) e para quem ele estava
// Escondido — e gravado na mesma operação do efeito.

func newHistoryFixture(t *testing.T, store *fakeStatusEffectStore) *combatFixture {
	t.Helper()
	return newCombatFixture(t, withBystander, withNPCTarget, withStatusEffectStore(store))
}

// lastEvents são os eventos da última gravação bem-sucedida do store, na ordem.
func lastEvents(t *testing.T, store *fakeStatusEffectStore) []*statuseffect.Event {
	t.Helper()
	writes, _ := store.snapshot()
	if len(writes) == 0 {
		t.Fatal("the store got no write")
	}
	var out []*statuseffect.Event
	for _, w := range writes[len(writes)-1] {
		out = append(out, w.Event)
	}
	return out
}

func viewSet(ids ...uuid.UUID) map[uuid.UUID]masteraction.View {
	out := map[uuid.UUID]masteraction.View{}
	for _, id := range ids {
		out[id] = masteraction.ViewFull
	}
	return out
}

func assertEventViews(t *testing.T, who string, got map[uuid.UUID]masteraction.View, want ...uuid.UUID) {
	t.Helper()
	w := viewSet(want...)
	if got == nil || len(got) != len(w) {
		t.Fatalf("%s: views = %v, want %v", who, got, w)
	}
	for id, v := range w {
		if got[id] != v {
			t.Fatalf("%s: views = %v, want %v", who, got, w)
		}
	}
}

func applyAndWait(t *testing.T, master *websocket.Conn, mc *collector, payload map[string]any) game.StatusEffectsUpdatedPayload {
	t.Helper()
	before := mc.count(game.MsgTypeStatusEffectsUpdated)
	sendApplyStatus(t, master, payload)
	awaitStatusCount(t, "master", mc, before+1)
	return lastStatusUpdate(t, mc)
}

func TestStatusHistory_StampedOutsideTurn(t *testing.T) {
	store := &fakeStatusEffectStore{}
	f := newHistoryFixture(t, store)
	sceneID, roundID := f.session.GetActiveScene().GetID(), f.session.GetActiveRound().GetID()
	matchID := f.session.GetMatchUUID()
	f.seedStatusBoard(t, statusNPCInSight)
	master, player, blind, mc, _, _ := f.connectTable(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	defer blind.Close()  //nolint:errcheck

	before := time.Now().Add(-time.Second)
	applyProneAndWait(t, master, mc, f.npcID)
	evs := lastEvents(t, store)
	if len(evs) != 1 || evs[0] == nil {
		t.Fatalf("events = %v, want one", evs)
	}
	ev := evs[0]
	if ev.ID == uuid.Nil || ev.Change != statuseffect.ChangeApplied {
		t.Fatalf("id/change = %s/%s", ev.ID, ev.Change)
	}
	if ev.MatchID != matchID || ev.SceneID != sceneID || ev.RoundID != roundID {
		t.Fatalf("match/scene/round = %s/%s/%s, want %s/%s/%s", ev.MatchID, ev.SceneID, ev.RoundID, matchID, sceneID, roundID)
	}
	if ev.TurnID != nil || ev.ActionID != nil {
		t.Fatalf("turn/action = %v/%v, want nil outside a turn", ev.TurnID, ev.ActionID)
	}
	assertEventViews(t, "prone on the NPC in sight", ev.Views, f.playerUUID, f.bystanderUUID)
	if _, ok := ev.Views[f.masterUUID]; ok {
		t.Fatal("the master is in views")
	}
	if !ev.Listing.Everyone || ev.HiddenFrom != nil {
		t.Fatalf("listing/hiddenFrom = %+v/%v, want everyone/nil", ev.Listing, ev.HiddenFrom)
	}
	if ev.HappenedAt.Before(before) || !ev.HappenedAt.Equal(ev.HappenedAt.Truncate(time.Microsecond)) {
		t.Fatalf("happenedAt = %v, want now truncated to microseconds", ev.HappenedAt)
	}
}

func TestStatusHistory_UnconsciousBringsTwoEvents(t *testing.T) {
	store := &fakeStatusEffectStore{}
	f := newHistoryFixture(t, store)
	master, player, blind, mc, _, _ := f.connectNoBoard(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	defer blind.Close()  //nolint:errcheck

	applyAndWait(t, master, mc, map[string]any{"characterIds": []string{f.victimID.String()}, "kind": "unconscious"})
	evs := lastEvents(t, store)
	if len(evs) != 2 || evs[0] == nil || evs[1] == nil || evs[0].ID == evs[1].ID {
		t.Fatalf("events = %+v, want two distinct", evs)
	}
	for _, ev := range evs {
		if ev.Change != statuseffect.ChangeApplied {
			t.Fatalf("change = %s, want applied", ev.Change)
		}
	}
}

func TestStatusHistory_ViewsFollowSight(t *testing.T) {
	store := &fakeStatusEffectStore{}
	f := newHistoryFixture(t, store)
	f.seedStatusBoard(t, statusNPCBehindWall, moveBoardWall)
	master, player, blind, mc, _, _ := f.connectTable(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	defer blind.Close()  //nolint:errcheck

	applyProneAndWait(t, master, mc, f.npcID)
	assertEventViews(t, "NPC behind the wall", lastEvents(t, store)[0].Views)

	applyAndWait(t, master, mc, map[string]any{"characterIds": []string{f.victimID.String()}, "kind": "poisoned"})
	ev := lastEvents(t, store)[0]
	assertEventViews(t, "poisoned on the PC", ev.Views, f.playerUUID, f.bystanderUUID)
	if ev.Listing.Everyone || !slices.Equal(ev.Listing.Players, []uuid.UUID{f.playerUUID}) {
		t.Fatalf("listing = %+v, want [owner]", ev.Listing)
	}
}

func TestStatusHistory_HiddenKeepsWhoWasLooking(t *testing.T) {
	store := &fakeStatusEffectStore{}
	f := newHistoryFixture(t, store)
	f.seedStatusBoard(t, statusNPCInSight)
	master, player, blind, mc, _, _ := f.connectTable(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	defer blind.Close()  //nolint:errcheck

	hidden := applyAndWait(t, master, mc, map[string]any{"characterIds": []string{f.npcID.String()}, "kind": "hidden"})
	ev := lastEvents(t, store)[0]
	// Os dois viam o NPC logo antes: entram, e por isso não estão em HiddenFrom.
	assertEventViews(t, "hidden applied", ev.Views, f.playerUUID, f.bystanderUUID)
	if ev.HiddenFrom != nil {
		t.Fatalf("hiddenFrom = %v, want nil — whoever saw it is in views", ev.HiddenFrom)
	}
	if ev.Listing.Everyone || len(ev.Listing.Players) != 0 {
		t.Fatalf("listing = %+v, want master only (hidden on an NPC)", ev.Listing)
	}

	// Escondido para os dois agora: o próximo evento não tem ninguém em views e os dois em
	// HiddenFrom, com o id do Escondido.
	hid := hidden.StatusEffects[0].ID
	applyProneAndWait(t, master, mc, f.npcID)
	ev = lastEvents(t, store)[0]
	assertEventViews(t, "prone while hidden", ev.Views)
	want := map[uuid.UUID]uuid.UUID{f.playerUUID: hid, f.bystanderUUID: hid}
	if len(ev.HiddenFrom) != 2 || ev.HiddenFrom[f.playerUUID] != hid || ev.HiddenFrom[f.bystanderUUID] != hid {
		t.Fatalf("hiddenFrom = %v, want %v", ev.HiddenFrom, want)
	}

	// Remover o Escondido: os dois voltam a ver logo depois — entram em views.
	sendWS(t, master, string(game.MsgTypeRemoveStatusEffect), map[string]any{"statusEffectId": hid})
	awaitStatusCount(t, "master", mc, mc.count(game.MsgTypeStatusEffectsUpdated)+1)
	writes, _ := store.snapshot()
	last := writes[len(writes)-1]
	if len(last) != 1 || last[0].Event == nil || last[0].Event.Change != statuseffect.ChangeRemoved {
		t.Fatalf("last write = %+v, want the hidden's removal", last)
	}
	assertEventViews(t, "hidden removed", last[0].Event.Views, f.playerUUID, f.bystanderUUID)
}

func TestStatusHistory_InsideTurnCarriesTurn(t *testing.T) {
	store := &fakeStatusEffectStore{}
	f := newHistoryFixture(t, store)
	master, player, blind, mc, _, _ := f.connectNoBoard(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	defer blind.Close()  //nolint:errcheck
	turnID := f.openAttackTurn(t, master, player, mc)
	actionID := lastTurnOpened(t, mc).ActionID
	var seatedID uuid.UUID
	for _, pr := range lastResolutionUpdated(t, mc).PendingReactions {
		if pr.ActorID == f.victimID && pr.Seated {
			seatedID = pr.ReactionID
		}
	}
	if seatedID == uuid.Nil {
		t.Fatal("no seated reaction for the victim")
	}

	applyAndWait(t, master, mc, map[string]any{
		"characterIds": []string{f.bystanderID.String()}, "kind": "prone", "actionId": actionID})
	applyAndWait(t, master, mc, map[string]any{
		"characterIds": []string{f.npcID.String()}, "kind": "prone", "actionId": seatedID})
	applyProneAndWait(t, master, mc, f.attackerID)
	if writes, attempts := store.snapshot(); len(writes) != 0 || attempts != 0 {
		t.Fatalf("store got %d write(s) (%d attempt(s)) with the turn open, want none", len(writes), attempts)
	}

	closeTurnAndWait(t, master, mc)
	f.awaitPersistedTurn(t, turnID)
	ops := lastCloseStatusOps(t, f)
	if len(ops) != 3 {
		t.Fatalf("close ops = %d, want 3", len(ops))
	}
	anchors := []uuid.UUID{actionID, seatedID, uuid.Nil}
	for i, w := range ops {
		if w.Event == nil || w.Event.TurnID == nil || *w.Event.TurnID != turnID {
			t.Fatalf("op %d: event = %+v, want turn %s", i, w.Event, turnID)
		}
		got := uuid.Nil
		if w.Event.ActionID != nil {
			got = *w.Event.ActionID
		}
		if got != anchors[i] {
			t.Fatalf("op %d: actionId = %s, want %s", i, got, anchors[i])
		}
	}
}

func TestStatusHistory_ActionIDMustBelongToTheOpenTurn(t *testing.T) {
	store := &fakeStatusEffectStore{}
	f := newHistoryFixture(t, store)
	master, player, blind, mc, pc, bc := f.connectNoBoard(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	defer blind.Close()  //nolint:errcheck

	refuse := func(label string, payload map[string]any, verb game.MessageType) {
		t.Helper()
		before := len(errorCodes(t, mc))
		sendWS(t, master, string(verb), payload)
		chatBarrier(t, master, mc, pc, bc)
		codes := errorCodes(t, mc)
		if len(codes) != before+1 || codes[len(codes)-1] != "status_effect_invalid" {
			t.Fatalf("%s: errors = %v, want a new status_effect_invalid", label, codes)
		}
	}

	refuse("no open turn", map[string]any{
		"characterIds": []string{f.victimID.String()}, "kind": "prone", "actionId": uuid.New()}, game.MsgTypeApplyStatusEffect)
	assertStatusCount(t, "master", mc, 0)

	applied := applyProneAndWait(t, master, mc, f.victimID)
	id := applied.StatusEffects[0].ID
	f.openAttackTurn(t, master, player, mc)
	refuse("unknown action on edit", map[string]any{
		"statusEffectId": id, "alias": "x", "actionId": uuid.New()}, game.MsgTypeEditStatusEffect)
	refuse("unknown action on remove", map[string]any{
		"statusEffectId": id, "actionId": uuid.New()}, game.MsgTypeRemoveStatusEffect)
	if got := lastStatusUpdate(t, mc).StatusEffects; len(got) != 1 || got[0].Alias != "" {
		t.Fatalf("a refused command changed the effect: %+v", got)
	}
}

func TestStatusHistory_FailedWriteKeepsItsEvents(t *testing.T) {
	store := &fakeStatusEffectStore{}
	f := newHistoryFixture(t, store)
	master, player, blind, mc, _, _ := f.connectNoBoard(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	defer blind.Close()  //nolint:errcheck

	store.setFail(errors.New("connection reset"))
	applyProneAndWait(t, master, mc, f.victimID)
	store.setFail(nil)
	applyProneAndWait(t, master, mc, f.bystanderID)

	tried := store.triedWrites()
	if len(tried) != 2 || len(tried[0]) != 1 || tried[0][0].Event == nil {
		t.Fatalf("tried = %+v, want the failed write then the retry", tried)
	}
	failedID := tried[0][0].Event.ID
	writes, _ := store.snapshot()
	var seen int
	for _, w := range writes {
		for _, op := range w {
			if op.Event != nil && op.Event.ID == failedID {
				seen++
			}
		}
	}
	if seen != 1 {
		t.Fatalf("the failed write's event was written %d time(s), want once with the same id", seen)
	}
	if writes[0][0].Event.ID != failedID {
		t.Fatalf("retry event id = %s, want %s", writes[0][0].Event.ID, failedID)
	}
}

func TestStatusHistory_EndMatchCarriesEvents(t *testing.T) {
	store := &fakeStatusEffectStore{}
	now := &fakeNow{t: time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)}
	ends := &fakeMatchEndRepo{}
	f := newCombatFixture(t, withRoomDeps(func(d *game.RoomDeps) {
		d.StatusEffectsUC = appmatch.NewStatusEffectsUC()
		d.StatusEffectRepo = store
		d.ClockUC = appmatch.NewClockCommandsUC(fakeMatchGetter{storyStart: time.Date(1999, 4, 1, 0, 0, 0, 0, time.UTC)})
		d.ClockRepo = &fakeClockRepo{}
		d.Now = now.Now
		d.MatchEndRepo = ends
	}))
	f.session.SetClock(clock.NewAt(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)))
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc := collectFrom(master)
	if !mc.await(game.MsgTypeMatchFullState, 2*time.Second) {
		t.Fatal("the master never got match_full_state")
	}

	store.setFail(errors.New("connection reset"))
	applyProneAndWait(t, master, mc, f.victimID)
	sendWS(t, master, string(game.MsgTypeEndMatch), map[string]any{"storyAt": "2000-01-01T05:00:00"})
	awaitMatchEnded(t, mc)
	ws := ends.writes()
	if len(ws) != 1 || len(ws[0].StatusEffects) != 1 {
		t.Fatalf("PersistMatchEnd = %+v, want one call with the pending write", ws)
	}
	ev := ws[0].StatusEffects[0].Event
	if ev == nil || ev.ID == uuid.Nil || ev.Change != statuseffect.ChangeApplied || !ev.HappenedAt.Equal(now.t) {
		t.Fatalf("event = %+v, want the applied event stamped at %v", ev, now.t)
	}
}

// Um fechamento que falha devolve as gravações de status à fila com o evento intacto (o turno
// dele inclusive); o próximo fechamento as grava uma vez só.
func TestStatusHistory_FailedCloseKeepsTheEventsTurn(t *testing.T) {
	store := &fakeStatusEffectStore{}
	f := newHistoryFixture(t, store)
	master, player, blind, mc, _, _ := f.connectNoBoard(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	defer blind.Close()  //nolint:errcheck

	first := f.openAttackTurn(t, master, player, mc)
	applyProneAndWait(t, master, mc, f.bystanderID)
	f.roundRepo.setFailPersist(errors.New("connection reset"))
	closeTurnAndWait(t, master, mc)
	f.roundRepo.setFailPersist(nil)

	before := mc.count(game.MsgTypeTurnOpened)
	f.enqueueAttack(t, player)
	if !awaitCount(mc, game.MsgTypeActionQueued, 2, 2*time.Second) {
		t.Fatal("the second attack was never queued")
	}
	sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
	if !awaitCount(mc, game.MsgTypeTurnOpened, before+1, 2*time.Second) {
		t.Fatal("no second turn opened")
	}
	second := lastTurnOpened(t, mc).TurnID
	before = mc.count(game.MsgTypeTurnClosed)
	sendWS(t, master, string(game.MsgTypeCloseTurn), game.CloseTurnPayload{Confirm: true})
	if !awaitCount(mc, game.MsgTypeTurnClosed, before+1, 3*time.Second) {
		t.Fatal("the second turn never closed")
	}
	f.awaitPersistedTurn(t, second)

	ops := lastCloseStatusOps(t, f)
	if len(ops) != 1 || ops[0].Event == nil || ops[0].Event.TurnID == nil || *ops[0].Event.TurnID != first {
		t.Fatalf("second close ops = %+v, want the first turn's event, its TurnID kept", ops)
	}
	var seen int
	for _, cd := range f.roundRepo.closeData() {
		for _, w := range cd.StatusEffects {
			if w.Event != nil && w.Event.ID == ops[0].Event.ID {
				seen++
			}
		}
	}
	if writes, _ := store.snapshot(); seen != 1 || len(writes) != 0 {
		t.Fatalf("the event was written %d time(s) by closes and %d by the store, want once by the close", seen, len(writes))
	}
}

// A âncora pode ser uma reação anexada (ainda não aberta) ou uma já aberta.
func TestStatusHistory_AnchorsOnAttachedAndOpenedReactions(t *testing.T) {
	store := &fakeStatusEffectStore{}
	f := newHistoryFixture(t, store)
	master, player, blind, mc, _, _ := f.connectNoBoard(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	defer blind.Close()  //nolint:errcheck

	f.enqueueAttackAt(t, player, f.attackerID, f.npcID)
	if !mc.await(game.MsgTypeActionQueued, 2*time.Second) {
		t.Fatal("the attack was never queued")
	}
	sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
	if !mc.await(game.MsgTypeTurnOpened, 2*time.Second) {
		t.Fatal("no turn opened")
	}
	turnID, actionID := lastTurnOpened(t, mc).TurnID, lastTurnOpened(t, mc).ActionID
	sendWS(t, master, string(game.MsgTypeAttachReaction), game.ActionPayload{
		ActorID: f.npcID, ReactToID: actionID, ReactionKind: "dodge", Dodge: &game.DodgePayload{},
	})
	if !mc.await(game.MsgTypeReactionAttached, 2*time.Second) {
		t.Fatal("no reaction_attached")
	}
	reactionID := lastReactionAttached(t, mc).ReactionID

	applyAndWait(t, master, mc, map[string]any{
		"characterIds": []string{f.victimID.String()}, "kind": "prone", "actionId": reactionID})
	sendWS(t, master, string(game.MsgTypeOpenReaction), game.OpenReactionPayload{ReactionID: reactionID})
	if !mc.await(game.MsgTypeReactionOpened, 2*time.Second) {
		t.Fatal("no reaction_opened")
	}
	applyAndWait(t, master, mc, map[string]any{
		"characterIds": []string{f.bystanderID.String()}, "kind": "prone", "actionId": reactionID})
	if codes := errorCodes(t, mc); len(codes) != 0 {
		t.Fatalf("errors = %v, want none", codes)
	}

	closeTurnAndWait(t, master, mc)
	f.awaitPersistedTurn(t, turnID)
	ops := lastCloseStatusOps(t, f)
	if len(ops) != 2 {
		t.Fatalf("close ops = %d, want 2", len(ops))
	}
	for i, w := range ops {
		if w.Event == nil || w.Event.ActionID == nil || *w.Event.ActionID != reactionID {
			t.Fatalf("op %d: event = %+v, want anchored on the reaction %s", i, w.Event, reactionID)
		}
	}
}
