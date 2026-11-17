package statuseffect

import "slices"

// Note é uma anotação só para o mestre.
type Note struct {
	Kind         Kind
	Domain       Domain
	Conduct      string
	Irresistible bool
}

// Summary é o resumo de bloqueios de um conjunto de efeitos.
type Summary struct {
	CannotAct, CannotReact, NoDisplacement       bool
	CannotActBy, CannotReactBy, NoDisplacementBy []Kind // quem causa (mensagens de erro)
	LockedDomains                                map[Domain][]Kind
	ForbiddenReactions                           map[Reaction][]Kind
	Notes                                        []Note
}

// Summarize lê só as peças que bloqueiam; penalidades, autoFail etc. são ignorados aqui.
func Summarize(effects []Effect) Summary {
	var s Summary
	for _, e := range effects {
		for _, p := range e.Pieces() {
			switch p.Kind {
			case PieceNoAct:
				s.CannotAct = true
				s.CannotActBy = addKind(s.CannotActBy, e.Kind)
			case PieceNoReact:
				s.CannotReact = true
				s.CannotReactBy = addKind(s.CannotReactBy, e.Kind)
			case PieceNoDisplacement:
				s.NoDisplacement = true
				s.NoDisplacementBy = addKind(s.NoDisplacementBy, e.Kind)
			case PieceDomainLocked:
				if s.LockedDomains == nil {
					s.LockedDomains = map[Domain][]Kind{}
				}
				s.LockedDomains[p.Domain] = addKind(s.LockedDomains[p.Domain], e.Kind)
			case PieceReactionRestricted:
				if p.Restriction != RestrictionForbidden {
					continue
				}
				if s.ForbiddenReactions == nil {
					s.ForbiddenReactions = map[Reaction][]Kind{}
				}
				s.ForbiddenReactions[p.Reaction] = addKind(s.ForbiddenReactions[p.Reaction], e.Kind)
			case PieceDomainWarning:
				s.Notes = append(s.Notes, Note{Kind: e.Kind, Domain: p.Domain})
			case PieceConduct:
				s.Notes = append(s.Notes, Note{Kind: e.Kind, Conduct: p.Conduct, Irresistible: p.Irresistible})
			}
		}
	}
	return s
}

// addKind insere o tipo uma única vez, mantendo a lista ordenada.
func addKind(kinds []Kind, k Kind) []Kind {
	if slices.Contains(kinds, k) {
		return kinds
	}
	kinds = append(kinds, k)
	slices.Sort(kinds)
	return kinds
}
