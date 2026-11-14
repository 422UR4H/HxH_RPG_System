package rules

import (
	"context"
	"slices"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/item"
)

type WeaponResponse struct {
	Name       string `json:"name"`
	Dice       []int  `json:"dice"`
	FlatDamage int    `json:"flatDamage"`
}

type GetWeaponsBody struct {
	Weapons []WeaponResponse `json:"weapons"`
}

type GetWeaponsResponse struct {
	CacheControl string `header:"Cache-Control"`
	Body         GetWeaponsBody
}

// GetWeaponsHandler serve o catálogo de armas do sistema (não o do personagem; esse é
// GET /charactersheets/{uuid}/combat-catalogue). A tabela é imutável durante a vida do
// processo, então a lista é montada uma vez, na ordem estável de enum.GetAllWeaponNames.
// Uma arma que o catálogo não traz é pulada. Os dados em dice vêm na ordem do catálogo;
// quem exibe ordena.
func GetWeaponsHandler(
	catalogue *item.WeaponsManager,
) func(context.Context, *struct{}) (*GetWeaponsResponse, error) {
	var weapons []WeaponResponse
	for _, name := range enum.GetAllWeaponNames() {
		dice, err := catalogue.GetDice(name)
		if err != nil {
			continue
		}
		flat, _ := catalogue.GetDamage(name)
		weapons = append(weapons, WeaponResponse{
			Name: string(name), Dice: slices.Clone(dice), FlatDamage: flat,
		})
	}
	return func(_ context.Context, _ *struct{}) (*GetWeaponsResponse, error) {
		// Cópia por resposta: quem recebe pode mexer em Dice sem afetar a próxima.
		out := make([]WeaponResponse, len(weapons))
		for i, w := range weapons {
			out[i] = WeaponResponse{Name: w.Name, Dice: slices.Clone(w.Dice), FlatDamage: w.FlatDamage}
		}
		return &GetWeaponsResponse{
			CacheControl: "public, max-age=300",
			Body:         GetWeaponsBody{Weapons: out},
		}, nil
	}
}
