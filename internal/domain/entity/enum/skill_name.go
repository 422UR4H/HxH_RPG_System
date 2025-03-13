package enum

import "fmt"

type SkillName uint8

const (
	// PHYSICALS
	// Resistance
	Vitality SkillName = iota
	Energy
	Defense

	// Strength: Ao treinar e aprender uma arma, o personagem treina a skill Strike
	// essa skill é relacionada à sua arma, então não é comum (common)
	Push
	Grab
	CarryCapacity

	// Agility
	Velocity
	Accelerate
	Brake

	// Action Speed => TODO: change to other name
	AttackSpeed // TODO: change to ActionSpeed
	Repel
	Feint

	// Flexibility: Permite técnicas de combate mais versáteis, como chutes altos ou esquivas complexas
	// Pode ser usado para habilidades que exigem movimentos incomuns, como ataques com ângulos inesperados
	// Ex.: Hisoka demonstra flexibilidade em suas técnicas de combate, usando seu corpo de maneiras imprevisíveis
	Acrobatics
	Evasion // Permite forçar esquiva sem sair do lugar ou se movimentar muito
	Sneak

	// Dexterity?
	Reflex
	Accuracy // crítico
	Stealth

	// Sense: Percepção ou Acuidade Sensorial
	// Ex.: Kurapika usa sua percepção apurada ao detectar inimigos e mentiras através da habilidade Chain Jail
	Vision
	Hearing
	Smell
	Tact
	Taste
	Balance

	// Constitution
	Heal
	Breath
	Tenacity

	// Instinct
	Intuition // ?

	// MENTALS
	// Resilience
	// Adaptability
	// Weighting
	// Creativity

	// SPIRITUALS
	// Spirit
	Nen
	Focus
	WillPower
)

func (sn SkillName) String() string {
	switch sn {
	case Vitality:
		return "Vitality"
	case Energy:
		return "Energy"
	case Defense:
		return "Defense"
	case Push:
		return "Push"
	case Grab:
		return "Grab"
	case CarryCapacity:
		return "CarryCapacity"
	case Velocity:
		return "Velocity"
	case Accelerate:
		return "Accelerate"
	case Brake:
		return "Brake"
	case AttackSpeed:
		return "AttackSpeed"
	case Repel:
		return "Repel"
	case Feint:
		return "Feint"
	case Acrobatics:
		return "Acrobatics"
	case Evasion:
		return "Evasion"
	case Sneak:
		return "Sneak"
	case Reflex:
		return "Reflex"
	case Accuracy:
		return "Accuracy"
	case Stealth:
		return "Stealth"
	case Vision:
		return "Vision"
	case Hearing:
		return "Hearing"
	case Smell:
		return "Smell"
	case Tact:
		return "Tact"
	case Taste:
		return "Taste"
	case Balance:
		return "Balance"
	case Heal:
		return "Heal"
	case Breath:
		return "Breath"
	case Tenacity:
		return "Tenacity"
	case Intuition:
		return "Intuition"
	case Nen:
		return "Nen"
	case Focus:
		return "Focus"
	case WillPower:
		return "WillPower"
	}
	return "Unknown"
}

func SkillNameFrom(s string) (SkillName, error) {
	switch s {
	case "Vitality":
		return Vitality, nil
	case "Energy":
		return Energy, nil
	case "Defense":
		return Defense, nil
	case "Push":
		return Push, nil
	case "Grab":
		return Grab, nil
	case "CarryCapacity":
		return CarryCapacity, nil
	case "Velocity":
		return Velocity, nil
	case "Accelerate":
		return Accelerate, nil
	case "Brake":
		return Brake, nil
	case "AttackSpeed":
		return AttackSpeed, nil
	case "Repel":
		return Repel, nil
	case "Feint":
		return Feint, nil
	case "Acrobatics":
		return Acrobatics, nil
	case "Evasion":
		return Evasion, nil
	case "Sneak":
		return Sneak, nil
	case "Reflex":
		return Reflex, nil
	case "Accuracy":
		return Accuracy, nil
	case "Stealth":
		return Stealth, nil
	case "Vision":
		return Vision, nil
	case "Hearing":
		return Hearing, nil
	case "Smell":
		return Smell, nil
	case "Tact":
		return Tact, nil
	case "Taste":
		return Taste, nil
	case "Balance":
		return Balance, nil
	case "Heal":
		return Heal, nil
	case "Breath":
		return Breath, nil
	case "Tenacity":
		return Tenacity, nil
	case "Intuition":
		return Intuition, nil
	case "Nen":
		return Nen, nil
	case "Focus":
		return Focus, nil
	case "WillPower":
		return WillPower, nil
	default:
		return 0, fmt.Errorf("invalid skill name: %s", s)
	}
}
