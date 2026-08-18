package matchsession_test

import (
	"errors"
	"testing"

	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

// fakePieceSource is a test double for matchsession.PiecePositionSource.
type fakePieceSource struct {
	positions map[uuid.UUID][]service.Point2D
}

func (f *fakePieceSource) PlayerPiecePositions(playerID uuid.UUID) []service.Point2D {
	return f.positions[playerID]
}

func (f *fakePieceSource) setPosition(playerID uuid.UUID, pts ...service.Point2D) {
	if f.positions == nil {
		f.positions = make(map[uuid.UUID][]service.Point2D)
	}
	f.positions[playerID] = pts
}

func TestNewMatchSession(t *testing.T) {
	matchUUID := uuid.New()
	playerUUID := uuid.New()

	participant := makeParticipant(matchUUID, &playerUUID)
	sheet := &csSheet.CharacterSheet{}
	sheets := map[uuid.UUID]*csSheet.CharacterSheet{participant.Sheet.UUID: sheet}

	s := matchsession.NewMatchSession(matchUUID, sheets, []*match.Participant{participant})

	if s == nil {
		t.Fatal("expected non-nil MatchSession")
	}
	if s.GetActiveRound() == nil {
		t.Error("expected non-nil activeRound on new session")
	}
	if s.GetActiveRound().GetMode() != enum.Free {
		t.Error("expected initial round mode to be Free")
	}
}

func TestMatchSession_GetCharSheet(t *testing.T) {
	matchUUID := uuid.New()
	playerUUID := uuid.New()
	participant := makeParticipant(matchUUID, &playerUUID)
	sheetUUID := participant.Sheet.UUID
	sheet := &csSheet.CharacterSheet{}
	// Keyed by sheet UUID: the combat entity is the character, not the player.
	sheets := map[uuid.UUID]*csSheet.CharacterSheet{sheetUUID: sheet}
	s := matchsession.NewMatchSession(matchUUID, sheets, []*match.Participant{participant})

	t.Run("returns the sheet for a known character", func(t *testing.T) {
		got, err := s.GetCharSheet(sheetUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != sheet {
			t.Error("expected the same sheet pointer")
		}
	})

	t.Run("the player UUID is no longer a valid key", func(t *testing.T) {
		if _, err := s.GetCharSheet(playerUUID); !errors.Is(err, matchsession.ErrCharSheetNotFound) {
			t.Errorf("expected ErrCharSheetNotFound, got %v", err)
		}
	})

	t.Run("returns ErrCharSheetNotFound for an unknown character", func(t *testing.T) {
		if _, err := s.GetCharSheet(uuid.New()); !errors.Is(err, matchsession.ErrCharSheetNotFound) {
			t.Errorf("expected ErrCharSheetNotFound, got %v", err)
		}
	})
}

func TestNewMatchSession_HoldsNPCs(t *testing.T) {
	matchUUID := uuid.New()
	// An NPC: PlayerUUID nil, MasterUUID set. It used to be dropped silently.
	npc := makeParticipant(matchUUID, nil)
	masterUUID := uuid.New()
	npc.Sheet.MasterUUID = &masterUUID
	npcSheetUUID := npc.Sheet.UUID

	npcSheet := &csSheet.CharacterSheet{}
	sheets := map[uuid.UUID]*csSheet.CharacterSheet{npcSheetUUID: npcSheet}

	s := matchsession.NewMatchSession(matchUUID, sheets, []*match.Participant{npc})

	t.Run("the NPC sheet is reachable by sheet UUID", func(t *testing.T) {
		got, err := s.GetCharSheet(npcSheetUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != npcSheet {
			t.Error("expected the NPC sheet pointer")
		}
	})

	t.Run("the NPC has a CharacterStatus", func(t *testing.T) {
		st, err := s.GetCharacterStatus(npcSheetUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if st == nil {
			t.Fatal("expected a non-nil status")
		}
		if st.Stance != match.StanceNone {
			t.Errorf("expected StanceNone, got %q", st.Stance)
		}
	})

	t.Run("the NPC has no authorization entry", func(t *testing.T) {
		// Authorization stays per player; an NPC has no player to authorize.
		if got := len(s.PlayerIDs()); got != 0 {
			t.Errorf("expected no player IDs, got %d", got)
		}
		if got := len(s.GetCharToPlayer()); got != 0 {
			t.Errorf("expected no charToPlayer entries, got %d", got)
		}
	})
}

func TestMatchSession_GetCharacterStatus(t *testing.T) {
	matchUUID := uuid.New()
	playerUUID := uuid.New()
	participant := makeParticipant(matchUUID, &playerUUID)
	s := matchsession.NewMatchSession(matchUUID, nil, []*match.Participant{participant})

	t.Run("every participant gets a status", func(t *testing.T) {
		if _, err := s.GetCharacterStatus(participant.Sheet.UUID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns ErrCharacterStatusNotFound for an unknown character", func(t *testing.T) {
		if _, err := s.GetCharacterStatus(uuid.New()); !errors.Is(err, matchsession.ErrCharacterStatusNotFound) {
			t.Errorf("expected ErrCharacterStatusNotFound, got %v", err)
		}
	})

	t.Run("the status is mutable through the session", func(t *testing.T) {
		st, _ := s.GetCharacterStatus(participant.Sheet.UUID)
		st.ActionBar.RecordSpeed(20)

		again, _ := s.GetCharacterStatus(participant.Sheet.UUID)
		if len(again.ActionBar.Speeds) != 1 {
			t.Error("expected the session to hand back the same status pointer")
		}
	})
}

func TestMatchSession_CategorizeTarget(t *testing.T) {
	matchUUID := uuid.New()
	playerUUID := uuid.New()
	participant := makeParticipant(matchUUID, &playerUUID)
	s := matchsession.NewMatchSession(matchUUID, nil, []*match.Participant{participant})

	t.Run("a piece CharacterID is a character", func(t *testing.T) {
		// Action.TargetID carries the piece's CharacterID, which is the sheet UUID.
		got := s.CategorizeTarget(participant.Sheet.UUID)
		if got != service.TargetKindCharacter {
			t.Errorf("expected TargetKindCharacter, got %q", got)
		}
	})

	t.Run("a wall ID is a wall segment", func(t *testing.T) {
		wallID := uuid.New()
		s.SyncMapState([]mapentity.WallSegment{{ID: wallID.String()}}, mapentity.GridShape{})

		if got := s.CategorizeTarget(wallID); got != service.TargetKindWallSegment {
			t.Errorf("expected TargetKindWallSegment, got %q", got)
		}
	})

	t.Run("anything else is unknown", func(t *testing.T) {
		if got := s.CategorizeTarget(uuid.New()); got != service.TargetKindUnknown {
			t.Errorf("expected TargetKindUnknown, got %q", got)
		}
	})
}

// ── helpers ──────────────────────────────────────────────────────────────────

func makeParticipant(matchUUID uuid.UUID, playerUUID *uuid.UUID) *match.Participant {
	return &match.Participant{
		UUID:      uuid.New(),
		MatchUUID: matchUUID,
		Sheet: csEntity.Summary{
			UUID:       uuid.New(),
			PlayerUUID: playerUUID,
		},
	}
}

// mustOpen opens the next action and hands back the turn it opened.
func mustOpen(t *testing.T, s *matchsession.MatchSession) *turn.Turn {
	t.Helper()
	tr, err := s.OpenNextAction()
	if err != nil {
		t.Fatalf("OpenNextAction: %v", err)
	}
	return tr.Opened
}

func enqueueAttack(t *testing.T, s *matchsession.MatchSession, playerUUID, charID uuid.UUID) *action.Action {
	t.Helper()
	a := action.NewAction(charID, nil, uuid.Nil, nil, action.ActionSpeed{},
		nil, nil, &action.Attack{}, nil, nil, nil, nil)
	if err := s.EnqueueAction(playerUUID, a); err != nil {
		t.Fatalf("EnqueueAction: %v", err)
	}
	return a
}

func mustOpenNext(t *testing.T, s *matchsession.MatchSession) *matchsession.TurnTransition {
	t.Helper()
	tr, err := s.OpenNextAction()
	if err != nil {
		t.Fatalf("OpenNextAction: %v", err)
	}
	return tr
}

// closeExhaustedRound drives the round the way production does: open until the session
// reports there is nothing left that can pay, and only THEN close.
//
// Calling CloseRound directly after an open would fail with ErrRoundHasOpenTurn — the turn
// under the baton is closed by the open that finds nothing, not by the round close.
func closeExhaustedRound(t *testing.T, s *matchsession.MatchSession) {
	t.Helper()
	for i := 0; i < 20; i++ {
		tr, err := s.OpenNextAction()
		if err != nil {
			t.Fatalf("OpenNextAction: %v", err)
		}
		if tr.RoundExhausted {
			if _, err := s.CloseRound(); err != nil {
				t.Fatalf("CloseRound: %v", err)
			}
			return
		}
	}
	t.Fatal("the round never ran out — the gate is letting through more than it should")
}

func makeAction(actorID uuid.UUID) *action.Action {
	return action.NewAction(actorID, nil, uuid.Nil, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil)
}

// sessionWithParticipants builds a session and returns it together with the sheet UUID of
// each player's character, in the order the players were given. The tests need the sheet
// UUIDs because that is what an action's actorID carries.
func sessionWithParticipants(playerUUIDs ...uuid.UUID) (*matchsession.MatchSession, []uuid.UUID) {
	matchUUID := uuid.New()
	participants := make([]*match.Participant, len(playerUUIDs))
	charIDs := make([]uuid.UUID, len(playerUUIDs))
	for i, id := range playerUUIDs {
		pID := id
		participants[i] = makeParticipant(matchUUID, &pID)
		charIDs[i] = participants[i].Sheet.UUID
	}
	return matchsession.NewMatchSession(matchUUID, nil, participants), charIDs
}

func makeActionWithSpeed(actorID uuid.UUID, speed int) *action.Action {
	return action.NewAction(actorID, nil, uuid.Nil, nil, action.ActionSpeed{RollCheck: action.RollCheck{Result: speed}}, nil, nil, nil, nil, nil, nil, nil)
}

func TestMatchSession_EnqueueAction(t *testing.T) {
	matchUUID := uuid.New()
	playerUUID := uuid.New()
	participant := makeParticipant(matchUUID, &playerUUID)
	charID := participant.Sheet.UUID
	s := matchsession.NewMatchSession(matchUUID, nil, []*match.Participant{participant})

	t.Run("enqueues an action whose actor is the player's character", func(t *testing.T) {
		// The combat entity is the character: actorID is the sheet UUID, the same ID the
		// board piece carries and the same ID a TargetID carries.
		a := makeAction(charID)
		if err := s.EnqueueAction(playerUUID, a); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns ErrParticipantNotFound for an unknown player", func(t *testing.T) {
		a := makeAction(charID)
		err := s.EnqueueAction(uuid.New(), a)
		if !errors.Is(err, matchsession.ErrParticipantNotFound) {
			t.Errorf("expected ErrParticipantNotFound, got %v", err)
		}
	})

	t.Run("a player cannot act through a character they do not own", func(t *testing.T) {
		a := makeAction(uuid.New()) // some other character
		err := s.EnqueueAction(playerUUID, a)
		if !errors.Is(err, matchsession.ErrActionActorMismatch) {
			t.Errorf("expected ErrActionActorMismatch, got %v", err)
		}
	})

	t.Run("the player UUID is no longer a valid actor", func(t *testing.T) {
		// It used to be exactly this, and it is what left the resolver unable to find the
		// actor's sheet.
		a := makeAction(playerUUID)
		err := s.EnqueueAction(playerUUID, a)
		if !errors.Is(err, matchsession.ErrActionActorMismatch) {
			t.Errorf("expected ErrActionActorMismatch, got %v", err)
		}
	})
}

func TestMatchSession_OpenNextAction(t *testing.T) {
	t.Run("opens Turn from highest-priority action in queue", func(t *testing.T) {
		playerA := uuid.New()
		playerB := uuid.New()
		s, chars := sessionWithParticipants(playerA, playerB)

		aHigh := makeActionWithSpeed(chars[0], 10)
		aLow := makeActionWithSpeed(chars[1], 3)
		s.EnqueueAction(playerA, aHigh) //nolint:errcheck
		s.EnqueueAction(playerB, aLow)  //nolint:errcheck

		tr, err := s.OpenNextAction()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tr.Closed != nil {
			t.Error("expected nil closed turn on first OpenNextAction")
		}
		if tr.Opened == nil {
			t.Fatal("expected non-nil opened turn")
		}
		if tr.Opened.GetAction().Speed.Result != 10 {
			t.Errorf("expected speed 10, got %d", tr.Opened.GetAction().Speed.Result)
		}
	})

	t.Run("closes previous open turn before opening next", func(t *testing.T) {
		playerA, playerB := uuid.New(), uuid.New()
		s, chars := sessionWithParticipants(playerA, playerB)
		s.EnqueueAction(playerA, makeActionWithSpeed(chars[0], 10)) //nolint:errcheck
		s.EnqueueAction(playerB, makeActionWithSpeed(chars[1], 5))  //nolint:errcheck

		tr1, _ := s.OpenNextAction()
		first := tr1.Opened
		tr2, err := s.OpenNextAction()
		closed := tr2.Closed
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if closed == nil {
			t.Fatal("expected closed turn to be non-nil on second call")
		}
		if closed != first {
			t.Error("expected closed turn to be the first opened turn")
		}
		if first.GetFinishedAt() == nil {
			t.Error("expected first turn to be closed")
		}
	})

	t.Run("returns service.ErrQueueEmpty when queue is empty", func(t *testing.T) {
		s := matchsession.NewMatchSession(uuid.New(), nil, nil)
		_, err := s.OpenNextAction()
		if !errors.Is(err, service.ErrQueueEmpty) {
			t.Errorf("expected ErrQueueEmpty, got %v", err)
		}
	})
}

func makeReactionTo(actorID, targetActionID uuid.UUID) *action.Action {
	return action.NewAction(actorID, nil, targetActionID, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil)
}

func TestMatchSession_AttachReaction(t *testing.T) {
	t.Run("attaches reaction to current turn and returns resolution", func(t *testing.T) {
		playerA, playerB := uuid.New(), uuid.New()
		s, chars := sessionWithParticipants(playerA, playerB)
		s.EnqueueAction(playerA, makeActionWithSpeed(chars[0], 10)) //nolint:errcheck
		opened := mustOpen(t, s)
		act := opened.GetAction()
		actionID := act.GetID()

		reaction := makeReactionTo(chars[1], actionID)
		res, err := s.AttachReaction(reaction)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res == nil {
			t.Fatal("expected non-nil TurnResolution")
		}
		if len(opened.GetReactions()) != 1 {
			t.Errorf("expected 1 reaction, got %d", len(opened.GetReactions()))
		}
	})

	t.Run("returns ErrReactionNotCompatible for wrong target", func(t *testing.T) {
		playerA := uuid.New()
		s, chars := sessionWithParticipants(playerA)
		s.EnqueueAction(playerA, makeActionWithSpeed(chars[0], 5)) //nolint:errcheck
		s.OpenNextAction()                                         //nolint:errcheck

		reaction := makeReactionTo(chars[0], uuid.New()) // wrong target
		_, err := s.AttachReaction(reaction)
		if !errors.Is(err, service.ErrReactionNotCompatible) {
			t.Errorf("expected ErrReactionNotCompatible, got %v", err)
		}
	})
}

func TestMatchSession_CloseTurn(t *testing.T) {
	t.Run("closes current open turn", func(t *testing.T) {
		playerA := uuid.New()
		s, chars := sessionWithParticipants(playerA)
		s.EnqueueAction(playerA, makeActionWithSpeed(chars[0], 5)) //nolint:errcheck
		opened := mustOpen(t, s)

		closed, err := s.CloseTurn()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if closed == nil {
			t.Fatal("expected non-nil closed turn")
		}
		if closed != opened {
			t.Error("expected closed turn to be the opened turn")
		}
		if closed.GetFinishedAt() == nil {
			t.Error("expected finishedAt to be set")
		}
	})

	t.Run("returns ErrNoCurrentTurn when no turns exist", func(t *testing.T) {
		s := matchsession.NewMatchSession(uuid.New(), nil, nil)
		_, err := s.CloseTurn()
		if !errors.Is(err, service.ErrNoCurrentTurn) {
			t.Errorf("expected ErrNoCurrentTurn, got %v", err)
		}
	})
}

func TestMatchSession_CloseRound(t *testing.T) {
	t.Run("closes round and starts a new one with same mode", func(t *testing.T) {
		playerA := uuid.New()
		s, chars := sessionWithParticipants(playerA)
		s.EnqueueAction(playerA, makeActionWithSpeed(chars[0], 5)) //nolint:errcheck
		s.OpenNextAction()                                         //nolint:errcheck
		s.CloseTurn()                                              //nolint:errcheck

		closedRound, err := s.CloseRound()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if closedRound == nil {
			t.Fatal("expected non-nil closed round")
		}
		if closedRound.GetFinishedAt() == nil {
			t.Error("expected finishedAt to be set on closed round")
		}
		if s.GetActiveRound() == closedRound {
			t.Error("expected activeRound to be a new round after CloseRound")
		}
		if s.GetActiveRound().GetMode() != enum.Free {
			t.Error("expected new round to preserve the previous round mode")
		}
	})

	t.Run("returns ErrRoundHasOpenTurn when a turn is still open", func(t *testing.T) {
		playerA := uuid.New()
		s, chars := sessionWithParticipants(playerA)
		s.EnqueueAction(playerA, makeActionWithSpeed(chars[0], 5)) //nolint:errcheck
		s.OpenNextAction()                                         //nolint:errcheck
		// turn is still open — no CloseTurn called

		_, err := s.CloseRound()
		if !errors.Is(err, matchsession.ErrRoundHasOpenTurn) {
			t.Errorf("expected ErrRoundHasOpenTurn, got %v", err)
		}
	})
}

func TestMatchSession_PullAction(t *testing.T) {
	t.Run("opens Turn for specific action UUID", func(t *testing.T) {
		playerA, playerB := uuid.New(), uuid.New()
		s, chars := sessionWithParticipants(playerA, playerB)
		aTarget := makeActionWithSpeed(chars[0], 3)
		aOther := makeActionWithSpeed(chars[1], 10)
		s.EnqueueAction(playerA, aTarget) //nolint:errcheck
		s.EnqueueAction(playerB, aOther)  //nolint:errcheck
		targetID := aTarget.GetID()

		tr, err := s.PullAction(targetID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := tr.Opened.GetAction()
		if got.GetID() != targetID {
			t.Errorf("expected action %v, got %v", targetID, got.GetID())
		}
	})

	t.Run("returns service.ErrActionNotFound for unknown UUID", func(t *testing.T) {
		s := matchsession.NewMatchSession(uuid.New(), nil, nil)
		_, err := s.PullAction(uuid.New())
		if !errors.Is(err, service.ErrActionNotFound) {
			t.Errorf("expected ErrActionNotFound, got %v", err)
		}
	})
}

func TestMatchSession_GetMatchUUID(t *testing.T) {
	id := uuid.New()
	s := matchsession.NewMatchSession(id, nil, nil)
	if s.GetMatchUUID() != id {
		t.Errorf("expected %v, got %v", id, s.GetMatchUUID())
	}
}

func TestMatchSession_GetActiveScene(t *testing.T) {
	s := matchsession.NewMatchSession(uuid.New(), nil, nil)
	if s.GetActiveScene() == nil {
		t.Fatal("expected non-nil active scene")
	}
	if s.GetActiveScene().GetCategory() != enum.Roleplay {
		t.Errorf("expected initial scene category Roleplay, got %v", s.GetActiveScene().GetCategory())
	}
}

func TestMatchSession_ChangeScene(t *testing.T) {
	t.Run("changes scene and resets round when no open turn", func(t *testing.T) {
		s := matchsession.NewMatchSession(uuid.New(), nil, nil)
		originalScene := s.GetActiveScene()
		originalRound := s.GetActiveRound()

		oldScene, oldRound, err := s.ChangeScene(enum.Battle, "Arena fight")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if oldScene != originalScene {
			t.Error("expected returned old scene to be the original")
		}
		if oldRound != originalRound {
			t.Error("expected returned old round to be the original")
		}
		if oldScene.GetFinishedAt() == nil {
			t.Error("expected old scene to be closed")
		}
		if oldRound.GetFinishedAt() == nil {
			t.Error("expected old round to be closed")
		}
		if s.GetActiveScene() == originalScene {
			t.Error("expected new active scene after ChangeScene")
		}
		if s.GetActiveScene().GetCategory() != enum.Battle {
			t.Errorf("expected new scene category Battle, got %v", s.GetActiveScene().GetCategory())
		}
		if s.GetActiveRound() == originalRound {
			t.Error("expected new active round after ChangeScene")
		}
	})

	t.Run("returns ErrRoundHasOpenTurn when turn is open", func(t *testing.T) {
		playerA := uuid.New()
		s, chars := sessionWithParticipants(playerA)
		s.EnqueueAction(playerA, makeActionWithSpeed(chars[0], 5)) //nolint:errcheck
		s.OpenNextAction()                                         //nolint:errcheck

		_, _, err := s.ChangeScene(enum.Battle, "desc")
		if !errors.Is(err, matchsession.ErrRoundHasOpenTurn) {
			t.Errorf("expected ErrRoundHasOpenTurn, got %v", err)
		}
	})
}

func TestMatchSession_EnqueueMasterAction(t *testing.T) {
	t.Run("enqueues master action on current open turn", func(t *testing.T) {
		playerA := uuid.New()
		s, chars := sessionWithParticipants(playerA)
		s.EnqueueAction(playerA, makeActionWithSpeed(chars[0], 5)) //nolint:errcheck
		opened := mustOpen(t, s)

		ma := action.NewMasterAction()
		if err := s.EnqueueMasterAction(ma); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(opened.GetMasterActions()) != 1 {
			t.Errorf("expected 1 master action on turn, got %d", len(opened.GetMasterActions()))
		}
		if ma.GetHappenedAt().IsZero() {
			t.Error("expected happenedAt to be set by EnqueueMasterAction")
		}
	})

	t.Run("returns ErrNoActiveTurn when no open turn", func(t *testing.T) {
		s := matchsession.NewMatchSession(uuid.New(), nil, nil)
		ma := action.NewMasterAction()
		err := s.EnqueueMasterAction(ma)
		if !errors.Is(err, matchsession.ErrNoActiveTurn) {
			t.Errorf("expected ErrNoActiveTurn, got %v", err)
		}
	})
}

func TestMatchSession_PersistenceFlags(t *testing.T) {
	t.Run("new session has flags false", func(t *testing.T) {
		s := matchsession.NewMatchSession(uuid.New(), nil, nil)
		if s.IsRoundPersisted() {
			t.Error("expected roundPersisted false on new session")
		}
		if s.IsScenePersisted() {
			t.Error("expected scenePersisted false on new session")
		}
	})

	t.Run("MarkRoundPersisted sets both flags", func(t *testing.T) {
		s := matchsession.NewMatchSession(uuid.New(), nil, nil)
		s.MarkRoundPersisted()
		if !s.IsRoundPersisted() {
			t.Error("expected roundPersisted true after MarkRoundPersisted")
		}
		if !s.IsScenePersisted() {
			t.Error("expected scenePersisted true after MarkRoundPersisted")
		}
	})

	t.Run("NewMatchSessionWithState has flags true", func(t *testing.T) {
		sc := scene.NewScene(enum.Battle, "Arena")
		r := round.NewRound(enum.Free)
		s := matchsession.NewMatchSessionWithState(uuid.New(), nil, nil, sc, r)
		if !s.IsRoundPersisted() {
			t.Error("expected roundPersisted true from WithState ctor")
		}
		if !s.IsScenePersisted() {
			t.Error("expected scenePersisted true from WithState ctor")
		}
		if s.GetActiveScene() != sc {
			t.Error("expected same scene pointer")
		}
		if s.GetActiveRound() != r {
			t.Error("expected same round pointer")
		}
	})

	t.Run("ChangeScene resets flags to false", func(t *testing.T) {
		sc := scene.NewScene(enum.Battle, "Arena")
		r := round.NewRound(enum.Free)
		s := matchsession.NewMatchSessionWithState(uuid.New(), nil, nil, sc, r)
		_, _, err := s.ChangeScene(enum.Roleplay, "Town")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.IsRoundPersisted() {
			t.Error("expected roundPersisted false after ChangeScene")
		}
		if s.IsScenePersisted() {
			t.Error("expected scenePersisted false after ChangeScene")
		}
	})
}

func TestSession_RecomputeVisibility_RecordsSeenWallsInMemory(t *testing.T) {
	matchUUID := uuid.New()
	playerUUID := uuid.New()
	sheetUUID := uuid.New()

	participant := &match.Participant{
		UUID:      uuid.New(),
		MatchUUID: matchUUID,
		Sheet: csEntity.Summary{
			UUID:       sheetUUID,
			PlayerUUID: &playerUUID,
		},
	}
	s := matchsession.NewMatchSession(matchUUID, nil, []*match.Participant{participant})

	squareGrid := mapentity.GridShape{
		Kind:      mapentity.GridKindSquare,
		Cols:      20,
		Rows:      20,
		CellSize:  64,
		SkewRatio: 1,
	}
	wall := mapentity.WallSegment{
		ID: "w1", P1: [2]float64{100, 0}, P2: [2]float64{100, 100},
		WallType: mapentity.WallTypeWall, Material: mapentity.WallMaterialStone,
		Sense: mapentity.SenseSight, HP: 100, MaxHP: 100,
	}
	s.SyncMapState([]mapentity.WallSegment{wall}, squareGrid)
	s.SyncPlayerMemories(nil, fog.FogModeExplored)

	src := &fakePieceSource{}
	src.setPosition(playerUUID, service.Point2D{X: 50, Y: 50})
	s.SetPieceSource(src)

	polys, err := s.RecomputeVisibility(playerUUID)
	if err != nil {
		t.Fatal(err)
	}
	if len(polys) == 0 {
		t.Fatal("expected non-empty visibility polygons")
	}

	memory, ok := s.GetPlayerMemory(playerUUID)
	if !ok || !memory.Has(fog.FeatureWall, "w1") {
		t.Fatal("expected the wall in line of sight to be recorded in the player's memory")
	}

	// Second recompute from the same spot must not panic or duplicate the entry.
	if _, err := s.RecomputeVisibility(playerUUID); err != nil {
		t.Fatal(err)
	}
	memory, _ = s.GetPlayerMemory(playerUUID)
	if len(memory.Seen) != 1 {
		t.Fatalf("re-recompute from same position should not duplicate memory entries, got %d", len(memory.Seen))
	}
}

// fixedSource makes every die land on the same face, so a test can name exact numbers.
type fixedSource struct{ face int }

func (f fixedSource) RollDie(_ enum.DieSides) int { return f.face }

func TestMatchSession_GetRules(t *testing.T) {
	s := matchsession.NewMatchSession(uuid.New(), nil, nil)
	rules := s.GetRules()

	if rules.DiceSet != match.DiceSet2D10 {
		t.Errorf("DiceSet = %q, want 2d10", rules.DiceSet)
	}
	if rules.LadderStep != 10 {
		t.Errorf("LadderStep = %d, want 10", rules.LadderStep)
	}
	if rules.PassiveValue() != 11 {
		t.Errorf("PassiveValue() = %d, want 11", rules.PassiveValue())
	}
}

func TestMatchSession_EnqueueAction_RollsTheDiceOnce(t *testing.T) {
	matchUUID := uuid.New()
	playerUUID := uuid.New()
	participant := makeParticipant(matchUUID, &playerUUID)
	charID := participant.Sheet.UUID
	s := matchsession.NewMatchSession(matchUUID, nil, []*match.Participant{participant})
	// Race mode, so actionSpeed is a real test and not the Free round's passive value —
	// this test is about dice falling, not about which mode derives what.
	s.GetActiveRound().SetMode(enum.Race)
	s.SetRollSource(fixedSource{face: 7})

	sword := enum.Sword
	atk := &action.Attack{
		Weapon: &sword,
		Hit:    action.RollCheck{SkillName: enum.Accuracy.String()},
	}
	a := action.NewAction(charID, nil, uuid.Nil, nil, action.ActionSpeed{},
		nil, nil, atk, nil, nil, nil, nil)

	if err := s.EnqueueAction(playerUUID, a); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("the hit dice fell, from the match dice set", func(t *testing.T) {
		if len(a.Attack.Hit.Attempts.Primary) != 2 {
			t.Fatalf("hit Primary = %v, want 2 dice", a.Attack.Hit.Attempts.Primary)
		}
		if a.Attack.Hit.Attempts.Primary[0] != 7 {
			t.Errorf("hit die = %d, want 7 from the scripted source", a.Attack.Hit.Attempts.Primary[0])
		}
		if len(a.Attack.Hit.Attempts.Secondary) != 2 {
			t.Error("both sets must fall up front, so a later advantage never re-rolls")
		}
	})

	t.Run("the damage dice fell, from the weapon's own set", func(t *testing.T) {
		// A Sword is D10 + D4 — two dice, not the match's 2 D10 by coincidence but by the
		// weapon's own definition.
		if len(a.Attack.Damage.Attempts.Primary) != 2 {
			t.Errorf("damage Primary = %v, want the sword's 2 dice", a.Attack.Damage.Attempts.Primary)
		}
		if len(a.Attack.Damage.Attempts.Secondary) != 0 {
			t.Error("damage has no advantage, so there is no second set")
		}
	})

	t.Run("the action speed dice fell", func(t *testing.T) {
		if a.Speed.Attempts.IsEmpty() {
			t.Error("expected the actionSpeed dice to fall on arrival too")
		}
	})
}

func TestMatchSession_EnqueueAction_NeverRerolls(t *testing.T) {
	matchUUID := uuid.New()
	playerUUID := uuid.New()
	participant := makeParticipant(matchUUID, &playerUUID)
	s := matchsession.NewMatchSession(matchUUID, nil, []*match.Participant{participant})
	s.SetRollSource(fixedSource{face: 3})

	a := action.NewAction(participant.Sheet.UUID, nil, uuid.Nil, nil, action.ActionSpeed{},
		nil, nil, &action.Attack{Hit: action.RollCheck{}}, nil, nil, nil, nil)
	// Dice that already fell — the master edited, the action came back around, whatever.
	a.Attack.Hit.Attempts = action.RollAttempts{Primary: []int{10, 10}, Secondary: []int{1, 1}}

	if err := s.EnqueueAction(playerUUID, a); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Attack.Hit.Attempts.Primary[0] != 10 {
		t.Error("dice that already fell must never be rolled again")
	}
}

// fixedTopFaceSource lands every die on its top face.
type fixedTopFaceSource struct{}

func (fixedTopFaceSource) RollDie(sides enum.DieSides) int { return sides.GetSides() }

func buildPlainSheet(t *testing.T) *csSheet.CharacterSheet {
	t.Helper()
	playerUUID := uuid.New()
	cs, err := csSheet.NewCharacterSheetFactory().Build(
		&playerUUID, nil, nil,
		csSheet.CharacterProfile{NickName: "Test", FullName: "Test Subject"},
		nil, nil, nil,
	)
	if err != nil {
		t.Fatalf("factory.Build error: %v", err)
	}
	return cs
}

func currentHP(t *testing.T, cs *csSheet.CharacterSheet) int {
	t.Helper()
	bar, ok := cs.GetAllStatusBar()[enum.Health]
	if !ok {
		t.Fatal("the sheet has no health bar")
	}
	return bar.GetCurrent()
}

func TestMatchSession_DamageIsAppliedOnlyOnTurnClose(t *testing.T) {
	matchUUID := uuid.New()
	playerA, playerB := uuid.New(), uuid.New()
	pA := makeParticipant(matchUUID, &playerA)
	pB := makeParticipant(matchUUID, &playerB)
	attacker, victim := pA.Sheet.UUID, pB.Sheet.UUID

	sheets := map[uuid.UUID]*csSheet.CharacterSheet{
		attacker: buildPlainSheet(t),
		victim:   buildPlainSheet(t),
	}
	s := matchsession.NewMatchSession(matchUUID, sheets, []*match.Participant{pA, pB})
	// Every die lands on its top face, so the hit clears the passive dodge of 11 for sure.
	s.SetRollSource(fixedTopFaceSource{})

	sword := enum.Sword
	atk := &action.Attack{Weapon: &sword, Hit: action.RollCheck{SkillName: enum.Accuracy.String()}}
	a := action.NewAction(attacker, []uuid.UUID{victim}, uuid.Nil, nil,
		action.ActionSpeed{RollCheck: action.RollCheck{Result: 10}},
		nil, nil, atk, nil, nil, nil, nil)

	if err := s.EnqueueAction(playerA, a); err != nil {
		t.Fatalf("EnqueueAction: %v", err)
	}
	hpBefore := currentHP(t, sheets[victim])

	// Opening the turn resolves it as a dry run.
	tr, err := s.OpenNextAction()
	if err != nil {
		t.Fatalf("OpenNextAction: %v", err)
	}

	t.Run("the master sees the projection before anything is applied", func(t *testing.T) {
		if tr.OpenedResolution == nil || len(tr.OpenedResolution.CharacterResults) != 1 {
			t.Fatalf("expected one character result, got %+v", tr.OpenedResolution)
		}
		if tr.OpenedResolution.CharacterResults[0].EffectiveDamage <= 0 {
			t.Fatal("expected projected damage from a maximum-roll attack")
		}
		if got := currentHP(t, sheets[victim]); got != hpBefore {
			t.Errorf("HP = %d, want %d — a projection must not touch the sheet", got, hpBefore)
		}
	})

	projected := tr.OpenedResolution.CharacterResults[0].EffectiveDamage

	t.Run("closing the turn applies it exactly once", func(t *testing.T) {
		// Enqueue a second action so there is something to open, which closes the first.
		a2 := action.NewAction(attacker, nil, uuid.Nil, nil,
			action.ActionSpeed{RollCheck: action.RollCheck{Result: 1}},
			nil, nil, nil, nil, nil, nil, nil)
		if err := s.EnqueueAction(playerA, a2); err != nil {
			t.Fatalf("EnqueueAction: %v", err)
		}
		tr2, err := s.OpenNextAction()
		if err != nil {
			t.Fatalf("OpenNextAction: %v", err)
		}
		if tr2.Closed == nil {
			t.Fatal("expected the first turn to close")
		}
		if len(tr2.Damaged) != 1 {
			t.Fatalf("expected 1 damaged character, got %d", len(tr2.Damaged))
		}
		if tr2.Damaged[0].CharacterID != victim {
			t.Errorf("damaged = %v, want %v", tr2.Damaged[0].CharacterID, victim)
		}
		if tr2.Damaged[0].Damage != projected {
			t.Errorf("applied %d, projected %d — they must agree", tr2.Damaged[0].Damage, projected)
		}
		if got := currentHP(t, sheets[victim]); got != hpBefore-projected {
			t.Errorf("HP = %d, want %d", got, hpBefore-projected)
		}
	})
}

func TestMatchSession_EnqueueAction_DerivesTheSpeed(t *testing.T) {
	matchUUID := uuid.New()
	playerUUID := uuid.New()
	participant := makeParticipant(matchUUID, &playerUUID)
	charID := participant.Sheet.UUID
	sheets := map[uuid.UUID]*csSheet.CharacterSheet{charID: buildPlainSheet(t)}

	t.Run("a Race actionSpeed rolls Legerity plus the dice", func(t *testing.T) {
		s := matchsession.NewMatchSession(matchUUID, sheets, []*match.Participant{participant})
		s.GetActiveRound().SetMode(enum.Race)
		s.SetRollSource(fixedSource{face: 6})

		a := action.NewAction(charID, nil, uuid.Nil, nil,
			action.ActionSpeed{RollCheck: action.RollCheck{SkillName: enum.Legerity.String()}},
			nil, nil, &action.Attack{}, nil, nil, nil, nil)
		if err := s.EnqueueAction(playerUUID, a); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Two D10 at 6 each, plus the sheet's Legerity.
		want := 12 + skillValue(t, sheets[charID], enum.Legerity)
		if a.Speed.Result != want {
			t.Errorf("Speed.Result = %d, want %d", a.Speed.Result, want)
		}
		if bars := a.Bars(); len(bars) != 1 || bars[0] != action.BarAction {
			t.Errorf("Bars() = %v, want just the action bar", bars)
		}
	})

	t.Run("a Free actionSpeed takes the passive value and rolls nothing", func(t *testing.T) {
		s := matchsession.NewMatchSession(matchUUID, sheets, []*match.Participant{participant})
		// A new session starts in Free.
		s.SetRollSource(fixedSource{face: 10})

		a := action.NewAction(charID, nil, uuid.Nil, nil, action.ActionSpeed{},
			nil, nil, &action.Attack{}, nil, nil, nil, nil)
		if err := s.EnqueueAction(playerUUID, a); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := 11 + skillValue(t, sheets[charID], enum.Legerity)
		if a.Speed.Result != want {
			t.Errorf("Speed.Result = %d, want the passive %d — rolling has zero expected gain", a.Speed.Result, want)
		}
	})

	t.Run("a Dash rolls Accelerate into the move bar", func(t *testing.T) {
		s := matchsession.NewMatchSession(matchUUID, sheets, []*match.Participant{participant})
		s.GetActiveRound().SetMode(enum.Race)
		s.SetRollSource(fixedSource{face: 4})

		a := action.NewAction(charID, nil, uuid.Nil, nil, action.ActionSpeed{},
			nil,
			&action.Move{Category: enum.Dash, Speed: &action.RollCheck{SkillName: enum.Accelerate.String()}},
			nil, nil, nil, nil, nil)
		if err := s.EnqueueAction(playerUUID, a); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := 8 + skillValue(t, sheets[charID], enum.Accelerate)
		if a.Move.FinalSpeed != want {
			t.Errorf("Move.FinalSpeed = %d, want %d", a.Move.FinalSpeed, want)
		}
		if a.SpeedOn(action.BarMove) != want {
			t.Errorf("SpeedOn(move) = %d, want %d", a.SpeedOn(action.BarMove), want)
		}
		if bars := a.Bars(); len(bars) != 1 || bars[0] != action.BarMove {
			t.Errorf("Bars() = %v, want just the move bar", bars)
		}
	})

	t.Run("a Shift takes the passive value of Brake and consumes no dice", func(t *testing.T) {
		s := matchsession.NewMatchSession(matchUUID, sheets, []*match.Participant{participant})
		s.GetActiveRound().SetMode(enum.Race)
		src := &countingSource{face: 9}
		s.SetRollSource(src)

		a := action.NewAction(charID, nil, uuid.Nil, nil, action.ActionSpeed{},
			nil,
			&action.Move{Category: enum.Shift, Speed: &action.RollCheck{SkillName: enum.Brake.String()}},
			nil, nil, nil, nil, nil)
		if err := s.EnqueueAction(playerUUID, a); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := 11 + skillValue(t, sheets[charID], enum.Brake)
		if a.Move.FinalSpeed != want {
			t.Errorf("Move.FinalSpeed = %d, want the passive %d", a.Move.FinalSpeed, want)
		}
		if !a.Move.Speed.Attempts.IsEmpty() {
			t.Error("a Shift rolls nothing: a phantom roll here silently drains a scripted source and shifts every number downstream")
		}
	})
}

func TestMatchSession_EnqueueAction_CombinedActionKeepsBothSpeeds(t *testing.T) {
	matchUUID := uuid.New()
	playerUUID := uuid.New()
	participant := makeParticipant(matchUUID, &playerUUID)
	charID := participant.Sheet.UUID
	sheets := map[uuid.UUID]*csSheet.CharacterSheet{charID: buildPlainSheet(t)}

	s := matchsession.NewMatchSession(matchUUID, sheets, []*match.Participant{participant})
	s.GetActiveRound().SetMode(enum.Race)
	s.SetRollSource(fixedSource{face: 5})

	sword := enum.Sword
	a := action.NewAction(charID, nil, uuid.Nil, nil, action.ActionSpeed{},
		nil,
		&action.Move{Category: enum.Dash, Position: [3]int{2, 2, 0}},
		&action.Attack{Weapon: &sword, Hit: action.RollCheck{SkillName: enum.Accuracy.String()}},
		nil, nil, nil, nil)
	if err := s.EnqueueAction(playerUUID, a); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("it stays ONE action in the queue", func(t *testing.T) {
		if pending := s.PendingActions(); len(pending) != 1 {
			t.Fatalf("queued %d actions, want 1 — an investida is a single action with the movement inside it", len(pending))
		}
	})

	t.Run("it charges both bars", func(t *testing.T) {
		bars := a.Bars()
		if len(bars) != 2 {
			t.Fatalf("Bars() = %v, want both", bars)
		}
	})

	t.Run("both speeds are derived and both survive on the action", func(t *testing.T) {
		wantAction := 10 + skillValue(t, sheets[charID], enum.Legerity)
		wantMove := 10 + skillValue(t, sheets[charID], enum.Accelerate)
		if a.SpeedOn(action.BarAction) != wantAction {
			t.Errorf("actionSpeed = %d, want %d", a.SpeedOn(action.BarAction), wantAction)
		}
		if a.SpeedOn(action.BarMove) != wantMove {
			t.Errorf("moveSpeed = %d, want %d", a.SpeedOn(action.BarMove), wantMove)
		}
	})
}

// skillValue reads a skill off a sheet the same way the engine does, so a test never hard
// codes a number the factory owns.
func skillValue(t *testing.T, cs *csSheet.CharacterSheet, name enum.SkillName) int {
	t.Helper()
	v, err := cs.GetValueForTestOfSkill(name)
	if err != nil {
		t.Fatalf("GetValueForTestOfSkill(%s): %v", name, err)
	}
	return v
}

// countingSource is a scripted source that also reports how many dice it handed out, so a
// test can prove a passive check rolled nothing.
type countingSource struct {
	face  int
	rolls int
}

func (c *countingSource) RollDie(_ enum.DieSides) int {
	c.rolls++
	return c.face
}

// scriptedFaces hands out faces in order and repeats the last one once exhausted.
type scriptedFaces struct {
	faces []int
	i     int
}

func (s *scriptedFaces) RollDie(_ enum.DieSides) int {
	if len(s.faces) == 0 {
		return 1
	}
	if s.i >= len(s.faces) {
		return s.faces[len(s.faces)-1]
	}
	f := s.faces[s.i]
	s.i++
	return f
}

func TestMatchSession_OpenNextAction_UsesTheBarEconomy(t *testing.T) {
	matchUUID := uuid.New()
	p1UUID, p2UUID := uuid.New(), uuid.New()
	p1 := makeParticipant(matchUUID, &p1UUID)
	p2 := makeParticipant(matchUUID, &p2UUID)
	sheets := map[uuid.UUID]*csSheet.CharacterSheet{
		p1.Sheet.UUID: buildPlainSheet(t),
		p2.Sheet.UUID: buildPlainSheet(t),
	}
	s := matchsession.NewMatchSession(matchUUID, sheets, []*match.Participant{p1, p2})
	s.GetActiveRound().SetMode(enum.Race)

	// Two D10 per action, scripted: p1 gets 4+4 = 8, p2 gets 9+9 = 18.
	s.SetRollSource(&scriptedFaces{faces: []int{4, 4, 4, 4, 9, 9, 9, 9}})

	enqueue := func(pl uuid.UUID, charID uuid.UUID) {
		t.Helper()
		a := action.NewAction(charID, nil, uuid.Nil, nil, action.ActionSpeed{},
			nil, nil, &action.Attack{}, nil, nil, nil, nil)
		if err := s.EnqueueAction(pl, a); err != nil {
			t.Fatalf("EnqueueAction: %v", err)
		}
	}
	enqueue(p1UUID, p1.Sheet.UUID)
	enqueue(p2UUID, p2.Sheet.UUID)

	tr, err := s.OpenNextAction()
	if err != nil {
		t.Fatalf("OpenNextAction: %v", err)
	}

	t.Run("the faster character opens first, not whoever was inserted first", func(t *testing.T) {
		openedAction := tr.Opened.GetAction()
		if openedAction.GetActorID() != p2.Sheet.UUID {
			t.Error("p2 rolled higher and must open first — the queue finally has a priority")
		}
	})

	t.Run("the action bar priced at the slowest pending speed", func(t *testing.T) {
		price, frozen := s.GetActiveRound().Price(action.BarAction)
		if !frozen {
			t.Fatal("opening the first action must freeze the price")
		}
		want := 8 + skillValue(t, sheets[p1.Sheet.UUID], enum.Legerity)
		if price != want {
			t.Errorf("price = %d, want p1's speed %d", price, want)
		}
	})

	t.Run("the opened action was recorded as having acted", func(t *testing.T) {
		_, acted := s.BarState(p2.Sheet.UUID, action.BarAction)
		if len(acted) != 1 {
			t.Fatalf("acted = %v, want exactly the one speed that opened", acted)
		}
		_, p1Acted := s.BarState(p1.Sheet.UUID, action.BarAction)
		if len(p1Acted) != 0 {
			t.Error("a pending action must not be recorded as acted")
		}
	})
}

func TestMatchSession_OpenNextAction_ReportsExhaustion(t *testing.T) {
	matchUUID := uuid.New()
	playerUUID := uuid.New()
	participant := makeParticipant(matchUUID, &playerUUID)
	charID := participant.Sheet.UUID
	sheets := map[uuid.UUID]*csSheet.CharacterSheet{charID: buildPlainSheet(t)}
	s := matchsession.NewMatchSession(matchUUID, sheets, []*match.Participant{participant})
	s.GetActiveRound().SetMode(enum.Race)
	s.SetRollSource(fixedSource{face: 5})

	a := action.NewAction(charID, nil, uuid.Nil, nil, action.ActionSpeed{},
		nil, nil, &action.Attack{}, nil, nil, nil, nil)
	if err := s.EnqueueAction(playerUUID, a); err != nil {
		t.Fatalf("EnqueueAction: %v", err)
	}
	if _, err := s.OpenNextAction(); err != nil {
		t.Fatalf("first open: %v", err)
	}

	tr, err := s.OpenNextAction()

	t.Run("exhaustion is a report, not an error", func(t *testing.T) {
		if err != nil {
			t.Errorf("err = %v, want nil: an exhausted Race round is a normal outcome", err)
		}
		if !tr.RoundExhausted {
			t.Error("nothing pending passes its gate, so the round is exhausted")
		}
		if tr.Opened != nil {
			t.Error("nothing opened")
		}
	})

	t.Run("the turn under the baton still closed and still applied", func(t *testing.T) {
		if tr.Closed == nil {
			t.Error("the open turn must close on the way out, exactly as it does on a normal open")
		}
	})
}

func TestMatchSession_FreeRoundStillReportsAnEmptyQueue(t *testing.T) {
	matchUUID := uuid.New()
	playerUUID := uuid.New()
	participant := makeParticipant(matchUUID, &playerUUID)
	s := matchsession.NewMatchSession(matchUUID, nil, []*match.Participant{participant})

	_, err := s.OpenNextAction()
	if !errors.Is(err, service.ErrQueueEmpty) {
		t.Errorf("err = %v, want ErrQueueEmpty: Free has no economy, so an empty queue is still an error", err)
	}
}

func TestMatchSession_CloseRound_SettlesTheBars(t *testing.T) {
	matchUUID := uuid.New()
	p1UUID, p2UUID := uuid.New(), uuid.New()
	p1 := makeParticipant(matchUUID, &p1UUID)
	p2 := makeParticipant(matchUUID, &p2UUID)
	sheets := map[uuid.UUID]*csSheet.CharacterSheet{
		p1.Sheet.UUID: buildPlainSheet(t),
		p2.Sheet.UUID: buildPlainSheet(t),
	}

	newRacingSession := func(faces []int) *matchsession.MatchSession {
		s := matchsession.NewMatchSession(matchUUID, sheets, []*match.Participant{p1, p2})
		s.GetActiveRound().SetMode(enum.Race)
		s.SetRollSource(&scriptedFaces{faces: faces})
		return s
	}

	t.Run("a character who acted once carries the leftover", func(t *testing.T) {
		// p1 rolls 4+4 = 8, p2 rolls 7+7 = 14. Legerity is 0 on a factory sheet, so those
		// ARE the speeds. Price = 8. p2 keeps 14 − 8 = 6, under the ceiling of 8; p1 keeps 0.
		s := newRacingSession([]int{4, 4, 4, 4, 7, 7, 7, 7})
		enqueueAttack(t, s, p1UUID, p1.Sheet.UUID)
		enqueueAttack(t, s, p2UUID, p2.Sheet.UUID)

		closeExhaustedRound(t, s)

		carry, acted := s.BarState(p2.Sheet.UUID, action.BarAction)
		if carry != 6 {
			t.Errorf("p2 carry = %v, want 6", carry)
		}
		if len(acted) != 0 {
			t.Error("the round's speed history must be cleared; the balance is what crosses over")
		}
		if p1Carry, _ := s.BarState(p1.Sheet.UUID, action.BarAction); p1Carry != 0 {
			t.Errorf("p1 carry = %v, want 0 — the slowest of the round starts the next one from zero", p1Carry)
		}
	})

	t.Run("the carry is capped at the round price", func(t *testing.T) {
		// p1 = 2+2 = 4 (the price), p2 = 10+10 = 20. p2's leftover of 16 blows past the
		// ceiling of 4 and is clipped to it.
		s := newRacingSession([]int{2, 2, 2, 2, 10, 10, 10, 10})
		enqueueAttack(t, s, p1UUID, p1.Sheet.UUID)
		enqueueAttack(t, s, p2UUID, p2.Sheet.UUID)

		closeExhaustedRound(t, s)

		if carry, _ := s.BarState(p2.Sheet.UUID, action.BarAction); carry != 4 {
			t.Errorf("carry = %v, want the ceiling 4 — standing time may not compound", carry)
		}
	})

	t.Run("a character who sent nothing carries the floor", func(t *testing.T) {
		// Only p1 acts, at 6+6 = 12, which is therefore also the price. p1 closes at 0, and
		// p2 — who never sent anything — closes at the floor, the same number as the ceiling.
		s := newRacingSession([]int{6, 6, 6, 6})
		enqueueAttack(t, s, p1UUID, p1.Sheet.UUID)

		closeExhaustedRound(t, s)

		if carry, _ := s.BarState(p2.Sheet.UUID, action.BarAction); carry != 12 {
			t.Errorf("p2 carry = %v, want the floor 12: reading the fight instead of acting is a legitimate trade", carry)
		}
		if carry, _ := s.BarState(p1.Sheet.UUID, action.BarAction); carry != 0 {
			t.Errorf("p1 carry = %v, want 0", carry)
		}
	})

	t.Run("a bar that never priced is left exactly as it was", func(t *testing.T) {
		s := newRacingSession([]int{6, 6, 6, 6})
		enqueueAttack(t, s, p1UUID, p1.Sheet.UUID)

		closeExhaustedRound(t, s)

		if carry, _ := s.BarState(p1.Sheet.UUID, action.BarMove); carry != 0 {
			t.Errorf("move carry = %v, want 0 — nobody moved, so no round happened on that bar", carry)
		}
	})

	t.Run("an action that never reached the price keeps its full roll for the next round", func(t *testing.T) {
		// p1 acts at 7+7 = 14, pricing the bar at 14. Only THEN does p2 send an action worth
		// 1+1 = 2, which cannot pay and sits the round out.
		s := newRacingSession([]int{7, 7, 7, 7})
		enqueueAttack(t, s, p1UUID, p1.Sheet.UUID)
		mustOpenNext(t, s)

		s.SetRollSource(&scriptedFaces{faces: []int{1, 1, 1, 1}})
		enqueueAttack(t, s, p2UUID, p2.Sheet.UUID)

		pending := s.PendingActions()
		if len(pending) != 1 {
			t.Fatalf("pending = %d, want 1", len(pending))
		}
		speedBefore := pending[0].SpeedOn(action.BarAction)

		closeExhaustedRound(t, s)

		after := s.PendingActions()
		if len(after) != 1 {
			t.Fatalf("pending after close = %d, want the action still queued", len(after))
		}
		if after[0].SpeedOn(action.BarAction) != speedBefore {
			t.Errorf("speed = %d, want %d unchanged: it pays nothing and goes to the next round whole",
				after[0].SpeedOn(action.BarAction), speedBefore)
		}
		if carry, _ := s.BarState(p2.Sheet.UUID, action.BarAction); carry < 0 {
			t.Errorf("carry = %v — sitting the round out must never put the bar in debt", carry)
		}
	})
}
