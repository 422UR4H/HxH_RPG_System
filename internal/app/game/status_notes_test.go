package game

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

func TestStatusNotesFor(t *testing.T) {
	c := uuid.New()
	sum := statuseffect.Summarize([]statuseffect.Effect{
		{ID: uuid.New(), CharacterID: c, Kind: statuseffect.KindInjured, Params: statuseffect.Params{
			Injuries: map[statuseffect.Domain]statuseffect.InjuryLevel{statuseffect.DomainAttack: statuseffect.InjuryDisabled},
		}},
		{ID: uuid.New(), CharacterID: c, Kind: statuseffect.KindCompelled, Params: statuseffect.Params{Conduct: "Fugir"}},
	})

	t.Run("a domain warning only for a domain the action uses", func(t *testing.T) {
		move := action.NewAction(c, nil, uuid.Nil, nil, action.ActionSpeed{},
			nil, &action.Move{Category: enum.Dash}, nil, nil, nil, nil, nil)
		got := statusNotesFor(sum, move)
		if len(got) != 1 || got[0].Kind != "compelled" || got[0].Conduct != "Fugir" {
			t.Fatalf("notes = %+v, want only the conduct", got)
		}
	})

	t.Run("an attack carries the warning and the conduct", func(t *testing.T) {
		atk := action.NewAction(c, nil, uuid.Nil, nil, action.ActionSpeed{},
			nil, nil, &action.Attack{}, nil, nil, nil, nil)
		got := statusNotesFor(sum, atk)
		if len(got) != 2 || got[0].Kind != "injured" || got[0].Domain != "attack" {
			t.Fatalf("notes = %+v, want the attack warning then the conduct", got)
		}
	})

	t.Run("no statuses, no notes", func(t *testing.T) {
		atk := action.NewAction(c, nil, uuid.Nil, nil, action.ActionSpeed{},
			nil, nil, &action.Attack{}, nil, nil, nil, nil)
		if got := statusNotesFor(statuseffect.Summary{}, atk); got != nil {
			t.Fatalf("notes = %+v, want nil", got)
		}
	})
}
