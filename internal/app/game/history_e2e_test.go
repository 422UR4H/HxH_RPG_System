package game_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchevent"
	"github.com/google/uuid"
)

// B15 (spec §4.5): the history keeps what is not a turn. A scene and a round are rows the
// moment they are born — rehydration, change_scene, a round closing by exhaustion — and a
// regime change is recorded in match_events the instant it is applied.
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

// Um round que fecha por exaustão fecha a sua linha, e o round que nasce no lugar dele vira
// linha na hora — antes de qualquer turno fechar nele.
func TestE2E_AnExhaustedRoundsSuccessorIsARowAtBirth(t *testing.T) {
	f := newCombatFixture(t)
	round1 := f.session.GetActiveRound().GetID()
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc, pc := collectFrom(master), collectFrom(player)

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
	turnID := lastTurnOpened(t, mc).TurnID
	sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
	if !mc.await(game.MsgTypeRoundClosed, 2*time.Second) {
		t.Fatal("the round ran out and nobody was told")
	}

	// The same open_next_action closed the turn AND the round: the turn belongs to the round it
	// closed in, not to the one that opened after it.
	if rd, ok := f.roundRepo.roundOfPersistedTurn(turnID); !ok || rd != round1 {
		t.Fatalf("the last turn of the exhausted round was persisted under round %s (ok=%v), want %s", rd, ok, round1)
	}

	if got := f.roundRepo.closedRoundIDs(); len(got) != 1 || got[0] != round1 {
		t.Fatalf("CloseRound calls = %v, want exactly the exhausted round %s — it is a row, so it closes", got, round1)
	}
	var successor uuid.UUID
	for _, id := range f.roundRepo.ensuredRoundIDs() {
		if id != round1 {
			successor = id
		}
	}
	if successor == uuid.Nil {
		t.Fatalf("the round born after the exhaustion was never made a row: ensured %v", f.roundRepo.ensuredRoundIDs())
	}
}
