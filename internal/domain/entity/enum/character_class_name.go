package enum

import "fmt"

type CharacterClassName uint8

const (
	Swordsman CharacterClassName = iota
	Samurai
	Ninja
	Rogue
	Netrunner
	Pirate
	Mercenary
	Terrorist
	Monk
	// MafiaMan
	Military
	Hunter
	WeaponsMaster
	Athlete
	Tribal
	Experiment
	Circus
)

func (ccn CharacterClassName) String() string {
	switch ccn {
	case Swordsman:
		return "Swordsman"
	case Samurai:
		return "Samurai"
	case Ninja:
		return "Ninja"
	case Rogue:
		return "Rogue"
	case Netrunner:
		return "Netrunner"
	case Pirate:
		return "Pirate"
	case Mercenary:
		return "Mercenary"
	case Terrorist:
		return "Terrorist"
	case Monk:
		return "Monk"
	// case MafiaMan:
	// 	return "Mafia Man"
	case Military:
		return "Military"
	case Hunter:
		return "Hunter"
	case WeaponsMaster:
		return "Weapons Master"
	case Athlete:
		return "Athlete"
	case Tribal:
		return "Tribal"
	case Experiment:
		return "Experiment"
	case Circus:
		return "Circus"
	}
	return "Unknown"
}

func CharacterClassNameFrom(s string) (CharacterClassName, error) {
	switch s {
	case "Swordsman":
		return Swordsman, nil
	case "Samurai":
		return Samurai, nil
	case "Ninja":
		return Ninja, nil
	case "Rogue":
		return Rogue, nil
	case "Netrunner":
		return Netrunner, nil
	case "Pirate":
		return Pirate, nil
	case "Mercenary":
		return Mercenary, nil
	case "Terrorist":
		return Terrorist, nil
	case "Monk":
		return Monk, nil
	// case "Mafia Man":
	// 	return MafiaMan
	case "Military":
		return Military, nil
	case "Hunter":
		return Hunter, nil
	case "WeaponsMaster":
		return WeaponsMaster, nil
	case "Athlete":
		return Athlete, nil
	case "Tribal":
		return Tribal, nil
	case "Experiment":
		return Experiment, nil
	case "Circus":
		return Circus, nil
	default:
		return 0, fmt.Errorf("invalid character class name: %s", s)
	}
}
