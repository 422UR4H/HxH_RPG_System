package statuseffect

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"
	"strings"
	"unicode/utf8"
)

const maxConductRunes = 255

// fieldLabel é o campo como o mestre o chama, com o artigo ("os pontos").
var fieldLabel = map[string]string{
	"sense": "um sentido", "aptitude": "uma aptidão", "domains": "as ações bloqueadas",
	"injuries": "as lesões", "level": "a gravidade", "faces": "as faces do Tato",
	"points": "os pontos", "hpPerRound": "quanto de vida o efeito tira por round", "penalty": "a penalidade",
	"disadvantage": "a desvantagem", "conduct": "a conduta", "irresistible": "o irresistível",
	"controllerId": "quem controla",
}

// foreignLabel completa "Esse efeito não …" para o campo que o tipo não aceita.
var foreignLabel = map[string]string{
	"sense": "tem sentido", "aptitude": "tem aptidão", "domains": "tem ações bloqueadas",
	"injuries": "tem lesões", "level": "tem gravidade", "faces": "tem faces do Tato",
	"points": "tem pontos", "hpPerRound": "tem dano por round", "penalty": "tem penalidade",
	"disadvantage": "tem desvantagem", "conduct": "tem conduta", "irresistible": "pode ser irresistível",
	"controllerId": "tem controlador",
}

// missingText é a frase de um campo exigido: "Informe os pontos do efeito."
func missingText(field string) string {
	if field == "hpPerRound" {
		return "Informe quanto de vida o efeito tira por round."
	}
	return "Informe " + fieldLabel[field] + " do efeito."
}

// invalid é o campo que o tipo não aceita: "Esse efeito não tem pontos."
func invalid(field string) error {
	return errtext.Wrap(ErrInvalidParams, "Esse efeito não "+foreignLabel[field]+".")
}

// missing é o campo que o tipo exige e veio vazio ou fora da faixa: "Informe os pontos do efeito."
func missing(field string) error {
	return errtext.Wrap(ErrInvalidParams, missingText(field))
}

// allowed declara quais campos de ParamsInput um tipo aceita.
type allowed struct {
	domains, injuries, level, faces, points, hp, penalty, disadvantage, conduct, irresistible, controller bool
}

// Normalize valida os parâmetros de acordo com o tipo e aplica os padrões.
// O que não pertence ao tipo deve vir vazio.
func Normalize(k Kind, s Sense, a Aptitude, in ParamsInput) (Params, error) {
	if _, err := ParseKind(string(k)); err != nil {
		return Params{}, err
	}
	if k == KindSense {
		if _, err := ParseSense(string(s)); err != nil {
			return Params{}, err
		}
	} else if s != "" {
		return Params{}, invalid("sense")
	}
	if k == KindWeakened {
		if _, err := ParseAptitude(string(a)); err != nil {
			return Params{}, err
		}
	} else if a != "" {
		return Params{}, invalid("aptitude")
	}

	var al allowed
	switch k {
	case KindParalyzed:
		al = allowed{domains: true, points: true}
	case KindStupefied, KindStunned:
		al = allowed{points: true}
	case KindInjured:
		al = allowed{injuries: true}
	case KindSlowed:
		al = allowed{penalty: true}
	case KindSense:
		al = allowed{level: true, faces: s == SenseTact, penalty: true, disadvantage: true}
	case KindWeakened:
		al = allowed{penalty: true, disadvantage: true}
	case KindBleeding, KindBurning, KindFreezing, KindPoisoned:
		al = allowed{hp: true}
	case KindCompelled:
		al = allowed{conduct: true, irresistible: true}
	case KindControlled:
		al = allowed{controller: true}
	}
	if err := rejectForeign(al, in); err != nil {
		return Params{}, err
	}

	var p Params
	switch k {
	case KindParalyzed:
		doms, err := normDomains(in.Domains)
		if err != nil {
			return Params{}, err
		}
		p.Domains = doms
		pts, err := requirePositive(in.Points, "points")
		if err != nil {
			return Params{}, err
		}
		p.Points = pts
	case KindStupefied, KindStunned:
		pts, err := requirePositive(in.Points, "points")
		if err != nil {
			return Params{}, err
		}
		p.Points = pts
	case KindInjured:
		if len(in.Injuries) == 0 {
			return Params{}, missing("injuries")
		}
		p.Injuries = make(map[Domain]InjuryLevel, len(in.Injuries))
		for d, l := range in.Injuries {
			if _, err := ParseDomain(string(d)); err != nil {
				return Params{}, err
			}
			if _, err := ParseInjuryLevel(string(l)); err != nil {
				return Params{}, err
			}
			p.Injuries[d] = l
		}
	case KindSlowed:
		pen, err := requirePositive(in.Penalty, "penalty")
		if err != nil {
			return Params{}, err
		}
		p.Penalty = pen
	case KindSense:
		if _, err := ParseSenseLevel(string(in.Level)); err != nil {
			return Params{}, err
		}
		p.Level = in.Level
		if in.Level == SenseNullified {
			if in.Disadvantage != nil {
				return Params{}, errtext.Wrap(ErrInvalidParams, "Um sentido anulado não aceita desvantagem.")
			}
			if in.Penalty != nil {
				return Params{}, errtext.Wrap(ErrInvalidParams, "Um sentido anulado não aceita penalidade.")
			}
		} else if err := disadvantageOrPenalty(&p, in); err != nil {
			return Params{}, err
		}
		if s == SenseTact {
			faces, err := normFaces(in.Faces)
			if err != nil {
				return Params{}, err
			}
			p.Faces = faces
		}
	case KindWeakened:
		if err := disadvantageOrPenalty(&p, in); err != nil {
			return Params{}, err
		}
	case KindBleeding, KindBurning, KindFreezing, KindPoisoned:
		p.HPPerRound = 1
		if in.HPPerRound != nil {
			hp, err := requirePositive(in.HPPerRound, "hpPerRound")
			if err != nil {
				return Params{}, err
			}
			p.HPPerRound = hp
		}
	case KindCompelled:
		c := strings.TrimSpace(in.Conduct)
		if n := utf8.RuneCountInString(c); n < 1 || n > maxConductRunes {
			return Params{}, errtext.Wrap(ErrInvalidParams, "Informe a conduta, com até 255 caracteres.")
		}
		p.Conduct = c
		p.Irresistible = in.Irresistible != nil && *in.Irresistible
	case KindControlled:
		if in.ControllerID != nil {
			id := *in.ControllerID
			p.ControllerID = &id
		}
	}
	return p, nil
}

func rejectForeign(al allowed, in ParamsInput) error {
	switch {
	case !al.domains && len(in.Domains) > 0:
		return invalid("domains")
	case !al.injuries && len(in.Injuries) > 0:
		return invalid("injuries")
	case !al.level && in.Level != "":
		return invalid("level")
	case !al.faces && len(in.Faces) > 0:
		return invalid("faces")
	case !al.points && in.Points != nil:
		return invalid("points")
	case !al.hp && in.HPPerRound != nil:
		return invalid("hpPerRound")
	case !al.penalty && in.Penalty != nil:
		return invalid("penalty")
	case !al.disadvantage && in.Disadvantage != nil:
		return invalid("disadvantage")
	case !al.conduct && in.Conduct != "":
		return invalid("conduct")
	case !al.irresistible && in.Irresistible != nil:
		return invalid("irresistible")
	case !al.controller && in.ControllerID != nil:
		return invalid("controllerId")
	}
	return nil
}

func requirePositive(v *int, field string) (int, error) {
	if v == nil || *v < 1 {
		return 0, missing(field)
	}
	return *v, nil
}

// disadvantageOrPenalty: desvantagem (padrão true), penalidade >= 0, ao menos um dos dois.
func disadvantageOrPenalty(p *Params, in ParamsInput) error {
	p.Disadvantage = in.Disadvantage == nil || *in.Disadvantage
	if in.Penalty != nil {
		if *in.Penalty < 0 {
			return errtext.Wrap(ErrInvalidParams, "A penalidade não pode ser negativa.")
		}
		p.Penalty = *in.Penalty
	}
	if !p.Disadvantage && p.Penalty == 0 {
		return errtext.Wrap(ErrInvalidParams, "Deixe a desvantagem ligada ou informe uma penalidade.")
	}
	return nil
}

func normDomains(in []Domain) ([]Domain, error) {
	if len(in) == 0 {
		return AllDomains(), nil
	}
	seen := map[Domain]bool{}
	for _, d := range in {
		if _, err := ParseDomain(string(d)); err != nil {
			return nil, err
		}
		if seen[d] {
			return nil, errtext.Wrap(ErrInvalidParams, "As ações bloqueadas não podem se repetir.")
		}
		seen[d] = true
	}
	out := make([]Domain, 0, len(seen))
	for _, d := range AllDomains() {
		if seen[d] {
			out = append(out, d)
		}
	}
	return out, nil
}

func normFaces(in []TouchFace) ([]TouchFace, error) {
	if len(in) == 0 {
		return nil, errtext.Wrap(ErrInvalidParams, "Informe ao menos uma face do Tato.")
	}
	seen := map[TouchFace]bool{}
	for _, f := range in {
		if _, err := ParseTouchFace(string(f)); err != nil {
			return nil, err
		}
		if seen[f] {
			return nil, errtext.Wrap(ErrInvalidParams, "As faces do Tato não podem se repetir.")
		}
		seen[f] = true
	}
	var out []TouchFace
	for _, f := range allFaces {
		if seen[f] {
			out = append(out, f)
		}
	}
	return out, nil
}
