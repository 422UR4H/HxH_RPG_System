package battle_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/battle"
	"github.com/google/uuid"
)

func TestNewBlow(t *testing.T) {
	actorID, targetID := uuid.New(), uuid.New()
	sword := enum.Sword
	attack := action.Attack{
		Weapon: &sword,
		Hit:    action.RollCheck{SkillName: "Accuracy", SkillValue: 4},
		Damage: action.RollCheck{SkillName: "Strike"},
	}

	t.Run("carries the pairing it was built with", func(t *testing.T) {
		b := battle.NewBlow(actorID, targetID, attack, nil, nil, nil)

		if b.GetActorID() != actorID {
			t.Errorf("GetActorID() = %v, want %v", b.GetActorID(), actorID)
		}
		if b.GetTargetID() != targetID {
			t.Errorf("GetTargetID() = %v, want %v", b.GetTargetID(), targetID)
		}
		if b.GetAttack().Hit.SkillName != "Accuracy" {
			t.Errorf("GetAttack().Hit.SkillName = %q, want Accuracy", b.GetAttack().Hit.SkillName)
		}
	})

	t.Run("a passive defense leaves the defense nil", func(t *testing.T) {
		b := battle.NewBlow(actorID, targetID, attack, nil, nil, nil)
		if b.GetDefense() != nil {
			t.Error("expected a nil defense when the target defended passively")
		}
	})

	t.Run("an explicit defense is carried through", func(t *testing.T) {
		dagger := enum.Dagger
		def := &action.Defense{Weapon: &dagger}
		b := battle.NewBlow(actorID, targetID, attack, nil, def, nil)
		if b.GetDefense() == nil || b.GetDefense().Weapon == nil || *b.GetDefense().Weapon != enum.Dagger {
			t.Errorf("GetDefense() = %+v, want the dagger defense", b.GetDefense())
		}
	})

	t.Run("the skills each side put behind the exchange survive", func(t *testing.T) {
		atkSkill := &action.Skill{SkillName: enum.Accuracy.String()}
		defSkill := &action.Skill{SkillName: enum.Defense.String()}
		b := battle.NewBlow(actorID, targetID, attack, atkSkill, nil, defSkill)

		if b.GetAttackSkill() == nil || b.GetAttackSkill().SkillName != enum.Accuracy.String() {
			t.Errorf("GetAttackSkill() = %+v, want Accuracy", b.GetAttackSkill())
		}
		if b.GetDefenseSkill() == nil || b.GetDefenseSkill().SkillName != enum.Defense.String() {
			t.Errorf("GetDefenseSkill() = %+v, want Defense", b.GetDefenseSkill())
		}
	})
}
