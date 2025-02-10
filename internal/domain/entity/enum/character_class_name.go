package enum

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
