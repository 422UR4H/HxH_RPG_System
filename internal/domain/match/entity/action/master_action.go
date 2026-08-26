package action

import (
	"time"

	"github.com/google/uuid"
)

// ConditionField names which RollCheck inside an Action a condition edit lands on.
//
// A path rather than a pointer because the edit arrives over the wire, from a client that
// holds no Go references — and because naming the field is what the audit stores.
type ConditionField string

const (
	FieldSpeed     ConditionField = "speed"
	FieldHit       ConditionField = "hit"
	FieldDamage    ConditionField = "damage"
	FieldDodge     ConditionField = "dodge"
	FieldDefense   ConditionField = "defense"
	FieldRepel     ConditionField = "repel"
	FieldFeint     ConditionField = "feint"
	FieldMoveSpeed ConditionField = "moveSpeed"
)

// ConditionEdit is the master changing HOW one test is read: the dice bias, a flat
// adjustment, and the reason both are surfaced by.
//
// Field and SkillName are alternatives: SkillName targets one entry of Skills (each is a test
// with its own DC), Field targets one of the action's fixed checks. Setting both is a client
// bug and is refused at the boundary.
type ConditionEdit struct {
	Field     ConditionField
	SkillName string
	Condition RollCondition
}

type MasterAction struct {
	// ActionID names which action of the OPEN turn this lands on — the turn's own action, or
	// one of its reactions. Zero means the turn's own action.
	ActionID uuid.UUID
	TargetID []uuid.UUID
	Skills   []Skill
	// Conditions is the master changing how existing tests are read. Skills and TargetID
	// change WHICH tests exist; this changes how they are read. The two surfaces are
	// deliberately separate — see combat-engine.md § A edição do mestre.
	Conditions  []ConditionEdit
	Move        *Move
	Attack      *Attack
	ActionSpeed *RollCheck
	Interact    *Interact
	happenedAt  time.Time
	// Initiative *Initiative ?
	// Penalidade *Penalty ?
}

func NewMasterAction() *MasterAction {
	return &MasterAction{}
}

func (ma *MasterAction) GetHappenedAt() time.Time {
	return ma.happenedAt
}

func (ma *MasterAction) SetHappenedAt(t time.Time) {
	ma.happenedAt = t
}

func (ma *MasterAction) GetSkills() []Skill {
	skillsCopy := make([]Skill, len(ma.Skills))
	copy(skillsCopy, ma.Skills)
	return skillsCopy
}
