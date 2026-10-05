package actionwire_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/wire/actionwire"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/google/uuid"
)

// fixtureAction builds one action carrying every RollCheck-bearing field the cut table
// touches — attack (hit + damage), move (with a moveSpeed roll and a FinalSpeed), skills,
// and a feint — with every RollCheck fully populated (SkillValue, Attempts AND Result all
// non-zero), so that a level which cuts a field is unambiguously distinguishable from one
// that merely happened to carry a zero.
func fixtureAction() action.Action {
	rc := func(value, result int) action.RollCheck {
		return action.RollCheck{
			SkillName: enum.Legerity.String(), SkillValue: value,
			Attempts: action.RollAttempts{Primary: []int{3, 4}}, Result: result,
		}
	}
	feint := rc(2, 9)
	moveSpeed := rc(4, 13)
	weapon := enum.Dagger

	a := action.NewAction(
		uuid.New(), []uuid.UUID{uuid.New()}, uuid.Nil,
		[]action.Skill{{SkillName: enum.Legerity.String(), RollCheck: rc(4, 11)}},
		action.ActionSpeed{Bar: 2, RollCheck: rc(4, 14)},
		&feint,
		&action.Move{
			Category: enum.Dash, Position: [3]int{5, 5, 0},
			Speed: &moveSpeed, FinalSpeed: 13,
		},
		&action.Attack{
			Weapon: &weapon, Hit: rc(5, 17), Damage: rc(6, 12), RelativeVelocity: 8.5,
		},
		nil, nil, nil, nil,
	)
	return *a
}

func TestFromCutsByLevel(t *testing.T) {
	a := fixtureAction() // attack + move + skills + feint, every RollCheck with Attempts and Result
	full := actionwire.From(a, actionwire.Full)
	opened := actionwire.From(a, actionwire.Opened)
	decl := actionwire.From(a, actionwire.Declaration)

	t.Run("full keeps every number", func(t *testing.T) {
		if full.Attack.Hit.Result == nil || full.Speed.RollCheck.Result == nil || full.Move.FinalSpeed == nil {
			t.Fatal("full must keep hit, speed and finalSpeed")
		}
	})
	t.Run("opened keeps the speeds and cuts the rest", func(t *testing.T) {
		if opened.Speed.RollCheck.Result == nil || opened.Move.Speed.Result == nil || opened.Move.FinalSpeed == nil {
			t.Fatal("opened must keep actionSpeed and moveSpeed")
		}
		for name, rc := range map[string]actionwire.RollCheck{
			"hit": opened.Attack.Hit, "damage": opened.Attack.Damage, "skill": opened.Skills[0].RollCheck, "feint": *opened.Feint,
		} {
			if rc.Result != nil || rc.Attempts != nil || rc.SkillValue != nil || rc.SkillName == "" {
				t.Errorf("%s: want only the skill name, got %+v", name, rc)
			}
		}
	})
	t.Run("declaration keeps no number at all", func(t *testing.T) {
		if decl.Speed.RollCheck.Result != nil || decl.Move.FinalSpeed != nil || decl.Move.Speed.Result != nil {
			t.Fatal("declaration must carry no speed")
		}
		if decl.Move.Position == nil || *decl.Move.Position != a.Move.Position || len(decl.TargetID) != len(a.TargetID) || decl.Attack.Weapon == nil {
			t.Fatal("declaration must keep what the owner declared")
		}
	})
}

func TestFromCarriesConsumedActionIDs(t *testing.T) {
	id := uuid.New()
	a := fixtureAction()
	a.ConsumedActionIDs = []uuid.UUID{id}
	for name, lvl := range map[string]actionwire.Level{
		"full": actionwire.Full, "opened": actionwire.Opened, "declaration": actionwire.Declaration,
	} {
		t.Run(name, func(t *testing.T) {
			got := actionwire.From(a, lvl).ConsumedActionIDs
			if len(got) != 1 || got[0] != id {
				t.Fatalf("ConsumedActionIDs = %v, want [%v]", got, id)
			}
		})
	}
	t.Run("nil is off the wire", func(t *testing.T) {
		raw, err := json.Marshal(actionwire.From(fixtureAction(), actionwire.Full))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), "consumedActionIds") {
			t.Fatalf("consumedActionIds present on an action that consumed nothing: %s", raw)
		}
	})
}
