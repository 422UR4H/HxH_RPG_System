package game

import (
	"encoding/json"
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
		t.Errorf("Reaction = %+v, want nil — the turn held no reaction for this target", p.Targets[0].Reaction)
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

// racingSessionWithTwoQueued builds a Race session holding two characters with one action
// each in the queue and nothing opened — so no bar has frozen yet, and the master's view has a
// provisional price to show.
func racingSessionWithTwoQueued(t *testing.T) (*matchsession.MatchSession, uuid.UUID, uuid.UUID) {
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
	return s, char1, char2
}

// racingSessionWithTwoActors is racingSessionWithTwoQueued with the first action opened — so
// the bar has a frozen price, one character with a recorded speed and one still pending.
func racingSessionWithTwoActors(t *testing.T) (*matchsession.MatchSession, uuid.UUID, uuid.UUID) {
	t.Helper()
	s, char1, char2 := racingSessionWithTwoQueued(t)
	if _, err := s.OpenNextAction(); err != nil {
		t.Fatalf("OpenNextAction: %v", err)
	}
	return s, char1, char2
}

func TestNewBarsUpdatedPayload(t *testing.T) {
	t.Run("master", func(t *testing.T) {
		session, p1, p2 := racingSessionWithTwoActors(t)
		payload := newBarsUpdatedPayload(session, true)

		t.Run("carries the frozen price of each bar that priced", func(t *testing.T) {
			if _, ok := payload.Prices[string(action.BarAction)]; !ok {
				t.Error("the action bar priced and the table must see it")
			}
		})

		t.Run("carries one row per character, both bars, sorted by id", func(t *testing.T) {
			if len(payload.Characters) != 2 {
				t.Fatalf("characters = %d, want 2", len(payload.Characters))
			}
			for _, c := range payload.Characters {
				if c.CharacterID != p1 && c.CharacterID != p2 {
					t.Errorf("unexpected character %v", c.CharacterID)
				}
			}
			if payload.Characters[0].CharacterID.String() > payload.Characters[1].CharacterID.String() {
				t.Error("characters must be sorted by characterId")
			}
		})

		t.Run("carries the queued actor's speed as pending, the opened one's as acted", func(t *testing.T) {
			var acted, pending int
			for _, c := range payload.Characters {
				if len(c.ActionSpeeds) == 1 && len(c.PendingActionSpeeds) == 0 {
					acted++
				}
				if len(c.PendingActionSpeeds) == 1 && len(c.ActionSpeeds) == 0 {
					pending++
				}
			}
			if acted != 1 || pending != 1 {
				t.Errorf("acted rows = %d, pending rows = %d, want 1 and 1: %+v", acted, pending, payload.Characters)
			}
		})

		t.Run("carries provisionalPrices while nothing has frozen", func(t *testing.T) {
			queued, _, _ := racingSessionWithTwoQueued(t)
			p := newBarsUpdatedPayload(queued, true)
			if _, ok := p.ProvisionalPrices[string(action.BarAction)]; !ok {
				t.Errorf("provisionalPrices = %v, want the action bar's provisional price", p.ProvisionalPrices)
			}
			if _, ok := p.Prices[string(action.BarAction)]; ok {
				t.Error("nothing opened, so nothing froze")
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
					t.Errorf("the queue is secret — the order carries no action identity; found undeclared field %q in %s", key, raw)
				}
			}
		})
	})

	t.Run("player", func(t *testing.T) {
		queued, _, _ := racingSessionWithTwoQueued(t)
		opened, _, _ := racingSessionWithTwoActors(t)
		for name, session := range map[string]*matchsession.MatchSession{
			"nothing opened yet": queued, "one opened, one pending": opened,
		} {
			t.Run(name, func(t *testing.T) {
				raw, err := json.Marshal(newBarsUpdatedPayload(session, false))
				if err != nil {
					t.Fatal(err)
				}
				js := string(raw)
				// Pending actions and unopened dice are the master's secret (spec D1): a
				// player's payload carries none of the keys that would describe them.
				for _, secret := range []string{
					`"provisionalPrices"`, `"pendingActionSpeeds"`, `"pendingMoveSpeeds"`,
					`"deferredActionSpeeds"`, `"deferredMoveSpeeds"`,
				} {
					if strings.Contains(js, secret) {
						t.Errorf("the player view carries %s: %s", secret, js)
					}
				}
				if !strings.Contains(js, `"order":[]`) {
					t.Errorf("the player view's order must be an empty array, got %s", js)
				}
			})
		}

		t.Run("shows the opened speed and hides the pending one", func(t *testing.T) {
			p := newBarsUpdatedPayload(opened, false)
			var withSpeed int
			for _, c := range p.Characters {
				if len(c.ActionSpeeds) > 0 {
					withSpeed++
				}
			}
			if withSpeed != 1 {
				t.Errorf("rows with a public speed = %d, want 1 (the opened action only): %+v", withSpeed, p.Characters)
			}
		})
	})
}

// barsRecipient is a registered client with no socket: dispatchPerPlayer writes to its send
// buffer, which the test drains.
func barsRecipient(r *Room, id uuid.UUID) *Client {
	c := &Client{userUUID: id, send: make(chan []byte, 16), done: make(chan struct{})}
	r.clients[id] = c
	return c
}

// readBars pops the next message off a recipient's buffer and decodes it as bars_updated.
func readBars(t *testing.T, c *Client) BarsUpdatedPayload {
	t.Helper()
	select {
	case data := <-c.send:
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
		return p
	default:
		t.Fatal("no bars_updated reached the recipient")
		return BarsUpdatedPayload{}
	}
}

// TestBroadcastBars_StampsARisingSequence pins the ordering guarantee bars_updated needs.
//
// The payload is a FULL STATE snapshot and every recipient gets their own copy of it. Seq is
// stamped once per snapshot under r.mu, so the master's copy and the player's copy of one
// snapshot carry the same number, and a client can throw away anything older than what it
// already applied.
func TestBroadcastBars_StampsARisingSequence(t *testing.T) {
	session, _, _ := racingSessionWithTwoActors(t)
	masterID, playerID := uuid.New(), uuid.New()
	r := &Room{clients: map[uuid.UUID]*Client{}, masterUUID: masterID}
	master := barsRecipient(r, masterID)
	player := barsRecipient(r, playerID)

	r.broadcastBars(session)
	r.broadcastBars(session)

	for i, want := range []uint64{1, 2} {
		m, p := readBars(t, master), readBars(t, player)
		if m.Seq != want || p.Seq != want {
			t.Errorf("snapshot %d: master seq %d, player seq %d, want both %d — one stamp per "+
				"snapshot, shared by every copy", i+1, m.Seq, p.Seq, want)
		}
		if len(m.Order) == 0 {
			t.Error("the master's copy is the full view: something is pending, so it has an order")
		}
		if len(p.Order) != 0 {
			t.Errorf("the player's copy carries order %+v — pending actions are the master's secret", p.Order)
		}
	}
}

// TestNewRoom_BarsSeqOutrunsAnEarlierRoom pins the contract's "the counter never restarts"
// across Rooms. A client keeps the highest seq it applied and drops anything lower, through
// reconnects — so a Room that replaces another for the same match (the process restarted, or
// the room emptied and closed itself in Run) must not start counting from 0 again, or every
// bars_updated it sends loses to the old room's last one and the client's bars freeze.
func TestNewRoom_BarsSeqOutrunsAnEarlierRoom(t *testing.T) {
	session, _, _ := racingSessionWithTwoActors(t)
	matchID, masterID := uuid.New(), uuid.New()
	lastSeq := func(r *Room, n int) uint64 {
		t.Helper()
		c := barsRecipient(r, masterID)
		var seq uint64
		for range n {
			r.broadcastBars(session)
			seq = max(seq, readBars(t, c).Seq)
		}
		return seq
	}

	old := lastSeq(NewRoom(matchID, masterID, RoomDeps{}), 3)
	// The old room ran for a while before the new one replaced it — a millisecond is already
	// a thousand times more time than the three snapshots it sent.
	time.Sleep(time.Millisecond)
	replacement := lastSeq(NewRoom(matchID, masterID, RoomDeps{}), 1)

	if replacement <= old {
		t.Errorf("the replacement room's first seq is %d, not above the old room's last %d — a "+
			"client that reconnects into it would drop every bars_updated it sends", replacement, old)
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

// B13: the escape's verdict reaches the wire as the domain read it, plus the one thing the
// wire derives — awaitsMaster, "failed and no landing chosen" — and landing only when chosen.
func TestResolutionUpdatedPayloadCarriesTheEscape(t *testing.T) {
	landing := [3]int{7, 6, 0}
	cases := []struct {
		name   string
		escape *service.EscapeResult
		want   *EscapeResultPayload
		// wantJSON is the substring the marshalled target must contain.
		wantJSON string
	}{
		{"not an escape", nil, nil, ""},
		{
			"escaped",
			&service.EscapeResult{MovePassed: true, DodgePassed: true, Escaped: true},
			&EscapeResultPayload{Escaped: true, MovePassed: true, DodgePassed: true},
			`"escape":{"escaped":true,"movePassed":true,"dodgePassed":true,"awaitsMaster":false}`,
		},
		{
			"failed, nothing chosen",
			&service.EscapeResult{DodgePassed: true},
			&EscapeResultPayload{DodgePassed: true, AwaitsMaster: true},
			`"escape":{"escaped":false,"movePassed":false,"dodgePassed":true,"awaitsMaster":true}`,
		},
		{
			"failed, landing chosen",
			&service.EscapeResult{MovePassed: true, Landing: &landing},
			&EscapeResultPayload{MovePassed: true, Landing: &landing},
			`"awaitsMaster":false,"landing":[7,6,0]}`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := &service.TurnResolution{CharacterResults: []service.CharacterResult{{
				TargetID: uuid.New(), Escape: c.escape,
			}}}
			p := newResolutionUpdatedPayload(uuid.New(), res)
			got := p.Targets[0].Escape
			if (got == nil) != (c.want == nil) {
				t.Fatalf("escape = %+v, want %+v", got, c.want)
			}
			if got != nil {
				if got.Escaped != c.want.Escaped || got.MovePassed != c.want.MovePassed ||
					got.DodgePassed != c.want.DodgePassed || got.AwaitsMaster != c.want.AwaitsMaster ||
					(got.Landing == nil) != (c.want.Landing == nil) ||
					(got.Landing != nil && *got.Landing != *c.want.Landing) {
					t.Fatalf("escape = %+v, want %+v", *got, *c.want)
				}
			}
			raw, err := json.Marshal(p.Targets[0])
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if c.want == nil && strings.Contains(string(raw), `"escape"`) {
				t.Fatalf("a non-escape carries an escape key: %s", raw)
			}
			if c.wantJSON != "" && !strings.Contains(string(raw), c.wantJSON) {
				t.Fatalf("JSON = %s, want it to contain %s", raw, c.wantJSON)
			}
		})
	}
}

// TestResolutionPayloadProjectsAStoppedChain: a target the blow reached already stopped by
// an earlier parry was not hit, so avoided is true on the wire, and attackStopped says why.
func TestResolutionPayloadProjectsAStoppedChain(t *testing.T) {
	stopped, plain := uuid.New(), uuid.New()
	res := &service.TurnResolution{CharacterResults: []service.CharacterResult{
		{TargetID: stopped, Avoided: false, AttackStopped: true},
		{TargetID: plain},
	}}

	p := newResolutionUpdatedPayload(uuid.New(), res)

	if !p.Targets[0].Avoided || !p.Targets[0].AttackStopped {
		t.Errorf("stopped target = %+v, want avoided and attackStopped", p.Targets[0])
	}
	if p.Targets[1].Avoided || p.Targets[1].AttackStopped {
		t.Errorf("plain target = %+v, want neither flag", p.Targets[1])
	}
	raw, err := json.Marshal(p.Targets[1])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "attackStopped") {
		t.Errorf("attackStopped must be omitted when false: %s", raw)
	}
}

func TestResolutionUpdatedPayloadCarriesTheRegency(t *testing.T) {
	actionID := uuid.New()
	res := &service.TurnResolution{
		DamageSkill: "Grab",
		Conditions: []service.CheckCondition{
			{ActionID: actionID, Field: "hit", Condition: action.RollCondition{Bias: -1, Modifier: -2, Description: "escuridao"}},
			{ActionID: actionID, SkillName: "Evasion", Condition: action.RollCondition{Bias: 1}},
		},
	}
	raw, err := json.Marshal(newResolutionUpdatedPayload(uuid.New(), res))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got struct {
		DamageSkill string           `json:"damageSkill"`
		Conditions  []map[string]any `json:"conditions"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.DamageSkill != "Grab" {
		t.Errorf("damageSkill = %q, want Grab", got.DamageSkill)
	}
	if len(got.Conditions) != 2 {
		t.Fatalf("conditions = %v, want 2 entries", got.Conditions)
	}
	first := got.Conditions[0]
	if first["actionId"] != actionID.String() || first["field"] != "hit" ||
		first["bias"] != float64(-1) || first["modifier"] != float64(-2) || first["description"] != "escuridao" {
		t.Errorf("conditions[0] = %v", first)
	}
	if _, ok := first["skillName"]; ok {
		t.Error("conditions[0] carries skillName — field and skillName are alternatives")
	}
	if got.Conditions[1]["skillName"] != "Evasion" {
		t.Errorf("conditions[1] = %v, want skillName Evasion", got.Conditions[1])
	}

	empty, _ := json.Marshal(newResolutionUpdatedPayload(uuid.New(), &service.TurnResolution{}))
	if strings.Contains(string(empty), "conditions") || strings.Contains(string(empty), "damageSkill") {
		t.Errorf("an empty resolution carries the keys anyway: %s", empty)
	}
}
