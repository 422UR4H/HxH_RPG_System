package match

import (
	"context"
	"errors"
	"time"

	apiAuth "github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/application/auth"
	matchUC "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
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
	UUID         uuid.UUID           `json:"uuid"`
	ActorID      uuid.UUID           `json:"actorId"`
	TargetID     []uuid.UUID         `json:"targetId,omitempty"`
	ReactToID    *uuid.UUID          `json:"reactToId,omitempty"`
	ReactionKind string              `json:"reactionKind,omitempty"`
	Skills       []SkillResponse     `json:"skills,omitempty"`
	Speed        ActionSpeedResponse `json:"speed"`
	Feint        *RollCheckResponse  `json:"feint,omitempty"`
	Trigger      *TriggerResponse    `json:"trigger,omitempty"`
	Move         *MoveResponse       `json:"move,omitempty"`
	Attack       *AttackResponse     `json:"attack,omitempty"`
	Defense      *DefenseResponse    `json:"defense,omitempty"`
	Dodge        *DodgeResponse      `json:"dodge,omitempty"`
	Repel        *RepelResponse      `json:"repel,omitempty"`
	Interact     *InteractResponse   `json:"interact,omitempty"`
}

type SkillResponse struct {
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
		out.Skills = append(out.Skills, SkillResponse{
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
		})
	}
	for _, pr := range res.PendingReactions {
		out.PendingReactions = append(out.PendingReactions, PendingReactionResponse{
			ReactionID: pr.ReactionID, ActorID: pr.ActorID, Kind: pr.Kind,
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

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}
