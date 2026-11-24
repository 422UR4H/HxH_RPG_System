package statuseffect

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

// Test descreve um teste de dados para a leitura dos status: o que ele é, não como rola.
type Test struct {
	Skill     enum.SkillName // "" quando não há perícia
	Domain    Domain         // "" quando não é teste de domínio (velocidade de ação, skills[])
	Reaction  Reaction       // "" quando não é reação
	MoveSpeed bool
}

// RollEffect é o que os status de um personagem fazem a UM teste. Penalty é assinado: a
// penalidade de 3 entra como -3 (quem consome soma ao modificador); Bias é negativo para
// desvantagem.
type RollEffect struct {
	Bias, Penalty int
	AutoFail      bool
	RolledOnly    bool
	Kinds         []Kind // efeitos que contribuíram, ordenados e sem repetição
}

// RollEffectOf soma as peças de todos os efeitos que alcançam o teste. Não lê as restrições
// "forbidden": o bloqueio é do Summarize.
func RollEffectOf(effects []Effect, t Test) RollEffect {
	var r RollEffect
	apt := AptitudeOfSkill(t.Skill)
	sense, isSense := SenseOfSkill(t.Skill)
	for _, e := range effects {
		for _, p := range e.Pieces() {
			switch p.Kind {
			case PieceTestPenalty:
				if scopeReaches(p.Scope, t, apt, sense, isSense) {
					r.Bias -= p.Disadvantage
					r.Penalty -= p.Penalty
					r.Kinds = addKind(r.Kinds, e.Kind)
				}
			case PieceAutoFail:
				if senseReaches(p.Scope, sense, isSense) {
					r.AutoFail = true
					r.Kinds = addKind(r.Kinds, e.Kind)
				}
			case PieceReactionRestricted:
				if t.Reaction == "" || p.Reaction != t.Reaction {
					continue
				}
				switch p.Restriction {
				case RestrictionDisadvantage:
					r.Bias--
					r.Kinds = addKind(r.Kinds, e.Kind)
				case RestrictionRolledOnly:
					r.RolledOnly = true
					r.Kinds = addKind(r.Kinds, e.Kind)
				}
			}
		}
	}
	return r
}

func scopeReaches(s Scope, t Test, apt Aptitude, sense Sense, isSense bool) bool {
	switch {
	case s.Aptitude != "":
		return s.Aptitude == apt
	case s.Domain != "":
		return s.Domain == t.Domain
	case s.Sense != "":
		return senseReaches(s, sense, isSense)
	case s.MoveSpeed:
		return t.MoveSpeed
	}
	return false
}

// senseReaches diz se o escopo de sentido alcança a perícia do teste. No Tato só vale a face
// do toque (spec §16.4 item 38); sem face (o Atordoado trava o Tato inteiro) vale também.
func senseReaches(s Scope, sense Sense, isSense bool) bool {
	if s.Sense == "" || !isSense || s.Sense != sense {
		return false
	}
	return s.Sense != SenseTact || s.Face == "" || s.Face == FaceTouch
}

// AptitudeOfSkill diz a aptidão que pesa sobre a perícia: as do conjunto espiritual são
// espirituais; qualquer outra é física (não há perícia mental); sem perícia, sem aptidão.
func AptitudeOfSkill(s enum.SkillName) Aptitude {
	switch s {
	case "":
		return ""
	case enum.Focus, enum.WillPower, enum.SelfKnowledge, enum.Coa, enum.Mop, enum.Aop:
		return AptitudeSpiritual
	}
	return AptitudePhysical
}

// SenseOfSkill devolve o sentido de uma perícia de sentido.
func SenseOfSkill(s enum.SkillName) (Sense, bool) {
	for _, sn := range []Sense{SenseVision, SenseHearing, SenseSmell, SenseTact, SenseTaste} {
		if sn.Skill() == s {
			return sn, true
		}
	}
	return "", false
}
