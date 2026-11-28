package statuseffect_test

import (
	"slices"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	se "github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
)

func weakened(a se.Aptitude, dis bool, pen int) se.Effect {
	return se.Effect{Kind: se.KindWeakened, Aptitude: a, Params: se.Params{Disadvantage: dis, Penalty: pen}}
}

func sense(s se.Sense, lvl se.SenseLevel, dis bool, pen int, faces ...se.TouchFace) se.Effect {
	return se.Effect{Kind: se.KindSense, Sense: s,
		Params: se.Params{Level: lvl, Disadvantage: dis, Penalty: pen, Faces: faces}}
}

func TestRollEffectOf(t *testing.T) {
	hit := se.Test{Skill: enum.Force, Domains: []se.Domain{se.DomainAttack}}
	tests := []struct {
		name    string
		effects []se.Effect
		test    se.Test
		want    se.RollEffect
	}{
		{"weakened physical weighs on hit", []se.Effect{weakened(se.AptitudePhysical, true, 2)}, hit,
			se.RollEffect{Bias: -1, Penalty: -2, Kinds: []se.Kind{se.KindWeakened}}},
		{"weakened spiritual ignores hit", []se.Effect{weakened(se.AptitudeSpiritual, true, 2)}, hit, se.RollEffect{}},
		{"weakened number only", []se.Effect{weakened(se.AptitudePhysical, false, 3)}, hit,
			se.RollEffect{Penalty: -3, Kinds: []se.Kind{se.KindWeakened}}},
		{"weakened spiritual weighs on a spiritual skill", []se.Effect{weakened(se.AptitudeSpiritual, true, 0)},
			se.Test{Skill: enum.Focus}, se.RollEffect{Bias: -1, Kinds: []se.Kind{se.KindWeakened}}},
		{"injured hurt weighs on its domain",
			[]se.Effect{{Kind: se.KindInjured, Params: se.Params{Injuries: map[se.Domain]se.InjuryLevel{se.DomainAttack: se.InjuryHurt}}}},
			hit, se.RollEffect{Bias: -1, Kinds: []se.Kind{se.KindInjured}}},
		{"injured hurt ignores another domain",
			[]se.Effect{{Kind: se.KindInjured, Params: se.Params{Injuries: map[se.Domain]se.InjuryLevel{se.DomainAttack: se.InjuryHurt}}}},
			se.Test{Skill: enum.Reflex, Domains: []se.Domain{se.DomainDodge}}, se.RollEffect{}},
		{"vision nullified weighs on attack defense and dodge",
			[]se.Effect{sense(se.SenseVision, se.SenseNullified, false, 0)}, hit,
			se.RollEffect{Bias: -1, Kinds: []se.Kind{se.KindSense}}},
		{"vision nullified weighs on defense",
			[]se.Effect{sense(se.SenseVision, se.SenseNullified, false, 0)}, se.Test{Domains: []se.Domain{se.DomainDefense}},
			se.RollEffect{Bias: -1, Kinds: []se.Kind{se.KindSense}}},
		{"vision nullified weighs on dodge",
			[]se.Effect{sense(se.SenseVision, se.SenseNullified, false, 0)}, se.Test{Domains: []se.Domain{se.DomainDodge}},
			se.RollEffect{Bias: -1, Kinds: []se.Kind{se.KindSense}}},
		{"vision nullified auto fails a vision test",
			[]se.Effect{sense(se.SenseVision, se.SenseNullified, false, 0)}, se.Test{Skill: enum.Vision},
			se.RollEffect{AutoFail: true, Kinds: []se.Kind{se.KindSense}}},
		{"vision impaired on a vision skill",
			[]se.Effect{sense(se.SenseVision, se.SenseImpaired, true, 1)}, se.Test{Skill: enum.Vision},
			se.RollEffect{Bias: -1, Penalty: -1, Kinds: []se.Kind{se.KindSense}}},
		{"tact touch weighs on a tact skill test",
			[]se.Effect{sense(se.SenseTact, se.SenseImpaired, true, 0, se.FaceTouch)}, se.Test{Skill: enum.Tact},
			se.RollEffect{Bias: -1, Kinds: []se.Kind{se.KindSense}}},
		{"tact balance only ignores a tact skill test",
			[]se.Effect{sense(se.SenseTact, se.SenseImpaired, true, 0, se.FaceBalance)}, se.Test{Skill: enum.Tact},
			se.RollEffect{}},
		{"tact balance weighs on the four domains",
			[]se.Effect{sense(se.SenseTact, se.SenseImpaired, true, 2, se.FaceBalance)}, hit,
			se.RollEffect{Bias: -1, Penalty: -2, Kinds: []se.Kind{se.KindSense}}},
		{"tact balance nullified weighs one level on a domain",
			[]se.Effect{sense(se.SenseTact, se.SenseNullified, false, 0, se.FaceBalance)}, se.Test{Domains: []se.Domain{se.DomainMovement}},
			se.RollEffect{Bias: -1, Kinds: []se.Kind{se.KindSense}}},
		{"stunned auto fails every sense test",
			[]se.Effect{{Kind: se.KindStunned, Params: se.Params{Points: 1}}}, se.Test{Skill: enum.Smell},
			se.RollEffect{AutoFail: true, Kinds: []se.Kind{se.KindStunned}}},
		{"stunned auto fails the tact test whatever the face",
			[]se.Effect{{Kind: se.KindStunned, Params: se.Params{Points: 1}}}, se.Test{Skill: enum.Tact},
			se.RollEffect{AutoFail: true, Kinds: []se.Kind{se.KindStunned}}},
		{"dead auto fails every sense test", []se.Effect{{Kind: se.KindDead}}, se.Test{Skill: enum.Hearing},
			se.RollEffect{AutoFail: true, Kinds: []se.Kind{se.KindDead}}},
		{"slowed penalizes move speed", []se.Effect{{Kind: se.KindSlowed, Params: se.Params{Penalty: 2}}},
			se.Test{Skill: enum.Legerity, Domains: []se.Domain{se.DomainMovement}, MoveSpeed: true},
			se.RollEffect{Penalty: -2, Kinds: []se.Kind{se.KindSlowed}}},
		{"slowed ignores a test that is not move speed", []se.Effect{{Kind: se.KindSlowed, Params: se.Params{Penalty: 2}}},
			hit, se.RollEffect{}},
		{"prone weighs on reflex", []se.Effect{{Kind: se.KindProne}}, se.Test{Reaction: se.ReactionReflexDodge},
			se.RollEffect{Bias: -1, Kinds: []se.Kind{se.KindProne}}},
		{"prone weighs on repel", []se.Effect{{Kind: se.KindProne}}, se.Test{Reaction: se.ReactionRepel},
			se.RollEffect{Bias: -1, Kinds: []se.Kind{se.KindProne}}},
		{"prone weighs on escape", []se.Effect{{Kind: se.KindProne}}, se.Test{Reaction: se.ReactionEscape},
			se.RollEffect{Bias: -1, Kinds: []se.Kind{se.KindProne}}},
		{"prone ignores evasion", []se.Effect{{Kind: se.KindProne}}, se.Test{Reaction: se.ReactionEvasion}, se.RollEffect{}},
		{"open guard escape disadvantage on both tests", []se.Effect{{Kind: se.KindOpenGuard}},
			se.Test{Reaction: se.ReactionEscape, MoveSpeed: true},
			se.RollEffect{Bias: -1, Kinds: []se.Kind{se.KindOpenGuard}}},
		{"open guard default defense rolled only", []se.Effect{{Kind: se.KindOpenGuard}},
			se.Test{Domains: []se.Domain{se.DomainDefense}, Reaction: se.ReactionDefense},
			se.RollEffect{RolledOnly: true, Kinds: []se.Kind{se.KindOpenGuard}}},
		{"sources accumulate",
			[]se.Effect{weakened(se.AptitudePhysical, true, 0), sense(se.SenseVision, se.SenseNullified, false, 0)}, hit,
			se.RollEffect{Bias: -2, Kinds: []se.Kind{se.KindSense, se.KindWeakened}}},
		{"no effects is neutral", nil, hit, se.RollEffect{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := se.RollEffectOf(tt.effects, tt.test)
			if got.Bias != tt.want.Bias || got.Penalty != tt.want.Penalty || got.AutoFail != tt.want.AutoFail ||
				got.RolledOnly != tt.want.RolledOnly || !slices.Equal(got.Kinds, tt.want.Kinds) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

// R4: num teste que está em dois conjuntos (o repelir, ataque E defesa), um mesmo status pesa
// uma vez só, pela parte mais forte; status diferentes somam.
func TestRollEffectOfRepelWeighsEachEffectOnce(t *testing.T) {
	repel := se.Test{Skill: enum.Repel, Domains: []se.Domain{se.DomainAttack, se.DomainDefense}, Reaction: se.ReactionRepel}
	injured := func(ds ...se.Domain) se.Effect {
		m := map[se.Domain]se.InjuryLevel{}
		for _, d := range ds {
			m[d] = se.InjuryHurt
		}
		return se.Effect{Kind: se.KindInjured, Params: se.Params{Injuries: m}}
	}
	vision := sense(se.SenseVision, se.SenseNullified, false, 0)
	tests := []struct {
		name    string
		effects []se.Effect
		want    se.RollEffect
	}{
		{"vision nullified weighs one level", []se.Effect{vision},
			se.RollEffect{Bias: -1, Kinds: []se.Kind{se.KindSense}}},
		{"injured attack only weighs on repel", []se.Effect{injured(se.DomainAttack)},
			se.RollEffect{Bias: -1, Kinds: []se.Kind{se.KindInjured}}},
		{"injured attack and defense weighs one level", []se.Effect{injured(se.DomainAttack, se.DomainDefense)},
			se.RollEffect{Bias: -1, Kinds: []se.Kind{se.KindInjured}}},
		{"vision nullified plus injured attack sum", []se.Effect{vision, injured(se.DomainAttack)},
			se.RollEffect{Bias: -2, Kinds: []se.Kind{se.KindInjured, se.KindSense}}},
		{"vision nullified plus injured attack and defense sum per effect",
			[]se.Effect{vision, injured(se.DomainAttack, se.DomainDefense)},
			se.RollEffect{Bias: -2, Kinds: []se.Kind{se.KindInjured, se.KindSense}}},
		{"tact balance penalty weighs once",
			[]se.Effect{sense(se.SenseTact, se.SenseImpaired, true, 3, se.FaceBalance)},
			se.RollEffect{Bias: -1, Penalty: -3, Kinds: []se.Kind{se.KindSense}}},
		{"prone and vision nullified sum", []se.Effect{{Kind: se.KindProne}, vision},
			se.RollEffect{Bias: -2, Kinds: []se.Kind{se.KindProne, se.KindSense}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := se.RollEffectOf(tt.effects, repel)
			if got.Bias != tt.want.Bias || got.Penalty != tt.want.Penalty || got.AutoFail != tt.want.AutoFail ||
				got.RolledOnly != tt.want.RolledOnly || !slices.Equal(got.Kinds, tt.want.Kinds) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestAptitudeAndSenseOfSkill(t *testing.T) {
	for s, want := range map[enum.SkillName]se.Aptitude{
		enum.Focus: se.AptitudeSpiritual, enum.WillPower: se.AptitudeSpiritual, enum.SelfKnowledge: se.AptitudeSpiritual,
		enum.Coa: se.AptitudeSpiritual, enum.Mop: se.AptitudeSpiritual, enum.Aop: se.AptitudeSpiritual,
		enum.Force: se.AptitudePhysical, enum.Vision: se.AptitudePhysical, "": "",
	} {
		if got := se.AptitudeOfSkill(s); got != want {
			t.Errorf("AptitudeOfSkill(%q) = %q, want %q", s, got, want)
		}
	}
	if sn, ok := se.SenseOfSkill(enum.Taste); !ok || sn != se.SenseTaste {
		t.Errorf("SenseOfSkill(Taste) = %q, %v", sn, ok)
	}
	if _, ok := se.SenseOfSkill(enum.Force); ok {
		t.Error("Force is not a sense skill")
	}
	if _, ok := se.SenseOfSkill(""); ok {
		t.Error("no skill is not a sense skill")
	}
}
