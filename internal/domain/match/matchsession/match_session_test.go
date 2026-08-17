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

		closed, opened, err := s.OpenNextAction()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if closed != nil {
			t.Error("expected nil closed turn on first OpenNextAction")
		}
		if opened == nil {
			t.Fatal("expected non-nil opened turn")
		}
		if opened.GetAction().Speed.Result != 10 {
			t.Errorf("expected speed 10, got %d", opened.GetAction().Speed.Result)
		}
	})

	t.Run("closes previous open turn before opening next", func(t *testing.T) {
		playerA, playerB := uuid.New(), uuid.New()
		s, chars := sessionWithParticipants(playerA, playerB)
		s.EnqueueAction(playerA, makeActionWithSpeed(chars[0], 10)) //nolint:errcheck
		s.EnqueueAction(playerB, makeActionWithSpeed(chars[1], 5))  //nolint:errcheck

		_, first, _ := s.OpenNextAction()
		closed, _, err := s.OpenNextAction()
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
		_, _, err := s.OpenNextAction()
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
		_, opened, _ := s.OpenNextAction()
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
		_, opened, _ := s.OpenNextAction()

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

		_, opened, err := s.PullAction(targetID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := opened.GetAction()
		if got.GetID() != targetID {
			t.Errorf("expected action %v, got %v", targetID, got.GetID())
		}
	})

	t.Run("returns service.ErrActionNotFound for unknown UUID", func(t *testing.T) {
		s := matchsession.NewMatchSession(uuid.New(), nil, nil)
		_, _, err := s.PullAction(uuid.New())
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
		_, opened, _ := s.OpenNextAction()

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
