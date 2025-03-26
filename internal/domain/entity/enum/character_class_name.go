package enum

import "fmt"

type CharacterClassName string

const (
	Swordsman CharacterClassName = "Swordsman"
	Samurai   CharacterClassName = "Samurai"
	Ninja     CharacterClassName = "Ninja"
	Rogue     CharacterClassName = "Rogue"
	Netrunner CharacterClassName = "Netrunner"
	Pirate    CharacterClassName = "Pirate"
	Mercenary CharacterClassName = "Mercenary"
	Terrorist CharacterClassName = "Terrorist"
	Monk      CharacterClassName = "Monk"
	// MafiaMan
	Military      CharacterClassName = "Military"
	Hunter        CharacterClassName = "Hunter"
	WeaponsMaster CharacterClassName = "WeaponsMaster"
	Athlete       CharacterClassName = "Athlete"
	Tribal        CharacterClassName = "Tribal"
	Experiment    CharacterClassName = "Experiment"
	Circus        CharacterClassName = "Circus"
)

func (ccn CharacterClassName) String() string {
	return string(ccn)
}

func GetAllCharacterClasses() []CharacterClassName {
	return []CharacterClassName{
		Swordsman,
		Samurai,
		Ninja,
		Rogue,
		Netrunner,
		Pirate,
		Mercenary,
		Terrorist,
		Monk,
		Military,
		Hunter,
		WeaponsMaster,
		Athlete,
		Tribal,
		Experiment,
		Circus,
	}
}

func CharacterClassNameFrom(s string) (CharacterClassName, error) {
	for _, name := range GetAllCharacterClasses() {
		if s == name.String() {
			return name, nil
		}
	}
	return "", fmt.Errorf("%w%s: %s", ErrInvalidNameOf, "character class", s)
}
