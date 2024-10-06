package main

import (
	"fmt"

	"github.com/422UR4H/HxH_RPG_Environment.Domain/experience"
)

func main() {
	expTable := experience.NewDefaultExpTable()
	fmt.Println(expTable.ToString())
	fmt.Println(expTable.GetLvlByExp(29))
}
