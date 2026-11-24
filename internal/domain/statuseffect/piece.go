package statuseffect

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/google/uuid"
)

// PieceKind é o tipo de uma peça: o que um status faz, em unidades combináveis.
type PieceKind string

const (
	PieceNoAct              PieceKind = "noAct"
	PieceNoReact            PieceKind = "noReact"
	PieceDomainLocked       PieceKind = "domainLocked"
	PieceDomainWarning      PieceKind = "domainWarning"
	PieceReactionRestricted PieceKind = "reactionRestricted"
	PieceTestPenalty        PieceKind = "testPenalty"
	PieceAutoFail           PieceKind = "autoFail"
	PieceNoDisplacement     PieceKind = "noDisplacement"
	PieceLosesHP            PieceKind = "losesHp"
	PieceRecovery           PieceKind = "recovery"
	PieceTriggeredTest      PieceKind = "triggeredTest"
	PieceConduct            PieceKind = "conduct"
	PieceControl            PieceKind = "control"
	PieceCounter            PieceKind = "counter"
)

// Reaction é a reação afetada por uma restrição.
type Reaction string

const (
	ReactionReflexDodge Reaction = "reflexDodge"
	ReactionEvasion     Reaction = "evasion"
	ReactionDefense     Reaction = "defense"
	ReactionRepel       Reaction = "repel"
	ReactionEscape      Reaction = "escape"
)

// Restriction é o grau de restrição de uma reação.
type Restriction string

const (
	RestrictionForbidden    Restriction = "forbidden"
	RestrictionDisadvantage Restriction = "disadvantage"
	RestrictionRolledOnly   Restriction = "rolledOnly"
)

// Trigger é o gatilho de um teste automático.
type Trigger string

const (
	TriggerOnHit  Trigger = "onHit"
	TriggerOnMove Trigger = "onMove"
)

// Scope delimita o alcance de uma peça; só um campo é usado por vez.
type Scope struct {
	Domain    Domain
	Aptitude  Aptitude
	Sense     Sense
	Face      TouchFace // junto de Sense = Tact
	MoveSpeed bool
}

// TestRef aponta o teste: uma perícia ou, sem perícia, um atributo.
type TestRef struct {
	Skill     enum.SkillName
	Attribute enum.AttributeName
}

// Piece usa só os campos que o seu Kind pede; o resto fica zerado.
type Piece struct {
	Kind         PieceKind
	Domain       Domain // domainLocked, domainWarning
	Reaction     Reaction
	Restriction  Restriction
	Scope        Scope
	Disadvantage int // níveis de desvantagem (testPenalty)
	Penalty      int
	HPPerRound   int
	Test         TestRef
	Points       int
	Triggers     []Trigger
	OnFail       Kind
	Conduct      string
	Irresistible bool
	ControllerID *uuid.UUID
}

// Pieces devolve a composição do efeito (§5.1–§5.4).
func (e Effect) Pieces() []Piece {
	p := e.Params
	switch e.Kind {
	case KindParalyzed:
		return paralyzedPieces(p.Domains, p.Points, true)
	case KindStupefied:
		return stupefiedPieces(p.Points, true)
	case KindStunned:
		return stunnedPieces(p.Points, true)
	case KindUnconscious, KindDead:
		// Inconsciente herda o que o Atordoado faz, não como ele termina; o Morto contém o Inconsciente.
		return stunnedPieces(0, false)
	case KindRestrained:
		return []Piece{
			{Kind: PieceNoDisplacement},
			restrict(ReactionEscape, RestrictionForbidden),
		}
	case KindInjured:
		return injuredPieces(p.Injuries)
	case KindSlowed:
		return []Piece{{Kind: PieceTestPenalty, Scope: Scope{MoveSpeed: true}, Penalty: p.Penalty}}
	case KindSense:
		return sensePieces(e.Sense, p)
	case KindWeakened:
		return []Piece{penaltyPiece(Scope{Aptitude: e.Aptitude}, p)}
	case KindBleeding, KindBurning, KindFreezing, KindPoisoned:
		return []Piece{{Kind: PieceLosesHP, HPPerRound: p.HPPerRound}}
	case KindAsphyxiating:
		// Fôlego e afogamento: fase posterior (spec §5.3 Ar).
		return []Piece{{Kind: PieceCounter}}
	case KindProne:
		// Provisório até o movimento (spec §13): só rolar/deslizar (Caído), só a trajetória (No ar).
		return []Piece{
			restrict(ReactionReflexDodge, RestrictionDisadvantage),
			restrict(ReactionRepel, RestrictionDisadvantage),
			// TODO(movimento): provisório — o desenho do movimento decide por tipo de escape (rolar talvez sem penalidade); spec §16.4 itens 33 e 37.
			restrict(ReactionEscape, RestrictionDisadvantage),
		}
	case KindAirborne:
		// Provisório até o movimento (spec §13): só rolar/deslizar (Caído), só a trajetória (No ar).
		return []Piece{
			restrict(ReactionReflexDodge, RestrictionForbidden),
			restrict(ReactionEscape, RestrictionForbidden),
		}
	case KindOpenGuard:
		return []Piece{
			restrict(ReactionReflexDodge, RestrictionForbidden),
			restrict(ReactionRepel, RestrictionForbidden),
			restrict(ReactionDefense, RestrictionRolledOnly),
			restrict(ReactionEscape, RestrictionDisadvantage),
		}
	case KindCompelled:
		return []Piece{{Kind: PieceConduct, Conduct: p.Conduct, Irresistible: p.Irresistible}}
	case KindControlled:
		return []Piece{{Kind: PieceControl, ControllerID: p.ControllerID}}
	}
	return nil // hidden e desconhecidos: nenhuma peça
}

func restrict(r Reaction, rs Restriction) Piece {
	return Piece{Kind: PieceReactionRestricted, Reaction: r, Restriction: rs}
}

func tenacityRecovery(points int) Piece {
	return Piece{Kind: PieceRecovery, Test: TestRef{Skill: enum.Tenacity}, Points: points}
}

func paralyzedPieces(domains []Domain, points int, recovery bool) []Piece {
	var out []Piece
	moves := false
	for _, d := range domains {
		out = append(out, Piece{Kind: PieceDomainLocked, Domain: d})
		moves = moves || d == DomainMovement
	}
	if moves {
		out = append(out, Piece{Kind: PieceNoDisplacement})
	}
	if recovery {
		out = append(out, tenacityRecovery(points))
	}
	return out
}

func stupefiedPieces(points int, recovery bool) []Piece {
	// TODO Nen: "não usa habilidades" (spec §13).
	out := []Piece{{Kind: PieceNoAct}, {Kind: PieceNoReact}}
	if recovery {
		out = append(out, Piece{Kind: PieceRecovery, Test: TestRef{Attribute: enum.Resilience}, Points: points})
	}
	return out
}

func stunnedPieces(points int, recovery bool) []Piece {
	// TODO Nen: "não usa habilidades" (spec §13).
	out := paralyzedPieces(AllDomains(), 0, false)
	out = append(out, stupefiedPieces(0, false)...)
	for _, s := range []Sense{SenseVision, SenseHearing, SenseSmell, SenseTact, SenseTaste} {
		out = append(out, Piece{Kind: PieceAutoFail, Scope: Scope{Sense: s}})
	}
	if recovery {
		out = append(out, tenacityRecovery(points))
	}
	return out
}

func injuredPieces(injuries map[Domain]InjuryLevel) []Piece {
	var out []Piece
	for _, d := range AllDomains() {
		lvl, ok := injuries[d]
		if !ok {
			continue
		}
		out = append(out, Piece{Kind: PieceTestPenalty, Scope: Scope{Domain: d}, Disadvantage: 1})
		if lvl == InjuryDisabled {
			out = append(out, Piece{Kind: PieceDomainWarning, Domain: d})
		}
	}
	return out
}

func disadvantageLevels(p Params) int {
	if p.Disadvantage {
		return 1
	}
	return 0
}

func penaltyPiece(s Scope, p Params) Piece {
	return Piece{Kind: PieceTestPenalty, Scope: s, Disadvantage: disadvantageLevels(p), Penalty: p.Penalty}
}

func sensePieces(s Sense, p Params) []Piece {
	scopes := []Scope{{Sense: s}}
	if s == SenseTact {
		scopes = nil
		for _, f := range p.Faces {
			scopes = append(scopes, Scope{Sense: s, Face: f})
		}
	}
	nullified := p.Level == SenseNullified
	var out []Piece
	for _, sc := range scopes {
		if nullified {
			out = append(out, Piece{Kind: PieceAutoFail, Scope: sc})
		} else {
			out = append(out, penaltyPiece(sc, p))
		}
	}
	if nullified && s == SenseVision {
		out = append(out,
			Piece{Kind: PieceTestPenalty, Scope: Scope{Domain: DomainAttack}, Disadvantage: 1},
			Piece{Kind: PieceTestPenalty, Scope: Scope{Domain: DomainDefense}, Disadvantage: 1},
			Piece{Kind: PieceTestPenalty, Scope: Scope{Domain: DomainDodge}, Disadvantage: 1})
	}
	if s == SenseTact {
		for _, f := range p.Faces {
			if f != FaceBalance {
				continue
			}
			out = append(out, Piece{
				Kind:     PieceTriggeredTest,
				Triggers: []Trigger{TriggerOnHit, TriggerOnMove},
				Test:     TestRef{Skill: enum.Tact},
				OnFail:   KindProne,
			})
			// G1: o equilíbrio afeta ataque, defesa, esquiva e movimento.
			g1 := penaltyPiece(Scope{}, p)
			if nullified {
				g1 = Piece{Kind: PieceTestPenalty, Disadvantage: 1}
			}
			for _, d := range AllDomains() {
				pc := g1
				pc.Scope = Scope{Domain: d}
				out = append(out, pc)
			}
		}
	}
	return out
}
