package matchsession

import (
	"errors"
	"reflect"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/google/uuid"
)

func TestRefusalClause(t *testing.T) {
	if got := refusalClause(errtext.Sentence("O personagem está atordoado.")); got != "O personagem está atordoado" {
		t.Errorf("sentence: got %q", got)
	}
	if got := refusalClause(errors.New("pq: boom")); got != "algo impediu a volta" {
		t.Errorf("english error: got %q", got)
	}
}

// TestRestoreIgnoresLegacyDamageSkillOverride: rows of the old damage-skill swap
// (overridden_action_values.field = "damageSkill") still exist in databases. Undoing or
// restoring one must leave the attack alone — Force always measures the damage now.
func TestRestoreIgnoresLegacyDamageSkillOverride(t *testing.T) {
	sword := enum.Sword
	a := action.NewAction(uuid.New(), nil, uuid.Nil, nil, action.ActionSpeed{},
		nil, nil, &action.Attack{Weapon: &sword}, nil, nil, nil, nil)
	before := *a.Attack
	restoreOriginal(a, match.OverriddenValue{Field: "damageSkill", Original: enum.Grab})
	if !reflect.DeepEqual(*a.Attack, before) {
		t.Fatalf("the legacy row changed the attack: %+v, want %+v", *a.Attack, before)
	}
}
