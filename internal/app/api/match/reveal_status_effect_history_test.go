package match_test

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
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
	// eventSheet (efeito → ficha dos eventos dele, em qualquer partida) e participants (partida →
	// fichas) ligam a regra do RevealableInMatch real; nil = só hasHid é revelável.
	eventSheet   map[uuid.UUID]uuid.UUID
	participants map[uuid.UUID][]uuid.UUID
}

func (f *fakeRevealStore) ListEventsByMatch(context.Context, uuid.UUID) ([]statuseffect.EventRecord, error) {
	return nil, nil
}
func (f *fakeRevealStore) LatestListings(context.Context, []uuid.UUID) (map[uuid.UUID]statuseffect.Audience, error) {
	return map[uuid.UUID]statuseffect.Audience{}, nil
}
func (f *fakeRevealStore) RevealableInMatch(_ context.Context, m, eff uuid.UUID) (bool, error) {
	if f.eventSheet == nil {
		return eff == f.hasHid, nil
	}
	sheet, ok := f.eventSheet[eff]
	return ok && slices.Contains(f.participants[m], sheet), nil
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

// TestRevealStatusEffectFromAnotherMatch (revisão final do PR 2, achado 3): um Escondido aplicado
// na partida A e ainda ativo na B esconde turnos da B, e o concealedBy da B o nomeia — o mestre
// da B o revela (204) e o histórico da B mostra o turno. Um efeito de ficha fora da B e o
// uuid.Nil são 404.
func TestRevealStatusEffectFromAnotherMatch(t *testing.T) {
	master, reader := uuid.New(), uuid.New()
	matchA, matchB := uuid.New(), uuid.New()
	npc, outsiderSheet := uuid.New(), uuid.New()
	hid, foreign := uuid.New(), uuid.New()
	opened := time.Now().Add(-time.Hour).UTC().Truncate(time.Microsecond)
	act := action.NewAction(npc, nil, uuid.Nil, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil)

	store := &fakeRevealStore{
		reveals:      map[uuid.UUID]time.Time{},
		eventSheet:   map[uuid.UUID]uuid.UUID{hid: npc, foreign: outsiderSheet}, // eventos gravados na A
		participants: map[uuid.UUID][]uuid.UUID{matchA: {npc, outsiderSheet}, matchB: {npc}},
	}
	matchRepo := &testutil.MockMatchRepo{
		GetMatchFn: func(_ context.Context, id uuid.UUID) (*matchEntity.Match, error) {
			return &matchEntity.Match{UUID: id, MasterUUID: master, IsPublic: true}, nil
		},
		ListParticipantsByMatchUUIDFn: func(context.Context, uuid.UUID) ([]*matchEntity.Participant, error) {
			return []*matchEntity.Participant{{Sheet: csEntity.Summary{UUID: npc}}}, nil
		},
	}
	rounds := &fakeRoundTree{scenes: []match.HistoryScene{{UUID: uuid.New(), Rounds: []match.HistoryRound{{
		UUID: uuid.New(), Turns: []match.HistoryTurn{{
			UUID: uuid.New(), CreatedAt: opened, FinishedAt: opened.Add(time.Second), Action: *act,
			ActionHiddenFrom: match.HiddenFrom{npc: {Effect: hid}},
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
	reveal := func(eff uuid.UUID) int {
		return api.PostCtx(as(master), "/matches/"+matchB.String()+"/status-effects/"+eff.String()+"/reveal").Code
	}
	turnsB := func() int {
		t.Helper()
		resp := api.GetCtx(as(reader), "/matches/"+matchB.String()+"/history")
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

	if n := turnsB(); n != 0 {
		t.Fatalf("before: B's history shows %d turns, want 0", n)
	}
	if code := reveal(foreign); code != http.StatusNotFound {
		t.Fatalf("an effect whose sheet is not in B: %d, want 404", code)
	}
	if code := reveal(uuid.Nil); code != http.StatusNotFound {
		t.Fatalf("uuid.Nil: %d, want 404", code)
	}
	if code := reveal(hid); code != http.StatusNoContent {
		t.Fatalf("the Hidden applied in A, revealed from B: %d, want 204", code)
	}
	if n := turnsB(); n != 1 {
		t.Fatalf("after: B's history shows %d turns, want 1", n)
	}
}
