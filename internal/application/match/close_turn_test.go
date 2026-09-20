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
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

// turnFixture is an attacker/victim session with the attack already open as the current
// turn — a real Attack, not a bare Speed action, so the victim's reaction has a chain step to
// land on and TestCloseTurnUC's confirmed close can see a CharacterResult come out of it.
type turnFixture struct {
	session      *matchsession.MatchSession
	masterUUID   uuid.UUID
	victimPlayer uuid.UUID
	victimChar   uuid.UUID
}

func newTurnFixture(t *testing.T) *turnFixture {
	t.Helper()
	matchUUID := uuid.New()
	attackerPlayer, victimPlayer := uuid.New(), uuid.New()
	attacker := &matchDomain.Participant{
		UUID:      uuid.New(),
		MatchUUID: matchUUID,
		Sheet:     csEntity.Summary{UUID: uuid.New(), PlayerUUID: &attackerPlayer},
	}
	victim := &matchDomain.Participant{
		UUID:      uuid.New(),
		MatchUUID: matchUUID,
		Sheet:     csEntity.Summary{UUID: uuid.New(), PlayerUUID: &victimPlayer},
	}
	sheets := map[uuid.UUID]*csSheet.CharacterSheet{
		attacker.Sheet.UUID: buildPlainSheet(t),
		victim.Sheet.UUID:   buildPlainSheet(t),
	}
	session := matchsession.NewMatchSession(matchUUID, sheets, []*matchDomain.Participant{attacker, victim})
	// topFaceSource guarantees the hit lands and clears the passive dodge, exactly as
	// racingSessionWithAttackThatHits does in open_next_action_test.go — same package,
	// helper reused rather than duplicated.
	session.SetRollSource(topFaceSource{})

	sword := enum.Sword
	atk := &action.Attack{Weapon: &sword, Hit: action.RollCheck{SkillName: enum.Accuracy.String()}}
	a := action.NewAction(attacker.Sheet.UUID, []uuid.UUID{victim.Sheet.UUID}, uuid.Nil, nil,
		action.ActionSpeed{RollCheck: action.RollCheck{Result: 10}},
		nil, nil, atk, nil, nil, nil, nil)
	if err := session.EnqueueAction(attackerPlayer, a); err != nil {
		t.Fatalf("EnqueueAction: %v", err)
	}
	if _, err := session.OpenNextAction(); err != nil {
		t.Fatalf("OpenNextAction: %v", err)
	}

	return &turnFixture{
		session:      session,
		masterUUID:   uuid.New(),
		victimPlayer: victimPlayer,
		victimChar:   victim.Sheet.UUID,
	}
}

// attachReaction attaches one reaction from the victim to the open turn's action, and
// deliberately never opens it — the scenario close_turn's refusal exists for.
func (f *turnFixture) attachReaction(t *testing.T) {
	t.Helper()
	act := f.session.GetActiveRound().CurrentTurn().GetAction()
	reaction := action.NewAction(f.victimChar, nil, act.GetID(), nil,
		action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil)
	if _, err := f.session.AttachReaction(f.victimPlayer, reaction); err != nil {
		t.Fatalf("AttachReaction: %v", err)
	}
}

// noopStatusWriter discards every write. CloseTurnUC's persistence is a straight call into
// persistDamage, already covered end to end by open_next_action_test.go's fakeStatusWriter;
// here Execute only needs a writer that does not panic.
type noopStatusWriter struct{}

func (noopStatusWriter) UpdateStatusBars(
	_ context.Context, _ string, _, _, _ status.IStatusBar,
) error {
	return nil
}

func TestCloseTurnUC(t *testing.T) {
	t.Run("refuses when a reaction was attached and never opened", func(t *testing.T) {
		f := newTurnFixture(t) // session with one open turn
		f.attachReaction(t)    // one reaction, not opened

		uc := match.NewCloseTurnUC(&noopStatusWriter{})
		res, err := uc.Execute(context.Background(), f.session, f.masterUUID, f.masterUUID, false)
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if len(res.Refused) != 1 {
			t.Fatalf("Refused = %d, want 1", len(res.Refused))
		}
		if res.ClosedTurn != nil {
			t.Fatal("the turn closed despite the refusal")
		}
	})

	t.Run("closes with confirm, and the unopened reaction still counts", func(t *testing.T) {
		f := newTurnFixture(t)
		f.attachReaction(t)

		uc := match.NewCloseTurnUC(&noopStatusWriter{})
		res, err := uc.Execute(context.Background(), f.session, f.masterUUID, f.masterUUID, true)
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if res.ClosedTurn == nil {
			t.Fatal("confirm:true did not close the turn")
		}
		if res.Resolution == nil || len(res.Resolution.CharacterResults) == 0 {
			t.Fatal("the unopened reaction's target produced no character result")
		}
	})

	t.Run("closes without confirm when nothing is pending", func(t *testing.T) {
		f := newTurnFixture(t)
		uc := match.NewCloseTurnUC(&noopStatusWriter{})
		res, err := uc.Execute(context.Background(), f.session, f.masterUUID, f.masterUUID, false)
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if res.ClosedTurn == nil {
			t.Fatal("a clean close was refused")
		}
	})

	t.Run("only the master may close", func(t *testing.T) {
		f := newTurnFixture(t)
		uc := match.NewCloseTurnUC(&noopStatusWriter{})
		_, err := uc.Execute(context.Background(), f.session, f.masterUUID, uuid.New(), true)
		if !errors.Is(err, match.ErrNotMatchMaster) {
			t.Fatalf("err = %v, want ErrNotMatchMaster", err)
		}
	})
}
