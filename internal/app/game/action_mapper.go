package game

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/google/uuid"
)

// buildAction maps an ActionPayload received from the WebSocket client to an Action domain entity.
// actorID is always the authenticated client's UUID — never trusted from the payload.
func buildAction(actorID uuid.UUID, p ActionPayload) *action.Action {
	var dodge *action.Dodge
	if p.Dodge != nil {
		var rc action.RollCheck
		if p.Dodge.RollCheck != nil {
			rc = action.RollCheck{SkillName: p.Dodge.RollCheck.SkillName}
		}
		dodge = &action.Dodge{
			Category:  enum.DodgeCategory(p.Dodge.Category),
			RollCheck: rc,
		}
	}
	var move *action.Move
	if p.Move != nil {
		move = &action.Move{
			Category: enum.MoveCategory(p.Move.Category),
			From:     p.Move.From,
			Position: p.Move.Position,
		}
		// TODO: map Speed, Charge, FinalSpeed once frontend contract is finalized
	}
	var interact *action.Interact
	if p.Interact != nil {
		interact = &action.Interact{Kind: action.InteractKind(p.Interact.Kind)}
	}
	// TODO: map Attack, Defense, Feint, Skills, Speed once frontend payload contract is finalized
	return action.NewAction(
		actorID, p.TargetID, p.ReactToID,
		nil, action.ActionSpeed{},
		nil, move, nil, nil, dodge, nil, interact,
	)
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
