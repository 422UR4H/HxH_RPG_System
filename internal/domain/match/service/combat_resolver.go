package service

import (
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/battle"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/google/uuid"
)

// TurnResolution is the snapshot of a Turn's combat result.
type TurnResolution struct {
	ActionResult    RollResult
	ReactionResults []ReactionResult
	Blows           []*battle.Blow
	IsSettled       bool
}

// RollResult holds the outcome of a single dice roll check.
type RollResult struct {
	SkillName  string
	SkillValue int
	DiceRolled []int
	Total      int
}

// ReactionResult holds the outcome of one reaction within the Turn.
type ReactionResult struct {
	ReactorID uuid.UUID
	Roll      RollResult
}

// CombatResolver is a stateless domain service that calculates Turn resolution.
type CombatResolver struct{}

// Resolve calculates the current resolution snapshot for the given Turn.
// sheets maps participant UUIDs to their character sheets; nil is valid.
func (cr CombatResolver) Resolve(t *turn.Turn, sheets map[uuid.UUID]*csSheet.CharacterSheet) *TurnResolution {
	res := &TurnResolution{
		IsSettled: t.GetFinishedAt() != nil,
	}

	// TODO: implement ActionResult calculation using RollCalculator + sheets

	reactions := t.GetReactions()
	res.ReactionResults = make([]ReactionResult, len(reactions))
	for i, r := range reactions {
		// TODO: implement per-reaction resolution
		res.ReactionResults[i] = ReactionResult{ReactorID: r.ReactToID}
	}

	// TODO: populate Blows from attack/defense collision
	return res
}
