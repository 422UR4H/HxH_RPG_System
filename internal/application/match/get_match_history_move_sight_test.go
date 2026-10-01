package match_test

import (
	"context"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/application/testutil"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

// The history shows each reader a move only as they saw it live (owner decision, 2026-10-01):
// the fog at read time is not the fog of that moment, so nothing is recomputed — the verdict
// recorded at the opening (actions.move_views) and at the settled resolution (the escape's
// landingViews) decides, and a row recorded before either existed fails closed.
func TestGetMatchHistoryShowsEachReaderTheMoveAsTheySawIt(t *testing.T) {
	masterUUID, ownerUUID, readerUUID, matchUUID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	actorID, escaperID := uuid.New(), uuid.New()

	dash := func() action.Action {
		return *action.NewAction(actorID, nil, uuid.Nil, nil, action.ActionSpeed{}, nil,
			&action.Move{Category: enum.Dash, From: &[3]int{1, 1, 0}, Position: [3]int{4, 4, 0}},
			nil, nil, nil, nil, nil)
	}
	landing := [3]int{7, 6, 0}
	escapeOf := func() *service.TurnResolution {
		return &service.TurnResolution{IsSettled: true, CharacterResults: []service.CharacterResult{{
			TargetID: escaperID,
			Escape:   &service.EscapeResult{DodgePassed: true, Landing: &landing},
		}}}
	}

	tests := []struct {
		name   string
		reader uuid.UUID
		// moveViews/landingViews are what was recorded live; nil is a row from before.
		moveViews    map[uuid.UUID]masteraction.View
		landingViews map[uuid.UUID]map[uuid.UUID]masteraction.View
		wantSight    match.MoveSight
		wantLanding  bool
	}{
		{
			name: "the master: the whole move and the landing, whatever was recorded", reader: masterUUID,
			wantSight: match.MoveSightWhole, wantLanding: true,
		},
		{
			name: "the actor's owner: the whole move, nothing recorded for them", reader: ownerUUID,
			moveViews: map[uuid.UUID]masteraction.View{},
			wantSight: match.MoveSightWhole,
		},
		{
			name: "a reader who saw the destination: the whole move", reader: readerUUID,
			moveViews:    map[uuid.UUID]masteraction.View{readerUUID: masteraction.ViewFull},
			landingViews: map[uuid.UUID]map[uuid.UUID]masteraction.View{escaperID: {readerUUID: masteraction.ViewFull}},
			wantSight:    match.MoveSightWhole, wantLanding: true,
		},
		{
			name: "a reader who saw only the piece leave: the origin", reader: readerUUID,
			moveViews: map[uuid.UUID]masteraction.View{readerUUID: masteraction.ViewLeft},
			wantSight: match.MoveSightOrigin,
		},
		{
			name: "a reader who saw nothing: the category alone, no landing", reader: readerUUID,
			moveViews:    map[uuid.UUID]masteraction.View{uuid.New(): masteraction.ViewFull},
			landingViews: map[uuid.UUID]map[uuid.UUID]masteraction.View{escaperID: {uuid.New(): masteraction.ViewFull}},
			wantSight:    match.MoveSightNone,
		},
		{
			name: "a row from before the views: fails closed", reader: readerUUID,
			wantSight: match.MoveSightNone,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stored := match.HistoryTurn{
				UUID: uuid.New(), FinishedAt: time.Now(),
				Action: dash(), Reactions: []action.Action{}, Resolution: escapeOf(),
				MoveViews: tt.moveViews, LandingViews: tt.landingViews,
			}
			uc := match.NewGetMatchHistoryUC(
				&testutil.MockMatchRepo{
					GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
						return &matchEntity.Match{UUID: matchUUID, MasterUUID: masterUUID, IsPublic: true}, nil
					},
					ListParticipantsByMatchUUIDFn: func(_ context.Context, _ uuid.UUID) ([]*matchEntity.Participant, error) {
						return []*matchEntity.Participant{
							{Sheet: csEntity.Summary{UUID: actorID, PlayerUUID: &ownerUUID}},
							{Sheet: csEntity.Summary{UUID: escaperID, PlayerUUID: &masterUUID}},
						}, nil
					},
				},
				&mockHistoryRoundRepo{fn: func(_ context.Context, _ uuid.UUID) ([]match.HistoryScene, error) {
					return historyWithTurns(stored), nil
				}},
				&mockParticipationChecker{}, nil, nil,
			)
			got, err := uc.Get(context.Background(), matchUUID, tt.reader)
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			tu := got.Scenes[0].Rounds[0].Turns[0]
			if tu.MoveSight != tt.wantSight {
				t.Errorf("MoveSight = %q, want %q", tu.MoveSight, tt.wantSight)
			}
			if shown := tu.ShownLandings[escaperID]; shown != tt.wantLanding {
				t.Errorf("landing shown = %v, want %v", shown, tt.wantLanding)
			}
			// Who saw what is not table data: the recorded views never leave the use case.
			if tu.MoveViews != nil || tu.LandingViews != nil {
				t.Errorf("the recorded views left the use case: move %v, landing %v", tu.MoveViews, tu.LandingViews)
			}
		})
	}

	t.Run("a turn whose action does not move: nothing to cut", func(t *testing.T) {
		stored := match.HistoryTurn{
			UUID: uuid.New(), FinishedAt: time.Now(),
			Action: actionWithFeint(actorID), Reactions: []action.Action{},
		}
		uc := match.NewGetMatchHistoryUC(
			&testutil.MockMatchRepo{
				GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{UUID: matchUUID, MasterUUID: masterUUID, IsPublic: true}, nil
				},
				ListParticipantsByMatchUUIDFn: func(_ context.Context, _ uuid.UUID) ([]*matchEntity.Participant, error) {
					return nil, nil
				},
			},
			&mockHistoryRoundRepo{fn: func(_ context.Context, _ uuid.UUID) ([]match.HistoryScene, error) {
				return historyWithTurns(stored), nil
			}},
			&mockParticipationChecker{}, nil, nil,
		)
		got, err := uc.Get(context.Background(), matchUUID, readerUUID)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if s := got.Scenes[0].Rounds[0].Turns[0].MoveSight; s != match.MoveSightWhole {
			t.Errorf("MoveSight = %q, want whole (there is no move)", s)
		}
	})
}

// The move of a REACTION is the same news: where an escaping character's piece went. Its action
// never reaches the table live, so the only way a reader learned that destination is the piece
// itself — the piece_moved of an escape that ESCAPED, judged at the close and recorded per player
// (LandingViews, keyed by the escaping character). A failed escape's move.position is a
// destination the piece never reached: nobody but the master and the owner is ever shown it.
func TestGetMatchHistoryShowsAReactionsMoveOnlyToWhoSawThePieceGoThere(t *testing.T) {
	masterUUID, ownerUUID, readerUUID, matchUUID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	attackerID, escaperID := uuid.New(), uuid.New()
	landing := [3]int{7, 6, 0}

	tests := []struct {
		name         string
		reader       uuid.UUID
		escaped      bool
		landingViews map[uuid.UUID]map[uuid.UUID]masteraction.View
		want         bool
	}{
		{name: "the master", reader: masterUUID, escaped: true, want: true},
		{name: "the escaper's owner, nothing recorded", reader: ownerUUID, escaped: true, want: true},
		{
			name: "escaped, the reader saw the piece arrive", reader: readerUUID, escaped: true,
			landingViews: map[uuid.UUID]map[uuid.UUID]masteraction.View{escaperID: {readerUUID: masteraction.ViewFull}},
			want:         true,
		},
		{
			name: "escaped, the reader did not", reader: readerUUID, escaped: true,
			landingViews: map[uuid.UUID]map[uuid.UUID]masteraction.View{escaperID: {uuid.New(): masteraction.ViewFull}},
		},
		{name: "escaped, a row from before the views: fails closed", reader: readerUUID, escaped: true},
		{
			name: "failed, the reader saw the landing: the attempted destination still never", reader: readerUUID,
			landingViews: map[uuid.UUID]map[uuid.UUID]masteraction.View{escaperID: {readerUUID: masteraction.ViewFull}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attack := *action.NewAction(attackerID, []uuid.UUID{escaperID}, uuid.Nil, nil, action.ActionSpeed{},
				nil, nil, &action.Attack{}, nil, nil, nil, nil)
			escape := *action.NewAction(escaperID, nil, attack.GetID(), nil, action.ActionSpeed{}, nil,
				&action.Move{Category: enum.Dash, Position: [3]int{9, 9, 0}}, nil, nil, &action.Dodge{}, nil, nil)
			escape.ReactionKind = action.ReactEscape
			esc := &service.EscapeResult{MovePassed: tt.escaped, DodgePassed: true, Escaped: tt.escaped}
			if !tt.escaped {
				esc.Landing = &landing
			}
			stored := match.HistoryTurn{
				UUID: uuid.New(), FinishedAt: time.Now(),
				Action: attack, Reactions: []action.Action{escape},
				Resolution: &service.TurnResolution{IsSettled: true, CharacterResults: []service.CharacterResult{{
					TargetID: escaperID, ReactionID: escape.GetID(), ReactionKind: string(action.ReactEscape), Escape: esc,
				}}},
				LandingViews: tt.landingViews,
			}
			uc := match.NewGetMatchHistoryUC(
				&testutil.MockMatchRepo{
					GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
						return &matchEntity.Match{UUID: matchUUID, MasterUUID: masterUUID, IsPublic: true}, nil
					},
					ListParticipantsByMatchUUIDFn: func(_ context.Context, _ uuid.UUID) ([]*matchEntity.Participant, error) {
						return []*matchEntity.Participant{{Sheet: csEntity.Summary{UUID: escaperID, PlayerUUID: &ownerUUID}}}, nil
					},
				},
				&mockHistoryRoundRepo{fn: func(_ context.Context, _ uuid.UUID) ([]match.HistoryScene, error) {
					return historyWithTurns(stored), nil
				}},
				&mockParticipationChecker{}, nil, nil,
			)
			got, err := uc.Get(context.Background(), matchUUID, tt.reader)
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			tu := got.Scenes[0].Rounds[0].Turns[0]
			if shown := tu.ShownReactionMoves[escape.GetID()]; shown != tt.want {
				t.Errorf("reaction move shown = %v, want %v", shown, tt.want)
			}
		})
	}
}
