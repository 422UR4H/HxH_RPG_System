package sheet

import (
	"context"
	"errors"
	"log"
	"net/http"

	apiAuth "github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/application/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	cs "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/item"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type GetCombatCatalogueRequest struct {
	UUID string `path:"uuid" required:"true" doc:"UUID of the character sheet"`
}

// WeaponOptionResponse is one weapon this character can attack with, with the numbers that
// decide what it does. The numbers travel because the bottom sheet has to show what the weapon
// does BEFORE the player picks it — and because the front duplicating this table is how the
// two sides start disagreeing.
type WeaponOptionResponse struct {
	Name             string `json:"name"`
	Dice             []int  `json:"dice"`
	FlatDamage       int    `json:"flatDamage"`
	DefenseBonus     int    `json:"defenseBonus"`
	ProficiencyLevel int    `json:"proficiencyLevel"`
}

type GetCombatCatalogueResponseBody struct {
	Weapons []WeaponOptionResponse `json:"weapons"`
	// Skills is the vocabulary the wire accepts, not a menu. Phase 6 puts NO skill selector in
	// the bottom sheet: the chain of tests is not executed yet, and a control the player moves
	// that changes nothing is worse than no control. This list exists so the front never
	// invents a string again — combat_strength was born of its absence.
	Skills []string `json:"skills"`
}

type GetCombatCatalogueResponse struct {
	Body   GetCombatCatalogueResponseBody `json:"body"`
	Status int                            `json:"status"`
}

// GetCombatCatalogueHandler serves what THIS character can attack with.
//
// It is deliberately not the system's weapon catalogue: it is the character's own
// proficiencies — what they know how to wield — plus the bare-handed blow, always. When an
// inventory exists this becomes "what they are carrying"; proficiency is today's best
// approximation of it.
//
// Authorization is IGetCharacterSheet's, reused on purpose: whoever may read the sheet may read
// its catalogue. Do not grow a third visibility policy here.
func GetCombatCatalogueHandler(
	uc cs.IGetCharacterSheet,
) func(context.Context, *GetCombatCatalogueRequest) (*GetCombatCatalogueResponse, error) {

	return func(ctx context.Context, req *GetCombatCatalogueRequest) (*GetCombatCatalogueResponse, error) {
		userUUID, ok := ctx.Value(apiAuth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		charSheetID, err := uuid.Parse(req.UUID)
		if err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}

		characterSheet, err := uc.GetCharacterSheet(ctx, charSheetID, userUUID)
		if err != nil {
			switch {
			case errors.Is(err, cs.ErrCharacterSheetNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, campaign.ErrCampaignNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, auth.ErrInsufficientPermissions):
				return nil, huma.Error403Forbidden(err.Error())
			default:
				log.Printf("[ERROR] GetCombatCatalogue uuid=%s: %v", req.UUID, err)
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}

		catalogue := item.NewWeaponsManagerFactory().Build()

		// Fist starts at level 0 and is overwritten by the real proficiency when the character
		// has one. Seeding it first is what makes "always present, never duplicated" one rule
		// instead of two.
		levels := map[enum.WeaponName]int{enum.Fist: 0}
		for name, prof := range characterSheet.GetCommonProficiencies() {
			levels[name] = prof.GetLevel()
		}

		weapons := make([]WeaponOptionResponse, 0, len(levels))
		for _, name := range enum.GetAllWeaponNames() {
			level, ok := levels[name]
			if !ok {
				continue
			}
			dice, err := catalogue.GetDice(name)
			if err != nil {
				// A proficiency in a weapon the catalogue does not carry is a data fault, not a
				// request fault: skip it rather than fail the whole list.
				log.Printf("[WARN] combat catalogue: weapon %s missing from catalogue", name)
				continue
			}
			damage, _ := catalogue.GetDamage(name)
			defense, _ := catalogue.GetDefense(name)
			weapons = append(weapons, WeaponOptionResponse{
				Name:             name.String(),
				Dice:             dice,
				FlatDamage:       damage,
				DefenseBonus:     defense,
				ProficiencyLevel: level,
			})
		}

		skills := make([]string, 0, len(enum.AllSkillNames()))
		for _, s := range enum.AllSkillNames() {
			skills = append(skills, s.String())
		}

		return &GetCombatCatalogueResponse{
			Body:   GetCombatCatalogueResponseBody{Weapons: weapons, Skills: skills},
			Status: http.StatusOK,
		}, nil
	}
}
