package characterclass

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/proficiency"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type CharacterClassFactory struct{}

func NewCharacterClassFactory() *CharacterClassFactory {
	return &CharacterClassFactory{}
}

func (ccf *CharacterClassFactory) Build() map[enum.CharacterClassName]CharacterClass {
	classMap := make(map[enum.CharacterClassName]CharacterClass)

	classMap[enum.Swordsman] = BuildSwordsman()
	classMap[enum.Samurai] = BuildSamurai()
	classMap[enum.Ninja] = BuildNinja()
	classMap[enum.Rogue] = BuildRogue()
	classMap[enum.Netrunner] = BuildNetrunner()
	classMap[enum.Pirate] = BuildPirate()
	classMap[enum.Mercenary] = BuildMercenary()
	classMap[enum.Terrorist] = BuildTerrorist()
	classMap[enum.Monk] = BuildMonk()
	// MafiaMan => is weak - difficulty balancing
	classMap[enum.Military] = BuildMilitary()
	classMap[enum.Hunter] = BuildHunter()
	classMap[enum.WeaponsMaster] = BuildWeaponsMaster()

	// all the classes below are very customizable, deal with that first
	// Athlete
	// Tribal => change name
	// Experiment
	// Circus

	return classMap
}

func BuildSwordsman() CharacterClass {
	profile := *NewClassProfile(
		enum.Swordsman, "", "The Swordsman is a master of swords", "",
	)
	skills := map[enum.SkillName]int{
		enum.Energy: 69,

		enum.Velocity:   210,
		enum.Accelerate: 328,
		enum.Brake:      69,

		enum.AttackSpeed: 127,
		enum.Repel:       494,
		enum.Feint:       210,

		enum.Acrobatics: 69,
		enum.Evasion:    494,

		enum.Reflex:   494,
		enum.Accuracy: 69,

		enum.Vision: 494,
		enum.Tact:   210,

		enum.Heal:   69,
		enum.Breath: 69,
	}
	weapons := []enum.WeaponName{ // x3
		enum.Scimitar, enum.Rapier, enum.Sword, enum.Longsword, enum.Katana,
	}
	// TODO: receipt 1.0 with constructor, initializated in main
	expTable := experience.NewExpTable(1.0)
	exp := experience.NewExperience(expTable)
	jointProficiencies := map[string]proficiency.JointProficiency{
		"Sword Master": *proficiency.NewJointProficiency(
			*exp, "Sword Master", weapons,
		),
	}
	jointProfExps := make(map[string]int)
	jointProfExps["Sword Master"] = 328

	mentals := map[enum.AttributeName]int{
		enum.Resilience:   210,
		enum.Adaptability: 210,
		enum.Weighting:    127,
	}
	swordsman := *NewCharacterClass(
		profile,
		nil,
		skills,
		nil,
		nil,
		jointProficiencies,
		jointProfExps,
		mentals,
		nil,
	)
	return swordsman
}

func BuildSamurai() CharacterClass {
	profile := *NewClassProfile(
		enum.Samurai, "", "The Samurai is a master of katana", "",
	)
	// TODO: fix exp of increase below
	skills := map[enum.SkillName]int{
		enum.Energy: 69,

		enum.Velocity:   210,
		enum.Accelerate: 328,
		enum.Brake:      69,

		enum.AttackSpeed: 127,
		enum.Repel:       494,
		enum.Feint:       210,

		enum.Acrobatics: 69,
		enum.Evasion:    494,

		enum.Reflex:   494,
		enum.Accuracy: 69,

		enum.Vision: 494,
		enum.Tact:   210,

		enum.Heal:   69,
		enum.Breath: 69,
	}
	proficiencies := map[enum.WeaponName]int{
		enum.Katana: 1040, // or 725
	}
	mentals := map[enum.AttributeName]int{
		enum.Resilience: 210,
		enum.Weighting:  127,
	}

	samurai := *NewCharacterClass(
		profile,
		nil,
		skills,
		nil,
		proficiencies,
		nil,
		nil,
		mentals,
		nil,
	)
	return samurai
}

func BuildNinja() CharacterClass {
	profile := *NewClassProfile(
		enum.Ninja, "", "The Ninja is a master of killing and has knowledge of the underworld", "",
	)
	skills := map[enum.SkillName]int{
		enum.Vitality: 210,

		enum.Push: 210,

		enum.Velocity:   210,
		enum.Accelerate: 328,
		enum.Brake:      69,

		enum.AttackSpeed: 127,
		enum.Repel:       494,
		enum.Feint:       210,

		enum.Acrobatics: 69,
		enum.Evasion:    494,
		enum.Sneak:      494,

		enum.Reflex:   494,
		enum.Accuracy: 69,
		enum.Stealth:  69,

		enum.Vision:  494,
		enum.Hearing: 494,
		enum.Tact:    210,

		enum.Heal:     69,
		enum.Breath:   69,
		enum.Tenacity: 69,
	}
	profsAllowed := []enum.WeaponName{enum.Dagger, enum.Katana, enum.Katar}
	distribution := &Distribution{
		ProficiencyPoints:    []int{127},
		ProficienciesAllowed: profsAllowed,
	}
	proficiencies := map[enum.WeaponName]int{
		enum.ThrowingDagger: 127,
	}
	mentals := map[enum.AttributeName]int{
		enum.Resilience:   210,
		enum.Adaptability: 210,
		enum.Weighting:    127,
	}
	categories := []enum.CategoryName{
		enum.Reinforcement, enum.Transmutation, enum.Materialization, enum.Emission,
	}

	ninja := *NewCharacterClass(
		profile,
		distribution,
		skills,
		nil,
		proficiencies,
		nil,
		nil,
		mentals,
		categories,
	)
	return ninja
}

func BuildRogue() CharacterClass {
	profile := *NewClassProfile(
		enum.Rogue, "", "The Rogue excels in stealth and roguery", "",
	)
	skills := map[enum.SkillName]int{
		enum.Energy: 69,

		enum.Velocity:   210,
		enum.Accelerate: 328,
		enum.Brake:      69,

		enum.AttackSpeed: 127,
		enum.Repel:       494,
		enum.Feint:       210,

		enum.Acrobatics: 69,
		enum.Evasion:    494,
		enum.Sneak:      494,

		enum.Reflex:   494,
		enum.Accuracy: 69,
		enum.Stealth:  69,

		enum.Vision:  494,
		enum.Hearing: 494,
		enum.Tact:    210,
	}
	mentals := map[enum.AttributeName]int{
		enum.Adaptability: 210,
		enum.Creativity:   328,
		enum.Weighting:    127,
	}
	proficiencies := map[enum.WeaponName]int{
		enum.Dagger: 127, enum.ThrowingDagger: 127,
	}
	categories := []enum.CategoryName{
		enum.Transmutation,
		enum.Materialization,
		enum.Manipulation,
		enum.Emission,
	}

	rogue := *NewCharacterClass(
		profile,
		nil,
		skills,
		nil,
		proficiencies,
		nil,
		nil,
		mentals,
		categories,
	)
	return rogue
}

func BuildNetrunner() CharacterClass {
	profile := *NewClassProfile(
		enum.Netrunner, "", "The Netrunner is a master of digital stealth and hacking", "",
	)
	skills := map[enum.SkillName]int{
		enum.Energy: 69,

		enum.Velocity:   127,
		enum.Accelerate: 127,
		enum.Brake:      69,

		enum.AttackSpeed: 69,
		enum.Repel:       127,

		enum.Acrobatics: 210,
		enum.Evasion:    210,
		enum.Sneak:      127,

		enum.Reflex:   328,
		enum.Accuracy: 210,
		enum.Stealth:  494,

		enum.Vision:  69,
		enum.Hearing: 210,
	}
	mentals := map[enum.AttributeName]int{
		enum.Resilience:   69,
		enum.Adaptability: 210,
		enum.Creativity:   494,
		enum.Weighting:    210,
	}
	proficiencies := map[enum.WeaponName]int{
		enum.Pistol38: 210,
	}
	categories := []enum.CategoryName{
		enum.Transmutation,
		enum.Materialization,
		enum.Manipulation,
		enum.Emission,
		enum.Specialization,
	}

	netrunner := *NewCharacterClass(
		profile,
		nil,
		skills,
		nil,
		proficiencies,
		nil,
		nil,
		mentals,
		categories,
	)
	return netrunner
}

func BuildPirate() CharacterClass {
	description := "The Pirate is a master of naval combat and treasure hunting. " +
		"He carring he carries a little monkey, parrot or cockatoo that can help him"
	profile := *NewClassProfile(
		enum.Pirate, "", description, "",
	)
	skills := map[enum.SkillName]int{
		enum.Energy: 69,

		enum.Push:          210,
		enum.Grab:          210,
		enum.CarryCapacity: 210,

		enum.Velocity:   69,
		enum.Accelerate: 127,

		enum.AttackSpeed: 69,
		enum.Repel:       210,
		enum.Feint:       69,

		enum.Acrobatics: 210,
		enum.Evasion:    69,
		enum.Sneak:      69,

		enum.Reflex:   210,
		enum.Accuracy: 210,

		enum.Vision:  210,
		enum.Hearing: 69,
		enum.Tact:    494,

		enum.Breath: 328,
	}
	mentals := map[enum.AttributeName]int{
		enum.Resilience:   127,
		enum.Adaptability: 127,
		enum.Weighting:    69,
	}
	proficiencies := map[enum.WeaponName]int{
		enum.Pistol38: 210,
	}

	pirate := *NewCharacterClass(
		profile,
		nil,
		skills,
		nil,
		proficiencies,
		nil,
		nil,
		mentals,
		nil,
	)
	return pirate
}

func BuildMercenary() CharacterClass {
	description := "the mercenary is nothing more than an assassin, " +
		"but also focused on face-to-face combat"
	profile := *NewClassProfile(
		enum.Mercenary, "", description, "",
	)
	skills := map[enum.SkillName]int{
		enum.Vitality: 127,
		enum.Defense:  127,
		enum.Energy:   69,

		enum.Push:          69,
		enum.Grab:          69,
		enum.CarryCapacity: 127,

		enum.Velocity:   127,
		enum.Accelerate: 69,

		enum.AttackSpeed: 69,
		enum.Repel:       69,
		enum.Feint:       69,

		enum.Acrobatics: 69,
		enum.Evasion:    69,

		enum.Reflex:   127,
		enum.Accuracy: 69,
		enum.Stealth:  210,

		enum.Vision:  69,
		enum.Hearing: 210,
		enum.Tact:    69,

		enum.Breath:   127,
		enum.Tenacity: 127,
	}
	mentals := map[enum.AttributeName]int{
		enum.Resilience:   69,
		enum.Adaptability: 69,
		enum.Weighting:    69,
	}
	profsAllowed := []enum.WeaponName{
		enum.Dagger,
		enum.ThrowingDagger,
		enum.Scimitar,
		enum.Scythe,
		enum.Longscythe,
		enum.Katar,
		enum.Crossbow,
		enum.Pistol38,
		enum.Rifle,
	}
	distribution := &Distribution{
		ProficiencyPoints:    []int{210, 127},
		ProficienciesAllowed: profsAllowed,
	}
	categories := []enum.CategoryName{
		enum.Reinforcement,
		enum.Transmutation,
		enum.Materialization,
		enum.Emission,
	}

	mercenary := *NewCharacterClass(
		profile,
		distribution,
		skills,
		nil,
		nil,
		nil,
		nil,
		mentals,
		categories,
	)
	return mercenary
}

func BuildTerrorist() CharacterClass {
	description := "The Terrorist is a master of explosives and stealth " +
		"without losing brute strength"
	profile := *NewClassProfile(
		enum.Terrorist, "", description, "",
	)
	skills := map[enum.SkillName]int{
		enum.Vitality: 328,
		enum.Defense:  210,

		enum.Push:          127,
		enum.Grab:          127,
		enum.CarryCapacity: 210,

		enum.Velocity: 127,

		enum.Repel: 127,

		enum.Acrobatics: 69,

		enum.Reflex:   127,
		enum.Accuracy: 69,
		enum.Stealth:  328,

		enum.Vision:  69,
		enum.Hearing: 210,
		enum.Smell:   69,

		enum.Breath:   127,
		enum.Tenacity: 127,
	}
	mentals := map[enum.AttributeName]int{
		enum.Resilience:   69,
		enum.Adaptability: 69,
		enum.Creativity:   210,
		enum.Weighting:    69,
	}
	proficiencies := map[enum.WeaponName]int{
		enum.Pistol38: 210, enum.Bomb: 328,
	}

	terrorist := *NewCharacterClass(
		profile,
		nil,
		skills,
		nil,
		proficiencies,
		nil,
		nil,
		mentals,
		nil,
	)
	return terrorist
}

func BuildMonk() CharacterClass {
	description := "The Monk is a master of martial arts, with excellent body awareness"
	profile := *NewClassProfile(
		enum.Monk, "", description, "",
	)
	skills := map[enum.SkillName]int{
		enum.Vitality: 210,
		enum.Defense:  210,

		enum.Push:          127,
		enum.Grab:          127,
		enum.CarryCapacity: 127,

		enum.Velocity:   127,
		enum.Accelerate: 127,
		enum.Brake:      127,

		enum.AttackSpeed: 127,
		enum.Repel:       127,
		enum.Feint:       127,

		enum.Acrobatics: 69,
		enum.Evasion:    69,
		enum.Sneak:      69,

		enum.Reflex:   127,
		enum.Accuracy: 69,
		enum.Stealth:  127,

		enum.Vision:  127,
		enum.Hearing: 127,
		enum.Smell:   69,
		enum.Tact:    127,

		enum.Heal:     210,
		enum.Breath:   210,
		enum.Tenacity: 210,
	}
	mentals := map[enum.AttributeName]int{
		enum.Resilience:   210,
		enum.Adaptability: 210,
		enum.Creativity:   69,
		enum.Weighting:    210,
	}
	proficiencies := map[enum.WeaponName]int{
		enum.Staff: 210, enum.Fist: 210,
	}
	categories := []enum.CategoryName{
		enum.Reinforcement,
		enum.Manipulation,
		enum.Materialization,
	}

	monk := *NewCharacterClass(
		profile,
		nil,
		skills,
		nil,
		proficiencies,
		nil,
		nil,
		mentals,
		categories,
	)
	return monk
}

func BuildMilitary() CharacterClass {
	description := "The Military is proficient with fire weapons and " +
		"has access to military areas and government data"
	profile := *NewClassProfile(
		enum.Military, "", description, "",
	)
	skills := map[enum.SkillName]int{
		enum.Vitality: 210,
		enum.Defense:  210,

		enum.Push:          210,
		enum.Grab:          69,
		enum.CarryCapacity: 127,

		enum.Velocity: 127,

		enum.Reflex:   127,
		enum.Accuracy: 328,
		enum.Stealth:  210,

		enum.Vision:  210,
		enum.Hearing: 210,
		enum.Smell:   127,

		enum.Heal:     69,
		enum.Breath:   127,
		enum.Tenacity: 210,
	}
	mentals := map[enum.AttributeName]int{
		enum.Resilience:   210,
		enum.Adaptability: 210,
	}
	proficiencies := map[enum.WeaponName]int{
		enum.Ar15: 127, enum.Pistol38: 127, enum.Rifle: 69,
	}
	categories := []enum.CategoryName{
		enum.Reinforcement,
		enum.Emission,
		enum.Transmutation,
		enum.Materialization,
	}

	military := *NewCharacterClass(
		profile,
		nil,
		skills,
		nil,
		proficiencies,
		nil,
		nil,
		mentals,
		categories,
	)
	return military
}

func BuildHunter() CharacterClass {
	// TODO: review description
	description := "The Hunter class is skilled in tracking and survival, " +
		"connected with nature and adept at utilizing resources."
	profile := *NewClassProfile(
		enum.Hunter, "", description, "",
	)
	skills := map[enum.SkillName]int{
		enum.Energy: 69,

		enum.CarryCapacity: 127,

		enum.Velocity:   210,
		enum.Accelerate: 127,
		enum.Brake:      69,

		enum.AttackSpeed: 127,
		enum.Repel:       69,

		enum.Acrobatics: 127,
		enum.Evasion:    210,
		enum.Sneak:      210,

		enum.Reflex:   328,
		enum.Accuracy: 328,
		enum.Stealth:  328,

		enum.Vision:  328,
		enum.Hearing: 210,
		enum.Smell:   210,
		enum.Tact:    69,
		enum.Taste:   69,

		enum.Heal:   127,
		enum.Breath: 127,
	}
	mentals := map[enum.AttributeName]int{
		enum.Resilience:   127,
		enum.Adaptability: 210,
		enum.Creativity:   210,
		enum.Weighting:    210,
	}
	proficiencies := map[enum.WeaponName]int{
		enum.Dagger: 127,
	}
	profsAllowed := []enum.WeaponName{
		enum.ThrowingDagger, enum.Bow, enum.Longbow,
	}
	distribution := &Distribution{
		ProficiencyPoints:    []int{127, 127},
		ProficienciesAllowed: profsAllowed,
	}
	categories := []enum.CategoryName{
		enum.Reinforcement,
		enum.Emission,
		enum.Transmutation,
		enum.Materialization,
	}

	hunter := *NewCharacterClass(
		profile,
		distribution,
		skills,
		nil,
		proficiencies,
		nil,
		nil,
		mentals,
		categories,
	)
	return hunter
}

func BuildWeaponsMaster() CharacterClass {
	description := "The Weapon Master has mastered a unique martial art, " +
		"capable of replicating it for any weapon he uses."
	profile := *NewClassProfile(
		enum.WeaponsMaster, "", description, "",
	)
	skills := map[enum.SkillName]int{
		enum.Vitality: 69,
		enum.Defense:  127,
		enum.Energy:   69,

		enum.Push:          127,
		enum.Grab:          127,
		enum.CarryCapacity: 210,

		enum.Velocity:   210,
		enum.Accelerate: 69,
		enum.Brake:      69,

		enum.AttackSpeed: 69,
		enum.Repel:       127,
		enum.Feint:       127,

		enum.Acrobatics: 69,
		enum.Evasion:    127,
		enum.Sneak:      69,

		enum.Reflex:   210,
		enum.Accuracy: 127,
		enum.Stealth:  69,

		enum.Vision:  210,
		enum.Hearing: 69,
		enum.Tact:    127,

		enum.Breath: 69,
	}
	mentals := map[enum.AttributeName]int{
		enum.Resilience:   69,
		enum.Adaptability: 210,
		enum.Creativity:   69,
		enum.Weighting:    127,
	}

	weapons := []enum.WeaponName{ // x5?
		enum.Dagger, enum.ThrowingDagger, enum.Halberd,
		enum.Bow, enum.Longbow, enum.Staff, enum.Scimitar,
		enum.Rapier, enum.Whip, enum.Club, enum.Longclub,
		enum.Sword, enum.Longsword, enum.Scythe, enum.Longscythe,
		enum.Katana, enum.Katar, enum.Spear, enum.Longspear,
		enum.Axe, enum.Longaxe, enum.ThrowingAxe,
		enum.Hammer, enum.Warhammer, enum.ThrowingHammer,
		enum.Massa, enum.Mangual, enum.Longmass,
		enum.Pickaxe, enum.Fist, enum.Trident, enum.Tchaco,
	}
	// TODO: receipt 1.0 with constructor, initialized in main
	expTable := experience.NewExpTable(1.0)
	exp := experience.NewExperience(expTable)
	jointProficiencies := map[string]proficiency.JointProficiency{
		"Mastery of Weapons": *proficiency.NewJointProficiency(
			*exp, "Mastery of Weapons", weapons,
		),
	}
	jointProfExps := make(map[string]int)
	jointProfExps["Mastery of Weapons"] = 328

	weaponsMaster := *NewCharacterClass(
		profile,
		nil,
		skills,
		nil,
		nil,
		jointProficiencies,
		jointProfExps,
		mentals,
		nil,
	)
	return weaponsMaster
}
