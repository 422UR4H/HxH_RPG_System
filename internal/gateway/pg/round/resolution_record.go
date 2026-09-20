package round

import (
	"encoding/json"
	"log"

	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

// The persisted shape of a settled resolution.
//
// It is an EXPLICIT record rather than json.Marshal on service.TurnResolution, for two
// reasons that are not style:
//
//   - battle.Blow has none of its fields exported, so it would serialize as {} — a silent
//     hole. It is also derived and carries no numbers (see its own doc), so it is dropped
//     here on purpose rather than half-written.
//   - match.Scope keeps kind and id private, so a Payout's Against would be lost the same
//     silent way. It travels through Scope.Kind()/ID() and comes back through ScopeFrom.
//
// Being explicit cuts both ways: every omission below is CHOSEN, not accidental, and has to
// be justified in this comment or in the record's own fields. Three fields of
// service.TurnResolution are dropped on purpose:
//
//   - Blows ([]*battle.Blow) — the same reason as above: Blow carries no numbers by its own
//     doc, only the shape of the exchange. The arithmetic it points at is already on
//     CharacterResult, which this record does keep.
//   - ReactionResults ([]service.ReactionResult) — Resolve's own TODO says this branch is
//     unimplemented: every entry is built as ReactionResult{ReactorID: r.ReactToID} with a
//     zero Roll. Persisting a stub that carries no real roll would be worse than omitting
//     it — it would look settled.
//   - PendingReactions ([]service.PendingReaction) — transient master-side to-do state (see
//     its own doc comment on TurnResolution): the reactions attached but not yet given the
//     floor. It describes what has NOT happened yet, not an outcome of the collision, so it
//     has nothing to persist once the turn is closed and settled.
//
// Tags are camelCase, like every other wire shape in this repo.
type resolutionRecord struct {
	IsSettled    bool                    `json:"isSettled"`
	ActionResult rollResultRecord        `json:"actionResult"`
	Characters   []characterResultRecord `json:"characters"`
	// WallResults holds the wall's ID, not its full mapentity.WallSegment: the wall's
	// geometry and live state already live in the map tables, and copying them into a turn
	// snapshot would create a second source of truth that drifts the moment the wall
	// changes again. Only the outcome of THIS turn's hit on it is this record's business.
	WallResults []wallResultRecord `json:"wallResults,omitempty"`
}

type rollResultRecord struct {
	SkillName         string `json:"skillName"`
	SkillValue        int    `json:"skillValue"`
	DiceRolled        []int  `json:"diceRolled"`
	Total             int    `json:"total"`
	IsCritical        bool   `json:"isCritical"`
	IsCriticalFailure bool   `json:"isCriticalFailure"`
	Margin            *int   `json:"margin,omitempty"`
}

// rollOutcomeRecord mirrors service.RollOutcome field-for-field, in order, for the same
// direct-type-conversion reason as rollResultRecord below: RollOutcome and RollResult are
// different structs (RollOutcome carries Bias/Modifier/Passive/DiceTotal, the DERIVED numbers
// the action's own RollCheck.Attempts JSON does not have) and must not be conflated. This is
// what Task 6 exists to save for Hit/Dodge/Defense: the dice themselves are already
// persisted via the action row; the ladder-independent roll math is not, without this.
type rollOutcomeRecord struct {
	SkillName         string `json:"skillName"`
	SkillValue        int    `json:"skillValue"`
	Dice              []int  `json:"dice,omitempty"`
	DiceTotal         int    `json:"diceTotal"`
	Bias              int    `json:"bias"`
	Modifier          int    `json:"modifier"`
	Passive           bool   `json:"passive"`
	Total             int    `json:"total"`
	IsCritical        bool   `json:"isCritical"`
	IsCriticalFailure bool   `json:"isCriticalFailure"`
}

type characterResultRecord struct {
	TargetID uuid.UUID `json:"targetId"`
	// ReactionID is a real sentinel, not an omission: uuid.Nil already means "no reaction
	// was opened" throughout this codebase (see action.Action.ReactToID), so it is written
	// as-is rather than elided — omitempty is a no-op on a fixed-size [16]byte array anyway
	// (see modifierRecord.AgainstID below).
	ReactionID          uuid.UUID `json:"reactionId"`
	ReactionKind        string    `json:"reactionKind,omitempty"`
	ReactionTotal       int       `json:"reactionTotal,omitempty"`
	ReactionStopsAttack bool      `json:"reactionStopsAttack,omitempty"`
	AttackStopped       bool      `json:"attackStopped,omitempty"`
	Avoided             bool      `json:"avoided"`
	Defended            bool      `json:"defended"`
	// Hit, Dodge and Defense are the derived roll math Task 6 exists to save — see
	// rollOutcomeRecord's own doc. Defense is the zero value when a dodge already stopped
	// the attack, same as on service.CharacterResult itself.
	Hit             rollOutcomeRecord `json:"hit"`
	Dodge           rollOutcomeRecord `json:"dodge"`
	Defense         rollOutcomeRecord `json:"defense"`
	Ladder          ladderRecord      `json:"ladder"`
	DamageDice      []int             `json:"damageDice,omitempty"`
	RawDamage       int               `json:"rawDamage"`
	DefenseApplied  int               `json:"defenseApplied"`
	EffectiveDamage int               `json:"effectiveDamage"`
	Payouts         []modifierRecord  `json:"payouts,omitempty"`
}

type ladderRecord struct {
	Rung       string `json:"rung,omitempty"`
	Margin     int    `json:"margin,omitempty"`
	Difference int    `json:"difference,omitempty"`
}

type modifierRecord struct {
	Amount      int    `json:"amount"`
	Bias        int    `json:"bias"`
	Applies     string `json:"applies"`
	Source      string `json:"source"`
	AgainstKind string `json:"againstKind"`
	// AgainstID: omitempty is a no-op here (uuid.UUID is [16]byte; encoding/json never
	// elides a fixed-size array), and it is written as-is like ReactionID above — the zero
	// UUID is exactly what ScopeAnyone/ScopeAllBut's own "no single target" case already
	// means, not a hole in the data.
	AgainstID uuid.UUID `json:"againstId"`
	ExpiresAt string    `json:"expiresAt"`
	Reason    string    `json:"reason,omitempty"`
}

// wallResultRecord is one wall's outcome from this turn's attack — the wall's own current
// geometry/HP/open/locked state is NOT copied here; that lives in the map tables and is
// read from there. See resolutionRecord.WallResults for why.
type wallResultRecord struct {
	WallID          string `json:"wallId"`
	EffectiveDamage int    `json:"effectiveDamage"`
	ReboundDamage   int    `json:"reboundDamage"`
	Kind            string `json:"kind"`
}

// encodeResolution returns nil (SQL NULL) for a nil resolution. A turn that resolved nothing
// stores nothing — a zero-value record would read back as "a collision that produced zero",
// which is a different claim.
func encodeResolution(res *service.TurnResolution) ([]byte, error) {
	if res == nil {
		return nil, nil
	}
	rec := resolutionRecord{
		IsSettled:    res.IsSettled,
		ActionResult: rollResultRecord(res.ActionResult),
		Characters:   make([]characterResultRecord, 0, len(res.CharacterResults)),
	}
	for _, cr := range res.CharacterResults {
		out := characterResultRecord{
			TargetID: cr.TargetID, ReactionID: cr.ReactionID,
			ReactionKind: cr.ReactionKind, ReactionTotal: cr.ReactionTotal,
			ReactionStopsAttack: cr.ReactionStopsAttack, AttackStopped: cr.AttackStopped,
			Avoided: cr.Avoided, Defended: cr.Defended,
			Hit: rollOutcomeRecord(cr.Hit), Dodge: rollOutcomeRecord(cr.Dodge),
			Defense: rollOutcomeRecord(cr.Defense),
			Ladder: ladderRecord{
				Rung: string(cr.Ladder.Rung), Margin: cr.Ladder.Margin,
				Difference: cr.Ladder.Difference,
			},
			DamageDice: cr.DamageDice, RawDamage: cr.RawDamage,
			DefenseApplied: cr.DefenseApplied, EffectiveDamage: cr.EffectiveDamage,
		}
		for _, m := range cr.Payouts {
			out.Payouts = append(out.Payouts, modifierRecord{
				Amount: m.Amount, Bias: m.Bias, Applies: string(m.Applies),
				Source: string(m.Source), AgainstKind: m.Against.Kind(),
				AgainstID: m.Against.ID(), ExpiresAt: string(m.ExpiresAt), Reason: m.Reason,
			})
		}
		rec.Characters = append(rec.Characters, out)
	}
	for _, wr := range res.WallResults {
		rec.WallResults = append(rec.WallResults, wallResultRecord{
			WallID: wr.UpdatedWall.ID, EffectiveDamage: wr.EffectiveDamage,
			ReboundDamage: wr.ReboundDamage, Kind: string(wr.Kind),
		})
	}
	return json.Marshal(rec)
}

// DecodeResolution rebuilds a settled resolution read back from turns.resolution. Exported
// because the history read path in this package and its tests both need it.
//
// A row that will not decode returns nil and logs: a turn whose stored collision is
// unreadable is still a turn that happened, and the history must show the declaration rather
// than fail the whole match's history over one bad row.
func DecodeResolution(raw []byte) *service.TurnResolution {
	if len(raw) == 0 {
		return nil
	}
	var rec resolutionRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		log.Printf("DecodeResolution: %v", err)
		return nil
	}
	out := &service.TurnResolution{
		IsSettled:        rec.IsSettled,
		ActionResult:     service.RollResult(rec.ActionResult),
		CharacterResults: make([]service.CharacterResult, 0, len(rec.Characters)),
	}
	for _, c := range rec.Characters {
		cr := service.CharacterResult{
			TargetID: c.TargetID, ReactionID: c.ReactionID,
			ReactionKind: c.ReactionKind, ReactionTotal: c.ReactionTotal,
			ReactionStopsAttack: c.ReactionStopsAttack, AttackStopped: c.AttackStopped,
			Avoided: c.Avoided, Defended: c.Defended,
			Hit: service.RollOutcome(c.Hit), Dodge: service.RollOutcome(c.Dodge),
			Defense: service.RollOutcome(c.Defense),
			Ladder: service.LadderOutcome{
				Rung: service.LadderRung(c.Ladder.Rung), Margin: c.Ladder.Margin,
				Difference: c.Ladder.Difference,
			},
			DamageDice: c.DamageDice, RawDamage: c.RawDamage,
			DefenseApplied: c.DefenseApplied, EffectiveDamage: c.EffectiveDamage,
		}
		for _, m := range c.Payouts {
			cr.Payouts = append(cr.Payouts, match.Modifier{
				Amount: m.Amount, Bias: m.Bias, Applies: match.Dimension(m.Applies),
				Source: match.Source(m.Source), Against: match.ScopeFrom(m.AgainstKind, m.AgainstID),
				ExpiresAt: match.Lifetime(m.ExpiresAt), Reason: m.Reason,
			})
		}
		out.CharacterResults = append(out.CharacterResults, cr)
	}
	for _, wr := range rec.WallResults {
		out.WallResults = append(out.WallResults, service.WallResult{
			UpdatedWall:     mapentity.WallSegment{ID: wr.WallID},
			EffectiveDamage: wr.EffectiveDamage,
			ReboundDamage:   wr.ReboundDamage,
			Kind:            service.WallResultKind(wr.Kind),
		})
	}
	return out
}
