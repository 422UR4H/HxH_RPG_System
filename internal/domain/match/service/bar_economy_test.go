package service_test

import (
	"math"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
)

// closeTo compares derived values without pretending float64 is exact. A third of a round's
// speeds is not representable in binary, and asserting equality on it would be a test that
// fails for a reason that has nothing to do with the rules.
func closeTo(got, want float64) bool { return math.Abs(got-want) < 1e-9 }

func TestBarEconomy_Mean(t *testing.T) {
	eco := service.BarEconomy{}
	tests := []struct {
		name  string
		acted []int
		want  float64
	}{
		{"no action yet", nil, 0},
		{"one action is its own average", []int{23}, 23},
		{"an exact average", []int{23, 17}, 20},
		{"a fractional average keeps the half", []int{17, 12}, 14.5},
		{"a third is kept too", []int{20, 10, 10}, 13.0 + 1.0/3.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := eco.Mean(tt.acted); !closeTo(got, tt.want) {
				t.Errorf("Mean(%v) = %v, want %v", tt.acted, got, tt.want)
			}
		})
	}
}

func TestBarEconomy_KeyAndBalance(t *testing.T) {
	eco := service.BarEconomy{}
	const price = 11

	t.Run("the canonical example, action by action", func(t *testing.T) {
		// p2 opens the round: nothing acted yet, so the key is the roll itself.
		if got := eco.Key(0, nil, 23, price); !closeTo(got, 23) {
			t.Errorf("p2 first key = %v, want 23", got)
		}
		if got := eco.Key(0, nil, 20, price); !closeTo(got, 20) {
			t.Errorf("p1 first key = %v, want 20", got)
		}
		if got := eco.Key(0, nil, 11, price); !closeTo(got, 11) {
			t.Errorf("p3 first key = %v, want 11", got)
		}
		// p2's second action: the average moves to 20 and one action has been paid for.
		// It enters the queue at 9 — BELOW the price — and still acts.
		if got := eco.Key(0, []int{23}, 17, price); !closeTo(got, 9) {
			t.Errorf("p2 second key = %v, want 9", got)
		}
	})

	t.Run("the leftover after the actions that acted", func(t *testing.T) {
		if got := eco.Balance(0, []int{23}, price); !closeTo(got, 12) {
			t.Errorf("p2 leftover after one action = %v, want 12", got)
		}
		if got := eco.Balance(0, []int{23, 17}, price); !closeTo(got, -2) {
			t.Errorf("p2 leftover after two actions = %v, want -2", got)
		}
	})

	t.Run("the carry-over sums into the bar before anything is paid", func(t *testing.T) {
		// Round two for a character that carried +9 and rolls 20 against a price of 11.
		if got := eco.Key(9, nil, 20, price); !closeTo(got, 29) {
			t.Errorf("Key with carry = %v, want 29", got)
		}
		if got := eco.Balance(9, []int{20}, price); !closeTo(got, 18) {
			t.Errorf("Balance with carry = %v, want 18", got)
		}
	})

	t.Run("a fractional carry survives instead of being rounded away", func(t *testing.T) {
		// A character who acted twice at 17 and 12 against a price of 11 leaves 14.5 − 22.
		if got := eco.Balance(0, []int{17, 12}, price); !closeTo(got, -7.5) {
			t.Errorf("Balance = %v, want -7.5 — the half must not be truncated", got)
		}
	})
}

func TestBarEconomy_IsEligible(t *testing.T) {
	eco := service.BarEconomy{}
	const price = 11

	t.Run("first action of the round — the gate is the BAR, not the raw roll", func(t *testing.T) {
		if !eco.IsEligible(0, nil, 11, price) {
			t.Error("landing exactly on the price must act")
		}
		if eco.IsEligible(0, nil, 10, price) {
			t.Error("below the price with no credit must sit out the round")
		}
		if !eco.IsEligible(5, nil, 10, price) {
			t.Error("credit must be able to rescue a roll below the price")
		}
	})

	t.Run("second action onward — the gate is the leftover", func(t *testing.T) {
		// p2 has acted once at 23 against a price of 11: leftover 12, so a second
		// action is granted BEFORE it is rolled.
		if !eco.IsEligible(0, []int{23}, 17, price) {
			t.Error("a leftover of 12 must buy another action")
		}
		// p1 has acted once at 20: leftover 9, not enough.
		if eco.IsEligible(0, []int{20}, 20, price) {
			t.Error("a leftover of 9 must not buy another action, however good the new roll")
		}
	})

	t.Run("a granted right is not revoked by a bad roll", func(t *testing.T) {
		// The very case that breaks if the key is used as the gate: key 9 < price 11.
		if eco.Key(0, []int{23}, 17, price) >= price {
			t.Fatal("precondition: this key is below the price")
		}
		if !eco.IsEligible(0, []int{23}, 17, price) {
			t.Error("eligibility is decided before the new roll; the key only orders")
		}
	})
}

func TestBarEconomy_CloseBalance(t *testing.T) {
	eco := service.BarEconomy{}
	const price = 11

	t.Run("the canonical closing balances", func(t *testing.T) {
		if got := eco.CloseBalance(0, []int{20}, price); !closeTo(got, 9) {
			t.Errorf("p1 close = %v, want +9", got)
		}
		if got := eco.CloseBalance(0, []int{11}, price); !closeTo(got, 0) {
			t.Errorf("p3 close = %v, want 0", got)
		}
		if got := eco.CloseBalance(0, []int{23, 17}, price); !closeTo(got, -2) {
			t.Errorf("p2 close = %v, want -2", got)
		}
	})

	t.Run("the ceiling is the round price", func(t *testing.T) {
		if got := eco.CloseBalance(9, []int{20}, price); !closeTo(got, 11) {
			t.Errorf("close = %v, want the ceiling 11 — 18 may not be carried", got)
		}
	})

	t.Run("standing still carries the floor, which is the same number", func(t *testing.T) {
		if got := eco.CloseBalance(0, nil, price); !closeTo(got, 11) {
			t.Errorf("close = %v, want 11: not acting trades an action for time", got)
		}
		if got := eco.CloseBalance(9, nil, price); !closeTo(got, 11) {
			t.Errorf("close = %v, want 11 — credit does not stack past the ceiling", got)
		}
		if got := eco.CloseBalance(-2, nil, price); !closeTo(got, 9) {
			t.Errorf("close = %v, want 9: a debtor recovers, but only up to the floor", got)
		}
	})
}
