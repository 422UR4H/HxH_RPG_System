package service

import (
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/battle"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/google/uuid"
)

// TargetKind identifies the entity type a UUID refers to in an active match.
type TargetKind string

const (
	TargetKindCharacter   TargetKind = "character"    // checked first in CategorizeTarget
	TargetKindWallSegment TargetKind = "wall_segment"
	TargetKindUnknown     TargetKind = "unknown"
	// TODO: TargetKindFloorTile, TargetKindItem — future phases
)

// TargetReader allows TurnResolver to categorize and read action targets
// without importing matchsession (prevents circular imports).
// *matchsession.MatchSession implements this interface implicitly.
type TargetReader interface {
	CategorizeTarget(id uuid.UUID) TargetKind
	GetWall(id string) (mapentity.WallSegment, bool)
}

// WallResultKind discriminates attack vs interact outcomes in WallResult.
type WallResultKind string

const (
	WallResultKindAttack   WallResultKind = "attack"
	WallResultKindInteract WallResultKind = "interact"
)

// WallResult is the computed outcome of one player action targeting a wall.
type WallResult struct {
	UpdatedWall     mapentity.WallSegment
	EffectiveDamage int
	ReboundDamage   int // melee rebound candidate; TODO: apply to actor if melee, subtract actor Defense
	Kind            WallResultKind
}

// TurnResolution is the snapshot of a Turn's result — character combat, wall
// interactions, or any mix thereof.
type TurnResolution struct {
	ActionResult    RollResult
	ReactionResults []ReactionResult
	Blows           []*battle.Blow
	WallResults     []WallResult
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

// TurnResolver is a stateless domain service that calculates Turn resolution
// for any action type: character combat, wall attacks, door interactions, etc.
type TurnResolver struct{}

// Resolve calculates the current resolution snapshot for the given Turn.
// sheets maps participant UUIDs to their character sheets; nil is valid.
// targets is used to categorize action targets; nil disables wall routing.
func (tr TurnResolver) Resolve(
	t *turn.Turn,
	sheets map[uuid.UUID]*csSheet.CharacterSheet,
	targets TargetReader,
) *TurnResolution {
	res := &TurnResolution{
		IsSettled: t.GetFinishedAt() != nil,
	}

	if targets != nil {
		a := t.GetAction()
		for _, targetID := range a.TargetID {
			switch targets.CategorizeTarget(targetID) {
			case TargetKindCharacter:
				// TODO: implement character combat rolls (existing path)

			case TargetKindWallSegment:
				wall, ok := targets.GetWall(targetID.String())
				if !ok {
					continue
				}
				if a.Attack != nil {
					rawDamage := 0 // TODO: extract from a.Attack.Damage roll when contrato finalizar
					sdr := ApplyStructuralDamage(wall, rawDamage)
					res.WallResults = append(res.WallResults, WallResult{
						UpdatedWall:     sdr.UpdatedWall,
						EffectiveDamage: sdr.EffectiveDamage,
						ReboundDamage:   sdr.ReboundDamage,
						Kind:            WallResultKindAttack,
					})
				}
				if a.Interact != nil {
					updated, ok := ApplyWallInteract(wall, a.Interact)
					if ok {
						res.WallResults = append(res.WallResults, WallResult{
							UpdatedWall: updated,
							Kind:        WallResultKindInteract,
						})
					}
				}

			case TargetKindUnknown:
				// TODO: record unknown-target error in resolution for caller to surface
			}
		}
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
