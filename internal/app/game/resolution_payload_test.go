package game

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"
	"time"

	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

func TestNewResolutionUpdatedPayload(t *testing.T) {
	turnID, targetID := uuid.New(), uuid.New()
	margin := 7
	res := &service.TurnResolution{
		IsSettled: false,
		ActionResult: service.RollResult{
			SkillName:  "Accuracy",
			SkillValue: 4,
			DiceRolled: []int{10, 8},
			Total:      22,
			IsCritical: false,
			Margin:     &margin,
		},
		CharacterResults: []service.CharacterResult{{
			TargetID:        targetID,
			Avoided:         false,
			Defended:        true,
			RawDamage:       14,
			EffectiveDamage: 14,
		}},
	}

	p := newResolutionUpdatedPayload(turnID, res)

	t.Run("carries the turn and the action roll", func(t *testing.T) {
		if p.TurnID != turnID {
			t.Errorf("TurnID = %v, want %v", p.TurnID, turnID)
		}
		if p.Action.Total != 22 || len(p.Action.DiceRolled) != 2 {
			t.Errorf("Action = %+v, want total 22 and the two dice", p.Action)
		}
		if p.Action.Margin == nil || *p.Action.Margin != 7 {
			t.Errorf("Margin = %v, want 7", p.Action.Margin)
		}
	})

	t.Run("carries the projected damage per target", func(t *testing.T) {
		if len(p.Targets) != 1 {
			t.Fatalf("Targets = %+v, want one entry", p.Targets)
		}
		if p.Targets[0].ProjectedDamage != 14 || !p.Targets[0].Defended {
			t.Errorf("Targets[0] = %+v", p.Targets[0])
		}
	})

	t.Run("serializes as camelCase", func(t *testing.T) {
		b, err := json.Marshal(p)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		s := string(b)
		for _, key := range []string{`"turnId"`, `"isSettled"`, `"diceRolled"`, `"projectedDamage"`} {
			if !strings.Contains(s, key) {
				t.Errorf("payload is missing %s: %s", key, s)
			}
		}
	})
}

func TestResolutionPayload_CarriesTheReaction(t *testing.T) {
	targetID := uuid.New()
	res := &service.TurnResolution{
		CharacterResults: []service.CharacterResult{{
			TargetID:        targetID,
			ReactionKind:    string(action.ReactRepel),
			Ladder:          service.LadderOutcome{Rung: service.RungNearMiss, Margin: -4, Difference: 4},
			Avoided:         true,
			RawDamage:       12,
			EffectiveDamage: 0,
		}},
	}
	p := newResolutionUpdatedPayload(uuid.New(), res)
	if len(p.Targets) != 1 || p.Targets[0].Reaction == nil {
		t.Fatal("the master has to see what the target answered with")
	}
	got := p.Targets[0].Reaction
	if got.Kind != "repel" || got.Rung != "near_miss" || got.Difference != 4 {
		t.Fatalf("reaction payload = %+v, want the kind, the rung and the difference", got)
	}
	if p.Targets[0].ProjectedDamage != 0 {
		t.Error("a parry is zero damage, not reduced damage")
	}
}

func TestResolutionPayload_NoReactionOpenedOmitsIt(t *testing.T) {
	res := &service.TurnResolution{
		CharacterResults: []service.CharacterResult{{
			TargetID:     uuid.New(),
			ReactionKind: "",
		}},
	}
	p := newResolutionUpdatedPayload(uuid.New(), res)
	if len(p.Targets) != 1 {
		t.Fatalf("Targets = %+v, want one entry", p.Targets)
	}
	if p.Targets[0].Reaction != nil {
		t.Errorf("Reaction = %+v, want nil — nothing was opened, so there is no answer to report", p.Targets[0].Reaction)
	}
}

func TestNewResolutionUpdatedPayload_NilResolution(t *testing.T) {
	turnID := uuid.New()
	p := newResolutionUpdatedPayload(turnID, nil)
	if p.TurnID != turnID {
		t.Errorf("TurnID = %v, want %v", p.TurnID, turnID)
	}
	if len(p.Targets) != 0 {
		t.Errorf("Targets = %+v, want empty", p.Targets)
	}
	// An empty slice rather than null, so a client can iterate it unconditionally.
	b, _ := json.Marshal(p)
	if !strings.Contains(string(b), `"targets":[]`) {
		t.Errorf("expected an empty array for targets, got %s", string(b))
	}
}

// newCombatSheet builds a minimal character sheet for combat fixtures. This package's tests
// use the INTERNAL test package (package game), so it cannot reach combat_e2e_test.go's copy,
// which lives in the external package game_test.
func newCombatSheet(t *testing.T) *csSheet.CharacterSheet {
	t.Helper()
	playerUUID := uuid.New()
	cs, err := csSheet.NewCharacterSheetFactory().Build(
		&playerUUID, nil, nil,
		csSheet.CharacterProfile{NickName: "Combatant", FullName: "Combat Test Subject"},
		nil, nil, nil,
	)
	if err != nil {
		t.Fatalf("factory.Build error: %v", err)
	}
	return cs
}

// topFaceSource lands every die on its top face, so the numbers in the assertions are exact.
type topFaceSource struct{}

func (topFaceSource) RollDie(sides enum.DieSides) int { return sides.GetSides() }

// racingSessionWithTwoActors builds a Race session holding two characters, enqueues one
// action for each, and opens the first — so the bar has a frozen price, one character with a
// recorded speed and one still pending.
func racingSessionWithTwoActors(t *testing.T) (*matchsession.MatchSession, uuid.UUID, uuid.UUID) {
	t.Helper()
	matchUUID := uuid.New()
	playerUUID := uuid.New()
	char1, char2 := uuid.New(), uuid.New()
	participants := []*match.Participant{
		{UUID: uuid.New(), MatchUUID: matchUUID, Sheet: csEntity.Summary{UUID: char1, PlayerUUID: &playerUUID}},
		{UUID: uuid.New(), MatchUUID: matchUUID, Sheet: csEntity.Summary{UUID: char2, PlayerUUID: &playerUUID}},
	}
	sheets := map[uuid.UUID]*csSheet.CharacterSheet{
		char1: newCombatSheet(t),
		char2: newCombatSheet(t),
	}
	s := matchsession.NewMatchSession(matchUUID, sheets, participants)
	s.GetActiveRound().SetMode(enum.Race)
	s.SetRollSource(topFaceSource{})

	for _, charID := range []uuid.UUID{char1, char2} {
		a := action.NewAction(charID, nil, uuid.Nil, nil, action.ActionSpeed{},
			nil, nil, &action.Attack{}, nil, nil, nil, nil)
		if err := s.EnqueueAction(playerUUID, a); err != nil {
			t.Fatalf("EnqueueAction: %v", err)
		}
	}
	if _, err := s.OpenNextAction(); err != nil {
		t.Fatalf("OpenNextAction: %v", err)
	}
	return s, char1, char2
}

func TestNewBarsUpdatedPayload(t *testing.T) {
	session, p1, p2 := racingSessionWithTwoActors(t)

	payload := newBarsUpdatedPayload(session)

	t.Run("carries the frozen price of each bar that priced", func(t *testing.T) {
		if _, ok := payload.Prices[string(action.BarAction)]; !ok {
			t.Error("the action bar priced and the table must see it")
		}
	})

	t.Run("carries one row per character, both bars", func(t *testing.T) {
		if len(payload.Characters) != 2 {
			t.Fatalf("characters = %d, want 2", len(payload.Characters))
		}
		for _, c := range payload.Characters {
			if c.CharacterID != p1 && c.CharacterID != p2 {
				t.Errorf("unexpected character %v", c.CharacterID)
			}
		}
	})

	t.Run("carries the projected order, and nothing that identifies an action", func(t *testing.T) {
		if len(payload.Order) == 0 {
			t.Fatal("something is pending, so the general bar has an order to show")
		}
		// Keys descend: this IS the order the master will open them in.
		for i := 1; i < len(payload.Order); i++ {
			if payload.Order[i-1].Key < payload.Order[i].Key {
				t.Error("the order must be sorted by key, highest first")
			}
		}
		raw, err := json.Marshal(payload.Order[0])
		if err != nil {
			t.Fatal(err)
		}
		// Whitelist, not a blocklist: a blocklist of forbidden substrings ("actionId",
		// "attack", ...) only catches leaks under the names someone thought to list — a
		// field named weaponId or abilityId would sail straight through. Asserting the
		// exact key set means any new field fails closed by default, and adding one to
		// BarSlotPayload forces a conscious edit to the allowed set below. Do not revert
		// this to a substring blocklist.
		allowed := map[string]bool{"actorId": true, "bars": true, "key": true}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil {
			t.Fatal(err)
		}
		for key := range fields {
			if !allowed[key] {
				t.Errorf("the queue is secret — only the bar and the order are public; found undeclared field %q in %s", key, raw)
			}
		}
	})
}

// TestBroadcastBars_StampsARisingSequence pins the ordering guarantee bars_updated needs.
//
// The payload is a FULL STATE snapshot and it reaches the broadcast channel from a detached
// goroutine, so two opens in quick succession race on that send and the older snapshot can be
// delivered last. Nothing else corrects it — the snapshot that closes the round is the last one
// the table gets. Seq is what lets a client throw the stale one away.
func TestBroadcastBars_StampsARisingSequence(t *testing.T) {
	session, _, _ := racingSessionWithTwoActors(t)
	r := &Room{broadcast: make(chan []byte, 8)}

	r.broadcastBars(session)
	r.broadcastBars(session)

	seqs := make([]uint64, 0, 2)
	for i := 0; i < 2; i++ {
		select {
		case data := <-r.broadcast:
			var msg Message
			if err := json.Unmarshal(data, &msg); err != nil {
				t.Fatalf("unmarshal message: %v", err)
			}
			if msg.Type != MsgTypeBarsUpdated {
				t.Fatalf("message type = %q, want %q", msg.Type, MsgTypeBarsUpdated)
			}
			var p BarsUpdatedPayload
			if err := json.Unmarshal(msg.Payload, &p); err != nil {
				t.Fatalf("unmarshal bars_updated: %v", err)
			}
			seqs = append(seqs, p.Seq)
		case <-time.After(2 * time.Second):
			t.Fatalf("only %d of 2 bars_updated reached the broadcast channel", i)
		}
	}

	sort.Slice(seqs, func(i, j int) bool { return seqs[i] < seqs[j] })
	// Sorted, because the sends race by design — what is pinned is the stamp, which is taken
	// under r.mu at snapshot time and is exactly what a receiver reorders by.
	if seqs[0] != 1 || seqs[1] != 2 {
		t.Errorf("seqs = %v, want [1 2] — the counter is stamped at snapshot time and rises by one", seqs)
	}
}

// TestResolutionUpdatedPayloadCarriesEngineFaults pins the wire half of the two faults the
// resolver used to swallow. The TODOs that marked them asked for them to be surfaced in the
// resolution "for caller to surface" — this is that caller.
func TestResolutionUpdatedPayloadCarriesEngineFaults(t *testing.T) {
	turnID, ghostID := uuid.New(), uuid.New()
	res := &service.TurnResolution{
		IsSettled: true,
		Errors: []service.ResolutionError{{
			Subject: ghostID,
			Kind:    service.ResolutionErrUnknownTarget,
			Detail:  "action target is neither a character nor a wall segment",
		}},
	}

	p := newResolutionUpdatedPayload(turnID, res)

	if len(p.Errors) != 1 {
		t.Fatalf("Errors = %+v, want the one fault the resolution reported", p.Errors)
	}
	if p.Errors[0].Kind != string(service.ResolutionErrUnknownTarget) {
		t.Errorf("Kind = %q, want %q", p.Errors[0].Kind, service.ResolutionErrUnknownTarget)
	}
	if p.Errors[0].Subject != ghostID {
		t.Errorf("Subject = %s, want %s", p.Errors[0].Subject, ghostID)
	}

	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"errors"`) {
		t.Errorf("the wire shape has no errors key: %s", raw)
	}

	t.Run("a clean resolution omits the key entirely", func(t *testing.T) {
		clean, err := json.Marshal(newResolutionUpdatedPayload(turnID, &service.TurnResolution{}))
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if strings.Contains(string(clean), `"errors"`) {
			t.Errorf("a clean resolution advertised an errors key: %s", clean)
		}
	})
}

// TestResolutionUpdatedPayloadCarriesPayouts is what makes the projection fix REACHABLE.
//
// ProjectResolution stopped zeroing Payouts for a reaction whose label was not demoted — but
// no wire field carried them, so a repel's penalty still reached nobody and the fix was
// theoretical. reacoes.md: "a penalidade de quem aparou vale contra todo mundo — qualquer um
// pode aproveitar". Someone has to be able to read it.
func TestResolutionUpdatedPayloadCarriesPayouts(t *testing.T) {
	turnID, targetID, attacker := uuid.New(), uuid.New(), uuid.New()

	res := &service.TurnResolution{
		IsSettled: true,
		CharacterResults: []service.CharacterResult{{
			TargetID:     targetID,
			ReactionKind: string(action.ReactRepel),
			Ladder:       service.LadderOutcome{Rung: service.RungNearMiss, Difference: 3},
			Payouts: []match.Modifier{{
				Amount: -3, Applies: match.DimActionSpeed, Source: match.SourceSystem,
				Against: match.ScopeAnyone(), ExpiresAt: match.LifetimeNextTurn,
				Reason: "repel: near miss penalty",
			}},
		}},
	}

	p := newResolutionUpdatedPayload(turnID, res)
	if len(p.Targets) != 1 || len(p.Targets[0].Payouts) != 1 {
		t.Fatalf("Targets = %+v, want one target carrying one payout", p.Targets)
	}
	got := p.Targets[0].Payouts[0]
	if got.Amount != -3 || got.Applies != string(match.DimActionSpeed) ||
		got.Source != string(match.SourceSystem) || got.ExpiresAt != string(match.LifetimeNextTurn) ||
		got.Reason != "repel: near miss penalty" {
		t.Errorf("payout = %+v, want the near-miss penalty verbatim", got)
	}
	if got.AgainstKind != match.ScopeAnyone().Kind() {
		t.Errorf("againstKind = %q, want %q — the scope is what says who may exploit it",
			got.AgainstKind, match.ScopeAnyone().Kind())
	}

	t.Run("the scope's target survives when there is one", func(t *testing.T) {
		res.CharacterResults[0].Payouts[0].Against = match.ScopeOnly(attacker)
		p := newResolutionUpdatedPayload(turnID, res)
		got := p.Targets[0].Payouts[0]
		if got.AgainstKind != match.ScopeOnly(attacker).Kind() || got.AgainstID != attacker {
			t.Errorf("against = %q/%s, want only/%s", got.AgainstKind, got.AgainstID, attacker)
		}
	})

	t.Run("a target with no payout omits the key", func(t *testing.T) {
		clean := &service.TurnResolution{
			IsSettled:        true,
			CharacterResults: []service.CharacterResult{{TargetID: targetID}},
		}
		raw, err := json.Marshal(newResolutionUpdatedPayload(turnID, clean))
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if strings.Contains(string(raw), `"payouts"`) {
			t.Errorf("a target with no payout advertised a payouts key: %s", raw)
		}
	})
}

// TestProjectedPayoutsReachTheRightRecipients drives the DOMAIN projection and the WIRE
// mapper together, because the two halves only mean something as a pair: the deny-list lives
// in service.ProjectResolution and the field that makes it observable lives here.
func TestProjectedPayoutsReachTheRightRecipients(t *testing.T) {
	repelTarget, dodgeTarget, third := uuid.New(), uuid.New(), uuid.New()
	turnID := uuid.New()

	res := &service.TurnResolution{
		IsSettled: true,
		CharacterResults: []service.CharacterResult{
			{
				TargetID: repelTarget, ReactionKind: string(action.ReactRepel),
				Payouts: []match.Modifier{{
					Amount: -3, Applies: match.DimActionSpeed, Against: match.ScopeAnyone(),
					Reason: "repel: near miss penalty",
				}},
			},
			{
				TargetID: dodgeTarget, ReactionKind: string(action.ReactClosedDodge),
				Payouts: []match.Modifier{{
					Amount: 4, Applies: match.DimDodge, Against: match.ScopeAllBut(uuid.New()),
					Reason: "closed dodge reserve",
				}},
			},
		},
	}

	v := service.Viewer{Owns: map[uuid.UUID]bool{third: true}}
	p := newResolutionUpdatedPayload(turnID, service.ProjectResolution(res, v))

	byTarget := map[uuid.UUID]CharacterResultPayload{}
	for _, cr := range p.Targets {
		byTarget[cr.TargetID] = cr
	}

	repel := byTarget[repelTarget]
	if repel.Reaction == nil || repel.Reaction.Kind != string(action.ReactRepel) {
		t.Fatalf("the repel was demoted: %+v", repel.Reaction)
	}
	if len(repel.Payouts) != 1 || repel.Payouts[0].Reason != "repel: near miss penalty" {
		t.Errorf("a third party cannot see the repel penalty: %+v — the rule says anyone "+
			"may exploit it", repel.Payouts)
	}

	dodge := byTarget[dodgeTarget]
	if dodge.Reaction == nil || dodge.Reaction.Kind != string(action.ReactDodge) {
		t.Fatalf("the closed dodge was not demoted: %+v", dodge.Reaction)
	}
	if len(dodge.Payouts) != 0 {
		t.Errorf("the closed dodge's reserve leaked to a third party: %+v", dodge.Payouts)
	}
}
