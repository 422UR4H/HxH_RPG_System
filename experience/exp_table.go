package experience

import (
	"fmt"
	"math"
)

const (
	MAX_LVL int8    = 101
	A_PARAM float64 = 3.7
	B_PARAM float64 = 2.8
)

type ExpTable struct {
	Coefficient    float64
	baseTable      [MAX_LVL]int
	aggregateTable [MAX_LVL]int
}

func expTableFunction(lvl int) float64 {
	return (1700.0 / (1.0 + math.Pow(math.E, A_PARAM/10.0*(12.0-float64(lvl))))) +
		(1800.0 / (1.0 + math.Pow(math.E, A_PARAM/10.0*(38.0-float64(lvl))))) +
		(2000.0 / (1.0 + math.Pow(math.E, B_PARAM/10.0*(74.0-float64(lvl)))))
}

func NewExpTable(coefficient float64) *ExpTable {
	expTable := &ExpTable{Coefficient: coefficient}
	expTable.baseTable[0] = 0
	expTable.aggregateTable[0] = 0

	for i := 1; i < int(MAX_LVL); i++ {
		currExp := int(coefficient * expTableFunction(i))
		expTable.baseTable[i] = currExp
		expTable.aggregateTable[i] = expTable.aggregateTable[i-1] + currExp
	}
	return expTable
}

func (e *ExpTable) GetBaseExpByLvl(lvl int) int {
	return e.baseTable[lvl]
}

func (e *ExpTable) GetAggregateExpByLvl(lvl int) int {
	return e.aggregateTable[lvl]
}

func (e *ExpTable) GetLvlByExp(exp int) int {
	for lvl := len(e.aggregateTable); lvl >= 0; lvl-- {
		if e.aggregateTable[lvl] < exp {
			return lvl
		}
	}
	return 0
}

func (e *ExpTable) ToString() string {
	expTable := "=============================\n"
	expTable += "Coef: "
	expTable += fmt.Sprintf("%.1f\n", e.Coefficient)
	expTable += "Lvl\t| Base\t| Total\n"
	for i := 0; i < int(MAX_LVL); i++ {
		expTable += " " + fmt.Sprint(i) + "\t| "
		expTable += fmt.Sprint(e.baseTable[i]) + "\t| "
		expTable += fmt.Sprint(e.aggregateTable[i]) + "\n"
	}
	expTable += "=============================\n"
	return expTable
}
