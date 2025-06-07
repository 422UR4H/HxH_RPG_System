package experience

type ICascadeUpgrade interface {
	CascadeUpgrade(values *UpgradeCascade)
	GetLevel() int
}
