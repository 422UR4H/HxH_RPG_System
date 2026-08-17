package battle

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/google/uuid"
)

// Blow is one attack meeting one target: the pairing of the attack with the defense that
// answered it, and the skills each side put behind them.
//
// It carries no numbers on purpose. The rolled outcome of a collision lives in
// service.CharacterResult, because the domain service imports this package and the reverse
// would be an import cycle. Blow is the *shape* of the exchange; the service holds the
// arithmetic.
type Blow struct {
	actorID      uuid.UUID
	targetID     uuid.UUID
	attack       action.Attack
	attackSkill  *action.Skill
	defense      *action.Defense // nil when the target defended passively
	defenseSkill *action.Skill
}

func NewBlow(
	actorID, targetID uuid.UUID,
	attack action.Attack,
	attackSkill *action.Skill,
	defense *action.Defense,
	defenseSkill *action.Skill,
) *Blow {
	return &Blow{
		actorID:      actorID,
		targetID:     targetID,
		attack:       attack,
		attackSkill:  attackSkill,
		defense:      defense,
		defenseSkill: defenseSkill,
	}
}

func (b *Blow) GetActorID() uuid.UUID          { return b.actorID }
func (b *Blow) GetTargetID() uuid.UUID         { return b.targetID }
func (b *Blow) GetAttack() action.Attack       { return b.attack }
func (b *Blow) GetAttackSkill() *action.Skill  { return b.attackSkill }
func (b *Blow) GetDefense() *action.Defense    { return b.defense }
func (b *Blow) GetDefenseSkill() *action.Skill { return b.defenseSkill }
