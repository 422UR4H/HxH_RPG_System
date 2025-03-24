package enum

import "fmt"

type SkillName string

const (
	// PHYSICALS
	// Resistance
	Vitality SkillName = "Vitality"
	Energy   SkillName = "Energy"
	Defense  SkillName = "Defense"

	// Strength: Ao treinar e aprender uma arma, o personagem treina a skill Strike
	// essa skill é relacionada à sua arma, então não é comum (common)
	Push          SkillName = "Push"
	Grab          SkillName = "Grab"
	CarryCapacity SkillName = "CarryCapacity"

	// Agility
	Velocity   SkillName = "Velocity"
	Accelerate SkillName = "Accelerate"
	Brake      SkillName = "Brake"

	// Action Speed => TODO: change to other name
	AttackSpeed SkillName = "AttackSpeed" // TODO: change to ActionSpeed
	Repel       SkillName = "Repel"
	Feint       SkillName = "Feint"

	// Flexibility: Permite técnicas de combate mais versáteis, como chutes altos ou esquivas complexas
	// Pode ser usado para habilidades que exigem movimentos incomuns, como ataques com ângulos inesperados
	// Ex.: Hisoka demonstra flexibilidade em suas técnicas de combate, usando seu corpo de maneiras imprevisíveis
	Acrobatics SkillName = "Acrobatics"
	Evasion    SkillName = "Evasion" // Permite forçar esquiva sem sair do lugar ou se movimentar muito
	Sneak      SkillName = "Sneak"

	// Dexterity?
	Reflex   SkillName = "Reflex"
	Accuracy SkillName = "Accuracy" // crítico
	Stealth  SkillName = "Stealth"

	// Sense: Percepção ou Acuidade Sensorial
	// Ex.: Kurapika usa sua percepção apurada ao detectar inimigos e mentiras através da habilidade Chain Jail
	Vision  SkillName = "Vision"
	Hearing SkillName = "Hearing"
	Smell   SkillName = "Smell"
	Tact    SkillName = "Tact"
	Taste   SkillName = "Taste"

	// Constitution
	Heal     SkillName = "Heal"
	Breath   SkillName = "Breath"
	Tenacity SkillName = "Tenacity"

	// MENTALS
	// Resilience
	// Adaptability
	// Weighting
	// Creativity

	// SPIRITUALS
	// Spirit
	Nen       SkillName = "Nen"
	Focus     SkillName = "Focus"
	WillPower SkillName = "WillPower"
)

func (sn SkillName) String() string {
	return string(sn)
}

func AllSkillNames() []SkillName {
	return []SkillName{
		Vitality, Energy, Defense,
		Push, Grab, CarryCapacity,
		Velocity, Accelerate, Brake,
		AttackSpeed, Repel, Feint,
		Acrobatics, Evasion, Sneak,
		Reflex, Accuracy, Stealth,
		Vision, Hearing, Smell, Tact, Taste,
		Heal, Breath, Tenacity,
		Nen, Focus, WillPower,
	}
}

func SkillNameFrom(s string) (SkillName, error) {
	for _, name := range AllSkillNames() {
		if s == name.String() {
			return name, nil
		}
	}
	return "", fmt.Errorf("invalid skill name: %s", s)
}
