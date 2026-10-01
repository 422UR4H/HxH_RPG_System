// Package actionwire is the ONE wire format for an action or reaction, shared by every
// surface that ever sends one: the REST Action History, the master's queue, turn_opened, and
// reconciliation's ownQueue (see design spec §4.1, B1/B2/B12). Before this package, the shape
// lived once, privately, in internal/app/api/match/get_match_history.go — and the WebSocket
// delivery package could not import the REST delivery package to reuse it without a layering
// violation. Now both import this one, delivery-only, no-I/O package instead.
//
// The SAME struct shape serves all three cut levels (see Level in from.go): what changes
// between them is not the type, only how many of the numbers From fills in. That is why every
// number field below is a pointer with `omitempty` — nil at a level that cuts it, filled
// (even with a zero value) at a level that keeps it. At Level Full every pointer is always
// non-nil, which is what makes the REST history's JSON byte-identical to what it was before
// this package existed (see get_match_history_golden_test.go).
//
// This package carries NO visibility deny-list. Feint/Trigger nil-ing, ReactionKind demotion,
// and the stripped Evasion skill entry are service.ProjectAction's job, and ProjectAction runs
// BEFORE From ever sees the action — see From's own doc.
package actionwire

import "github.com/google/uuid"

// Action is one action or reaction, cut to whatever Level the caller asked From for.
//
// When the deny-list already ran (service.ProjectAction, upstream — see From's doc), Feint and
// Trigger are nil, ReactionKind is demoted, and a stripped Evasion skill entry is simply
// absent. None of that is this package's business; what IS this package's business is how
// many of the numbers that survived the deny-list this recipient's level entitles them to.
type Action struct {
	UUID         uuid.UUID     `json:"uuid"`
	ActorID      uuid.UUID     `json:"actorId"`
	TargetID     []uuid.UUID   `json:"targetId,omitempty"`
	ReactToID    *uuid.UUID    `json:"reactToId,omitempty"`
	ReactionKind string        `json:"reactionKind,omitempty"`
	Skills       []ActionSkill `json:"skills,omitempty"`
	Speed        ActionSpeed   `json:"speed"`
	Feint        *RollCheck    `json:"feint,omitempty"`
	Trigger      *Trigger      `json:"trigger,omitempty"`
	Move         *Move         `json:"move,omitempty"`
	Attack       *Attack       `json:"attack,omitempty"`
	Defense      *Defense      `json:"defense,omitempty"`
	Dodge        *Dodge        `json:"dodge,omitempty"`
	Repel        *Repel        `json:"repel,omitempty"`
	Interact     *Interact     `json:"interact,omitempty"`
	// SystemBias is the engine-imposed advantage/disadvantage this action was charged under:
	// 0 for a plain action, -1 for a reaction that displaced a queued one. It is a third,
	// engine-owned origin — neither the master's RollCondition nor the character's
	// ModifierLedger (see action.Action.SystemBias) — and it is here because it is the REASON
	// a roll on this surface came out as it did.
	//
	// Public for the same reason RollCheck.Attempts is: the bias is public by omission. Both
	// dice sets and the result already travel to every viewer at Full/Opened, so WHICH set the
	// engine read is already derivable there — withholding the field would only force the
	// client into the algebra this repo avoids on purpose (see CharacterResult.ReactionTotal).
	// It is NOT itself cut by Level: every level that keeps this Action at all keeps this
	// field, since it never carries a number the cut table lists.
	//
	// RollCheck.Context (the master's RollCondition) is NOT the same call and stays off every
	// surface: the master's intervention already has one of its own, in
	// overridden_action_values.
	//
	// omitempty keeps it off the overwhelming majority of actions, which were charged nothing.
	SystemBias int `json:"systemBias,omitempty"`
}

// ActionSkill is one skill test this action rolled — a plain attribute/skill check outside
// the speed, attack or reaction axes (e.g. a Perception check bundled into a combined
// action).
//
// Named ActionSkill, not the bare Skill the design spec's own type list uses, for a reason
// outside this package's control: huma's OpenAPI schema registry keys components by bare Go
// type name across every package reachable from a registered route (see
// api/routes_registration_test.go's own doc, which predicts exactly this collision class by
// name), and internal/app/api/sheet already registers an unrelated DTO also named Skill.
// get_match_history.go's own ActionSkillResponse — the type this one replaces — carried the
// same Action- prefix for the same reason, before this package existed. The JSON tag stays
// unprefixed ("skillName" etc. on the fields, and this type is never itself keyed by name on
// the wire — only nested under "skills" on Action) so no byte of the REST history's output
// changes; only this Go export's name does.
type ActionSkill struct {
	SkillName  string    `json:"skillName"`
	Difficulty *int      `json:"difficulty,omitempty"`
	RollCheck  RollCheck `json:"rollCheck"`
}

// RollCheck is one test's skill name and, where the recipient's Level keeps them, its numbers.
//
// SkillName always travels — every level, including Declaration, tells the recipient WHICH
// skill was tested, because that alone is what the owner declared. SkillValue, Attempts and
// Result are the numbers the cut table (§4.1) may withhold: nil at a level that cuts them,
// non-nil (even pointing at a zero value) at a level that keeps them. At Full, all three are
// always non-nil — see From's own doc for why that keeps the REST history byte-identical to
// what it emitted before this package existed.
//
// The numbers travel to every viewer they reach at all: public by omission is the rule, and a
// third party deducing a hidden Evasion from the numbers is impossible without them (see
// service.ProjectAction's own doc). Only the closed reactions' LABEL and the Evasion skill
// entry itself are on the deny list, and both are handled upstream, not here.
type RollCheck struct {
	SkillName  string        `json:"skillName"`
	SkillValue *int          `json:"skillValue,omitempty"`
	Attempts   *RollAttempts `json:"attempts,omitempty"`
	Result     *int          `json:"result,omitempty"`
}

// RollAttempts is both dice sets a RollCheck rolled — see action.RollAttempts for why both
// travel together.
type RollAttempts struct {
	Primary   []int `json:"primary,omitempty"`
	Secondary []int `json:"secondary,omitempty"`
}

// Trigger is presence-only: the domain Trigger carries no fields yet (see action.Trigger's
// own TODO), so its wire shape is deliberately an empty object — what matters here is whether
// this recipient is entitled to know a trigger exists at all, which the deny-list decided
// upstream, not the Level this package cuts by.
type Trigger struct{}

// ActionSpeed is the actionSpeed roll — Bar always travels, RollCheck's numbers follow the cut
// table's "speed.rollCheck" row: kept at Full and Opened, cut to the skill name alone at
// Declaration.
type ActionSpeed struct {
	Bar       int       `json:"bar"`
	RollCheck RollCheck `json:"rollCheck"`
}

// Move is the movement half of an action. Category, From and Position are the DECLARATION —
// what the owner committed to — and From fills them at every level. Speed and
// FinalSpeed are the moveSpeed roll and its derived result, cut by the same "moveSpeed" row as
// ActionSpeed above: kept at Full and Opened, cut to the skill name alone (Speed) or absent
// entirely (FinalSpeed) at Declaration. Charge follows the OTHER row instead — same as
// attack.hit/damage/charge, skills[], feint, defense, dodge, repel — kept only at Full.
//
// From and Position are also WHERE a piece stands and goes, which is not a Level question:
// the game server nils them per recipient after From, by the piece's own fog gate (room.go's
// turnActionWireLocked, owner decision 2026-10-01), and the REST history by the verdict that
// gate recorded live (HistoryTurn.MoveSight). From never does — see its own doc.
type Move struct {
	Category string `json:"category"`
	// From is the actor's piece position on the server's board at enqueue time (B6, spec
	// §4.3 "B5, B6 e B10") — never the client's declared origin, which the server discards.
	// nil (omitted on the wire) means the actor had no piece to check against. Convention:
	// [a, b, z], (a, b) = (col, row) square or (q, r) axial hex, z not read.
	From *[3]int `json:"from,omitempty"`
	// Position is where the move puts the piece. Always set by From; nil (omitted on the wire)
	// only when the game server withheld it from a recipient who cannot see that cell. The
	// REST history (Full, never gated) always carries it.
	Position   *[3]int    `json:"position,omitempty"`
	Speed      *RollCheck `json:"speed,omitempty"`
	Charge     *RollCheck `json:"charge,omitempty"`
	FinalSpeed *int       `json:"finalSpeed,omitempty"`
}

// Attack is the strike half of an action. Weapon, Spread and RelativeVelocity are declaration
// and always travel; Hit, Damage and Charge follow the numbers row — kept only at Full.
type Attack struct {
	Weapon           *string    `json:"weapon,omitempty"`
	Hit              RollCheck  `json:"hit"`
	Damage           RollCheck  `json:"damage"`
	Charge           *RollCheck `json:"charge,omitempty"`
	Spread           string     `json:"spread,omitempty"`
	RelativeVelocity float64    `json:"relativeVelocity"`
}

// Defense is a defense reaction's weapon and roll. RollCheck follows the numbers row — kept
// only at Full.
type Defense struct {
	Weapon    *string   `json:"weapon,omitempty"`
	RollCheck RollCheck `json:"rollCheck"`
}

// Dodge is a dodge reaction's roll. WHICH dodge (active, closed, escape, closed escape) is
// ReactionKind's business, on Action itself — see action.Dodge's own doc. RollCheck follows
// the numbers row — kept only at Full.
type Dodge struct {
	RollCheck RollCheck `json:"rollCheck"`
}

// Repel is a repel reaction's weapon and roll. RollCheck follows the numbers row — kept only
// at Full.
type Repel struct {
	Weapon    *string   `json:"weapon,omitempty"`
	RollCheck RollCheck `json:"rollCheck"`
}

// Interact is presence-only beyond its Kind, which always travels — it is declaration, not a
// number the cut table touches.
type Interact struct {
	Kind string `json:"kind"`
}
