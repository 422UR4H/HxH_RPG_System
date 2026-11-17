package statuseffect_test

import (
	"slices"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	se "github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

func pieceEq(a, b se.Piece) bool {
	if a.Kind != b.Kind || a.Domain != b.Domain || a.Reaction != b.Reaction ||
		a.Restriction != b.Restriction || a.Scope != b.Scope ||
		a.Disadvantage != b.Disadvantage || a.Penalty != b.Penalty ||
		a.HPPerRound != b.HPPerRound || a.Test != b.Test || a.Points != b.Points ||
		a.OnFail != b.OnFail || a.Conduct != b.Conduct || a.Irresistible != b.Irresistible {
		return false
	}
	if !slices.Equal(a.Triggers, b.Triggers) {
		return false
	}
	if (a.ControllerID == nil) != (b.ControllerID == nil) {
		return false
	}
	return a.ControllerID == nil || *a.ControllerID == *b.ControllerID
}

func hasPiece(ps []se.Piece, p se.Piece) bool {
	return slices.ContainsFunc(ps, func(q se.Piece) bool { return pieceEq(p, q) })
}

// sameSet compara listas ignorando a ordem.
func sameSet(got, want []se.Piece) bool {
	if len(got) != len(want) {
		return false
	}
	for _, w := range want {
		if !hasPiece(got, w) {
			return false
		}
	}
	return true
}

func TestPieces_Composition(t *testing.T) {
	ctrl := uuid.UUID{1}
	tenacity := se.TestRef{Skill: enum.Tenacity}
	resilience := se.TestRef{Attribute: enum.Resilience}
	tact := se.TestRef{Skill: enum.Tact}

	pen := func(s se.Scope, dis, n int) se.Piece {
		return se.Piece{Kind: se.PieceTestPenalty, Scope: s, Disadvantage: dis, Penalty: n}
	}
	dom := func(d se.Domain) se.Scope { return se.Scope{Domain: d} }
	g1 := func(dis, n int) []se.Piece {
		var out []se.Piece
		for _, d := range se.AllDomains() {
			out = append(out, pen(dom(d), dis, n))
		}
		return out
	}

	tests := []struct {
		name string
		e    se.Effect
		want []se.Piece
	}{
		{"paralyzed all domains", se.Effect{Kind: se.KindParalyzed, Params: se.Params{Domains: se.AllDomains(), Points: 3}}, []se.Piece{
			{Kind: se.PieceDomainLocked, Domain: se.DomainAttack}, {Kind: se.PieceDomainLocked, Domain: se.DomainDefense},
			{Kind: se.PieceDomainLocked, Domain: se.DomainMovement}, {Kind: se.PieceDomainLocked, Domain: se.DomainDodge},
			{Kind: se.PieceNoDisplacement}, {Kind: se.PieceRecovery, Test: tenacity, Points: 3},
		}},
		{"paralyzed attack only has no noDisplacement", se.Effect{Kind: se.KindParalyzed, Params: se.Params{Domains: []se.Domain{se.DomainAttack}, Points: 2}}, []se.Piece{
			{Kind: se.PieceDomainLocked, Domain: se.DomainAttack}, {Kind: se.PieceRecovery, Test: tenacity, Points: 2},
		}},
		{"stupefied", se.Effect{Kind: se.KindStupefied, Params: se.Params{Points: 4}}, []se.Piece{
			{Kind: se.PieceNoAct}, {Kind: se.PieceNoReact}, {Kind: se.PieceRecovery, Test: resilience, Points: 4},
		}},
		{"restrained", se.Effect{Kind: se.KindRestrained}, []se.Piece{
			{Kind: se.PieceNoDisplacement},
			{Kind: se.PieceReactionRestricted, Reaction: se.ReactionEscape, Restriction: se.RestrictionForbidden},
		}},
		{"injured hurt and disabled", se.Effect{Kind: se.KindInjured, Params: se.Params{Injuries: map[se.Domain]se.InjuryLevel{
			se.DomainAttack: se.InjuryHurt, se.DomainMovement: se.InjuryDisabled}}}, []se.Piece{
			pen(dom(se.DomainAttack), 1, 0), pen(dom(se.DomainMovement), 1, 0),
			{Kind: se.PieceDomainWarning, Domain: se.DomainMovement},
		}},
		{"slowed", se.Effect{Kind: se.KindSlowed, Params: se.Params{Penalty: 2}}, []se.Piece{
			pen(se.Scope{MoveSpeed: true}, 0, 2),
		}},
		{"vision impaired", se.Effect{Kind: se.KindSense, Sense: se.SenseVision, Params: se.Params{Level: se.SenseImpaired, Disadvantage: true, Penalty: 1}}, []se.Piece{
			pen(se.Scope{Sense: se.SenseVision}, 1, 1),
		}},
		{"vision nullified", se.Effect{Kind: se.KindSense, Sense: se.SenseVision, Params: se.Params{Level: se.SenseNullified}}, []se.Piece{
			{Kind: se.PieceAutoFail, Scope: se.Scope{Sense: se.SenseVision}},
			pen(dom(se.DomainAttack), 1, 0), pen(dom(se.DomainDefense), 1, 0),
		}},
		{"hearing nullified has no domain penalty", se.Effect{Kind: se.KindSense, Sense: se.SenseHearing, Params: se.Params{Level: se.SenseNullified}}, []se.Piece{
			{Kind: se.PieceAutoFail, Scope: se.Scope{Sense: se.SenseHearing}},
		}},
		{"tact touch has no triggeredTest", se.Effect{Kind: se.KindSense, Sense: se.SenseTact, Params: se.Params{Level: se.SenseImpaired, Disadvantage: true, Faces: []se.TouchFace{se.FaceTouch}}}, []se.Piece{
			pen(se.Scope{Sense: se.SenseTact, Face: se.FaceTouch}, 1, 0),
		}},
		{"tact two faces nullified", se.Effect{Kind: se.KindSense, Sense: se.SenseTact, Params: se.Params{Level: se.SenseNullified, Faces: []se.TouchFace{se.FaceTouch, se.FacePreemption}}}, []se.Piece{
			{Kind: se.PieceAutoFail, Scope: se.Scope{Sense: se.SenseTact, Face: se.FaceTouch}},
			{Kind: se.PieceAutoFail, Scope: se.Scope{Sense: se.SenseTact, Face: se.FacePreemption}},
		}},
		{"tact balance impaired (G1)", se.Effect{Kind: se.KindSense, Sense: se.SenseTact, Params: se.Params{Level: se.SenseImpaired, Disadvantage: true, Penalty: 2, Faces: []se.TouchFace{se.FaceBalance}}},
			append([]se.Piece{
				pen(se.Scope{Sense: se.SenseTact, Face: se.FaceBalance}, 1, 2),
				{Kind: se.PieceTriggeredTest, Triggers: []se.Trigger{se.TriggerOnHit, se.TriggerOnMove}, Test: tact, OnFail: se.KindProne},
			}, g1(1, 2)...)},
		{"tact balance nullified (G1)", se.Effect{Kind: se.KindSense, Sense: se.SenseTact, Params: se.Params{Level: se.SenseNullified, Faces: []se.TouchFace{se.FaceBalance}}},
			append([]se.Piece{
				{Kind: se.PieceAutoFail, Scope: se.Scope{Sense: se.SenseTact, Face: se.FaceBalance}},
				{Kind: se.PieceTriggeredTest, Triggers: []se.Trigger{se.TriggerOnHit, se.TriggerOnMove}, Test: tact, OnFail: se.KindProne},
			}, g1(1, 0)...)},
		{"weakened", se.Effect{Kind: se.KindWeakened, Aptitude: se.AptitudeMental, Params: se.Params{Disadvantage: true, Penalty: 1}}, []se.Piece{
			pen(se.Scope{Aptitude: se.AptitudeMental}, 1, 1),
		}},
		{"weakened penalty only", se.Effect{Kind: se.KindWeakened, Aptitude: se.AptitudePhysical, Params: se.Params{Penalty: 3}}, []se.Piece{
			pen(se.Scope{Aptitude: se.AptitudePhysical}, 0, 3),
		}},
		{"bleeding", se.Effect{Kind: se.KindBleeding, Params: se.Params{HPPerRound: 1}}, []se.Piece{{Kind: se.PieceLosesHP, HPPerRound: 1}}},
		{"burning", se.Effect{Kind: se.KindBurning, Params: se.Params{HPPerRound: 2}}, []se.Piece{{Kind: se.PieceLosesHP, HPPerRound: 2}}},
		{"freezing", se.Effect{Kind: se.KindFreezing, Params: se.Params{HPPerRound: 1}}, []se.Piece{{Kind: se.PieceLosesHP, HPPerRound: 1}}},
		{"poisoned", se.Effect{Kind: se.KindPoisoned, Params: se.Params{HPPerRound: 1}}, []se.Piece{{Kind: se.PieceLosesHP, HPPerRound: 1}}},
		{"asphyxiating", se.Effect{Kind: se.KindAsphyxiating}, []se.Piece{{Kind: se.PieceCounter}}},
		{"prone", se.Effect{Kind: se.KindProne}, []se.Piece{
			{Kind: se.PieceReactionRestricted, Reaction: se.ReactionReflexDodge, Restriction: se.RestrictionDisadvantage},
			{Kind: se.PieceReactionRestricted, Reaction: se.ReactionRepel, Restriction: se.RestrictionDisadvantage},
		}},
		{"airborne", se.Effect{Kind: se.KindAirborne}, []se.Piece{
			{Kind: se.PieceReactionRestricted, Reaction: se.ReactionReflexDodge, Restriction: se.RestrictionForbidden},
			{Kind: se.PieceReactionRestricted, Reaction: se.ReactionEscape, Restriction: se.RestrictionForbidden},
		}},
		{"openGuard", se.Effect{Kind: se.KindOpenGuard}, []se.Piece{
			{Kind: se.PieceReactionRestricted, Reaction: se.ReactionReflexDodge, Restriction: se.RestrictionForbidden},
			{Kind: se.PieceReactionRestricted, Reaction: se.ReactionRepel, Restriction: se.RestrictionForbidden},
			{Kind: se.PieceReactionRestricted, Reaction: se.ReactionDefense, Restriction: se.RestrictionRolledOnly},
			{Kind: se.PieceReactionRestricted, Reaction: se.ReactionEscape, Restriction: se.RestrictionDisadvantage},
		}},
		{"compelled", se.Effect{Kind: se.KindCompelled, Params: se.Params{Conduct: "obey", Irresistible: true}}, []se.Piece{
			{Kind: se.PieceConduct, Conduct: "obey", Irresistible: true},
		}},
		{"controlled", se.Effect{Kind: se.KindControlled, Params: se.Params{ControllerID: &ctrl}}, []se.Piece{
			{Kind: se.PieceControl, ControllerID: &ctrl},
		}},
		{"hidden", se.Effect{Kind: se.KindHidden}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.e.Pieces()
			if !sameSet(got, tt.want) {
				t.Errorf("pieces mismatch\n got: %+v\nwant: %+v", got, tt.want)
			}
		})
	}
}

func withoutRecovery(ps []se.Piece) []se.Piece {
	var out []se.Piece
	for _, p := range ps {
		if p.Kind != se.PieceRecovery {
			out = append(out, p)
		}
	}
	return out
}

func subset(t *testing.T, sub, super []se.Piece) {
	t.Helper()
	for _, p := range sub {
		if !hasPiece(super, p) {
			t.Errorf("piece %+v missing from the containing status", p)
		}
	}
}

func TestPieces_Containment(t *testing.T) {
	paralyzedAll := se.Effect{Kind: se.KindParalyzed, Params: se.Params{Domains: se.AllDomains(), Points: 1}}
	stupefied := se.Effect{Kind: se.KindStupefied, Params: se.Params{Points: 1}}
	stunned := se.Effect{Kind: se.KindStunned, Params: se.Params{Points: 1}}
	unconscious := se.Effect{Kind: se.KindUnconscious}
	restrained := se.Effect{Kind: se.KindRestrained}
	paralyzedMove := se.Effect{Kind: se.KindParalyzed, Params: se.Params{Domains: []se.Domain{se.DomainMovement}, Points: 1}}

	t.Run("paralyzed contained in stunned", func(t *testing.T) {
		subset(t, withoutRecovery(paralyzedAll.Pieces()), stunned.Pieces())
	})
	t.Run("stupefied contained in stunned", func(t *testing.T) {
		subset(t, withoutRecovery(stupefied.Pieces()), stunned.Pieces())
	})
	t.Run("stunned contained in unconscious", func(t *testing.T) {
		subset(t, withoutRecovery(stunned.Pieces()), unconscious.Pieces())
	})
	t.Run("unconscious has no recovery", func(t *testing.T) {
		if len(withoutRecovery(unconscious.Pieces())) != len(unconscious.Pieces()) {
			t.Error("unconscious must not carry a recovery piece")
		}
	})
	t.Run("restrained noDisplacement contained in paralyzed movement", func(t *testing.T) {
		subset(t, []se.Piece{{Kind: se.PieceNoDisplacement}}, paralyzedMove.Pieces())
		if !hasPiece(restrained.Pieces(), se.Piece{Kind: se.PieceNoDisplacement}) {
			t.Error("restrained lost noDisplacement")
		}
	})
}
