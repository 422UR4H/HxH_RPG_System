package enum

import "strconv"

type DieSides int

const (
	D4   DieSides = 4
	D6   DieSides = 6
	D8   DieSides = 8
	D10  DieSides = 10
	D12  DieSides = 12
	D20  DieSides = 20
	D100 DieSides = 100
)

func (dt DieSides) String() string { // ex.: D10, D20
	return strconv.Itoa(int(dt))
}

func (dt DieSides) GetSides() int {
	return int(dt)
}
