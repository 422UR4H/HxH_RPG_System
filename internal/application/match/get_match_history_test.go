package match_test

import (
	"context"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/application/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/application/testutil"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

// mockHistoryRoundRepo is a minimal IRoundRepository for GetMatchHistoryUC tests — only
// FindMatchHistory is exercised; the rest of the interface is stubbed to satisfy it.
type mockHistoryRoundRepo struct {
	fn func(ctx context.Context, matchUUID uuid.UUID) ([]match.HistoryScene, error)
}

func (m *mockHistoryRoundRepo) FindMatchHistory(
	ctx context.Context, matchUUID uuid.UUID,
) ([]match.HistoryScene, error) {
	return m.fn(ctx, matchUUID)
}
func (m *mockHistoryRoundRepo) CloseRound(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}
func (m *mockHistoryRoundRepo) CloseSceneAndRound(_ context.Context, _, _ uuid.UUID, _ time.Time) error {
	return nil
}
func (m *mockHistoryRoundRepo) FindActiveSession(
	_ context.Context, _ uuid.UUID,
) (*matchsession.ActiveSessionData, error) {
	return nil, nil
}
func (m *mockHistoryRoundRepo) PersistTurnClose(_ context.Context, _ match.TurnCloseData) error {
	return nil
}

// actionWithClosedDodge builds a reaction shaped exactly like projection_test.go's own fixture:
// a closed dodge with the Evasion skill folded in, so the demotion this whole task exists to
// enforce has something real to strip.
func actionWithClosedDodge(actorID uuid.UUID) action.Action {
	a := action.NewAction(actorID, nil, uuid.New(),
		[]action.Skill{
			{SkillName: enum.Evasion.String()},
			{SkillName: enum.Legerity.String()},
		},
		action.ActionSpeed{}, nil, nil, nil, nil, &action.Dodge{}, nil, nil,
	)
	a.ReactionKind = action.ReactClosedDodge
	return *a
}

// actionWithFeint builds a plain action carrying a feint — the other deny-list item
// ProjectAction strips from anyone but the owner/master.
func actionWithFeint(actorID uuid.UUID) action.Action {
	a := action.NewAction(actorID, nil, uuid.Nil,
		[]action.Skill{{SkillName: enum.Legerity.String()}},
		action.ActionSpeed{}, &action.RollCheck{SkillName: enum.Legerity.String()},
		nil, nil, nil, nil, nil, nil,
	)
	return *a
}

func historyWithTurns(turns ...match.HistoryTurn) []match.HistoryScene {
	return []match.HistoryScene{{
		UUID: uuid.New(), Category: "combat",
		Rounds: []match.HistoryRound{{
			UUID: uuid.New(), Mode: "combat",
			Turns: turns,
		}},
	}}
}

func TestGetMatchHistoryUC(t *testing.T) {
	masterUUID := uuid.New()
	matchUUID := uuid.New()
	campaignUUID := uuid.New()

	privateMatch := &matchEntity.Match{
		UUID: matchUUID, MasterUUID: masterUUID, CampaignUUID: campaignUUID, IsPublic: false,
	}

	t.Run("a non-participant of a private match is refused", func(t *testing.T) {
		userUUID := uuid.New()
		matchMock := &testutil.MockMatchRepo{
			GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return privateMatch, nil
			},
			ListParticipantsByMatchUUIDFn: func(_ context.Context, _ uuid.UUID) ([]*matchEntity.Participant, error) {
				t.Fatal("participants should not be read once authorization already failed")
				return nil, nil
			},
		}
		roundMock := &mockHistoryRoundRepo{
			fn: func(_ context.Context, _ uuid.UUID) ([]match.HistoryScene, error) {
				t.Fatal("history should not be read once authorization already failed")
				return nil, nil
			},
		}
		checker := &mockParticipationChecker{fn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return false, nil
		}}

		uc := match.NewGetMatchHistoryUC(matchMock, roundMock, checker)
		_, err := uc.Get(context.Background(), matchUUID, userUUID)
		if err != auth.ErrInsufficientPermissions {
			t.Fatalf("got %v, want ErrInsufficientPermissions", err)
		}
	})

	t.Run("the master's viewer sees everything", func(t *testing.T) {
		actorID := uuid.New()
		turn := match.HistoryTurn{
			UUID: uuid.New(), FinishedAt: time.Now(),
			Action:    actionWithClosedDodge(actorID),
			Reactions: []action.Action{},
		}
		matchMock := &testutil.MockMatchRepo{
			GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return privateMatch, nil
			},
			ListParticipantsByMatchUUIDFn: func(_ context.Context, _ uuid.UUID) ([]*matchEntity.Participant, error) {
				return nil, nil
			},
		}
		roundMock := &mockHistoryRoundRepo{
			fn: func(_ context.Context, _ uuid.UUID) ([]match.HistoryScene, error) {
				return historyWithTurns(turn), nil
			},
		}
		checker := &mockParticipationChecker{}

		uc := match.NewGetMatchHistoryUC(matchMock, roundMock, checker)
		result, err := uc.Get(context.Background(), matchUUID, masterUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := result.Scenes[0].Rounds[0].Turns[0].Action
		if got.ReactionKind != action.ReactClosedDodge {
			t.Fatalf("master's ReactionKind = %q, want closedDodge", got.ReactionKind)
		}
		foundEvasion := false
		for _, s := range got.Skills {
			if s.SkillName == enum.Evasion.String() {
				foundEvasion = true
			}
		}
		if !foundEvasion {
			t.Fatal("the master lost the Evasion skill entry")
		}
	})

	t.Run("a player's viewer owns exactly their own characters", func(t *testing.T) {
		playerUUID := uuid.New()
		otherPlayerUUID := uuid.New()
		ownCharUUID := uuid.New()
		otherCharUUID := uuid.New()

		// The other character's action also carries a closed dodge label — a per-sheet secret
		// that stays gated by Owns regardless of settlement, unlike the feint, which task 3
		// made time-gated instead of class-gated.
		otherAction := actionWithFeint(otherCharUUID)
		otherAction.ReactionKind = action.ReactClosedDodge

		ownTurn := match.HistoryTurn{
			UUID: uuid.New(), FinishedAt: time.Now(),
			Action:    actionWithFeint(ownCharUUID),
			Reactions: []action.Action{},
		}
		otherTurn := match.HistoryTurn{
			UUID: uuid.New(), FinishedAt: time.Now(),
			Action:    otherAction,
			Reactions: []action.Action{},
		}

		matchMock := &testutil.MockMatchRepo{
			GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return privateMatch, nil
			},
			ListParticipantsByMatchUUIDFn: func(_ context.Context, _ uuid.UUID) ([]*matchEntity.Participant, error) {
				return []*matchEntity.Participant{
					{Sheet: csEntity.Summary{UUID: ownCharUUID, PlayerUUID: &playerUUID}},
					{Sheet: csEntity.Summary{UUID: otherCharUUID, PlayerUUID: &otherPlayerUUID}},
				}, nil
			},
		}
		roundMock := &mockHistoryRoundRepo{
			fn: func(_ context.Context, _ uuid.UUID) ([]match.HistoryScene, error) {
				return historyWithTurns(ownTurn, otherTurn), nil
			},
		}
		checker := &mockParticipationChecker{fn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return true, nil
		}}

		uc := match.NewGetMatchHistoryUC(matchMock, roundMock, checker)
		result, err := uc.Get(context.Background(), matchUUID, playerUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		turns := result.Scenes[0].Rounds[0].Turns
		if turns[0].Action.Feint == nil {
			t.Fatal("the player was projected away from their OWN action's feint")
		}
		// The feint itself is NOT a distinguisher here: turns[1] is a closed turn (see
		// FinishedAt above), and task 3 made the feint public once the turn is settled,
		// regardless of who owns the actor. What Owns still gates on a closed turn is the
		// closed-dodge label, so that is what proves Owns is scoped per-sheet.
		if turns[1].Action.ReactionKind != action.ReactDodge {
			t.Fatalf("the player saw another character's closed dodge label — Owns leaked "+
				"beyond their own sheets: %q", turns[1].Action.ReactionKind)
		}
	})

	t.Run("a third party's turn arrives with the closed dodge demoted", func(t *testing.T) {
		thirdPartyUUID := uuid.New()
		actorID := uuid.New()

		turn := match.HistoryTurn{
			UUID: uuid.New(), FinishedAt: time.Now(),
			Action:    actionWithFeint(actorID),
			Reactions: []action.Action{actionWithClosedDodge(actorID)},
		}
		matchMock := &testutil.MockMatchRepo{
			GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return privateMatch, nil
			},
			ListParticipantsByMatchUUIDFn: func(_ context.Context, _ uuid.UUID) ([]*matchEntity.Participant, error) {
				return nil, nil
			},
		}
		roundMock := &mockHistoryRoundRepo{
			fn: func(_ context.Context, _ uuid.UUID) ([]match.HistoryScene, error) {
				return historyWithTurns(turn), nil
			},
		}
		checker := &mockParticipationChecker{fn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return true, nil
		}}

		uc := match.NewGetMatchHistoryUC(matchMock, roundMock, checker)
		result, err := uc.Get(context.Background(), matchUUID, thirdPartyUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		reaction := result.Scenes[0].Rounds[0].Turns[0].Reactions[0]
		if reaction.ReactionKind != action.ReactDodge {
			t.Fatalf("ReactionKind = %q, want dodge — the label is the leak", reaction.ReactionKind)
		}
		for _, s := range reaction.Skills {
			if s.SkillName == enum.Evasion.String() {
				t.Fatal("the Evasion entry leaked to a third party")
			}
		}
	})
}

// TestGetMatchHistoryProjectsEngineFaults is the surface-level half of the claim that
// TurnResolution.Errors is master-only. The deny-list itself lives in
// service.ProjectResolution and is pinned there; what this proves is that the REST read path
// actually RUNS it over the stored resolution — the errors were persisted with the turn, so
// without the projection they would reach every participant.
func TestGetMatchHistoryProjectsEngineFaults(t *testing.T) {
	masterUUID, playerUUID, matchUUID := uuid.New(), uuid.New(), uuid.New()
	ghost := uuid.New()

	publicMatch := &matchEntity.Match{
		UUID: matchUUID, MasterUUID: masterUUID, CampaignUUID: uuid.New(), IsPublic: true,
	}
	stored := func() []match.HistoryScene {
		return historyWithTurns(match.HistoryTurn{
			UUID: uuid.New(), FinishedAt: time.Now(),
			Action:    actionWithFeint(uuid.New()),
			Reactions: []action.Action{},
			Resolution: &service.TurnResolution{
				IsSettled: true,
				Errors: []service.ResolutionError{{
					Subject: ghost,
					Kind:    service.ResolutionErrUnknownTarget,
					Detail:  "action target is neither a character nor a wall segment",
				}},
			},
		})
	}
	newUC := func() *match.GetMatchHistoryUC {
		return match.NewGetMatchHistoryUC(
			&testutil.MockMatchRepo{
				GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
					return publicMatch, nil
				},
				ListParticipantsByMatchUUIDFn: func(_ context.Context, _ uuid.UUID) ([]*matchEntity.Participant, error) {
					return nil, nil
				},
			},
			&mockHistoryRoundRepo{
				fn: func(_ context.Context, _ uuid.UUID) ([]match.HistoryScene, error) {
					return stored(), nil
				},
			},
			&mockParticipationChecker{},
		)
	}
	resolutionOf := func(t *testing.T, res *match.GetMatchHistoryResult) *service.TurnResolution {
		t.Helper()
		return res.Scenes[0].Rounds[0].Turns[0].Resolution
	}

	t.Run("the master reads the engine's faults", func(t *testing.T) {
		got, err := newUC().Get(context.Background(), matchUUID, masterUUID)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		errs := resolutionOf(t, got).Errors
		if len(errs) != 1 || errs[0].Subject != ghost {
			t.Fatalf("Errors = %+v, want the stored unknown-target fault", errs)
		}
	})

	t.Run("a player does not", func(t *testing.T) {
		got, err := newUC().Get(context.Background(), matchUUID, playerUUID)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if errs := resolutionOf(t, got).Errors; len(errs) != 0 {
			t.Fatalf("a player read the engine's diagnostics: %+v", errs)
		}
	})
}

// TestGetMatchHistoryRevealsFeintOfAClosedTurnToAThirdParty is the wiring half of task 3's
// claim, the counterpart to TestProjectActionRevealsFeintOnceTheTurnIsSettled in the service
// package: it is not enough that ProjectAction reads isSettled correctly, the history path
// actually has to PASS it the real fact. Before this task, GetMatchHistoryUC called
// ProjectAction without a time axis at all, so a feint stayed hidden from its target forever —
// this proves the fix reaches the caller, not just the function it calls.
func TestGetMatchHistoryRevealsFeintOfAClosedTurnToAThirdParty(t *testing.T) {
	masterUUID, thirdPartyUUID, actorID, matchUUID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	publicMatch := &matchEntity.Match{
		UUID: matchUUID, MasterUUID: masterUUID, CampaignUUID: uuid.New(), IsPublic: true,
	}
	closedTurn := match.HistoryTurn{
		UUID: uuid.New(), FinishedAt: time.Now(), // closed: FinishedAt is set
		Action:    actionWithFeint(actorID),
		Reactions: []action.Action{},
	}
	matchMock := &testutil.MockMatchRepo{
		GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
			return publicMatch, nil
		},
		ListParticipantsByMatchUUIDFn: func(_ context.Context, _ uuid.UUID) ([]*matchEntity.Participant, error) {
			return nil, nil
		},
	}
	roundMock := &mockHistoryRoundRepo{
		fn: func(_ context.Context, _ uuid.UUID) ([]match.HistoryScene, error) {
			return historyWithTurns(closedTurn), nil
		},
	}
	uc := match.NewGetMatchHistoryUC(matchMock, roundMock, &mockParticipationChecker{})

	// thirdPartyUUID is neither the master nor an owner of actorID — exactly the target who
	// fell for the feint and, before this task, would never have found out.
	result, err := uc.Get(context.Background(), matchUUID, thirdPartyUUID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got := result.Scenes[0].Rounds[0].Turns[0].Action.Feint; got == nil {
		t.Fatal("a closed turn's feint stayed hidden from a non-owner reader — it would be hidden forever")
	}
}
