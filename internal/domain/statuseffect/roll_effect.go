package statuseffect

import (
	"slices"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

// Test descreve um teste de dados para a leitura dos status: o que ele é, não como rola.
type Test struct {
	Skill enum.SkillName // "" quando não há perícia
	// Domains são os conjuntos da ação em que o teste está; vazio quando não é teste de conjunto
	// (velocidade de ação, skills[]). O repelir é o único teste em dois: ataque E defesa (R1).
	Domains   []Domain
	Reaction  Reaction // "" quando não é reação
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
//
// R4 — um status pesa uma vez só no mesmo teste, pelos conjuntos: a lista de conjuntos de um
// efeito (Visão anulada: ataque, defesa e esquiva; Tato no equilíbrio: os quatro; Lesionado: um
// por conjunto lesionado) diz até onde ele alcança, não quanto pesa. Num teste em dois conjuntos
// (o repelir) as peças de conjunto do MESMO efeito contam uma vez, pela maior desvantagem e pela
// maior penalidade entre elas. Efeitos diferentes continuam somando, e as peças dos outros
// escopos (aptidão, sentido, velocidade de movimento) e as de reação seguem como antes.
func RollEffectOf(effects []Effect, t Test) RollEffect {
	var r RollEffect
	apt := AptitudeOfSkill(t.Skill)
	sense, isSense := SenseOfSkill(t.Skill)
	for _, e := range effects {
		var domainHit bool
		var domainDis, domainPen int
		for _, p := range e.Pieces() {
			switch p.Kind {
			case PieceTestPenalty:
				if p.Scope.Domain != "" {
					if slices.Contains(t.Domains, p.Scope.Domain) {
						domainHit = true
						domainDis = max(domainDis, p.Disadvantage)
						domainPen = max(domainPen, p.Penalty)
					}
					continue
				}
				if scopeReaches(p.Scope, apt, sense, isSense, t.MoveSpeed) {
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
		if domainHit {
			r.Bias -= domainDis
			r.Penalty -= domainPen
			r.Kinds = addKind(r.Kinds, e.Kind)
		}
	}
	return r
}

// scopeReaches diz se um escopo que não é de conjunto alcança o teste (os de conjunto são lidos
// à parte, uma vez por efeito — R4 em RollEffectOf).
func scopeReaches(s Scope, apt Aptitude, sense Sense, isSense, moveSpeed bool) bool {
	switch {
	case s.Aptitude != "":
		return s.Aptitude == apt
	case s.Sense != "":
		return senseReaches(s, sense, isSense)
	case s.MoveSpeed:
		return moveSpeed
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
