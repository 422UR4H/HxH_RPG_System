package match

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	apiAuth "github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/wire/actionwire"
	"github.com/422UR4H/HxH_RPG_System/internal/application/auth"
	matchUC "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
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

// HistoryRoundResponse is one round of a scene's history, with the turns closed inside it and
// what happened inside it that is not a turn.
type HistoryRoundResponse struct {
	UUID       uuid.UUID             `json:"uuid"`
	Mode       string                `json:"mode"`
	CreatedAt  string                `json:"createdAt"`
	FinishedAt *string               `json:"finishedAt,omitempty"`
	Turns      []HistoryTurnResponse `json:"turns"`
	// Events is the round's regime changes and the master actions outside any turn the history
	// holds, in the order they happened (B15, spec §4.5, §4.8). Always a list — [] when nothing
	// happened — never null.
	Events []HistoryEventResponse `json:"events"`
}

// HistoryEventResponse is one entry of a round's events. Kind is the discriminator:
// "roundModeChanged" carries Payload ({"from", "to"}); "masterAction" carries MasterAction.
// UUID is the event's own, or the master action's; CreatedAt is when it happened.
type HistoryEventResponse struct {
	UUID         uuid.UUID                    `json:"uuid"`
	Kind         string                       `json:"kind"`
	CreatedAt    string                       `json:"createdAt"`
	Payload      json.RawMessage              `json:"payload,omitempty"`
	MasterAction *HistoryMasterActionResponse `json:"masterAction,omitempty"`
}

// HistoryMasterActionResponse is one master action as this reader saw it live — already run
// through masteraction.Record.ProjectFor by the use case (a `left` reader's move has no
// destination in Content). It deliberately has no field for Record.Views: who saw what is how
// the projection is decided, not something any reader — the master included — is shown.
type HistoryMasterActionResponse struct {
	UUID       uuid.UUID       `json:"uuid"`
	Kind       string          `json:"kind"`
	TurnID     *uuid.UUID      `json:"turnId,omitempty"`
	HappenedAt string          `json:"happenedAt"`
	Content    json.RawMessage `json:"content"`
}

// HistoryTurnResponse is one closed turn: the action that drove it, whatever reactions
// answered it, and the settled collision that resulted — each already run through this
// viewer's projection (service.ProjectAction/ProjectResolution) before it ever reached this
// struct, and then through actionwire.From(_, actionwire.Full): the REST history is the ONE
// surface that always asks for the Full cut level, so it keeps every number ProjectAction let
// through. See internal/app/wire/actionwire for the shared shape (Action and its field docs
// — this is where the doc comments that used to live on ActionResponse and its neighbours
// moved to) and for Opened/Declaration, the two narrower cuts the WebSocket surfaces use.
type HistoryTurnResponse struct {
	UUID       uuid.UUID               `json:"uuid"`
	CreatedAt  string                  `json:"createdAt"`
	FinishedAt string                  `json:"finishedAt"`
	Action     actionwire.Action       `json:"action"`
	Reactions  []actionwire.Action     `json:"reactions"`
	Resolution *TurnResolutionResponse `json:"resolution,omitempty"`
	// MasterActions are the master actions applied while this turn was open, as this reader
	// saw them live, in the order they happened (spec §4.8). Always a list, never null.
	MasterActions []HistoryMasterActionResponse `json:"masterActions"`
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
	// Escape is how an escape came out — absent for every reaction that does not displace.
	// The REST mirror of the WebSocket's EscapeResultPayload, same fields, same derivation of
	// awaitsMaster (see EscapeResultResponse).
	Escape *EscapeResultResponse `json:"escape,omitempty"`
}

// EscapeResultResponse is an escape's verdict: it escaped only if the movement AND the dodge
// both beat the attacker's hit (front-combat-phases.md §6A.5, B13). Landing is where the
// master put the piece of a FAILED escape; AwaitsMaster is "failed and no landing" — on a
// settled turn, which every history turn is, it means the piece stayed where it stood.
type EscapeResultResponse struct {
	Escaped      bool    `json:"escaped"`
	MovePassed   bool    `json:"movePassed"`
	DodgePassed  bool    `json:"dodgePassed"`
	AwaitsMaster bool    `json:"awaitsMaster"`
	Landing      *[3]int `json:"landing,omitempty"`
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
	events := make([]HistoryEventResponse, 0, len(r.Events))
	for _, e := range r.Events {
		if ev, ok := toHistoryEventResponse(e); ok {
			events = append(events, ev)
		}
	}
	return HistoryRoundResponse{
		UUID: r.UUID, Mode: r.Mode,
		CreatedAt:  r.CreatedAt.Format(time.RFC3339),
		FinishedAt: formatTimePtr(r.FinishedAt),
		Turns:      turns,
		Events:     events,
	}
}

// toHistoryEventResponse maps one round event; false for an entry whose Kind and pointer
// disagree, which the use case never builds.
func toHistoryEventResponse(e matchUC.HistoryEvent) (HistoryEventResponse, bool) {
	switch {
	case e.Kind == matchUC.HistoryEventRoundModeChanged && e.RoundModeChange != nil:
		return HistoryEventResponse{
			UUID: e.RoundModeChange.UUID, Kind: string(e.Kind),
			CreatedAt: e.At.Format(time.RFC3339),
			Payload:   e.RoundModeChange.Payload,
		}, true
	case e.Kind == matchUC.HistoryEventMasterAction && e.MasterAction != nil:
		ma := toHistoryMasterActionResponse(*e.MasterAction)
		return HistoryEventResponse{
			UUID: e.MasterAction.UUID, Kind: string(e.Kind),
			CreatedAt:    e.At.Format(time.RFC3339),
			MasterAction: &ma,
		}, true
	default:
		return HistoryEventResponse{}, false
	}
}

// toHistoryMasterActionResponse maps one already-projected master action. Record.Views is
// dropped here for every reader — see HistoryMasterActionResponse.
func toHistoryMasterActionResponse(r masteraction.Record) HistoryMasterActionResponse {
	return HistoryMasterActionResponse{
		UUID: r.UUID, Kind: string(r.Kind), TurnID: r.TurnUUID,
		HappenedAt: r.HappenedAt.Format(time.RFC3339),
		Content:    r.Content,
	}
}

func toHistoryTurnResponse(t matchUC.HistoryTurn) HistoryTurnResponse {
	reactions := make([]actionwire.Action, 0, len(t.Reactions))
	for _, r := range t.Reactions {
		reactions = append(reactions, actionwire.From(r, actionwire.Full))
	}
	masterActions := make([]HistoryMasterActionResponse, 0, len(t.MasterActions))
	for _, ma := range t.MasterActions {
		masterActions = append(masterActions, toHistoryMasterActionResponse(ma))
	}
	// The move's WHERE is cut after From, by the verdict the use case read off what this reader
	// saw live (MoveSight) — the order room.go's turnActionWireLocked applies the live gate in.
	// MoveViews/LandingViews themselves are never mapped: who saw what is not table data.
	act := actionwire.From(t.Action, actionwire.Full)
	if act.Move != nil {
		switch t.MoveSight {
		case matchUC.MoveSightOrigin:
			act.Move.Position = nil
		case matchUC.MoveSightNone:
			act.Move.From, act.Move.Position = nil, nil
		}
	}
	res := toTurnResolutionResponse(t.Resolution)
	if res != nil {
		// After awaitsMaster was derived from the real landing: a landing withheld from this
		// reader is not a landing the master never chose.
		for i := range res.Targets {
			if esc := res.Targets[i].Escape; esc != nil && t.HiddenLandings[res.Targets[i].TargetID] {
				esc.Landing = nil
			}
		}
	}
	return HistoryTurnResponse{
		UUID:          t.UUID,
		CreatedAt:     t.CreatedAt.Format(time.RFC3339),
		FinishedAt:    t.FinishedAt.Format(time.RFC3339),
		Action:        act,
		Reactions:     reactions,
		Resolution:    res,
		MasterActions: masterActions,
	}
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
			Escape:          toEscapeResultResponse(cr.Escape),
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

func toEscapeResultResponse(e *service.EscapeResult) *EscapeResultResponse {
	if e == nil {
		return nil
	}
	out := &EscapeResultResponse{
		Escaped: e.Escaped, MovePassed: e.MovePassed, DodgePassed: e.DodgePassed,
		AwaitsMaster: !e.Escaped && e.Landing == nil,
	}
	if e.Landing != nil {
		pos := *e.Landing
		out.Landing = &pos
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
