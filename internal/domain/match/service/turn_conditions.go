package service

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/google/uuid"
)

// CheckCondition is one master condition in force on one check of the open turn — what the
// master's editor needs to show what is in force after a reload (front-combat-phases.md §0.2).
// Field and SkillName are alternatives, exactly as in edit_action: Field names a fixed check,
// SkillName an entry of Skills.
type CheckCondition struct {
	ActionID  uuid.UUID
	Field     string
	SkillName string
	Condition action.RollCondition
}

// turnConditions lists every check of the turn that carries a master condition: the turn's own
// action first, then each attached reaction (opened or not) in the turn's order. Inside one
// action the order is fixed — speed, feint, hit, damage, dodge, defense, repel, moveSpeed, then
// each skill — so two reads of the same turn list the same thing in the same order.
//
// It reads the live action, which is where the condition lives ("the edited action IS the
// action", combat-engine.md); there is no second copy to keep in sync.
func turnConditions(t *turn.Turn) []CheckCondition {
	var out []CheckCondition
	out = appendActionConditions(out, t.GetAction())
	for _, r := range t.GetReactions() {
		out = appendActionConditions(out, r)
	}
	return out
}

func appendActionConditions(out []CheckCondition, a action.Action) []CheckCondition {
	add := func(field action.ConditionField, rc *action.RollCheck) {
		if rc == nil || rc.Context.Condition == nil {
			return
		}
		out = append(out, CheckCondition{
			ActionID: a.GetID(), Field: string(field), Condition: *rc.Context.Condition,
		})
	}
	add(action.FieldSpeed, &a.Speed.RollCheck)
	add(action.FieldFeint, a.Feint)
	if a.Attack != nil {
		add(action.FieldHit, &a.Attack.Hit)
		add(action.FieldDamage, &a.Attack.Damage)
	}
	if a.Dodge != nil {
		add(action.FieldDodge, &a.Dodge.RollCheck)
	}
	// "defense" names the declared Defense on a plain action and the DefaultDefense on a
	// reaction that keeps it (resolveRollCheck) — never both editable on the same action.
	if a.Defense != nil {
		add(action.FieldDefense, &a.Defense.RollCheck)
	}
	add(action.FieldDefense, a.DefaultDefense)
	if a.Repel != nil {
		add(action.FieldRepel, &a.Repel.RollCheck)
	}
	if a.Move != nil {
		add(action.FieldMoveSpeed, a.Move.Speed)
	}
	for _, s := range a.Skills {
		if s.Context.Condition == nil {
			continue
		}
		out = append(out, CheckCondition{
			ActionID: a.GetID(), SkillName: s.SkillName, Condition: *s.Context.Condition,
		})
	}
	return out
}
