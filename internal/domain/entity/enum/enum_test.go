package enum_test

import (
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func TestCharacterClassNameFrom(t *testing.T) {
	t.Run("valid names", func(t *testing.T) {
		tests := []struct {
			input    string
			expected enum.CharacterClassName
		}{
			{"Swordsman", enum.Swordsman},
			{"Ninja", enum.Ninja},
			{"Hunter", enum.Hunter},
			{"WeaponsMaster", enum.WeaponsMaster},
		}
		for _, tt := range tests {
			t.Run(tt.input, func(t *testing.T) {
				result, err := enum.CharacterClassNameFrom(tt.input)
				if err != nil {
					t.Fatalf("error = %v", err)
				}
				if result != tt.expected {
					t.Errorf("result = %v, want %v", result, tt.expected)
				}
			})
		}
	})

	t.Run("invalid name", func(t *testing.T) {
		_, err := enum.CharacterClassNameFrom("InvalidClass")
		if !errors.Is(err, enum.ErrInvalidNameOf) {
			t.Errorf("error = %v, want ErrInvalidNameOf", err)
		}
	})

	t.Run("empty string", func(t *testing.T) {
		_, err := enum.CharacterClassNameFrom("")
		if !errors.Is(err, enum.ErrInvalidNameOf) {
			t.Errorf("error = %v, want ErrInvalidNameOf", err)
		}
	})
}

func TestWeaponNameFrom(t *testing.T) {
	t.Run("valid names", func(t *testing.T) {
		tests := []struct {
			input    string
			expected enum.WeaponName
		}{
			{"Dagger", enum.Dagger},
			{"Katana", enum.Katana},
			{"Pistol38", enum.Pistol38},
			{"Ar15", enum.Ar15},
			{"ThrowingDagger", enum.ThrowingDagger},
		}
		for _, tt := range tests {
			t.Run(tt.input, func(t *testing.T) {
				result, err := enum.WeaponNameFrom(tt.input)
				if err != nil {
					t.Fatalf("error = %v", err)
				}
				if result != tt.expected {
					t.Errorf("result = %v, want %v", result, tt.expected)
				}
			})
		}
	})

	t.Run("invalid name", func(t *testing.T) {
		_, err := enum.WeaponNameFrom("Lightsaber")
		if !errors.Is(err, enum.ErrInvalidNameOf) {
			t.Errorf("error = %v, want ErrInvalidNameOf", err)
		}
	})
}

func TestCategoryNameFrom(t *testing.T) {
	t.Run("valid names", func(t *testing.T) {
		tests := []struct {
			input    string
			expected enum.CategoryName
		}{
			{"Reinforcement", enum.Reinforcement},
			{"Transmutation", enum.Transmutation},
			{"Materialization", enum.Materialization},
			{"Specialization", enum.Specialization},
			{"Manipulation", enum.Manipulation},
			{"Emission", enum.Emission},
		}
		for _, tt := range tests {
			t.Run(tt.input, func(t *testing.T) {
				result, err := enum.CategoryNameFrom(tt.input)
				if err != nil {
					t.Fatalf("error = %v", err)
				}
				if result != tt.expected {
					t.Errorf("result = %v, want %v", result, tt.expected)
				}
			})
		}
	})

	t.Run("invalid name", func(t *testing.T) {
		_, err := enum.CategoryNameFrom("Fire")
		if !errors.Is(err, enum.ErrInvalidNameOf) {
			t.Errorf("error = %v, want ErrInvalidNameOf", err)
		}
	})
}

func TestDieSides_GetSides(t *testing.T) {
	tests := []struct {
		name     string
		die      enum.DieSides
		expected int
	}{
		{"D4", enum.D4, 4},
		{"D6", enum.D6, 6},
		{"D8", enum.D8, 8},
		{"D10", enum.D10, 10},
		{"D12", enum.D12, 12},
		{"D20", enum.D20, 20},
		{"D100", enum.D100, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.die.GetSides() != tt.expected {
				t.Errorf("GetSides() = %d, want %d", tt.die.GetSides(), tt.expected)
			}
		})
	}
}

func TestGetAllCharacterClasses(t *testing.T) {
	classes := enum.GetAllCharacterClasses()
	if len(classes) != 16 {
		t.Errorf("GetAllCharacterClasses() len = %d, want 16", len(classes))
	}
}

func TestGetAllWeaponNames(t *testing.T) {
	weapons := enum.GetAllWeaponNames()
	if len(weapons) != 40 {
		t.Errorf("GetAllWeaponNames() len = %d, want 40", len(weapons))
	}
}

func TestAllNenCategoryNames(t *testing.T) {
	categories := enum.AllNenCategoryNames()
	if len(categories) != 6 {
		t.Errorf("AllNenCategoryNames() len = %d, want 6", len(categories))
	}
}
