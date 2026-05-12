package service_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
)

func TestRollCalculator_Calculate(t *testing.T) {
	calc := service.RollCalculator{}

	t.Run("returns int result for a RollCheck with no sheet", func(t *testing.T) {
		check := action.RollCheck{
			SkillName:  "Velocidade",
			SkillValue: 10,
		}

		result := calc.Calculate(check, nil)

		// Formula not implemented — result is 0 for now.
		_ = result
	})

	t.Run("does not panic when sheet is nil", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Calculate panicked with nil sheet: %v", r)
			}
		}()
		calc.Calculate(action.RollCheck{}, nil)
	})
}
