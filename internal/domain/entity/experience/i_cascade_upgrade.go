package experience

type ICascadeUpgrade interface {
	CascadeUpgrade(exp int)
	GetLevel() int
}
