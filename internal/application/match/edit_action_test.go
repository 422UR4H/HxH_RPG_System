package match_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	matchDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

// ─── fixture ────────────────────────────────────────────────────────────────
//
// editFixture is internal/app/game/combat_e2e_test.go's combatFixture minus the HTTP
// server, hub and handler: the edit use case is driven directly against the session, the way
// every other application/match test already does (see attach_reaction_test.go).

// scriptedFaces hands out faces in order and NEVER repeats: once exhausted, it records an
// overrun instead of silently replaying the last face — see combat_e2e_test.go's same-named
// type for the full rationale (a repeat can never be told apart from a genuine match by a
// test that relies on it). That one lives in internal/app/game, a different Go package, and
// is unexported there, so this package needs its own copy rather than importing it.
type scriptedFaces struct {
	faces   []int
	i       int
	overran bool // set once a roll asks for a face beyond what was scripted
}

func (s *scriptedFaces) RollDie(_ enum.DieSides) int {
	if s.i >= len(s.faces) {
		s.overran = true
		return -1 // an impossible face: anything that accidentally depends on it fails loudly
	}
	f := s.faces[s.i]
	s.i++
	return f
}

// consumed is how many faces the scripted source has handed out. It is the only way to prove
// "no die was re-rolled" — asserting on the numbers cannot distinguish a re-roll that
// happened to land the same.
func (s *scriptedFaces) consumed() int { return s.i }

type editFixture struct {
	matchUUID  uuid.UUID
	masterUUID uuid.UUID
	playerUUID uuid.UUID
	attackerID uuid.UUID // sheet UUID of the attacking character
	victimID   uuid.UUID // sheet UUID of the target
	// thirdCharacterID is a bystander: present in the session but never targeted by the
	// fixture's attack, on the shape of newCombatFixture's roster.
	thirdCharacterID uuid.UUID
	// actorID is set only by newOpenMoveFixture: attackerID/victimID name the two roles of an
	// attack fixture, but a pure-move fixture has just the one character.
	actorID uuid.UUID

	session    *matchsession.MatchSession
	rollSource *scriptedFaces

	actionID uuid.UUID // the open turn's own action
	// facesAtOpen is rollSource.consumed() right after the fixture opens the turn — the
	// baseline the advantage test checks an edit never moves.
	facesAtOpen int
	// primaryTotal is ActionResult.Total read off the Primary set alone, before any edit —
	// what the advantage test proves the edit reads PAST, onto the (scripted higher) Secondary
	// set.
	primaryTotal int
}

// currentResolution re-resolves the open turn without mutating it.
func (f *editFixture) currentResolution(t *testing.T) *service.TurnResolution {
	t.Helper()
	return f.session.ResolveTurn(f.openTurn())
}

// openTurn reads the session's currently open turn.
func (f *editFixture) openTurn() *turn.Turn {
	return f.session.GetActiveRound().CurrentTurn()
}

// newOpenAttackFixture builds a session with an attacker, a victim and a third character,
// enqueues one Sword attack from the attacker against the victim, and opens it — exactly the
// state a master's condition edit lands on.
//
// The dice budget: one Sword attack, in the session's default Free round mode, rolls the hit
// (a 2D10 test — 4 faces, Primary AND Secondary, because Roll always rolls both even with no
// advantage) and then the weapon's own damage dice (Sword is D10+D4, Primary only — damage has
// no advantage). Free mode never rolls actionSpeed, so that is 6 faces for the open action
// itself, plus 4 more reserved and unused by every RollCondition test: a Skills-edit test
// reuses this same fixture and adds one skill afterwards, which is its own 2D10 test.
//
// Secondary is scripted strictly higher than Primary on the hit check, so a later advantage
// edit provably switches which already-rolled set is read.
func newOpenAttackFixture(t *testing.T) *editFixture {
	t.Helper()

	f := &editFixture{
		matchUUID:        uuid.New(),
		masterUUID:       uuid.New(),
		playerUUID:       uuid.New(),
		attackerID:       uuid.New(),
		victimID:         uuid.New(),
		thirdCharacterID: uuid.New(),
	}

	// All three characters belong to the same player — enough here, exactly as
	// newCombatFixture: authorization is per player and only one client ever sends the attack.
	attacker := &matchDomain.Participant{
		UUID: uuid.New(), MatchUUID: f.matchUUID,
		Sheet: csEntity.Summary{UUID: f.attackerID, PlayerUUID: &f.playerUUID},
	}
	victim := &matchDomain.Participant{
		UUID: uuid.New(), MatchUUID: f.matchUUID,
		Sheet: csEntity.Summary{UUID: f.victimID, PlayerUUID: &f.playerUUID},
	}
	third := &matchDomain.Participant{
		UUID: uuid.New(), MatchUUID: f.matchUUID,
		Sheet: csEntity.Summary{UUID: f.thirdCharacterID, PlayerUUID: &f.playerUUID},
	}

	sheets := map[uuid.UUID]*csSheet.CharacterSheet{
		f.attackerID:       newEditSheet(t),
		f.victimID:         newEditSheet(t),
		f.thirdCharacterID: newEditSheet(t),
	}
	session := matchsession.NewMatchSession(
		f.matchUUID, sheets, []*matchDomain.Participant{attacker, victim, third},
	)

	f.rollSource = &scriptedFaces{faces: []int{
		3, 4, // Hit Primary: sum 7
		9, 8, // Hit Secondary: sum 17 — strictly higher, so advantage must switch sets
		5, 2, // Sword damage: D10, D4
		6, 7, 1, 9, // reserved for a later skill edit (2D10, unused by RollCondition tests)
	}}
	session.SetRollSource(f.rollSource)
	f.session = session

	weapon := enum.Sword
	act := action.NewAction(
		f.attackerID, []uuid.UUID{f.victimID}, uuid.Nil, nil,
		action.ActionSpeed{RollCheck: action.RollCheck{SkillName: enum.Legerity.String()}},
		nil, nil,
		&action.Attack{
			Weapon: &weapon,
			Hit:    action.RollCheck{SkillName: enum.Accuracy.String()},
			Damage: action.RollCheck{SkillName: enum.Push.String()},
		},
		nil, nil, nil, nil,
	)
	if err := session.EnqueueAction(f.playerUUID, act); err != nil {
		t.Fatalf("EnqueueAction: %v", err)
	}

	tr, err := session.OpenNextAction()
	if err != nil {
		t.Fatalf("OpenNextAction: %v", err)
	}
	openedAction := tr.Opened.GetAction()
	f.actionID = openedAction.GetID()
	f.facesAtOpen = f.rollSource.consumed()
	f.primaryTotal = f.currentResolution(t).ActionResult.Total
	return f
}

// newOpenAttackFixtureWithSkill is newOpenAttackFixture, but the attack carries one skill from
// the very start, so its dice fall the moment the action is enqueued — this is the "original"
// roll the removed-and-restored test proves comes back unchanged.
//
// Dice budget: rollActionDice walks Skills BEFORE Attack, so the skill's own 2D10 test (4
// faces) rolls first, then the hit (4 faces), then the weapon's damage (2 faces, Primary
// only — damage has no advantage). 10 faces total. Free mode never rolls actionSpeed.
func newOpenAttackFixtureWithSkill(t *testing.T, skillName string) *editFixture {
	t.Helper()

	f := &editFixture{
		matchUUID:        uuid.New(),
		masterUUID:       uuid.New(),
		playerUUID:       uuid.New(),
		attackerID:       uuid.New(),
		victimID:         uuid.New(),
		thirdCharacterID: uuid.New(),
	}

	attacker := &matchDomain.Participant{
		UUID: uuid.New(), MatchUUID: f.matchUUID,
		Sheet: csEntity.Summary{UUID: f.attackerID, PlayerUUID: &f.playerUUID},
	}
	victim := &matchDomain.Participant{
		UUID: uuid.New(), MatchUUID: f.matchUUID,
		Sheet: csEntity.Summary{UUID: f.victimID, PlayerUUID: &f.playerUUID},
	}
	third := &matchDomain.Participant{
		UUID: uuid.New(), MatchUUID: f.matchUUID,
		Sheet: csEntity.Summary{UUID: f.thirdCharacterID, PlayerUUID: &f.playerUUID},
	}

	sheets := map[uuid.UUID]*csSheet.CharacterSheet{
		f.attackerID:       newEditSheet(t),
		f.victimID:         newEditSheet(t),
		f.thirdCharacterID: newEditSheet(t),
	}
	session := matchsession.NewMatchSession(
		f.matchUUID, sheets, []*matchDomain.Participant{attacker, victim, third},
	)

	f.rollSource = &scriptedFaces{faces: []int{
		1, 2, // skill Primary: sum 3
		3, 4, // skill Secondary: sum 7
		5, 6, // Hit Primary
		7, 8, // Hit Secondary
		9, 1, // Sword damage: D10, D4
	}}
	session.SetRollSource(f.rollSource)
	f.session = session

	weapon := enum.Sword
	act := action.NewAction(
		f.attackerID, []uuid.UUID{f.victimID}, uuid.Nil,
		[]action.Skill{{SkillName: skillName, RollCheck: action.RollCheck{SkillName: skillName}}},
		action.ActionSpeed{RollCheck: action.RollCheck{SkillName: enum.Legerity.String()}},
		nil, nil,
		&action.Attack{
			Weapon: &weapon,
			Hit:    action.RollCheck{SkillName: enum.Accuracy.String()},
			Damage: action.RollCheck{SkillName: enum.Push.String()},
		},
		nil, nil, nil, nil,
	)
	if err := session.EnqueueAction(f.playerUUID, act); err != nil {
		t.Fatalf("EnqueueAction: %v", err)
	}

	tr, err := session.OpenNextAction()
	if err != nil {
		t.Fatalf("OpenNextAction: %v", err)
	}
	openedAction := tr.Opened.GetAction()
	f.actionID = openedAction.GetID()
	f.facesAtOpen = f.rollSource.consumed()
	return f
}

// newOpenMoveFixture builds a session with a single character and opens a pure-move action —
// Bars() answers just [BarMove], nothing on the action bar. Race mode, deliberately: outside
// Race, recordActed no-ops unconditionally (see MatchSession.recordActed), which would make
// the "does not start charging the action bar" test pass vacuously no matter what the edit
// does. Under Race, opening the action actually records a speed on the move bar, so the test
// can prove the action bar's Speeds are genuinely untouched by the edit.
//
// PullAction, not OpenNextAction: it is explicitly ungated ("the master's explicit pull_action
// stays ungated on purpose"), so a single queued action opens without wiring up the
// scheduler's Race eligibility gate.
//
// Dice budget: Race rolls actionSpeed (4 faces). The move is a Shift, which takes Brake's
// passive value and rolls nothing (see rollActionDice's Shift carve-out). 4 faces total.
func newOpenMoveFixture(t *testing.T) *editFixture {
	t.Helper()

	f := &editFixture{
		matchUUID:  uuid.New(),
		masterUUID: uuid.New(),
		playerUUID: uuid.New(),
		actorID:    uuid.New(),
	}

	actor := &matchDomain.Participant{
		UUID: uuid.New(), MatchUUID: f.matchUUID,
		Sheet: csEntity.Summary{UUID: f.actorID, PlayerUUID: &f.playerUUID},
	}
	sheets := map[uuid.UUID]*csSheet.CharacterSheet{
		f.actorID: newEditSheet(t),
	}
	session := matchsession.NewMatchSession(
		f.matchUUID, sheets, []*matchDomain.Participant{actor},
	)
	session.GetActiveRound().SetMode(enum.Race)

	f.rollSource = &scriptedFaces{faces: []int{
		1, 2, // actionSpeed Primary
		3, 4, // actionSpeed Secondary
	}}
	session.SetRollSource(f.rollSource)
	f.session = session

	act := action.NewAction(
		f.actorID, nil, uuid.Nil, nil,
		action.ActionSpeed{RollCheck: action.RollCheck{SkillName: enum.Legerity.String()}},
		nil,
		&action.Move{Category: enum.Shift, Speed: &action.RollCheck{SkillName: enum.Brake.String()}},
		nil, nil, nil, nil, nil,
	)
	if err := session.EnqueueAction(f.playerUUID, act); err != nil {
		t.Fatalf("EnqueueAction: %v", err)
	}

	tr, err := session.PullAction(act.GetID())
	if err != nil {
		t.Fatalf("PullAction: %v", err)
	}
	openedAction := tr.Opened.GetAction()
	f.actionID = openedAction.GetID()
	f.facesAtOpen = f.rollSource.consumed()
	return f
}

// skillOn locates one entry of the open turn's own action's Skills by name, or fails the
// test — how the skills tests read the exact dice a skill carries after an edit.
func (f *editFixture) skillOn(t *testing.T, name string) action.Skill {
	t.Helper()
	a := f.openTurn().ActionRef()
	for _, sk := range a.Skills {
		if sk.SkillName == name {
			return sk
		}
	}
	t.Fatalf("no skill named %q on the open action", name)
	return action.Skill{}
}

func newEditSheet(t *testing.T) *csSheet.CharacterSheet {
	t.Helper()
	playerUUID := uuid.New()
	cs, err := csSheet.NewCharacterSheetFactory().Build(
		&playerUUID, nil, nil,
		csSheet.CharacterProfile{NickName: "Combatant", FullName: "Edit Test Subject"},
		nil, nil, nil,
	)
	if err != nil {
		t.Fatalf("factory.Build error: %v", err)
	}
	return cs
}

// ─── the tests ──────────────────────────────────────────────────────────────

func TestEditActionRollCondition(t *testing.T) {
	t.Run("a flat modifier moves the total without touching the dice", func(t *testing.T) {
		f := newOpenAttackFixture(t) // one open turn, attack vs one target, scripted dice
		before := f.currentResolution(t)
		diceBefore := append([]int(nil), before.ActionResult.DiceRolled...)

		uc := match.NewEditActionUC()
		ma := action.NewMasterAction()
		ma.ActionID = f.actionID
		ma.Conditions = []action.ConditionEdit{{
			Field:     action.FieldHit,
			Condition: action.RollCondition{Modifier: 3, Description: "creative positioning"},
		}}

		res, err := uc.Execute(context.Background(), f.session, f.masterUUID, f.masterUUID, ma)
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if got, want := res.Resolution.ActionResult.Total, before.ActionResult.Total+3; got != want {
			t.Fatalf("Total = %d, want %d", got, want)
		}
		if !slices.Equal(res.Resolution.ActionResult.DiceRolled, diceBefore) {
			t.Fatalf("the dice changed: %v → %v", diceBefore, res.Resolution.ActionResult.DiceRolled)
		}
	})

	t.Run("advantage reads the other set that was already rolled", func(t *testing.T) {
		f := newOpenAttackFixture(t)
		uc := match.NewEditActionUC()
		ma := action.NewMasterAction()
		ma.ActionID = f.actionID
		ma.Conditions = []action.ConditionEdit{{
			Field: action.FieldHit, Condition: action.RollCondition{Bias: 1},
		}}

		res, err := uc.Execute(context.Background(), f.session, f.masterUUID, f.masterUUID, ma)
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		// The fixture scripts Secondary strictly higher than Primary, so advantage must
		// switch which set is read — proving RollAttempts is why the edit needs no re-roll.
		if res.Resolution.ActionResult.Total <= f.primaryTotal {
			t.Fatalf("advantage did not switch to the better set (total %d)",
				res.Resolution.ActionResult.Total)
		}
		if f.rollSource.consumed() != f.facesAtOpen {
			t.Fatalf("the edit consumed %d extra faces — a die was re-rolled",
				f.rollSource.consumed()-f.facesAtOpen)
		}
	})

	t.Run("the edit never moves the economy", func(t *testing.T) {
		f := newOpenAttackFixture(t)
		balanceBefore, speedsBefore := f.session.BarState(f.attackerID, action.BarAction)

		uc := match.NewEditActionUC()
		ma := action.NewMasterAction()
		ma.ActionID = f.actionID
		ma.Conditions = []action.ConditionEdit{{
			Field: action.FieldSpeed, Condition: action.RollCondition{Modifier: 12},
		}}
		if _, err := uc.Execute(context.Background(), f.session, f.masterUUID, f.masterUUID, ma); err != nil {
			t.Fatalf("Execute: %v", err)
		}

		balanceAfter, speedsAfter := f.session.BarState(f.attackerID, action.BarAction)
		if balanceAfter != balanceBefore || !slices.Equal(speedsAfter, speedsBefore) {
			t.Fatalf("the edit rewrote the economy: %v/%v → %v/%v",
				balanceBefore, speedsBefore, balanceAfter, speedsAfter)
		}
	})

	t.Run("only the master edits", func(t *testing.T) {
		f := newOpenAttackFixture(t)
		uc := match.NewEditActionUC()
		_, err := uc.Execute(context.Background(), f.session, f.masterUUID, uuid.New(),
			action.NewMasterAction())
		if !errors.Is(err, match.ErrNotMatchMaster) {
			t.Fatalf("err = %v, want ErrNotMatchMaster", err)
		}
	})

	t.Run("a field the action does not carry is an error, not a silent no-op", func(t *testing.T) {
		f := newOpenAttackFixture(t)
		uc := match.NewEditActionUC()
		ma := action.NewMasterAction()
		ma.ActionID = f.actionID
		// The fixture's open action is a plain attack: it has no Dodge.
		ma.Conditions = []action.ConditionEdit{{
			Field: action.FieldDodge, Condition: action.RollCondition{Modifier: 1},
		}}
		_, err := uc.Execute(context.Background(), f.session, f.masterUUID, f.masterUUID, ma)
		if !errors.Is(err, matchsession.ErrConditionTargetMissing) {
			t.Fatalf("err = %v, want ErrConditionTargetMissing", err)
		}
	})

	t.Run("field and skillName together is ambiguous, not a preference", func(t *testing.T) {
		f := newOpenAttackFixture(t)
		uc := match.NewEditActionUC()
		ma := action.NewMasterAction()
		ma.ActionID = f.actionID
		ma.Conditions = []action.ConditionEdit{{
			Field: action.FieldHit, SkillName: "whatever",
			Condition: action.RollCondition{Modifier: 1},
		}}
		_, err := uc.Execute(context.Background(), f.session, f.masterUUID, f.masterUUID, ma)
		if !errors.Is(err, matchsession.ErrAmbiguousConditionEdit) {
			t.Fatalf("err = %v, want ErrAmbiguousConditionEdit", err)
		}
	})
}

// TestApplyMasterAction_PreservesReactionSwapDisadvantage guards the OTHER half of
// ApplyMasterAction's re-derive: deriveSpeeds runs again on every condition edit (so a
// speed/moveSpeed edit reads through), but a reaction that displaced a queued action derived
// at systemBias -1 (see MatchSession.AttachReaction), and that -1 is transient — a parameter
// to Derive, never stored anywhere on its own. A hardcoded 0 on the re-derive would silently
// flatten the reaction's swap-disadvantage to neutral the instant the master edited ANY OTHER
// condition on it, and the corrupted Speed.Result is what persist_turn_close.go later writes
// straight into the speed JSONB column.
//
// This edits FieldRepel — unrelated to speed — and asserts the reaction's Speed.Result is
// unchanged. It fails against a version of ApplyMasterAction that re-derives at a literal 0.
func TestApplyMasterAction_PreservesReactionSwapDisadvantage(t *testing.T) {
	matchUUID := uuid.New()
	masterUUID := uuid.New()
	attackerPlayer, victimPlayer := uuid.New(), uuid.New()
	attackerID, victimID := uuid.New(), uuid.New()

	attacker := &matchDomain.Participant{
		UUID: uuid.New(), MatchUUID: matchUUID,
		Sheet: csEntity.Summary{UUID: attackerID, PlayerUUID: &attackerPlayer},
	}
	victim := &matchDomain.Participant{
		UUID: uuid.New(), MatchUUID: matchUUID,
		Sheet: csEntity.Summary{UUID: victimID, PlayerUUID: &victimPlayer},
	}
	sheets := map[uuid.UUID]*csSheet.CharacterSheet{
		attackerID: newEditSheet(t),
		victimID:   newEditSheet(t),
	}
	session := matchsession.NewMatchSession(matchUUID, sheets, []*matchDomain.Participant{attacker, victim})
	// Race mode: only there does actionSpeed roll and read the bias at all — Free takes the
	// passive value, which never consults bias (see RollCalculator.Derive's Passive branch).
	session.GetActiveRound().SetMode(enum.Race)

	src := &scriptedFaces{faces: []int{
		1, 1, 1, 1, // attacker's open action: Speed Primary + Secondary (value irrelevant)
		1, 1, 1, 1, // victim's own pending action: Speed Primary + Secondary
		9, 8, 2, 3, // reaction: Speed Primary (sum 17, the BETTER set) + Secondary (sum 5, worse)
		1, 1, 1, 1, // reaction: Repel Primary + Secondary
	}}
	session.SetRollSource(src)

	// The open action: the attacker "acts" against the victim with no Attack at all —
	// AttachReaction only requires the reactor to be a TARGET of the open action, never that
	// it carries a Hit.
	openAction := action.NewAction(
		attackerID, []uuid.UUID{victimID}, uuid.Nil, nil,
		action.ActionSpeed{RollCheck: action.RollCheck{SkillName: enum.Legerity.String()}},
		nil, nil, nil, nil, nil, nil, nil,
	)
	if err := session.EnqueueAction(attackerPlayer, openAction); err != nil {
		t.Fatalf("EnqueueAction(attacker): %v", err)
	}
	tr, err := session.OpenNextAction()
	if err != nil {
		t.Fatalf("OpenNextAction: %v", err)
	}
	openActionID := tr.Opened.ActionRef().GetID()

	// The victim's own queued action — the one the reaction below displaces.
	pending := action.NewAction(
		victimID, nil, uuid.Nil, nil,
		action.ActionSpeed{RollCheck: action.RollCheck{SkillName: enum.Legerity.String()}},
		nil, nil, nil, nil, nil, nil, nil,
	)
	if err := session.EnqueueAction(victimPlayer, pending); err != nil {
		t.Fatalf("EnqueueAction(victim pending): %v", err)
	}

	reaction := action.NewAction(
		victimID, []uuid.UUID{attackerID}, openActionID, nil,
		action.ActionSpeed{RollCheck: action.RollCheck{SkillName: enum.Legerity.String()}},
		nil, nil, nil, nil, nil, nil, nil,
	)
	reaction.Repel = &action.Repel{}
	reaction.ReactionKind = action.ReactRepel

	if _, err := session.AttachReaction(victimPlayer, reaction); err != nil {
		t.Fatalf("AttachReaction: %v", err)
	}
	if reaction.SystemBias != -1 {
		t.Fatalf("fixture is wrong: the reaction should have displaced the victim's pending "+
			"action and derived at systemBias -1, got %d", reaction.SystemBias)
	}
	speedBefore := reaction.Speed.Result

	// The master edits something else entirely on this SAME reaction — the repel condition,
	// not speed — and must not move the speed the reaction was actually charged under.
	uc := match.NewEditActionUC()
	ma := action.NewMasterAction()
	ma.ActionID = reaction.GetID()
	ma.Conditions = []action.ConditionEdit{{
		Field: action.FieldRepel, Condition: action.RollCondition{Modifier: 7},
	}}
	if _, err := uc.Execute(context.Background(), session, masterUUID, masterUUID, ma); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var after *action.Action
	for _, r := range session.GetActiveRound().CurrentTurn().GetReactions() {
		if r.GetID() == reaction.GetID() {
			rc := r
			after = &rc
			break
		}
	}
	if after == nil {
		t.Fatal("the reaction is no longer on the open turn after the edit")
	}
	if after.Speed.Result != speedBefore {
		t.Fatalf("an unrelated edit moved the reaction's speed: %d → %d — the swap-disadvantage "+
			"was silently erased", speedBefore, after.Speed.Result)
	}
}

func TestEditActionSkills(t *testing.T) {
	t.Run("adding a skill rolls dice for it — a first roll, not a re-roll", func(t *testing.T) {
		f := newOpenAttackFixture(t)
		facesBefore := f.rollSource.consumed()

		uc := match.NewEditActionUC()
		ma := action.NewMasterAction()
		ma.ActionID = f.actionID
		ma.Skills = []action.Skill{{SkillName: enum.Acrobatics.String()}}

		if _, err := uc.Execute(context.Background(), f.session, f.masterUUID, f.masterUUID, ma); err != nil {
			t.Fatalf("Execute: %v", err)
		}
		// One 2D10 test: four faces, because Roll always rolls Primary AND Secondary.
		if got := f.rollSource.consumed() - facesBefore; got != 4 {
			t.Fatalf("added skill consumed %d faces, want 4", got)
		}
		added := f.skillOn(t, enum.Acrobatics.String())
		if added.Attempts.IsEmpty() {
			t.Fatal("the added skill has no dice")
		}
	})

	t.Run("removing and re-adding a skill reuses its dice", func(t *testing.T) {
		f := newOpenAttackFixtureWithSkill(t, enum.Acrobatics.String())
		original := f.skillOn(t, enum.Acrobatics.String()).Attempts

		uc := match.NewEditActionUC()
		strip := action.NewMasterAction()
		strip.ActionID = f.actionID
		strip.Skills = []action.Skill{} // present and empty: remove them all
		if _, err := uc.Execute(context.Background(), f.session, f.masterUUID, f.masterUUID, strip); err != nil {
			t.Fatalf("strip: %v", err)
		}
		facesAfterStrip := f.rollSource.consumed()

		restore := action.NewMasterAction()
		restore.ActionID = f.actionID
		restore.Skills = []action.Skill{{SkillName: enum.Acrobatics.String()}}
		if _, err := uc.Execute(context.Background(), f.session, f.masterUUID, f.masterUUID, restore); err != nil {
			t.Fatalf("restore: %v", err)
		}

		if f.rollSource.consumed() != facesAfterStrip {
			t.Fatal("re-adding a removed skill rolled again — that is a free re-roll")
		}
		back := f.skillOn(t, enum.Acrobatics.String()).Attempts
		if !slices.Equal(back.Primary, original.Primary) ||
			!slices.Equal(back.Secondary, original.Secondary) {
			t.Fatalf("the dice came back different: %v → %v", original, back)
		}
	})

	t.Run("adding a skill to a pure move does not start charging the action bar", func(t *testing.T) {
		f := newOpenMoveFixture(t) // a move-only action, opened, charging BarMove alone
		_, speedsBefore := f.session.BarState(f.actorID, action.BarAction)

		uc := match.NewEditActionUC()
		ma := action.NewMasterAction()
		ma.ActionID = f.actionID
		ma.Skills = []action.Skill{{SkillName: enum.Acrobatics.String()}}
		if _, err := uc.Execute(context.Background(), f.session, f.masterUUID, f.masterUUID, ma); err != nil {
			t.Fatalf("Execute: %v", err)
		}

		_, speedsAfter := f.session.BarState(f.actorID, action.BarAction)
		if !slices.Equal(speedsAfter, speedsBefore) {
			t.Fatal("the edit re-priced the action bar; the order already played would move")
		}
	})
}

func TestEditActionTargets(t *testing.T) {
	t.Run("replacing the target list re-resolves against the new targets", func(t *testing.T) {
		f := newOpenAttackFixture(t)
		other := f.thirdCharacterID

		uc := match.NewEditActionUC()
		ma := action.NewMasterAction()
		ma.ActionID = f.actionID
		ma.TargetID = []uuid.UUID{other}

		res, err := uc.Execute(context.Background(), f.session, f.masterUUID, f.masterUUID, ma)
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if len(res.Resolution.CharacterResults) != 1 ||
			res.Resolution.CharacterResults[0].TargetID != other {
			t.Fatalf("the resolution still names the old target: %+v", res.Resolution.CharacterResults)
		}
	})
}
