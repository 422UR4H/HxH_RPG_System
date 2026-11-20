package match_test

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	apiMatch "github.com/422UR4H/HxH_RPG_System/internal/app/api/match"
	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/application/testutil"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

// fakeRoundTree é o IRoundRepository só com a árvore do histórico.
type fakeRoundTree struct {
	match.IRoundRepository
	scenes []match.HistoryScene
}

func (f *fakeRoundTree) FindMatchHistory(context.Context, uuid.UUID) ([]match.HistoryScene, error) {
	return f.scenes, nil
}

// fakeRevealStore guarda as revelações em memória e é o leitor, o gravador e o verificador de
// eventos dos dois casos de uso — a mesma fonte, como o Repository real.
type fakeRevealStore struct {
	mu      sync.Mutex
	hasHid  uuid.UUID
	reveals map[uuid.UUID]time.Time
}

func (f *fakeRevealStore) ListEventsByMatch(context.Context, uuid.UUID) ([]statuseffect.EventRecord, error) {
	return nil, nil
}
func (f *fakeRevealStore) LatestListings(context.Context, []uuid.UUID) (map[uuid.UUID]statuseffect.Audience, error) {
	return map[uuid.UUID]statuseffect.Audience{}, nil
}
func (f *fakeRevealStore) HasEventInMatch(_ context.Context, _, eff uuid.UUID) (bool, error) {
	return eff == f.hasHid, nil
}
func (f *fakeRevealStore) InsertReveal(_ context.Context, _, eff uuid.UUID, at time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reveals[eff] = at
	return nil
}
func (f *fakeRevealStore) RevealedAt(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]time.Time, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[uuid.UUID]time.Time{}
	for _, id := range ids {
		if at, ok := f.reveals[id]; ok {
			out[id] = at
		}
	}
	return out, nil
}

// TestRevealStatusEffectOpensTheHistory: o histórico como uma terceira conta, antes e depois de
// o mestre revelar o Escondido pelo POST — as duas rotas reais sobre o mesmo armazenamento.
func TestRevealStatusEffectOpensTheHistory(t *testing.T) {
	master, owner, reader := uuid.New(), uuid.New(), uuid.New()
	matchUUID, hid := uuid.New(), uuid.New()
	ownersPC := uuid.New()
	opened := time.Now().Add(-time.Hour).UTC().Truncate(time.Microsecond)
	turnUUID := uuid.New()
	act := action.NewAction(ownersPC, nil, uuid.Nil, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil)

	store := &fakeRevealStore{hasHid: hid, reveals: map[uuid.UUID]time.Time{}}
	matchRepo := &testutil.MockMatchRepo{
		GetMatchFn: func(context.Context, uuid.UUID) (*matchEntity.Match, error) {
			return &matchEntity.Match{UUID: matchUUID, MasterUUID: master, IsPublic: true}, nil
		},
		ListParticipantsByMatchUUIDFn: func(context.Context, uuid.UUID) ([]*matchEntity.Participant, error) {
			return []*matchEntity.Participant{{Sheet: csEntity.Summary{UUID: ownersPC, PlayerUUID: &owner}}}, nil
		},
	}
	rounds := &fakeRoundTree{scenes: []match.HistoryScene{{UUID: uuid.New(), Rounds: []match.HistoryRound{{
		UUID: uuid.New(), Turns: []match.HistoryTurn{{
			UUID: turnUUID, CreatedAt: opened, FinishedAt: opened.Add(time.Second), Action: *act,
			ActionHiddenFrom: match.HiddenFrom{ownersPC: {Effect: hid}},
		}},
	}}}}}
	historyUC := match.NewGetMatchHistoryUC(matchRepo, rounds, nil, nil, nil, nil).
		WithStatusEffectEvents(store).WithStatusEffectReveals(store)
	revealUC := match.NewRevealStatusEffectUC(matchRepo, store, store)

	_, api := humatest.New(t)
	huma.Register(api, huma.Operation{Method: http.MethodGet, Path: "/matches/{uuid}/history"},
		apiMatch.GetMatchHistoryHandler(historyUC))
	huma.Register(api, huma.Operation{
		Method: http.MethodPost, Path: "/matches/{matchUuid}/status-effects/{effectUuid}/reveal",
		DefaultStatus: http.StatusNoContent,
	}, apiMatch.RevealStatusEffectHandler(revealUC))

	as := func(u uuid.UUID) context.Context { return context.WithValue(context.Background(), auth.UserIDKey, u) }
	turns := func(who uuid.UUID) int {
		t.Helper()
		resp := api.GetCtx(as(who), "/matches/"+matchUUID.String()+"/history")
		if resp.Code != http.StatusOK {
			t.Fatalf("history: %d %s", resp.Code, resp.Body.String())
		}
		var w struct {
			Scenes []struct {
				Rounds []struct {
					Turns []json.RawMessage `json:"turns"`
				} `json:"rounds"`
			} `json:"scenes"`
		}
		if err := json.Unmarshal(resp.Body.Bytes(), &w); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		return len(w.Scenes[0].Rounds[0].Turns)
	}
	revealPath := "/matches/" + matchUUID.String() + "/status-effects/" + hid.String() + "/reveal"

	if n := turns(reader); n != 0 {
		t.Fatalf("before: the third account sees %d turns, want 0", n)
	}
	if resp := api.PostCtx(as(reader), revealPath); resp.Code != http.StatusForbidden {
		t.Fatalf("a player revealing: %d, want 403", resp.Code)
	}
	if n := turns(reader); n != 0 {
		t.Fatal("a refused reveal opened the history")
	}
	if resp := api.PostCtx(as(master), revealPath); resp.Code != http.StatusNoContent {
		t.Fatalf("reveal: %d %s", resp.Code, resp.Body.String())
	}
	if n := turns(reader); n != 1 {
		t.Fatalf("after: the third account sees %d turns, want 1", n)
	}
}
