package match_test

import (
	"context"
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/status"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	matchDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

func TestOpenNextActionUC(t *testing.T) {
	masterUUID := uuid.New()
	playerUUID := uuid.New()

	t.Run("returns ErrNotMatchMaster when caller is not master", func(t *testing.T) {
		session := matchsession.NewMatchSession(uuid.New(), nil, nil)
		uc := match.NewOpenNextActionUC(&fakeStatusWriter{}, nil)
		_, err := uc.Execute(context.Background(), session, masterUUID, uuid.New())
		if !errors.Is(err, match.ErrNotMatchMaster) {
			t.Errorf("expected ErrNotMatchMaster, got %v", err)
		}
	})

	t.Run("returns result with opened turn on success", func(t *testing.T) {
		pUUID := playerUUID
		matchUUID := uuid.New()
		p := &matchDomain.Participant{
			UUID:      uuid.New(),
			MatchUUID: matchUUID,
			Sheet:     csEntity.Summary{UUID: uuid.New(), PlayerUUID: &pUUID},
		}
		session := matchsession.NewMatchSession(matchUUID, nil, []*matchDomain.Participant{p})
		a := action.NewAction(p.Sheet.UUID, nil, uuid.Nil, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil)
		session.EnqueueAction(playerUUID, a) //nolint:errcheck

		uc := match.NewOpenNextActionUC(&fakeStatusWriter{}, nil)
		result, err := uc.Execute(context.Background(), session, masterUUID, masterUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.OpenedTurn == nil {
			t.Error("expected non-nil OpenedTurn")
		}
		if result.Resolution == nil {
			t.Error("expected non-nil Resolution")
		}
	})

	t.Run("returns ErrQueueEmpty when queue is empty", func(t *testing.T) {
		session := matchsession.NewMatchSession(uuid.New(), nil, nil)
		uc := match.NewOpenNextActionUC(&fakeStatusWriter{}, nil)
		_, err := uc.Execute(context.Background(), session, masterUUID, masterUUID)
		if !errors.Is(err, service.ErrQueueEmpty) {
			t.Errorf("expected ErrQueueEmpty, got %v", err)
		}
	})
}

// fakeStatusWriter records which sheets were persisted.
type fakeStatusWriter struct {
	calls []string
	err   error
}

func (f *fakeStatusWriter) UpdateStatusBars(
	_ context.Context, sheetUUID string, _, _, _ status.IStatusBar,
) error {
	f.calls = append(f.calls, sheetUUID)
	return f.err
}

func TestOpenNextActionUC_PersistsOnlyWhatTookDamage(t *testing.T) {
	// A transition that damaged nobody must not write.
	writer := &fakeStatusWriter{}
	playerA := uuid.New()
	session, chars := sessionWithPlayers(playerA)
	masterUUID := uuid.New()

	a := action.NewAction(chars[0], nil, uuid.Nil, nil,
		action.ActionSpeed{RollCheck: action.RollCheck{Result: 5}},
		nil, nil, nil, nil, nil, nil, nil)
	session.EnqueueAction(playerA, a) //nolint:errcheck

	uc := match.NewOpenNextActionUC(writer, nil)
	if _, err := uc.Execute(context.Background(), session, masterUUID, masterUUID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(writer.calls) != 0 {
		t.Errorf("expected no persistence when nothing took damage, got %v", writer.calls)
	}
}

func TestOpenNextActionUC_NilWriterIsSafe(t *testing.T) {
	// Delivery tests build these use cases without a gateway; a nil writer must no-op
	// rather than panic.
	session, chars := sessionWithPlayers(uuid.New())
	_ = chars
	masterUUID := uuid.New()
	uc := match.NewOpenNextActionUC(nil, nil)
	if _, err := uc.Execute(context.Background(), session, masterUUID, masterUUID); err == nil {
		t.Error("expected ErrQueueEmpty on an empty queue")
	}
}

// spyCloseRound records that the exhausted round was handed to the close use case.
type spyCloseRound struct {
	calls  int
	closed *round.Round
	err    error
}

func (s *spyCloseRound) Execute(
	ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID,
) (*round.Round, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	r, err := session.CloseRound()
	s.closed = r
	return r, err
}

// racingSessionWithOneAction builds a Race-mode session — one participant, a factory-built
// sheet, a fixed roll source — with a single enqueued attack: enough for one open to drain
// the queue and the next to find it empty, tripping RoundExhausted.
func racingSessionWithOneAction(t *testing.T, playerUUID uuid.UUID) (*matchsession.MatchSession, uuid.UUID) {
	t.Helper()
	matchUUID := uuid.New()
	pUUID := playerUUID
	p := &matchDomain.Participant{
		UUID:      uuid.New(),
		MatchUUID: matchUUID,
		Sheet:     csEntity.Summary{UUID: uuid.New(), PlayerUUID: &pUUID},
	}
	sheets := map[uuid.UUID]*csSheet.CharacterSheet{p.Sheet.UUID: buildPlainSheet(t)}
	session := matchsession.NewMatchSession(matchUUID, sheets, []*matchDomain.Participant{p})
	session.GetActiveRound().SetMode(enum.Race)
	session.SetRollSource(fixedSource{face: 5})

	a := action.NewAction(p.Sheet.UUID, nil, uuid.Nil, nil, action.ActionSpeed{},
		nil, nil, &action.Attack{}, nil, nil, nil, nil)
	if err := session.EnqueueAction(playerUUID, a); err != nil {
		t.Fatalf("EnqueueAction: %v", err)
	}
	return session, p.Sheet.UUID
}

// freeSessionWithNoActions builds a Free-mode session — no economy — with an empty queue.
func freeSessionWithNoActions(t *testing.T) *matchsession.MatchSession {
	t.Helper()
	return matchsession.NewMatchSession(uuid.New(), nil, nil)
}

// buildPlainSheet builds a minimal factory-valid character sheet, the same way
// TestMatchSession_CloseRound_SettlesTheBars does in the matchsession package.
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

// fixedSource lands every die on the same face — reproducible rolls for tests.
type fixedSource struct{ face int }

func (f fixedSource) RollDie(_ enum.DieSides) int { return f.face }

func TestOpenNextAction_ClosesAnExhaustedRound(t *testing.T) {
	masterUUID := uuid.New()
	playerUUID := uuid.New()
	session, charID := racingSessionWithOneAction(t, playerUUID)
	_ = charID

	spy := &spyCloseRound{}
	uc := match.NewOpenNextActionUC(nil, spy)

	// First open consumes the only action.
	if _, err := uc.Execute(context.Background(), session, masterUUID, masterUUID); err != nil {
		t.Fatalf("first open: %v", err)
	}

	res, err := uc.Execute(context.Background(), session, masterUUID, masterUUID)

	t.Run("no error — an exhausted round is a normal outcome", func(t *testing.T) {
		if err != nil {
			t.Errorf("err = %v, want nil", err)
		}
	})
	t.Run("the round was closed through CloseRoundUC", func(t *testing.T) {
		if spy.calls != 1 {
			t.Errorf("close calls = %d, want 1", spy.calls)
		}
		if res == nil || res.ClosedRound == nil {
			t.Error("the caller needs the closed round to announce round_closed")
		}
	})
	t.Run("nothing opened", func(t *testing.T) {
		if res != nil && res.OpenedTurn != nil {
			t.Error("there was nothing left to open")
		}
	})
}

func TestOpenNextAction_EmptyFreeQueueStillErrors(t *testing.T) {
	masterUUID := uuid.New()
	session := freeSessionWithNoActions(t)
	uc := match.NewOpenNextActionUC(nil, &spyCloseRound{})

	if _, err := uc.Execute(context.Background(), session, masterUUID, masterUUID); err == nil {
		t.Error("a Free round has no economy: an empty queue is still an error, as it always was")
	}
}
