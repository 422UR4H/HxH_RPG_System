package game

import (
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/google/uuid"
)

// buildAction maps an ActionPayload received from the WebSocket client to an Action domain
// entity.
//
// actorCharID is the acting CHARACTER's sheet UUID, taken from the payload by the caller and
// checked against the authenticated player by the session. The combat entity is the
// character: one person drives several of them.
//
// The mapper never rolls. The dice fall in MatchSession the moment the action is accepted,
// so a payload that is rejected costs nothing and an accepted one is rolled exactly once.
func buildAction(actorCharID uuid.UUID, p ActionPayload) (*action.Action, error) {
	skills, err := buildSkills(p.Skills)
	if err != nil {
		return nil, err
	}

	// actionSpeed is ALWAYS Legerity. Whatever the payload carried is discarded: the
	// player picks a move type or an attack, never the skill behind its speed.
	// ActionSpeedPayload.Bar is likewise ignored — which bar an action pays from is derived
	// from its content by Action.Bars(), never trusted from the client.
	speed := action.ActionSpeed{
		RollCheck: action.RollCheck{SkillName: enum.Legerity.String()},
	}

	feint, err := buildRollCheck(p.Feint)
	if err != nil {
		return nil, err
	}

	var move *action.Move
	if p.Move != nil {
		category, moveSkill, err := moveSpeedSkill(p.Move.Category)
		if err != nil {
			return nil, err
		}
		moveCharge, err := buildRollCheck(p.Move.Charge)
		if err != nil {
			return nil, err
		}
		move = &action.Move{
			Category: category,
			From:     p.Move.From,
			Position: p.Move.Position,
			// The skill comes from the category, never from the payload. The front shows the
			// tactical move types explicitly; switching Dash to Shift in the bottom sheet
			// switches the skill on its own.
			Speed:  &action.RollCheck{SkillName: moveSkill.String()},
			Charge: moveCharge,
		}
		// FinalSpeed is computed by the engine, never taken from the client.
	}

	var attack *action.Attack
	if p.Attack != nil {
		hit, err := buildRollCheck(&p.Attack.Hit)
		if err != nil {
			return nil, err
		}
		damage, err := buildRollCheck(&p.Attack.Damage)
		if err != nil {
			return nil, err
		}
		charge, err := buildRollCheck(p.Attack.Charge)
		if err != nil {
			return nil, err
		}
		weapon, err := buildWeaponName(p.Attack.Weapon)
		if err != nil {
			return nil, err
		}
		attack = &action.Attack{
			Weapon: weapon,
			Hit:    *hit,
			Damage: *damage,
			Charge: charge,
		}
	}

	var defense *action.Defense
	if p.Defense != nil {
		rc, err := buildRollCheck(&p.Defense.RollCheck)
		if err != nil {
			return nil, err
		}
		weapon, err := buildWeaponName(p.Defense.Weapon)
		if err != nil {
			return nil, err
		}
		defense = &action.Defense{Weapon: weapon, RollCheck: *rc}
	}

	var dodge *action.Dodge
	if p.Dodge != nil {
		rc, err := buildRollCheck(p.Dodge.RollCheck)
		if err != nil {
			return nil, err
		}
		dodge = &action.Dodge{Category: enum.DodgeCategory(p.Dodge.Category)}
		if rc != nil {
			dodge.RollCheck = *rc
		}
	}

	var interact *action.Interact
	if p.Interact != nil {
		interact = &action.Interact{Kind: action.InteractKind(p.Interact.Kind)}
	}

	return action.NewAction(
		actorCharID, p.TargetID, p.ReactToID,
		skills, speed,
		feint, move, attack, defense, dodge, nil, interact,
	), nil
}

// buildRollCheck crosses the string→enum boundary for a skill name. An unknown name is a
// client bug and comes back as an error here, where it can still be answered with a WS
// error, instead of contributing a silent zero deep inside the resolver.
func buildRollCheck(p *RollCheckPayload) (*action.RollCheck, error) {
	if p == nil {
		return nil, nil
	}
	if p.SkillName != "" {
		if _, err := enum.SkillNameFrom(p.SkillName); err != nil {
			return nil, err
		}
	}
	return &action.RollCheck{SkillName: p.SkillName}, nil
}

func buildSkills(ps []ActionSkillPayload) ([]action.Skill, error) {
	if len(ps) == 0 {
		return nil, nil
	}
	out := make([]action.Skill, 0, len(ps))
	for _, s := range ps {
		if _, err := enum.SkillNameFrom(s.SkillName); err != nil {
			return nil, err
		}
		out = append(out, action.Skill{
			SkillName:  s.SkillName,
			Difficulty: s.Difficulty,
			RollCheck:  action.RollCheck{SkillName: s.SkillName},
		})
	}
	return out, nil
}

func buildWeaponName(s *string) (*enum.WeaponName, error) {
	if s == nil {
		return nil, nil
	}
	name, err := enum.WeaponNameFrom(*s)
	if err != nil {
		return nil, err
	}
	return &name, nil
}

// moveSpeedSkill maps a movement category to the skill its speed is tested on.
//
//	Dash  (arrancada)            → Accelerate, rolled
//	Shift (controlled step)      → Brake, taken passively (see MatchSession.deriveSpeeds)
//
// enum.MoveCategory has five other values — Back (cait), Roll, Slide, Jump, FlatJump — and
// they are REFUSED here rather than mapped. Their skills belong to the movement slice, which
// is where they will actually be exercised. Mapping them by analogy would work silently and
// wrongly: a leap would cost like a controlled step, and nobody would find out until someone
// complained at the table.
func moveSpeedSkill(raw string) (enum.MoveCategory, enum.SkillName, error) {
	switch enum.MoveCategory(raw) {
	case enum.Dash:
		return enum.Dash, enum.Accelerate, nil
	case enum.Shift:
		return enum.Shift, enum.Brake, nil
	default:
		return "", "", fmt.Errorf("move category %q is not supported yet", raw)
	}
}

// buildMasterAction maps a MasterActionPayload received from the WebSocket client to a MasterAction domain entity.
// masterUUID is always the authenticated master's UUID — never trusted from the payload.
func buildMasterAction(masterUUID uuid.UUID, p MasterActionPayload) *action.MasterAction {
	_ = masterUUID
	ma := action.NewMasterAction()
	ma.TargetID = p.TargetIDs
	if p.ActionSpeed != nil {
		ma.ActionSpeed = &action.RollCheck{SkillName: p.ActionSpeed.SkillName}
	}
	for _, s := range p.Skills {
		ma.Skills = append(ma.Skills, action.Skill{SkillName: s.SkillName})
	}
	if p.Move != nil {
		// TODO: map Move fully once frontend contract is finalized
		_ = p.Move
	}
	if p.Attack != nil {
		// TODO: map Attack once frontend contract is finalized
		_ = p.Attack
	}
	if p.Interact != nil {
		ma.Interact = &action.Interact{Kind: action.InteractKind(p.Interact.Kind)}
	}
	return ma
}
