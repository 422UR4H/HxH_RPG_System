package statuseffect_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

// TestResolveListing confere, sobre a matriz inteira do TestListed (tipos público/escondido,
// cada forma de lista, PC/NPC, a regra da partida), que a lista resolvida responde igual ao
// Listed para o dono, para outro jogador e para uuid.Nil.
func TestResolveListing(t *testing.T) {
	owner, other := uuid.New(), uuid.New()
	subjects := map[string]statuseffect.Subject{
		"pc":  {OwnerID: owner},
		"npc": {OwnerID: owner, IsNPC: true},
	}
	audiences := map[string]*statuseffect.Audience{
		"default":       nil,
		"everyone":      {Everyone: true},
		"empty":         {Players: []uuid.UUID{}},
		"nil players":   {},
		"[other]":       {Players: []uuid.UUID{other}},
		"[owner]":       {Players: []uuid.UUID{owner}},
		"[owner,other]": {Players: []uuid.UUID{owner, other}},
	}
	kinds := []statuseffect.Kind{statuseffect.KindProne, statuseffect.KindBleeding, statuseffect.KindHidden}
	readers := map[string]uuid.UUID{"owner": owner, "other": other, "nil": uuid.Nil}

	for _, k := range kinds {
		for an, a := range audiences {
			for sn, s := range subjects {
				for _, pc := range []bool{false, true} {
					e := statuseffect.Effect{Kind: k, Audience: a}
					got := statuseffect.ResolveListing(e, s, pc)
					name := fmt.Sprintf("%s/%s/%s/pcPublic=%v", k, an, sn, pc)
					t.Run(name, func(t *testing.T) {
						if !got.Everyone && got.Players == nil {
							t.Fatalf("Players nil: %+v", got)
						}
						if got.Everyone && len(got.Players) > 0 {
							t.Fatalf("everyone with players: %+v", got)
						}
						for rn, u := range readers {
							want := statuseffect.Listed(e, statuseffect.Viewer{UserID: u}, s, pc)
							if got.Lists(u) != want {
								t.Errorf("%s: Lists=%v, Listed=%v", rn, got.Lists(u), want)
							}
						}
					})
				}
			}
		}
	}

	t.Run("an explicit list is copied", func(t *testing.T) {
		a := &statuseffect.Audience{Players: []uuid.UUID{other}}
		got := statuseffect.ResolveListing(statuseffect.Effect{Kind: statuseffect.KindBleeding, Audience: a}, subjects["pc"], false)
		a.Players[0] = owner
		if got.Players[0] != other {
			t.Fatal("ResolveListing shares the effect's slice")
		}
	})
}

func TestAudienceLists(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	if !(statuseffect.Audience{Everyone: true}).Lists(a) {
		t.Error("everyone must list anyone")
	}
	if (statuseffect.Audience{Players: []uuid.UUID{}}).Lists(a) {
		t.Error("empty list lists no one")
	}
	if !(statuseffect.Audience{Players: []uuid.UUID{b, a}}).Lists(a) || (statuseffect.Audience{Players: []uuid.UUID{b}}).Lists(a) {
		t.Error("players list membership")
	}
}

func TestVisibleInHistory(t *testing.T) {
	reader, hiddenID, effectID := uuid.New(), uuid.New(), uuid.New()
	at := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	everyone := statuseffect.Audience{Everyone: true}
	masterOnly := statuseffect.Audience{Players: []uuid.UUID{}}

	inViews := statuseffect.EventRecord{
		Event:  statuseffect.Event{HappenedAt: at, Views: map[uuid.UUID]masteraction.View{reader: masteraction.ViewFull}},
		Effect: statuseffect.Effect{ID: effectID},
	}
	notInViews := statuseffect.EventRecord{
		Event:  statuseffect.Event{HappenedAt: at, Views: map[uuid.UUID]masteraction.View{}},
		Effect: statuseffect.Effect{ID: effectID},
	}
	hiddenFromReader := statuseffect.EventRecord{
		Event: statuseffect.Event{HappenedAt: at, Views: map[uuid.UUID]masteraction.View{},
			HiddenFrom: map[uuid.UUID]uuid.UUID{reader: hiddenID}},
		Effect: statuseffect.Effect{ID: effectID},
	}
	reveals := func(m map[uuid.UUID]time.Time) statuseffect.RevealLookup {
		return func(id uuid.UUID) (time.Time, bool) {
			t, ok := m[id]
			return t, ok
		}
	}
	reading := func(latest statuseffect.Audience, has bool, rv statuseffect.RevealLookup) statuseffect.HistoryReading {
		return statuseffect.HistoryReading{Reader: reader, Latest: latest, HasLatest: has, RevealedAt: rv, AnchorVisible: true}
	}

	tests := []struct {
		name string
		rec  statuseffect.EventRecord
		h    statuseffect.HistoryReading
		want bool
	}{
		{"master always, even with no latest and a hidden anchor", notInViews,
			statuseffect.HistoryReading{Reader: reader, IsMaster: true}, true},
		{"in views and latest lists", inViews, reading(everyone, true, nil), true},
		{"in views, latest does not list", inViews, reading(masterOnly, true, nil), false},
		{"listed but not in views", notInViews, reading(everyone, true, nil), false},
		{"in views, no latest", inViews, reading(statuseffect.Audience{}, false, nil), false},
		{"hidden anchor hides it", inViews, statuseffect.HistoryReading{Reader: reader, Latest: everyone, HasLatest: true}, false},
		{"hidden anchor hides even a revealed effect", inViews,
			statuseffect.HistoryReading{Reader: reader, RevealedAt: reveals(map[uuid.UUID]time.Time{effectID: at})}, false},
		{"effect revealed after the event shows it to who was not in sight", notInViews,
			reading(masterOnly, true, reveals(map[uuid.UUID]time.Time{effectID: at.Add(time.Minute)})), true},
		{"effect revealed at the instant counts", notInViews,
			reading(statuseffect.Audience{}, false, reveals(map[uuid.UUID]time.Time{effectID: at})), true},
		{"effect revealed before the event does not cover it", notInViews,
			reading(everyone, true, reveals(map[uuid.UUID]time.Time{effectID: at.Add(-time.Second)})), false},
		{"hidden from reader, Hidden never revealed", hiddenFromReader, reading(everyone, true, nil), false},
		{"hidden from reader, Hidden revealed later and latest lists", hiddenFromReader,
			reading(everyone, true, reveals(map[uuid.UUID]time.Time{hiddenID: at.Add(time.Minute)})), true},
		{"hidden from reader, Hidden revealed later but latest does not list", hiddenFromReader,
			reading(masterOnly, true, reveals(map[uuid.UUID]time.Time{hiddenID: at.Add(time.Minute)})), false},
		{"hidden from reader, Hidden revealed before the event", hiddenFromReader,
			reading(everyone, true, reveals(map[uuid.UUID]time.Time{hiddenID: at.Add(-time.Minute)})), false},
		{"a reveal of the Hidden does not help who was not hidden-from", notInViews,
			reading(everyone, true, reveals(map[uuid.UUID]time.Time{hiddenID: at.Add(time.Minute)})), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := statuseffect.VisibleInHistory(tt.rec, tt.h); got != tt.want {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}
}
