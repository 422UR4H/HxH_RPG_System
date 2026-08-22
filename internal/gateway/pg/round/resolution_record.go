package round

import (
	"encoding/json"
	"log"

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
// Tags are camelCase, like every other wire shape in this repo.
type resolutionRecord struct {
	IsSettled    bool                    `json:"isSettled"`
	ActionResult rollResultRecord        `json:"actionResult"`
	Characters   []characterResultRecord `json:"characters"`
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

type characterResultRecord struct {
	TargetID            uuid.UUID        `json:"targetId"`
	ReactionID          uuid.UUID        `json:"reactionId,omitempty"`
	ReactionKind        string           `json:"reactionKind,omitempty"`
	ReactionTotal       int              `json:"reactionTotal,omitempty"`
	ReactionStopsAttack bool             `json:"reactionStopsAttack,omitempty"`
	AttackStopped       bool             `json:"attackStopped,omitempty"`
	Avoided             bool             `json:"avoided"`
	Defended            bool             `json:"defended"`
	Ladder              ladderRecord     `json:"ladder"`
	DamageDice          []int            `json:"damageDice,omitempty"`
	RawDamage           int              `json:"rawDamage"`
	DefenseApplied      int              `json:"defenseApplied"`
	EffectiveDamage     int              `json:"effectiveDamage"`
	Payouts             []modifierRecord `json:"payouts,omitempty"`
}

type ladderRecord struct {
	Rung       string `json:"rung,omitempty"`
	Margin     int    `json:"margin,omitempty"`
	Difference int    `json:"difference,omitempty"`
}

type modifierRecord struct {
	Amount      int       `json:"amount"`
	Bias        int       `json:"bias"`
	Applies     string    `json:"applies"`
	Source      string    `json:"source"`
	AgainstKind string    `json:"againstKind"`
	AgainstID   uuid.UUID `json:"againstId,omitempty"`
	ExpiresAt   string    `json:"expiresAt"`
	Reason      string    `json:"reason,omitempty"`
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
	return out
}
