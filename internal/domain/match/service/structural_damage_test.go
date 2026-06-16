package service_test

import (
	"testing"

	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
)

func TestApplyStructuralDamage(t *testing.T) {
	base := mapentity.WallSegment{
		ID:         "wall-1",
		HP:         40,
		MaxHP:      40,
		Resistance: 5,
		Destroyed:  false,
	}

	t.Run("indestructible wall (MaxHP=0) returns full rebound, HP unchanged", func(t *testing.T) {
		w := base
		w.HP = 0
		w.MaxHP = 0
		result := service.ApplyStructuralDamage(w, 20)
		if result.EffectiveDamage != 0 {
			t.Errorf("expected EffectiveDamage=0, got %d", result.EffectiveDamage)
		}
		if result.ReboundDamage != 20 {
			t.Errorf("expected ReboundDamage=20, got %d", result.ReboundDamage)
		}
		if result.UpdatedWall.Destroyed {
			t.Error("indestructible wall must not be marked Destroyed")
		}
	})

	t.Run("damage <= resistance: effective=0, rebound=rawDamage", func(t *testing.T) {
		w := base
		result := service.ApplyStructuralDamage(w, 3)
		if result.EffectiveDamage != 0 {
			t.Errorf("expected EffectiveDamage=0, got %d", result.EffectiveDamage)
		}
		if result.ReboundDamage != 3 {
			t.Errorf("expected ReboundDamage=3, got %d", result.ReboundDamage)
		}
		if result.UpdatedWall.HP != 40 {
			t.Errorf("expected HP=40 (no damage), got %d", result.UpdatedWall.HP)
		}
	})

	t.Run("damage > resistance: effective=raw-resistance, HP decremented", func(t *testing.T) {
		w := base
		result := service.ApplyStructuralDamage(w, 15)
		if result.EffectiveDamage != 10 {
			t.Errorf("expected EffectiveDamage=10, got %d", result.EffectiveDamage)
		}
		if result.ReboundDamage != 5 {
			t.Errorf("expected ReboundDamage=5 (=resistance), got %d", result.ReboundDamage)
		}
		if result.UpdatedWall.HP != 30 {
			t.Errorf("expected HP=30, got %d", result.UpdatedWall.HP)
		}
		if result.UpdatedWall.Destroyed {
			t.Error("wall still has HP, must not be Destroyed")
		}
	})

	t.Run("damage brings HP to 0: Destroyed=true", func(t *testing.T) {
		w := base
		result := service.ApplyStructuralDamage(w, 45)
		if result.UpdatedWall.HP != 0 {
			t.Errorf("expected HP=0, got %d", result.UpdatedWall.HP)
		}
		if !result.UpdatedWall.Destroyed {
			t.Error("expected Destroyed=true when HP reaches 0")
		}
	})

	t.Run("overkill damage: HP clamped to 0, Destroyed=true", func(t *testing.T) {
		w := base
		result := service.ApplyStructuralDamage(w, 200)
		if result.UpdatedWall.HP != 0 {
			t.Errorf("expected HP=0 (clamped), got %d", result.UpdatedWall.HP)
		}
		if !result.UpdatedWall.Destroyed {
			t.Error("expected Destroyed=true")
		}
	})
}
