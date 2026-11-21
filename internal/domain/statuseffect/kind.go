package statuseffect

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

// Kind é o tipo de efeito de status.
type Kind string

const (
	KindParalyzed    Kind = "paralyzed"    // Paralisado
	KindStupefied    Kind = "stupefied"    // Atônito
	KindStunned      Kind = "stunned"      // Atordoado
	KindUnconscious  Kind = "unconscious"  // Inconsciente
	KindRestrained   Kind = "restrained"   // Preso
	KindInjured      Kind = "injured"      // Lesionado
	KindSlowed       Kind = "slowed"       // Lento
	KindSense        Kind = "sense"        // Sentido afetado
	KindWeakened     Kind = "weakened"     // Debilitado
	KindBleeding     Kind = "bleeding"     // Sangrando
	KindBurning      Kind = "burning"      // Queimando
	KindFreezing     Kind = "freezing"     // Congelando
	KindPoisoned     Kind = "poisoned"     // Envenenado
	KindAsphyxiating Kind = "asphyxiating" // Asfixiando
	KindProne        Kind = "prone"        // Caído
	KindAirborne     Kind = "airborne"     // No ar
	KindOpenGuard    Kind = "openGuard"    // Guarda aberta
	KindCompelled    Kind = "compelled"    // Compelido
	KindControlled   Kind = "controlled"   // Controlado
	KindHidden       Kind = "hidden"       // Escondido
)

var allKinds = []Kind{
	KindParalyzed, KindStupefied, KindStunned, KindUnconscious, KindRestrained,
	KindInjured, KindSlowed, KindSense, KindWeakened,
	KindBleeding, KindBurning, KindFreezing, KindPoisoned,
	KindAsphyxiating, KindProne, KindAirborne, KindOpenGuard,
	KindCompelled, KindControlled, KindHidden,
}

func ParseKind(s string) (Kind, error) {
	for _, k := range allKinds {
		if string(k) == s {
			return k, nil
		}
	}
	return "", ErrUnknownKind
}

// kindLabel é o nome de cada tipo na mesa (a tabela de match-status-effects.md §3).
var kindLabel = map[Kind]string{
	KindParalyzed: "Paralisado", KindStupefied: "Atônito", KindStunned: "Atordoado",
	KindUnconscious: "Inconsciente", KindRestrained: "Preso", KindInjured: "Lesionado",
	KindSlowed: "Lento", KindSense: "Sentido afetado", KindWeakened: "Debilitado",
	KindBleeding: "Sangrando", KindBurning: "Queimando", KindFreezing: "Congelando",
	KindPoisoned: "Envenenado", KindAsphyxiating: "Asfixiando", KindProne: "Caído",
	KindAirborne: "No ar", KindOpenGuard: "Guarda aberta", KindCompelled: "Compelido",
	KindControlled: "Controlado", KindHidden: "Escondido",
}

// Label é o nome do tipo em português, para texto que o usuário lê (as recusas por status). O
// fio continua com o valor em inglês (string(k)); um tipo desconhecido sai como veio.
func (k Kind) Label() string {
	if l, ok := kindLabel[k]; ok {
		return l
	}
	return string(k)
}

// kindState é cada tipo numa forma que cabe depois de "está" ("está paralisado").
var kindState = map[Kind]string{
	KindParalyzed: "paralisado", KindStupefied: "atônito", KindStunned: "atordoado",
	KindUnconscious: "inconsciente", KindRestrained: "preso", KindInjured: "lesionado",
	KindSlowed: "lento", KindSense: "com um sentido afetado", KindWeakened: "debilitado",
	KindBleeding: "sangrando", KindBurning: "queimando", KindFreezing: "congelando",
	KindPoisoned: "envenenado", KindAsphyxiating: "asfixiando", KindProne: "caído",
	KindAirborne: "no ar", KindOpenGuard: "com a guarda aberta", KindCompelled: "compelido",
	KindControlled: "controlado", KindHidden: "escondido",
}

// StatePhrase é o tipo em forma de estado, para frases "está …" que o usuário lê (as recusas
// por status: "não pode atacar porque está paralisado"). Tipo desconhecido sai como veio.
func (k Kind) StatePhrase() string {
	if l, ok := kindState[k]; ok {
		return l
	}
	return string(k)
}

// PublicByDefault diz se, sem registro de audiência, todos veem o efeito (§9).
func (k Kind) PublicByDefault() bool {
	switch k {
	case KindProne, KindAirborne, KindRestrained, KindBurning, KindUnconscious,
		KindStunned, KindAsphyxiating, KindOpenGuard:
		return true
	}
	return false
}

// Sense é o sentido afetado.
type Sense string

const (
	SenseVision  Sense = "vision"
	SenseHearing Sense = "hearing"
	SenseSmell   Sense = "smell"
	SenseTact    Sense = "tact"
	SenseTaste   Sense = "taste"
)

func ParseSense(s string) (Sense, error) {
	switch v := Sense(s); v {
	case SenseVision, SenseHearing, SenseSmell, SenseTact, SenseTaste:
		return v, nil
	}
	return "", Wrap(ErrInvalidParams, "Esse sentido não existe.")
}

// Skill devolve a perícia correspondente ao sentido.
func (s Sense) Skill() enum.SkillName {
	switch s {
	case SenseVision:
		return enum.Vision
	case SenseHearing:
		return enum.Hearing
	case SenseSmell:
		return enum.Smell
	case SenseTact:
		return enum.Tact
	case SenseTaste:
		return enum.Taste
	}
	return ""
}

// Aptitude é a aptidão enfraquecida.
type Aptitude string

const (
	AptitudePhysical  Aptitude = "physical"
	AptitudeMental    Aptitude = "mental"
	AptitudeSpiritual Aptitude = "spiritual"
)

func ParseAptitude(s string) (Aptitude, error) {
	switch v := Aptitude(s); v {
	case AptitudePhysical, AptitudeMental, AptitudeSpiritual:
		return v, nil
	}
	return "", Wrap(ErrInvalidParams, "Essa aptidão não existe.")
}

// Ability devolve a habilidade (ability) correspondente à aptidão.
func (a Aptitude) Ability() enum.AbilityName {
	switch a {
	case AptitudePhysical:
		return enum.Physicals
	case AptitudeMental:
		return enum.Mentals
	case AptitudeSpiritual:
		return enum.Spirituals
	}
	return ""
}

// Domain é um domínio de ação restringível.
type Domain string

const (
	DomainAttack   Domain = "attack"
	DomainDefense  Domain = "defense"
	DomainMovement Domain = "movement"
	DomainDodge    Domain = "dodge"
)

// AllDomains devolve os quatro domínios, na ordem de declaração.
func AllDomains() []Domain {
	return []Domain{DomainAttack, DomainDefense, DomainMovement, DomainDodge}
}

func ParseDomain(s string) (Domain, error) {
	for _, d := range AllDomains() {
		if string(d) == s {
			return d, nil
		}
	}
	return "", Wrap(ErrInvalidParams, "Essa ação bloqueada não existe.")
}

// SenseLevel é a gravidade de um sentido afetado.
type SenseLevel string

const (
	SenseImpaired  SenseLevel = "impaired"
	SenseNullified SenseLevel = "nullified"
)

func ParseSenseLevel(s string) (SenseLevel, error) {
	switch v := SenseLevel(s); v {
	case SenseImpaired, SenseNullified:
		return v, nil
	}
	return "", Wrap(ErrInvalidParams, "Essa gravidade do sentido não existe.")
}

// InjuryLevel é a gravidade de uma lesão em um domínio.
type InjuryLevel string

const (
	InjuryHurt     InjuryLevel = "hurt"
	InjuryDisabled InjuryLevel = "disabled"
)

func ParseInjuryLevel(s string) (InjuryLevel, error) {
	switch v := InjuryLevel(s); v {
	case InjuryHurt, InjuryDisabled:
		return v, nil
	}
	return "", Wrap(ErrInvalidParams, "Essa gravidade da lesão não existe.")
}

// TouchFace é uma face do Tato.
type TouchFace string

const (
	FaceTouch      TouchFace = "touch"
	FaceBalance    TouchFace = "balance"
	FacePreemption TouchFace = "preemption"
)

var allFaces = []TouchFace{FaceTouch, FaceBalance, FacePreemption}

func ParseTouchFace(s string) (TouchFace, error) {
	for _, f := range allFaces {
		if string(f) == s {
			return f, nil
		}
	}
	return "", Wrap(ErrInvalidParams, "Essa face do Tato não existe.")
}
