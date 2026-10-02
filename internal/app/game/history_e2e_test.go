package game_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	roundentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchevent"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// B15 (spec §4.5): the history keeps what is not a turn. A scene and a round are rows the
// moment they are born — rehydration, change_scene, a round ending because no action can still
// pay its price — and a regime change is recorded in match_events the instant it is applied.
//
// Driven over real sockets against a real Room and session; the only fakes are the stores.

// fakeEventStore stands in for pgmatchevent.Repository: it keeps every Event it was handed.
type fakeEventStore struct {
	mu     sync.Mutex
	events []matchevent.Event
}

func (s *fakeEventStore) Insert(_ context.Context, e matchevent.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, e)
	return nil
}

func (s *fakeEventStore) snapshot() []matchevent.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]matchevent.Event(nil), s.events...)
}

func contains(ids []uuid.UUID, id uuid.UUID) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

func lastSceneChanged(t *testing.T, c *collector) game.SceneChangedPayload {
	t.Helper()
	msgs := c.snapshotMessages()
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Type == game.MsgTypeSceneChanged {
			var p game.SceneChangedPayload
			if err := json.Unmarshal(msgs[i].Payload, &p); err != nil {
				t.Fatalf("unmarshal scene_changed: %v", err)
			}
			return p
		}
	}
	t.Fatal("no scene_changed received")
	return game.SceneChangedPayload{}
}

// O mestre troca o regime e troca a cena sem que nenhum turno feche: as duas cenas (e seus
// rounds) viraram linhas quando nasceram, a troca de regime foi gravada com o de-para, a cena
// velha foi fechada no banco — e o arrastar entre turnos que vem depois é gravado sem turno, na
// cena nova.
func TestE2E_TheHistoryKeepsWhatIsNotATurn(t *testing.T) {
	f := newCombatFixture(t)
	f.seedBoard(t)
	// Read before anyone connects: the session is the fixture's, and nothing runs on it yet.
	scene1, round1 := f.session.GetActiveScene().GetID(), f.session.GetActiveRound().GetID()

	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc := collectFrom(master)

	// The rehydration made the active pair rows — no turn has closed yet.
	if !contains(f.roundRepo.ensuredSceneIDs(), scene1) || !contains(f.roundRepo.ensuredRoundIDs(), round1) {
		t.Fatalf("after rehydration: ensured scenes %v / rounds %v, want scene %s / round %s",
			f.roundRepo.ensuredSceneIDs(), f.roundRepo.ensuredRoundIDs(), scene1, round1)
	}

	sendWS(t, master, string(game.MsgTypeChangeRoundMode), game.ChangeRoundModePayload{Mode: string(enum.Race)})
	if !mc.await(game.MsgTypeRoundModeChanged, 2*time.Second) {
		t.Fatal("the regime switch was never announced")
	}
	evs := f.events.snapshot()
	if len(evs) != 1 {
		t.Fatalf("events = %d, want 1 roundModeChanged", len(evs))
	}
	ev := evs[0]
	if ev.Kind != matchevent.KindRoundModeChanged || ev.MatchUUID != f.matchUUID ||
		ev.SceneUUID != scene1 || ev.RoundUUID != round1 || ev.UUID == uuid.Nil || ev.CreatedAt.IsZero() {
		t.Fatalf("event = %+v, want roundModeChanged of round %s / scene %s", ev, round1, scene1)
	}
	var payload map[string]string
	if err := json.Unmarshal(ev.Payload, &payload); err != nil {
		t.Fatalf("payload %s: %v", ev.Payload, err)
	}
	if payload["from"] != string(enum.Free) || payload["to"] != string(enum.Race) || len(payload) != 2 {
		t.Fatalf("payload = %s, want {from: Free, to: Race}", ev.Payload)
	}

	// Switching to the regime the round is already in is not a change.
	sendWS(t, master, string(game.MsgTypeChangeRoundMode), game.ChangeRoundModePayload{Mode: string(enum.Race)})
	if !awaitCount(mc, game.MsgTypeRoundModeChanged, 2, 2*time.Second) {
		t.Fatal("the second switch was never answered")
	}
	if n := len(f.events.snapshot()); n != 1 {
		t.Fatalf("events = %d after a Race -> Race switch, want still 1", n)
	}

	sendWS(t, master, string(game.MsgTypeChangeScene), game.ChangeScenePayload{
		Category: string(enum.Roleplay), BriefInitialDescription: "Taverna",
	})
	if !mc.await(game.MsgTypeSceneChanged, 2*time.Second) {
		t.Fatal("scene_changed never arrived")
	}
	scene2 := lastSceneChanged(t, mc).SceneID
	if !contains(f.roundRepo.ensuredSceneIDs(), scene2) {
		t.Fatalf("the new scene %s was not made a row when it was born: ensured %v", scene2, f.roundRepo.ensuredSceneIDs())
	}
	closed := f.roundRepo.closedScenePairs()
	if len(closed) != 1 || closed[0] != [2]uuid.UUID{scene1, round1} {
		t.Fatalf("CloseSceneAndRound calls = %v, want exactly the old scene %s / round %s — they are rows, so they close", closed, scene1, round1)
	}

	// A drag between turns, in the new scene: recorded with no turn, on the new scene's round.
	sendMasterMove(t, master, f.attackerID, [3]int{6, 4, 0})
	recs := f.masterActions.await(t, 1, 2*time.Second)
	rec := recs[0]
	if rec.Kind != masteraction.KindMovePiece || rec.TurnUUID != nil {
		t.Fatalf("record = %s turn %v, want a movePiece with no turn", rec.Kind, rec.TurnUUID)
	}
	if rec.SceneUUID != scene2 {
		t.Fatalf("record scene = %s, want the new scene %s", rec.SceneUUID, scene2)
	}
	rounds := f.roundRepo.ensuredRoundIDs()
	if rec.RoundUUID == round1 || !contains(rounds, rec.RoundUUID) {
		t.Fatalf("record round = %s, want the new scene's round, ensured when born (ensured: %v)", rec.RoundUUID, rounds)
	}
}

// Carry-over of T5: a master action with no turn closed yet, then change_scene — the scene and
// round the master action already references are rows, so change_scene closes them.
func TestE2E_AChangeSceneAfterAMasterActionClosesTheRows(t *testing.T) {
	f := newCombatFixture(t)
	f.seedBoard(t)
	scene1, round1 := f.session.GetActiveScene().GetID(), f.session.GetActiveRound().GetID()
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc := collectFrom(master)

	sendMasterMove(t, master, f.attackerID, [3]int{6, 4, 0})
	if rec := f.masterActions.await(t, 1, 2*time.Second)[0]; rec.RoundUUID != round1 {
		t.Fatalf("record round = %s, want %s", rec.RoundUUID, round1)
	}

	sendWS(t, master, string(game.MsgTypeChangeScene), game.ChangeScenePayload{
		Category: string(enum.Battle), BriefInitialDescription: "Arena",
	})
	if !mc.await(game.MsgTypeSceneChanged, 2*time.Second) {
		t.Fatal("scene_changed never arrived")
	}
	closed := f.roundRepo.closedScenePairs()
	if len(closed) != 1 || closed[0] != [2]uuid.UUID{scene1, round1} {
		t.Fatalf("CloseSceneAndRound calls = %v, want the scene %s / round %s the master action points at", closed, scene1, round1)
	}
}

// raceRoundWithOneOpenTurn switches the round to Race, enqueues one attack and opens it: the
// next open_next_action closes that turn and finds nothing that can still pay its price, so it
// ends the round in the same command. Returns the round and the open turn.
func raceRoundWithOneOpenTurn(t *testing.T, f *combatFixture, master, player *websocket.Conn, mc, pc *collector) (uuid.UUID, uuid.UUID) {
	t.Helper()
	round1 := f.session.GetActiveRound().GetID() // read before anything is in flight
	sendWS(t, master, string(game.MsgTypeChangeRoundMode), game.ChangeRoundModePayload{Mode: string(enum.Race)})
	if !mc.await(game.MsgTypeRoundModeChanged, 2*time.Second) {
		t.Fatal("the regime switch was never announced")
	}
	f.enqueueAttack(t, player)
	if !pc.await(game.MsgTypeActionEnqueued, 2*time.Second) {
		t.Fatal("the action was never acknowledged as enqueued")
	}
	sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
	if !mc.await(game.MsgTypeTurnOpened, 2*time.Second) {
		t.Fatal("the first action never opened")
	}
	return round1, lastTurnOpened(t, mc).TurnID
}

// Um open_next_action que fecha o último turno e acaba o round grava tudo numa transação só
// (dono do produto, 2026-10-02): o turno, o fim do round e o round que nasce no lugar dele — um
// PersistTurnClose, e nada gravado à parte.
func TestE2E_TheRoundEndGoesInTheLastTurnsTransaction(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc, pc := collectFrom(master), collectFrom(player)

	round1, turnID := raceRoundWithOneOpenTurn(t, f, master, player, mc, pc)
	sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
	if !mc.await(game.MsgTypeRoundClosed, 2*time.Second) {
		t.Fatal("the round ended and nobody was told")
	}

	closes := f.roundRepo.closeData()
	if len(closes) != 1 || closes[0].Turn.GetID() != turnID {
		t.Fatalf("PersistTurnClose calls = %d, want exactly one, for the last turn %s", len(closes), turnID)
	}
	d := closes[0]
	// The turn belongs to the round it closed in, not to the one that opened after it.
	if d.Round.GetID() != round1 || d.Round.GetFinishedAt() == nil {
		t.Fatalf("the turn's round = %s finished %v, want the ended round %s with its finish", d.Round.GetID(), d.Round.GetFinishedAt(), round1)
	}
	if d.NextRound == nil || d.NextRound.GetID() == round1 || d.NextRound.GetFinishedAt() != nil {
		t.Fatalf("NextRound = %+v, want the open round born in place of %s", d.NextRound, round1)
	}
	if calls := f.roundRepo.roundCloseCalls(); len(calls) != 0 {
		t.Fatalf("PersistRoundClose calls = %d, want 0 — the turn's transaction carries the round's end", len(calls))
	}
	if contains(f.roundRepo.ensuredRoundIDs(), d.NextRound.GetID()) {
		t.Fatal("the successor was also written on its own, outside the turn's transaction")
	}
}

// Um round que acaba sem turno fechado no mesmo comando (nada aberto, nada na fila que pague o
// preço) grava o seu fim e o round seguinte numa transação deles — nunca um sem o outro.
func TestE2E_ARoundThatEndsWithNoTurnWritesItsEndAndSuccessorTogether(t *testing.T) {
	f := newCombatFixture(t)
	round1 := f.session.GetActiveRound().GetID()
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc := collectFrom(master)

	sendWS(t, master, string(game.MsgTypeChangeRoundMode), game.ChangeRoundModePayload{Mode: string(enum.Race)})
	if !mc.await(game.MsgTypeRoundModeChanged, 2*time.Second) {
		t.Fatal("the regime switch was never announced")
	}
	// Nothing open, nothing queued: no action can pay, so the round ends with no turn closing.
	sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
	if !mc.await(game.MsgTypeRoundClosed, 2*time.Second) {
		t.Fatal("the round ended and nobody was told")
	}

	calls := f.roundRepo.roundCloseCalls()
	if len(calls) != 1 {
		t.Fatalf("PersistRoundClose calls = %d, want exactly 1", len(calls))
	}
	c := calls[0]
	if c.closed.GetID() != round1 || c.closed.GetFinishedAt() == nil {
		t.Fatalf("closed = %s finished %v, want %s with its finish", c.closed.GetID(), c.closed.GetFinishedAt(), round1)
	}
	if c.next == nil || c.next.GetID() == round1 || c.next.GetFinishedAt() != nil {
		t.Fatalf("next = %+v, want the open round born in place of %s", c.next, round1)
	}
	if n := len(f.roundRepo.persistedTurnIDs()); n != 0 {
		t.Fatalf("PersistTurnClose calls = %d, want 0 — no turn closed", n)
	}
	if contains(f.roundRepo.ensuredRoundIDs(), c.next.GetID()) {
		t.Fatal("the successor was also written on its own, outside the round's transaction")
	}
}

// Se a transação do último turno falha, o turno se perde (logado) — mas o round acabou na mesa, e
// o seu fim e o round seguinte ainda vão juntos, numa transação deles: senão o banco ficaria com o
// round velho aberto e, assim que o novo virasse linha, com dois rounds abertos na cena.
func TestE2E_AFailedLastTurnStillWritesTheRoundEndWithItsSuccessor(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc, pc := collectFrom(master), collectFrom(player)

	round1, turnID := raceRoundWithOneOpenTurn(t, f, master, player, mc, pc)
	f.roundRepo.setFailPersist(errors.New("the transaction rolled back"))
	sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
	if !mc.await(game.MsgTypeRoundClosed, 2*time.Second) {
		t.Fatal("the round ended and nobody was told")
	}

	if contains(f.roundRepo.persistedTurnIDs(), turnID) {
		t.Fatal("the failing repository reports the turn persisted — the fake is wrong")
	}
	calls := f.roundRepo.roundCloseCalls()
	if len(calls) != 1 || calls[0].closed.GetID() != round1 || calls[0].closed.GetFinishedAt() == nil ||
		calls[0].next == nil || calls[0].next.GetID() == round1 {
		t.Fatalf("PersistRoundClose calls = %+v, want one: %s ended, with its successor", calls, round1)
	}
}

// F4: the scene/round writes run after r.mu is released, so the gateway must be handed a COPY
// taken under the lock — a live *Scene/*Round read there races the next mutation of the session
// (a regime switch, a round ending because no action can pay). Every pointer the round repository ever
// received is checked against the session's live objects: the pair the session started on
// (rehydration, a closed turn, a regime switch) and the pair change_scene installed.
func TestE2E_TheRoundRepositoryGetsACopyOfTheSceneAndRound(t *testing.T) {
	f := newCombatFixture(t)
	f.seedBoard(t)
	// Read before anyone connects: the session is the fixture's, and nothing runs on it yet.
	liveScene, liveRound := f.session.GetActiveScene(), f.session.GetActiveRound()

	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc := collectFrom(master)

	turnID := f.openAttackTurn(t, master, player, mc)
	sendWS(t, master, string(game.MsgTypeCloseTurn), game.CloseTurnPayload{Confirm: true})
	deadline := time.Now().Add(2 * time.Second)
	for !contains(f.roundRepo.persistedTurnIDs(), turnID) {
		if time.Now().After(deadline) {
			t.Fatal("the closed turn was never persisted")
		}
		time.Sleep(10 * time.Millisecond)
	}

	sendWS(t, master, string(game.MsgTypeChangeRoundMode), game.ChangeRoundModePayload{Mode: string(enum.Race)})
	if !mc.await(game.MsgTypeRoundModeChanged, 2*time.Second) {
		t.Fatal("the regime switch was never announced")
	}
	sendWS(t, master, string(game.MsgTypeChangeScene), game.ChangeScenePayload{
		Category: string(enum.Roleplay), BriefInitialDescription: "Taverna",
	})
	if !mc.await(game.MsgTypeSceneChanged, 2*time.Second) {
		t.Fatal("scene_changed never arrived")
	}
	newScene := lastSceneChanged(t, mc).SceneID
	deadline = time.Now().Add(2 * time.Second)
	for !contains(f.roundRepo.ensuredSceneIDs(), newScene) {
		if time.Now().After(deadline) {
			t.Fatal("the new scene was never ensured")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Safe to read: the room wrote the session, then called the repository, whose mutex the
	// read above already went through.
	live := map[any]string{
		liveScene: "the starting scene", liveRound: "the starting round",
		f.session.GetActiveScene(): "change_scene's new scene", f.session.GetActiveRound(): "change_scene's new round",
	}
	sawRaceCopy := false
	for _, p := range f.roundRepo.handedPointers() {
		if what, ok := live[p]; ok {
			t.Errorf("the round repository was handed the session's live object (%s), not a copy", what)
		}
		if rd, ok := p.(*roundentity.Round); ok && rd.GetID() == liveRound.GetID() && rd.GetMode() == enum.Race {
			sawRaceCopy = true
		}
	}
	if !sawRaceCopy {
		t.Error("no copy of the starting round carried the regime it was switched to — the copy " +
			"must be taken AFTER the switch, under the same lock")
	}
}
