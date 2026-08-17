package match_test

import (
	"context"
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/status"
	matchDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

func TestOpenNextActionUC(t *testing.T) {
	masterUUID := uuid.New()
	playerUUID := uuid.New()

	t.Run("returns ErrNotMatchMaster when caller is not master", func(t *testing.T) {
		session := matchsession.NewMatchSession(uuid.New(), nil, nil)
		uc := match.NewOpenNextActionUC(&fakeStatusWriter{})
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

		uc := match.NewOpenNextActionUC(&fakeStatusWriter{})
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
		uc := match.NewOpenNextActionUC(&fakeStatusWriter{})
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

	uc := match.NewOpenNextActionUC(writer)
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
	uc := match.NewOpenNextActionUC(nil)
	if _, err := uc.Execute(context.Background(), session, masterUUID, masterUUID); err == nil {
		t.Error("expected ErrQueueEmpty on an empty queue")
	}
}
