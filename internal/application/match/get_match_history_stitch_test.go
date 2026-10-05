package match_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/application/testutil"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchevent"
	"github.com/google/uuid"
)

type fakeEventLister struct{ events []matchevent.Event }

func (f *fakeEventLister) ListByMatch(_ context.Context, _ uuid.UUID) ([]matchevent.Event, error) {
	return f.events, nil
}

type fakeMasterActionLister struct{ recs []masteraction.Record }

func (f *fakeMasterActionLister) ListByMatch(_ context.Context, _ uuid.UUID) ([]masteraction.Record, error) {
	return f.recs, nil
}

// TestGetMatchHistoryStitchesEventsAndMasterActions is B15 + §4.8: the use case reads the tree,
// the round's regime changes and the match's master actions, and stitches the last two in —
// each master action projected as its reader saw it live, inside the turn it was recorded with
// when that turn is in the tree, and in the round's events otherwise.
func TestGetMatchHistoryStitchesEventsAndMasterActions(t *testing.T) {
	masterUUID, matchUUID := uuid.New(), uuid.New()
	sawAll, sawLeave, sawNothing := uuid.New(), uuid.New(), uuid.New()
	sceneUUID, roundUUID, editedTurn, turnUUID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	lostTurn := uuid.New() // recorded with a master action, never written: lost in a restart
	base := time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)
	at := func(sec int) time.Time { return base.Add(time.Duration(sec) * time.Second) }

	move := func(from, to [3]int) json.RawMessage {
		b, _ := json.Marshal(masteraction.PieceContent{CharacterID: "c", PieceID: "p", From: &from, To: &to})
		return b
	}
	pieceViews := map[uuid.UUID]masteraction.View{sawAll: masteraction.ViewFull, sawLeave: masteraction.ViewLeft}
	rec := func(kind masteraction.Kind, turn *uuid.UUID, happened time.Time, content json.RawMessage, views map[uuid.UUID]masteraction.View) masteraction.Record {
		return masteraction.Record{
			UUID: uuid.New(), MatchUUID: matchUUID, SceneUUID: sceneUUID, RoundUUID: roundUUID,
			MasterUUID: masterUUID, TurnUUID: turn, Kind: kind, Content: content, Views: views,
			HappenedAt: happened,
		}
	}

	note := rec(masteraction.KindTurnNote, &turnUUID, at(10), json.RawMessage(`{"note":"x"}`), map[uuid.UUID]masteraction.View{})
	inTurn := rec(masteraction.KindMovePiece, &turnUUID, at(11), move([3]int{1, 1, 0}, [3]int{2, 2, 0}), pieceViews)
	lost := rec(masteraction.KindRemovePiece, &lostTurn, at(20), json.RawMessage(`{"characterId":"c","pieceId":"p","from":[2,2,0]}`),
		map[uuid.UUID]masteraction.View{sawAll: masteraction.ViewFull})
	outside := rec(masteraction.KindMovePiece, nil, at(40), move([3]int{2, 2, 0}, [3]int{5, 5, 0}), pieceViews)
	strayRound := rec(masteraction.KindMovePiece, nil, at(41), move([3]int{0, 0, 0}, [3]int{1, 0, 0}), pieceViews)
	strayRound.RoundUUID = uuid.New() // a round the tree does not hold: logged and dropped

	modeChange := matchevent.Event{
		UUID: uuid.New(), MatchUUID: matchUUID, SceneUUID: sceneUUID, RoundUUID: roundUUID,
		Kind: matchevent.KindRoundModeChanged, Payload: json.RawMessage(`{"from":"Free","to":"Race"}`),
		CreatedAt: at(30),
	}
	strayEvent := modeChange
	strayEvent.UUID, strayEvent.RoundUUID = uuid.New(), uuid.New()

	tree := func() []match.HistoryScene {
		return []match.HistoryScene{{
			UUID: sceneUUID, Category: "battle",
			Rounds: []match.HistoryRound{{
				UUID: roundUUID, Mode: "Race",
				Turns: []match.HistoryTurn{
					// The master edited this turn's action: edit_action hangs a MasterAction on
					// the live turn, but it is NOT a master action — no Record exists for it,
					// so the turn's masterActions must still come out empty.
					{UUID: editedTurn, FinishedAt: at(5), Action: actionWithFeint(uuid.New()), Reactions: []action.Action{}},
					{UUID: turnUUID, FinishedAt: at(12), Action: actionWithFeint(uuid.New()), Reactions: []action.Action{}},
				},
			}},
		}}
	}

	newUC := func() *match.GetMatchHistoryUC {
		return match.NewGetMatchHistoryUC(
			&testutil.MockMatchRepo{
				GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{UUID: matchUUID, MasterUUID: masterUUID, IsPublic: true}, nil
				},
				ListParticipantsByMatchUUIDFn: func(_ context.Context, _ uuid.UUID) ([]*matchEntity.Participant, error) {
					return nil, nil
				},
			},
			&mockHistoryRoundRepo{fn: func(_ context.Context, _ uuid.UUID) ([]match.HistoryScene, error) {
				return tree(), nil
			}},
			&mockParticipationChecker{},
			&fakeEventLister{events: []matchevent.Event{strayEvent, modeChange}},
			// Deliberately out of time order: the use case orders, it does not trust the feed.
			&fakeMasterActionLister{recs: []masteraction.Record{outside, inTurn, strayRound, lost, note}},
		)
	}

	type entry struct {
		kind   match.HistoryEventKind
		uuid   uuid.UUID
		leftTo bool // a master action whose destination must have been erased
	}
	tests := []struct {
		name       string
		reader     uuid.UUID
		wantInTurn []uuid.UUID
		wantEvents []entry
		leftInTurn bool
	}{
		{
			name:       "the master sees every master action, inside and outside the turn, whole",
			reader:     masterUUID,
			wantInTurn: []uuid.UUID{note.UUID, inTurn.UUID},
			wantEvents: []entry{
				{kind: match.HistoryEventMasterAction, uuid: lost.UUID},
				{kind: match.HistoryEventRoundModeChanged, uuid: modeChange.UUID},
				{kind: match.HistoryEventMasterAction, uuid: outside.UUID},
			},
		},
		{
			name:       "a player who saw it whole reads it whole",
			reader:     sawAll,
			wantInTurn: []uuid.UUID{inTurn.UUID},
			wantEvents: []entry{
				{kind: match.HistoryEventMasterAction, uuid: lost.UUID},
				{kind: match.HistoryEventRoundModeChanged, uuid: modeChange.UUID},
				{kind: match.HistoryEventMasterAction, uuid: outside.UUID},
			},
		},
		{
			name:       "a player who only saw the piece leave reads the move without its destination",
			reader:     sawLeave,
			wantInTurn: []uuid.UUID{inTurn.UUID},
			leftInTurn: true,
			wantEvents: []entry{
				{kind: match.HistoryEventRoundModeChanged, uuid: modeChange.UUID},
				{kind: match.HistoryEventMasterAction, uuid: outside.UUID, leftTo: true},
			},
		},
		{
			name:       "a player who saw nothing reads no master action at all — only the public regime change",
			reader:     sawNothing,
			wantInTurn: []uuid.UUID{},
			wantEvents: []entry{
				{kind: match.HistoryEventRoundModeChanged, uuid: modeChange.UUID},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := newUC().Get(context.Background(), matchUUID, tt.reader)
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			round := res.Scenes[0].Rounds[0]

			if got := round.Turns[0].MasterActions; got == nil || len(got) != 0 {
				t.Fatalf("the edited turn's masterActions = %#v, want a non-nil empty slice — edit_action is not a master action", got)
			}

			got := round.Turns[1].MasterActions
			if got == nil || len(got) != len(tt.wantInTurn) {
				t.Fatalf("turn masterActions = %d (%#v), want %d", len(got), got, len(tt.wantInTurn))
			}
			for i, want := range tt.wantInTurn {
				if got[i].UUID != want {
					t.Fatalf("turn masterActions[%d] = %s, want %s (time order)", i, got[i].UUID, want)
				}
				if tt.reader != masterUUID && got[i].Views != nil {
					t.Fatalf("a player's projection carries Views: %v", got[i].Views)
				}
			}
			if len(got) > 0 && got[len(got)-1].Kind == masteraction.KindMovePiece {
				assertTo(t, got[len(got)-1].Content, !tt.leftInTurn)
			}

			if round.Events == nil || len(round.Events) != len(tt.wantEvents) {
				t.Fatalf("events = %d (%+v), want %d", len(round.Events), round.Events, len(tt.wantEvents))
			}
			for i, want := range tt.wantEvents {
				ev := round.Events[i]
				if ev.Kind != want.kind {
					t.Fatalf("events[%d].kind = %q, want %q", i, ev.Kind, want.kind)
				}
				switch want.kind {
				case match.HistoryEventRoundModeChanged:
					if ev.RoundModeChange == nil || ev.RoundModeChange.UUID != want.uuid || ev.MasterAction != nil {
						t.Fatalf("events[%d] = %+v, want the regime change %s alone", i, ev, want.uuid)
					}
					if !ev.At.Equal(modeChange.CreatedAt) {
						t.Fatalf("events[%d].At = %v, want %v", i, ev.At, modeChange.CreatedAt)
					}
				case match.HistoryEventMasterAction:
					if ev.MasterAction == nil || ev.MasterAction.UUID != want.uuid || ev.RoundModeChange != nil {
						t.Fatalf("events[%d] = %+v, want the master action %s alone", i, ev, want.uuid)
					}
					if !ev.At.Equal(ev.MasterAction.HappenedAt) {
						t.Fatalf("events[%d].At = %v, want its HappenedAt %v", i, ev.At, ev.MasterAction.HappenedAt)
					}
					if ev.MasterAction.Kind == masteraction.KindMovePiece {
						assertTo(t, ev.MasterAction.Content, !want.leftTo)
					}
				}
			}
		})
	}

	// Review focus 6, on its own: the turn a master action was recorded with was lost in a
	// restart (it only becomes a row when it closes). The master action did happen — the saved
	// board confirms it — so it must not vanish: it comes back as the round's event.
	t.Run("a master action whose turn was lost in a restart lands in the round's events", func(t *testing.T) {
		res, err := newUC().Get(context.Background(), matchUUID, masterUUID)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		round := res.Scenes[0].Rounds[0]
		for _, tu := range round.Turns {
			for _, ma := range tu.MasterActions {
				if ma.UUID == lost.UUID {
					t.Fatalf("the lost turn's master action landed inside turn %s", tu.UUID)
				}
			}
		}
		found := false
		for _, ev := range round.Events {
			if ev.Kind == match.HistoryEventMasterAction && ev.MasterAction.UUID == lost.UUID {
				found = true
				if ev.MasterAction.TurnUUID == nil || *ev.MasterAction.TurnUUID != lostTurn {
					t.Fatalf("the record's turnUuid was rewritten: %v", ev.MasterAction.TurnUUID)
				}
			}
		}
		if !found {
			t.Fatal("the master action whose turn was lost vanished from the history")
		}
	})

	t.Run("a history with no event and no master action still carries empty slices", func(t *testing.T) {
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
				return tree(), nil
			}},
			&mockParticipationChecker{},
			&fakeEventLister{}, &fakeMasterActionLister{},
		)
		res, err := uc.Get(context.Background(), matchUUID, masterUUID)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		round := res.Scenes[0].Rounds[0]
		if round.Events == nil || len(round.Events) != 0 {
			t.Fatalf("events = %#v, want a non-nil empty slice", round.Events)
		}
		for _, tu := range round.Turns {
			if tu.MasterActions == nil || len(tu.MasterActions) != 0 {
				t.Fatalf("turn %s masterActions = %#v, want a non-nil empty slice", tu.UUID, tu.MasterActions)
			}
		}
	})
}

// assertTo checks whether a piece Content still carries its destination.
func assertTo(t *testing.T, content json.RawMessage, wantTo bool) {
	t.Helper()
	var pc masteraction.PieceContent
	if err := json.Unmarshal(content, &pc); err != nil {
		t.Fatalf("unmarshal piece content %s: %v", content, err)
	}
	if wantTo && pc.To == nil {
		t.Fatalf("content %s lost its destination — this reader saw it", content)
	}
	if !wantTo && pc.To != nil {
		t.Fatalf("content %s kept its destination — this reader only saw the piece leave", content)
	}
	if pc.From == nil {
		t.Fatalf("content %s lost its origin", content)
	}
}
