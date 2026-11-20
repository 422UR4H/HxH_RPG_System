package match_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

// TestGetMatchHistoryCarriesStatusEffects: um evento de status fora de turno vai em
// rounds[].events com kind "statusEffect" e o efeito na forma do fio (statuswire); o de dentro
// do turno, em turns[].statusEffects (sempre lista, nunca null). A visibilidade e o public só
// saem quando o caso de uso os pôs (mestre).
func TestGetMatchHistoryCarriesStatusEffects(t *testing.T) {
	now := time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)
	turnUUID, pc, anchor := uuid.New(), uuid.New(), uuid.New()
	act := action.NewAction(uuid.New(), nil, uuid.Nil, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil)
	prone := statuseffect.Effect{ID: uuid.New(), CharacterID: pc, Kind: statuseffect.KindProne, AppliedAt: now}
	poisoned := statuseffect.Effect{ID: uuid.New(), CharacterID: pc, Kind: statuseffect.KindPoisoned,
		Params: statuseffect.Params{HPPerRound: 1}, AppliedAt: now}

	tree := func(master bool) []match.HistoryScene {
		out := match.HistoryStatusEffect{UUID: uuid.New(), Change: statuseffect.ChangeApplied, Effect: prone, HappenedAt: now}
		in := match.HistoryStatusEffect{UUID: uuid.New(), Change: statuseffect.ChangeRemoved, Effect: poisoned,
			TurnUUID: &turnUUID, ActionID: &anchor, HappenedAt: now.Add(time.Second)}
		if master {
			pub, priv := true, false
			out.Public, in.Public = &pub, &priv
			in.Effect.Audience = &statuseffect.Audience{Players: []uuid.UUID{}}
		}
		return []match.HistoryScene{{
			UUID: uuid.New(), Category: "battle", CreatedAt: now,
			Rounds: []match.HistoryRound{{
				UUID: uuid.New(), Mode: "Race", CreatedAt: now,
				Turns: []match.HistoryTurn{
					{UUID: turnUUID, CreatedAt: now, FinishedAt: now, Action: *act, Reactions: []action.Action{},
						StatusEffects: []match.HistoryStatusEffect{in}},
					// Antes do caso de uso (nil): ainda [] no fio.
					{UUID: uuid.New(), CreatedAt: now, FinishedAt: now, Action: *act, Reactions: []action.Action{}},
				},
				Events: []match.HistoryEvent{{Kind: match.HistoryEventStatusEffect, At: now, StatusEffect: &out}},
			}},
		}}
	}

	type wireEffect struct {
		Kind       string          `json:"kind"`
		Params     json.RawMessage `json:"params"`
		Visibility *struct {
			Mode string `json:"mode"`
		} `json:"visibility"`
		Public *bool `json:"public"`
	}
	type wireStatus struct {
		UUID         uuid.UUID  `json:"uuid"`
		Change       string     `json:"change"`
		CharacterID  uuid.UUID  `json:"characterId"`
		TurnID       *uuid.UUID `json:"turnId"`
		ActionID     *uuid.UUID `json:"actionId"`
		HappenedAt   string     `json:"happenedAt"`
		StatusEffect wireEffect `json:"statusEffect"`
	}
	type wire struct {
		Scenes []struct {
			Rounds []struct {
				Turns []struct {
					StatusEffects []wireStatus `json:"statusEffects"`
				} `json:"turns"`
				Events []struct {
					Kind         string      `json:"kind"`
					CreatedAt    string      `json:"createdAt"`
					StatusEffect *wireStatus `json:"statusEffect"`
				} `json:"events"`
			} `json:"rounds"`
		} `json:"scenes"`
	}
	decode := func(t *testing.T, body []byte) wire {
		t.Helper()
		var w wire
		if err := json.Unmarshal(body, &w); err != nil {
			t.Fatalf("unmarshal: %v\n%s", err, body)
		}
		return w
	}

	t.Run("a player's copy", func(t *testing.T) {
		body := getHistory(t, tree(false))
		if strings.Contains(string(body), `"statusEffects":null`) {
			t.Fatalf("statusEffects is null: %s", body)
		}
		if strings.Contains(string(body), `"visibility"`) || strings.Contains(string(body), `"public"`) {
			t.Fatalf("a player's history carries the viewer list: %s", body)
		}
		ro := decode(t, body).Scenes[0].Rounds[0]
		if len(ro.Events) != 1 || ro.Events[0].Kind != "statusEffect" || ro.Events[0].StatusEffect == nil {
			t.Fatalf("events = %+v", ro.Events)
		}
		ev := ro.Events[0].StatusEffect
		if ev.Change != "applied" || ev.CharacterID != pc || ev.TurnID != nil || ev.ActionID != nil ||
			ev.StatusEffect.Kind != "prone" || ev.HappenedAt != now.Format(time.RFC3339) {
			t.Fatalf("round status event = %+v", ev)
		}
		if ro.Events[0].CreatedAt != ev.HappenedAt {
			t.Fatalf("createdAt = %s, want happenedAt %s", ro.Events[0].CreatedAt, ev.HappenedAt)
		}
		if len(ro.Turns[1].StatusEffects) != 0 || ro.Turns[1].StatusEffects == nil {
			t.Fatalf("empty turn statusEffects = %v, want []", ro.Turns[1].StatusEffects)
		}
		in := ro.Turns[0].StatusEffects
		if len(in) != 1 || in[0].Change != "removed" || in[0].TurnID == nil || *in[0].TurnID != turnUUID ||
			in[0].ActionID == nil || *in[0].ActionID != anchor || string(in[0].StatusEffect.Params) != `{"hpPerRound":1}` {
			t.Fatalf("turn status events = %+v", in)
		}
	})

	t.Run("the master's copy has visibility and public", func(t *testing.T) {
		ro := decode(t, getHistory(t, tree(true))).Scenes[0].Rounds[0]
		out := ro.Events[0].StatusEffect.StatusEffect
		if out.Visibility == nil || out.Visibility.Mode != "default" || out.Public == nil || !*out.Public {
			t.Fatalf("master round event effect = %+v", out)
		}
		in := ro.Turns[0].StatusEffects[0].StatusEffect
		if in.Visibility == nil || in.Visibility.Mode != "players" || in.Public == nil || *in.Public {
			t.Fatalf("master turn event effect = %+v", in)
		}
	})
}

// TestGetMatchHistoryHiddenMarksAndNanoTimes: o mestre recebe concealedBy no turno e revealed
// nos eventos de status; o jogador, nenhum dos dois. happenedAt/createdAt de eventos de round,
// master actions e eventos de status levam a fração de segundo (o front ordena dentro do turno
// pelo instante).
func TestGetMatchHistoryHiddenMarksAndNanoTimes(t *testing.T) {
	now := time.Date(2026, 11, 19, 10, 0, 0, 123456000, time.UTC)
	turnUUID, hid := uuid.New(), uuid.New()
	act := action.NewAction(uuid.New(), nil, uuid.Nil, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil)
	eff := statuseffect.Effect{ID: uuid.New(), CharacterID: uuid.New(), Kind: statuseffect.KindProne, AppliedAt: now}
	tree := func(master bool) []match.HistoryScene {
		se := match.HistoryStatusEffect{UUID: uuid.New(), Change: statuseffect.ChangeApplied, Effect: eff,
			TurnUUID: &turnUUID, HappenedAt: now}
		tu := match.HistoryTurn{UUID: turnUUID, CreatedAt: now, FinishedAt: now, Action: *act,
			Reactions: []action.Action{}, StatusEffects: []match.HistoryStatusEffect{se}}
		if master {
			pub, rev := true, false
			tu.StatusEffects[0].Public, tu.StatusEffects[0].Revealed = &pub, &rev
			tu.ConcealedBy = []uuid.UUID{hid}
		}
		round := se
		round.TurnUUID = nil
		return []match.HistoryScene{{
			UUID: uuid.New(), Category: "battle", CreatedAt: now,
			Rounds: []match.HistoryRound{{
				UUID: uuid.New(), Mode: "Race", CreatedAt: now, Turns: []match.HistoryTurn{tu},
				Events: []match.HistoryEvent{{Kind: match.HistoryEventStatusEffect, At: now, StatusEffect: &round}},
			}},
		}}
	}
	type wire struct {
		Scenes []struct {
			Rounds []struct {
				Turns []struct {
					ConcealedBy   []uuid.UUID `json:"concealedBy"`
					StatusEffects []struct {
						HappenedAt string `json:"happenedAt"`
						Revealed   *bool  `json:"revealed"`
					} `json:"statusEffects"`
				} `json:"turns"`
				Events []struct {
					CreatedAt    string `json:"createdAt"`
					StatusEffect struct {
						HappenedAt string `json:"happenedAt"`
					} `json:"statusEffect"`
				} `json:"events"`
			} `json:"rounds"`
		} `json:"scenes"`
	}
	read := func(t *testing.T, body []byte) wire {
		t.Helper()
		var w wire
		if err := json.Unmarshal(body, &w); err != nil {
			t.Fatalf("unmarshal: %v\n%s", err, body)
		}
		return w
	}
	nano := now.Format(time.RFC3339Nano)

	t.Run("the master's marks", func(t *testing.T) {
		ro := read(t, getHistory(t, tree(true))).Scenes[0].Rounds[0]
		tu := ro.Turns[0]
		if len(tu.ConcealedBy) != 1 || tu.ConcealedBy[0] != hid {
			t.Fatalf("concealedBy = %v, want [%s]", tu.ConcealedBy, hid)
		}
		if r := tu.StatusEffects[0].Revealed; r == nil || *r {
			t.Fatalf("revealed = %v, want false", r)
		}
		if tu.StatusEffects[0].HappenedAt != nano || ro.Events[0].CreatedAt != nano || ro.Events[0].StatusEffect.HappenedAt != nano {
			t.Fatalf("times = %s / %s / %s, want %s", tu.StatusEffects[0].HappenedAt, ro.Events[0].CreatedAt,
				ro.Events[0].StatusEffect.HappenedAt, nano)
		}
	})

	t.Run("a player gets no marks", func(t *testing.T) {
		body := getHistory(t, tree(false))
		if strings.Contains(string(body), `"concealedBy"`) || strings.Contains(string(body), `"revealed"`) {
			t.Fatalf("a player's history carries the master's marks: %s", body)
		}
	})
}

// TestGetMatchHistoryServesReactionOpenings (Task 15b): cada reação aberta leva o openedAt dela
// (RFC3339Nano); a nunca aberta, nada. Só no histórico — o fio do WS não muda.
func TestGetMatchHistoryServesReactionOpenings(t *testing.T) {
	now := time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)
	openedAt := now.Add(1500 * time.Millisecond)
	act := action.NewAction(uuid.New(), nil, uuid.Nil, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil)
	mk := func() action.Action {
		return *action.NewAction(uuid.New(), nil, act.GetID(), nil, action.ActionSpeed{}, nil, nil, nil, nil, &action.Dodge{}, nil, nil)
	}
	opened, never := mk(), mk()
	tree := []match.HistoryScene{{
		UUID: uuid.New(), Category: "battle", CreatedAt: now,
		Rounds: []match.HistoryRound{{
			UUID: uuid.New(), Mode: "Race", CreatedAt: now,
			Turns: []match.HistoryTurn{{
				UUID: uuid.New(), CreatedAt: now, FinishedAt: now.Add(time.Minute), Action: *act,
				Reactions:        []action.Action{opened, never},
				ReactionOpenedAt: map[uuid.UUID]time.Time{opened.GetID(): openedAt},
			}},
		}},
	}}
	var w struct {
		Scenes []struct {
			Rounds []struct {
				Turns []struct {
					Reactions []map[string]json.RawMessage `json:"reactions"`
				} `json:"turns"`
			} `json:"rounds"`
		} `json:"scenes"`
	}
	body := getHistory(t, tree)
	if err := json.Unmarshal(body, &w); err != nil {
		t.Fatal(err)
	}
	rs := w.Scenes[0].Rounds[0].Turns[0].Reactions
	if len(rs) != 2 {
		t.Fatalf("reactions = %d, want 2", len(rs))
	}
	if got := string(rs[0]["openedAt"]); got != `"`+openedAt.Format(time.RFC3339Nano)+`"` {
		t.Fatalf("opened reaction openedAt = %s, want %s", got, openedAt.Format(time.RFC3339Nano))
	}
	if _, ok := rs[1]["openedAt"]; ok {
		t.Fatalf("never-opened reaction carries openedAt: %s", rs[1]["openedAt"])
	}
	if _, ok := rs[0]["uuid"]; !ok {
		t.Fatalf("the reaction lost its actionwire fields: %v", rs[0])
	}
}
