package statuswire_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/wire/statuswire"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

func TestFrom(t *testing.T) {
	ctrl := uuid.MustParse("00000000-0000-0000-0000-0000000000cc")
	cases := []struct {
		name   string
		effect statuseffect.Effect
		params string // JSON esperado de params
	}{
		{"paralyzed", statuseffect.Effect{Kind: statuseffect.KindParalyzed, Params: statuseffect.Params{
			Domains: []statuseffect.Domain{statuseffect.DomainAttack, statuseffect.DomainDodge}, Points: 3}},
			`{"domains":["attack","dodge"],"points":3}`},
		{"stupefied", statuseffect.Effect{Kind: statuseffect.KindStupefied, Params: statuseffect.Params{Points: 2}},
			`{"points":2}`},
		{"stunned", statuseffect.Effect{Kind: statuseffect.KindStunned, Params: statuseffect.Params{Points: 4}},
			`{"points":4}`},
		{"unconscious", statuseffect.Effect{Kind: statuseffect.KindUnconscious}, `{}`},
		{"restrained", statuseffect.Effect{Kind: statuseffect.KindRestrained}, `{}`},
		{"injured", statuseffect.Effect{Kind: statuseffect.KindInjured, Params: statuseffect.Params{
			Injuries: map[statuseffect.Domain]statuseffect.InjuryLevel{statuseffect.DomainMovement: statuseffect.InjuryDisabled}}},
			`{"injuries":{"movement":"disabled"}}`},
		{"slowed", statuseffect.Effect{Kind: statuseffect.KindSlowed, Params: statuseffect.Params{Penalty: 2}},
			`{"penalty":2}`},
		{"sense impaired", statuseffect.Effect{Kind: statuseffect.KindSense, Sense: statuseffect.SenseVision,
			Params: statuseffect.Params{Level: statuseffect.SenseImpaired, Disadvantage: true}},
			`{"level":"impaired","penalty":0,"disadvantage":true}`},
		{"sense nullified drops disadvantage and penalty", statuseffect.Effect{Kind: statuseffect.KindSense,
			Sense: statuseffect.SenseHearing, Params: statuseffect.Params{Level: statuseffect.SenseNullified, Disadvantage: true}},
			`{"level":"nullified"}`},
		{"tact carries faces", statuseffect.Effect{Kind: statuseffect.KindSense, Sense: statuseffect.SenseTact,
			Params: statuseffect.Params{Level: statuseffect.SenseNullified, Faces: []statuseffect.TouchFace{statuseffect.FaceBalance}}},
			`{"level":"nullified","faces":["balance"]}`},
		{"weakened", statuseffect.Effect{Kind: statuseffect.KindWeakened, Aptitude: statuseffect.AptitudeMental,
			Params: statuseffect.Params{Penalty: 1}},
			`{"penalty":1,"disadvantage":false}`},
		{"bleeding", statuseffect.Effect{Kind: statuseffect.KindBleeding, Params: statuseffect.Params{HPPerRound: 1}},
			`{"hpPerRound":1}`},
		{"burning", statuseffect.Effect{Kind: statuseffect.KindBurning, Params: statuseffect.Params{HPPerRound: 2}},
			`{"hpPerRound":2}`},
		{"freezing", statuseffect.Effect{Kind: statuseffect.KindFreezing, Params: statuseffect.Params{HPPerRound: 1}},
			`{"hpPerRound":1}`},
		{"poisoned", statuseffect.Effect{Kind: statuseffect.KindPoisoned, Params: statuseffect.Params{HPPerRound: 3}},
			`{"hpPerRound":3}`},
		{"asphyxiating", statuseffect.Effect{Kind: statuseffect.KindAsphyxiating}, `{}`},
		{"prone", statuseffect.Effect{Kind: statuseffect.KindProne}, `{}`},
		{"airborne", statuseffect.Effect{Kind: statuseffect.KindAirborne}, `{}`},
		{"openGuard", statuseffect.Effect{Kind: statuseffect.KindOpenGuard}, `{}`},
		{"compelled", statuseffect.Effect{Kind: statuseffect.KindCompelled, Params: statuseffect.Params{
			Conduct: "fugir", Irresistible: true}},
			`{"conduct":"fugir","irresistible":true}`},
		{"controlled", statuseffect.Effect{Kind: statuseffect.KindControlled, Params: statuseffect.Params{ControllerID: &ctrl}},
			`{"controllerId":"00000000-0000-0000-0000-0000000000cc"}`},
		{"hidden", statuseffect.Effect{Kind: statuseffect.KindHidden}, `{}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := json.Marshal(statuswire.From(c.effect).Params)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != c.params {
				t.Fatalf("params = %s, want %s", got, c.params)
			}
		})
	}

	t.Run("identity fields and no visibility", func(t *testing.T) {
		id := uuid.New()
		at := time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)
		e := statuseffect.Effect{ID: id, Kind: statuseffect.KindSense, Sense: statuseffect.SenseSmell,
			Alias: "nariz", AppliedAt: at, Audience: &statuseffect.Audience{Everyone: true},
			Params: statuseffect.Params{Level: statuseffect.SenseNullified}}
		w := statuswire.From(e)
		if w.ID != id || w.Kind != "sense" || w.Sense != "smell" || w.Alias != "nariz" || !w.AppliedAt.Equal(at) {
			t.Fatalf("wire = %+v", w)
		}
		if w.Visibility != nil || w.Public != nil {
			t.Fatalf("From must not fill visibility/public: %+v", w)
		}
	})
}

func TestVisibilityOf(t *testing.T) {
	p := uuid.New()
	cases := []struct {
		name string
		in   *statuseffect.Audience
		want string
	}{
		{"default", nil, `{"mode":"default"}`},
		{"public", &statuseffect.Audience{Everyone: true}, `{"mode":"public"}`},
		{"master only", &statuseffect.Audience{Players: []uuid.UUID{}}, `{"mode":"players"}`},
		{"players", &statuseffect.Audience{Players: []uuid.UUID{p}}, `{"mode":"players","playerIds":["` + p.String() + `"]}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, _ := json.Marshal(statuswire.VisibilityOf(c.in))
			if string(got) != c.want {
				t.Fatalf("got %s, want %s", got, c.want)
			}
		})
	}
}
