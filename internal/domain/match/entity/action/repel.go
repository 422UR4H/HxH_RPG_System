package action

import "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"

// Repel is the component of the hardest reaction in the catalogue: instead of dodging or
// parrying, the character hits the incoming blow so it does not reach them.
//
// It is shaped exactly like Defense — a weapon and a test — because it is the same gesture
// read against a different ladder. The weapon matters twice: it is what the repel is made
// with, and on a near miss (a parry) its defense is what reduces the blow travelling on to
// the next target.
type Repel struct {
	Weapon *enum.WeaponName
	RollCheck
}
