package match

import (
	"context"
	"errors"
	"time"

	apiAuth "github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/application/auth"
	matchUC "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type GetMatchHistoryRequest struct {
	UUID uuid.UUID `path:"uuid" required:"true" doc:"Match UUID"`
}

// GetMatchHistoryResponseBody is the nested Scene -> Round -> Turn -> Action tree, already
// projected per viewer by the use case. This handler never filters anything itself — see
// service.ProjectAction/ProjectResolution, which is the ONE deny-list both this surface and
// the WebSocket path run.
type GetMatchHistoryResponseBody struct {
	Scenes []HistorySceneResponse `json:"scenes"`
}

type GetMatchHistoryResponse struct {
	Body GetMatchHistoryResponseBody
}

// HistorySceneResponse is one scene the match was organised into, with the rounds it contains.
// The tree shape mirrors the domain read path (matchUC.HistoryScene) on purpose: the front
// renders action cards inside the scope of each scene, so flattening here would push the
// regrouping onto every consumer.
type HistorySceneResponse struct {
	UUID       uuid.UUID              `json:"uuid"`
	Category   string                 `json:"category"`
	BriefDesc  string                 `json:"briefDesc"`
	CreatedAt  string                 `json:"createdAt"`
	FinishedAt *string                `json:"finishedAt,omitempty"`
	Rounds     []HistoryRoundResponse `json:"rounds"`
}

// HistoryRoundResponse is one round of a scene's history, with the turns closed inside it.
type HistoryRoundResponse struct {
	UUID       uuid.UUID             `json:"uuid"`
	Mode       string                `json:"mode"`
	CreatedAt  string                `json:"createdAt"`
	FinishedAt *string               `json:"finishedAt,omitempty"`
	Turns      []HistoryTurnResponse `json:"turns"`
}

// HistoryTurnResponse is one closed turn: the action that drove it, whatever reactions
// answered it, and the settled collision that resulted — each already run through this
// viewer's projection before it ever reached this struct.
type HistoryTurnResponse struct {
	UUID       uuid.UUID               `json:"uuid"`
	CreatedAt  string                  `json:"createdAt"`
	FinishedAt string                  `json:"finishedAt"`
	Action     ActionResponse          `json:"action"`
	Reactions  []ActionResponse        `json:"reactions"`
	Resolution *TurnResolutionResponse `json:"resolution,omitempty"`
}

// ActionResponse is one action or reaction as THIS viewer is entitled to see it. Feint and
// Trigger are nil, ReactionKind is demoted, and a stripped Evasion skill entry is simply
// absent — all of that already happened upstream, in service.ProjectAction.
type ActionResponse struct {
	UUID         uuid.UUID             `json:"uuid"`
	ActorID      uuid.UUID             `json:"actorId"`
	TargetID     []uuid.UUID           `json:"targetId,omitempty"`
	ReactToID    *uuid.UUID            `json:"reactToId,omitempty"`
	ReactionKind string                `json:"reactionKind,omitempty"`
	Skills       []ActionSkillResponse `json:"skills,omitempty"`
	Speed        ActionSpeedResponse   `json:"speed"`
	Feint        *RollCheckResponse    `json:"feint,omitempty"`
	Trigger      *TriggerResponse      `json:"trigger,omitempty"`
	Move         *MoveResponse         `json:"move,omitempty"`
	Attack       *AttackResponse       `json:"attack,omitempty"`
	Defense      *DefenseResponse      `json:"defense,omitempty"`
	Dodge        *DodgeResponse        `json:"dodge,omitempty"`
	Repel        *RepelResponse        `json:"repel,omitempty"`
	Interact     *InteractResponse     `json:"interact,omitempty"`
	// SystemBias is the engine-imposed advantage/disadvantage this action was charged under:
	// 0 for a plain action, -1 for a reaction that displaced a queued one. It is a third,
	// engine-owned origin — neither the master's RollCondition nor the character's
	// ModifierLedger (see action.Action.SystemBias) — and it is here because it is the REASON
	// a roll on this surface came out as it did.
	//
	// Public for the same reason RollCheckResponse.attempts is: the bias is public by
	// omission. Both dice sets and the result already travel to every viewer, so WHICH set
	// the engine read is already derivable — withholding the field would only force the
	// client into the algebra this repo avoids on purpose (see CharacterResult.ReactionTotal).
	//
	// RollCheck.Context (the master's RollCondition) is NOT the same call and stays off every
	// surface: the master's intervention already has one of its own, in
	// overridden_action_values.
	//
	// omitempty keeps it off the overwhelming majority of actions, which were charged nothing.
	SystemBias int `json:"systemBias,omitempty"`
}

type ActionSkillResponse struct {
	SkillName  string            `json:"skillName"`
	Difficulty *int              `json:"difficulty,omitempty"`
	RollCheck  RollCheckResponse `json:"rollCheck"`
}

// RollCheckResponse is one test's dice and result. The numbers travel to every viewer — public
// by omission is the rule, and a third party deducing a hidden Evasion from the numbers is
// impossible without them (see projection.go's own doc). Only the closed reactions' LABEL and
// the Evasion skill entry itself are on the deny list, and both are handled upstream.
type RollCheckResponse struct {
	SkillName  string               `json:"skillName"`
	SkillValue int                  `json:"skillValue"`
	Attempts   RollAttemptsResponse `json:"attempts"`
	Result     int                  `json:"result"`
}

type RollAttemptsResponse struct {
	Primary   []int `json:"primary,omitempty"`
	Secondary []int `json:"secondary,omitempty"`
}

// TriggerResponse is presence-only: the domain Trigger carries no fields yet (see
// action.Trigger's own TODO), so its wire shape is deliberately an empty object — what matters
// here is whether this viewer is entitled to know a trigger exists at all.
type TriggerResponse struct{}

type ActionSpeedResponse struct {
	Bar       int               `json:"bar"`
	RollCheck RollCheckResponse `json:"rollCheck"`
}

type MoveResponse struct {
	Category   string             `json:"category"`
	From       [3]int             `json:"from,omitempty"`
	Position   [3]int             `json:"position"`
	Speed      *RollCheckResponse `json:"speed,omitempty"`
	Charge     *RollCheckResponse `json:"charge,omitempty"`
	FinalSpeed int                `json:"finalSpeed"`
}

type AttackResponse struct {
	Weapon           *string            `json:"weapon,omitempty"`
	Hit              RollCheckResponse  `json:"hit"`
	Damage           RollCheckResponse  `json:"damage"`
	Charge           *RollCheckResponse `json:"charge,omitempty"`
	Spread           string             `json:"spread,omitempty"`
	RelativeVelocity float64            `json:"relativeVelocity"`
}

type DefenseResponse struct {
	Weapon    *string           `json:"weapon,omitempty"`
	RollCheck RollCheckResponse `json:"rollCheck"`
}

type DodgeResponse struct {
	RollCheck RollCheckResponse `json:"rollCheck"`
}

type RepelResponse struct {
	Weapon    *string           `json:"weapon,omitempty"`
	RollCheck RollCheckResponse `json:"rollCheck"`
}

type InteractResponse struct {
	Kind string `json:"kind"`
}

// TurnResolutionResponse is one recipient's view of a turn's settled resolution — the same
// per-field split ResolutionUpdatedPayload uses on the WebSocket path (see message.go), rebuilt
// here because a REST delivery package does not import the WS delivery package. Both read off
// the SAME projected service.TurnResolution; only the wire shape is duplicated, never the
// deny-list itself.
type TurnResolutionResponse struct {
	IsSettled        bool                      `json:"isSettled"`
	Action           RollResultResponse        `json:"action"`
	Targets          []CharacterResultResponse `json:"targets"`
	PendingReactions []PendingReactionResponse `json:"pendingReactions,omitempty"`
	// Errors is every engine fault hit while computing this resolution — a target the engine
	// could not classify, a character on the board whose sheet it was not handed. Absent on a
	// clean resolution, which is the normal case, so its presence is the signal.
	//
	// MASTER-ONLY, exactly like PendingReactions above and by the same mechanism:
	// service.ProjectResolution strips it upstream for every other viewer, and this DTO
	// re-applies nothing. These are diagnostics about the engine rather than facts about the
	// fiction, and the master is the only person who can act on one.
	//
	// It is on this surface and not only on the WebSocket because the live path is EPHEMERAL:
	// "the master gets it live" assumes a master connected and looking at that instant. The
	// fault is persisted with the turn (resolution_record.go) for the reason written there —
	// a missing_sheet means a target produced no entry in Targets at all, and a history that
	// kept that silence would read back, a year later, as a turn that simply never aimed at
	// them. A DTO that decodes the row and then drops the field recreates exactly that
	// silence, one layer up, with the information already paid for and stored.
	//
	// It is NOT an error MESSAGE: a fault here does not mean the request failed, or that the
	// turn did. The turn resolved, these numbers are real, and one part of the collision is
	// missing from them.
	Errors []ResolutionErrorResponse `json:"errors,omitempty"`
}

// ResolutionErrorResponse is one engine fault, as the master's client reads it — the REST
// mirror of the WebSocket's ResolutionErrorPayload. Kind is the stable discriminator; Detail
// is prose for a human and must never be parsed.
type ResolutionErrorResponse struct {
	Subject uuid.UUID `json:"subject"`
	Kind    string    `json:"kind"`
	Detail  string    `json:"detail,omitempty"`
}

type RollResultResponse struct {
	SkillName         string `json:"skillName"`
	SkillValue        int    `json:"skillValue"`
	DiceRolled        []int  `json:"diceRolled"`
	Total             int    `json:"total"`
	IsCritical        bool   `json:"isCritical"`
	IsCriticalFailure bool   `json:"isCriticalFailure"`
	Margin            *int   `json:"margin,omitempty"`
}

type CharacterResultResponse struct {
	TargetID        uuid.UUID               `json:"targetId"`
	Avoided         bool                    `json:"avoided"`
	Defended        bool                    `json:"defended"`
	DodgeTotal      int                     `json:"dodgeTotal"`
	DefenseTotal    int                     `json:"defenseTotal"`
	RawDamage       int                     `json:"rawDamage"`
	DefenseApplied  int                     `json:"defenseApplied"`
	ProjectedDamage int                     `json:"projectedDamage"`
	Reaction        *ReactionResultResponse `json:"reaction,omitempty"`
	// Payouts is what this target's own reaction EARNED — a repel's bonus or penalty, a
	// closed dodge's reserve. Absent when it earned nothing, which is most reactions.
	//
	// Same deny-list as everything else on this surface, applied upstream by
	// service.ProjectResolution and NOT re-applied here: it is withheld on exactly one
	// condition, that the reaction's LABEL was demoted. The closed dodge's reserve is the
	// other half of that secret — its size says how much Evasion was folded in — so it
	// leaves with the label. A repel is never demoted, and reacoes.md is explicit that its
	// leftover is public: "a penalidade de quem aparou vale contra todo mundo — qualquer um
	// pode aproveitar". Whoever may exploit it has to be able to read it.
	Payouts []ModifierResponse `json:"payouts,omitempty"`
}

// ModifierResponse is one accumulated bonus or penalty a reaction wrote into its character's
// ledger — the REST mirror of the WebSocket's ModifierPayload, duplicated for the same reason
// every other shape here is: a REST delivery package does not import the WS one. Both read
// off the SAME projected service.TurnResolution.
//
// match.Scope keeps kind and id private, so the scope travels flattened through Kind()/ID(),
// exactly as the persisted record does (modifierRecord in resolution_record.go).
type ModifierResponse struct {
	Amount int `json:"amount"`
	Bias   int `json:"bias"`
	// Applies is which dimension this moves: "action_speed" or "dodge".
	Applies string `json:"applies"`
	// Source is "system" or "master".
	Source string `json:"source"`
	// AgainstKind ("anyone" | "only" | "all_but") is the whole point of a payout: it says WHO
	// may count it. AgainstID names the character the last two turn on, and is the zero UUID
	// for "anyone" — which is what that case already means, not a hole.
	AgainstKind string    `json:"againstKind"`
	AgainstID   uuid.UUID `json:"againstId"`
	// ExpiresAt is "end_of_turn", "next_turn" or "end_of_round".
	ExpiresAt string `json:"expiresAt"`
	Reason    string `json:"reason,omitempty"`
}

// ReactionResultResponse is what one target answered with. Kind is the SAME field
// service.ProjectResolution already demoted on CharacterResult.ReactionKind — a closed dodge
// still reads "dodge" here for anyone but the master or the owner.
type ReactionResultResponse struct {
	Kind        string    `json:"kind"`
	Total       int       `json:"total"`
	ReactionID  uuid.UUID `json:"reactionId"`
	Rung        string    `json:"rung,omitempty"`
	Margin      int       `json:"margin,omitempty"`
	Difference  int       `json:"difference,omitempty"`
	StopsAttack bool      `json:"stopsAttack"`
}

type PendingReactionResponse struct {
	ReactionID uuid.UUID `json:"reactionId"`
	ActorID    uuid.UUID `json:"actorId"`
	Kind       string    `json:"kind"`
}

func GetMatchHistoryHandler(
	uc matchUC.IGetMatchHistory,
) func(context.Context, *GetMatchHistoryRequest) (*GetMatchHistoryResponse, error) {
	return func(ctx context.Context, req *GetMatchHistoryRequest) (*GetMatchHistoryResponse, error) {
		userUUID, ok := ctx.Value(apiAuth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		result, err := uc.Get(ctx, req.UUID, userUUID)
		if err != nil {
			switch {
			case errors.Is(err, matchUC.ErrMatchNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, auth.ErrInsufficientPermissions):
				return nil, huma.Error403Forbidden(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}

		scenes := make([]HistorySceneResponse, 0, len(result.Scenes))
		for _, s := range result.Scenes {
			scenes = append(scenes, toHistorySceneResponse(s))
		}
		return &GetMatchHistoryResponse{
			Body: GetMatchHistoryResponseBody{Scenes: scenes},
		}, nil
	}
}

func toHistorySceneResponse(s matchUC.HistoryScene) HistorySceneResponse {
	rounds := make([]HistoryRoundResponse, 0, len(s.Rounds))
	for _, r := range s.Rounds {
		rounds = append(rounds, toHistoryRoundResponse(r))
	}
	return HistorySceneResponse{
		UUID: s.UUID, Category: s.Category, BriefDesc: s.BriefDesc,
		CreatedAt:  s.CreatedAt.Format(time.RFC3339),
		FinishedAt: formatTimePtr(s.FinishedAt),
		Rounds:     rounds,
	}
}

func toHistoryRoundResponse(r matchUC.HistoryRound) HistoryRoundResponse {
	turns := make([]HistoryTurnResponse, 0, len(r.Turns))
	for _, t := range r.Turns {
		turns = append(turns, toHistoryTurnResponse(t))
	}
	return HistoryRoundResponse{
		UUID: r.UUID, Mode: r.Mode,
		CreatedAt:  r.CreatedAt.Format(time.RFC3339),
		FinishedAt: formatTimePtr(r.FinishedAt),
		Turns:      turns,
	}
}

func toHistoryTurnResponse(t matchUC.HistoryTurn) HistoryTurnResponse {
	reactions := make([]ActionResponse, 0, len(t.Reactions))
	for _, r := range t.Reactions {
		reactions = append(reactions, toActionResponse(r))
	}
	return HistoryTurnResponse{
		UUID:       t.UUID,
		CreatedAt:  t.CreatedAt.Format(time.RFC3339),
		FinishedAt: t.FinishedAt.Format(time.RFC3339),
		Action:     toActionResponse(t.Action),
		Reactions:  reactions,
		Resolution: toTurnResolutionResponse(t.Resolution),
	}
}

func toActionResponse(a action.Action) ActionResponse {
	out := ActionResponse{
		UUID:         a.GetID(),
		ActorID:      a.GetActorID(),
		TargetID:     a.TargetID,
		ReactionKind: string(a.ReactionKind),
		Speed: ActionSpeedResponse{
			Bar:       a.Speed.Bar,
			RollCheck: toRollCheckResponse(a.Speed.RollCheck),
		},
	}
	if a.ReactToID != uuid.Nil {
		id := a.ReactToID
		out.ReactToID = &id
	}
	for _, s := range a.Skills {
		out.Skills = append(out.Skills, ActionSkillResponse{
			SkillName: s.SkillName, Difficulty: s.Difficulty,
			RollCheck: toRollCheckResponse(s.RollCheck),
		})
	}
	if a.Feint != nil {
		rc := toRollCheckResponse(*a.Feint)
		out.Feint = &rc
	}
	if a.Trigger != nil {
		out.Trigger = &TriggerResponse{}
	}
	if a.Move != nil {
		out.Move = &MoveResponse{
			Category: string(a.Move.Category), From: a.Move.From, Position: a.Move.Position,
			Speed: rollCheckPtr(a.Move.Speed), Charge: rollCheckPtr(a.Move.Charge),
			FinalSpeed: a.Move.FinalSpeed,
		}
	}
	if a.Attack != nil {
		out.Attack = &AttackResponse{
			Weapon: weaponPtr(a.Attack.Weapon),
			Hit:    toRollCheckResponse(a.Attack.Hit), Damage: toRollCheckResponse(a.Attack.Damage),
			Charge: rollCheckPtr(a.Attack.Charge), Spread: string(a.Attack.Spread),
			RelativeVelocity: a.Attack.RelativeVelocity,
		}
	}
	if a.Defense != nil {
		out.Defense = &DefenseResponse{
			Weapon: weaponPtr(a.Defense.Weapon), RollCheck: toRollCheckResponse(a.Defense.RollCheck),
		}
	}
	if a.Dodge != nil {
		out.Dodge = &DodgeResponse{RollCheck: toRollCheckResponse(a.Dodge.RollCheck)}
	}
	if a.Repel != nil {
		out.Repel = &RepelResponse{
			Weapon: weaponPtr(a.Repel.Weapon), RollCheck: toRollCheckResponse(a.Repel.RollCheck),
		}
	}
	if a.Interact != nil {
		out.Interact = &InteractResponse{Kind: string(a.Interact.Kind)}
	}
	out.SystemBias = a.SystemBias
	return out
}

func toRollCheckResponse(rc action.RollCheck) RollCheckResponse {
	return RollCheckResponse{
		SkillName: rc.SkillName, SkillValue: rc.SkillValue,
		Attempts: RollAttemptsResponse{Primary: rc.Attempts.Primary, Secondary: rc.Attempts.Secondary},
		Result:   rc.Result,
	}
}

func rollCheckPtr(rc *action.RollCheck) *RollCheckResponse {
	if rc == nil {
		return nil
	}
	out := toRollCheckResponse(*rc)
	return &out
}

func weaponPtr(w *enum.WeaponName) *string {
	if w == nil {
		return nil
	}
	s := string(*w)
	return &s
}

func toTurnResolutionResponse(res *service.TurnResolution) *TurnResolutionResponse {
	if res == nil {
		return nil
	}
	out := &TurnResolutionResponse{
		IsSettled: res.IsSettled,
		Action: RollResultResponse{
			SkillName: res.ActionResult.SkillName, SkillValue: res.ActionResult.SkillValue,
			DiceRolled: res.ActionResult.DiceRolled, Total: res.ActionResult.Total,
			IsCritical: res.ActionResult.IsCritical, IsCriticalFailure: res.ActionResult.IsCriticalFailure,
			Margin: res.ActionResult.Margin,
		},
		Targets: make([]CharacterResultResponse, 0, len(res.CharacterResults)),
	}
	for _, cr := range res.CharacterResults {
		out.Targets = append(out.Targets, CharacterResultResponse{
			TargetID: cr.TargetID, Avoided: cr.Avoided, Defended: cr.Defended,
			DodgeTotal: cr.Dodge.Total, DefenseTotal: cr.Defense.Total,
			RawDamage: cr.RawDamage, DefenseApplied: cr.DefenseApplied,
			ProjectedDamage: cr.EffectiveDamage,
			Reaction:        toReactionResultResponse(cr),
			Payouts:         toModifierResponses(cr.Payouts),
		})
	}
	for _, pr := range res.PendingReactions {
		out.PendingReactions = append(out.PendingReactions, PendingReactionResponse{
			ReactionID: pr.ReactionID, ActorID: pr.ActorID, Kind: pr.Kind,
		})
	}
	for _, e := range res.Errors {
		out.Errors = append(out.Errors, ResolutionErrorResponse{
			Subject: e.Subject, Kind: string(e.Kind), Detail: e.Detail,
		})
	}
	return out
}

func toReactionResultResponse(cr service.CharacterResult) *ReactionResultResponse {
	if cr.ReactionKind == "" {
		return nil
	}
	return &ReactionResultResponse{
		Kind: cr.ReactionKind, Total: cr.ReactionTotal, ReactionID: cr.ReactionID,
		Rung: string(cr.Ladder.Rung), Margin: cr.Ladder.Margin, Difference: cr.Ladder.Difference,
		StopsAttack: cr.ReactionStopsAttack,
	}
}

// toModifierResponses projects a reaction's payouts. It does NOT decide what a viewer may
// see — service.ProjectResolution already did — so this is a pure mapping. Keeping the
// deny-list in one place is what stops this surface and the WebSocket one from drifting.
func toModifierResponses(ms []domainMatch.Modifier) []ModifierResponse {
	if len(ms) == 0 {
		return nil
	}
	out := make([]ModifierResponse, 0, len(ms))
	for _, m := range ms {
		out = append(out, ModifierResponse{
			Amount:      m.Amount,
			Bias:        m.Bias,
			Applies:     string(m.Applies),
			Source:      string(m.Source),
			AgainstKind: m.Against.Kind(),
			AgainstID:   m.Against.ID(),
			ExpiresAt:   string(m.ExpiresAt),
			Reason:      m.Reason,
		})
	}
	return out
}

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}
