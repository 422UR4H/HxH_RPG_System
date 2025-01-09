package skill

// Passive skill upgrade is similar to Domain/experience/i_trigger_cascade_exp.go
type IPassive interface {
	UpgradeStatus(exp int)
}
