package match_test

import (
	"reflect"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/google/uuid"
)

func TestResourceBar_SecretSpeeds(t *testing.T) {
	r1, r2 := uuid.New(), uuid.New()

	tests := []struct {
		name       string
		build      func(b *match.ResourceBar)
		wantSpeeds []int
		wantPublic []int
	}{
		{
			name: "a secret speed is recorded but not public",
			build: func(b *match.ResourceBar) {
				b.RecordSpeed(20)
				b.RecordSecretSpeed(r1, 14)
			},
			wantSpeeds: []int{20, 14},
			wantPublic: []int{20},
		},
		{
			name: "revealing it makes it public",
			build: func(b *match.ResourceBar) {
				b.RecordSpeed(20)
				b.RecordSecretSpeed(r1, 14)
				b.Reveal(r1)
			},
			wantSpeeds: []int{20, 14},
			wantPublic: []int{20, 14},
		},
		{
			name: "reveal of an unknown id is a no-op",
			build: func(b *match.ResourceBar) {
				b.RecordSecretSpeed(r1, 14)
				b.Reveal(r2)
			},
			wantSpeeds: []int{14},
			wantPublic: []int{},
		},
		{
			name: "RevealAll opens every secret",
			build: func(b *match.ResourceBar) {
				b.RecordSpeed(20)
				b.RecordSecretSpeed(r1, 14)
				b.RecordSecretSpeed(r2, 11)
				b.RevealAll()
			},
			wantSpeeds: []int{20, 14, 11},
			wantPublic: []int{20, 14, 11},
		},
		{
			name: "a duplicate value removes exactly one occurrence",
			build: func(b *match.ResourceBar) {
				b.RecordSpeed(20)
				b.RecordSecretSpeed(r1, 20)
			},
			wantSpeeds: []int{20, 20},
			wantPublic: []int{20},
		},
		{
			name: "ResetRound clears pending secrets",
			build: func(b *match.ResourceBar) {
				b.RecordSecretSpeed(r1, 14)
				b.ResetRound()
				b.RecordSpeed(9)
			},
			wantSpeeds: []int{9},
			wantPublic: []int{9},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b match.ResourceBar
			tt.build(&b)
			if !reflect.DeepEqual(b.Speeds, tt.wantSpeeds) {
				t.Errorf("Speeds = %v, want %v", b.Speeds, tt.wantSpeeds)
			}
			got := b.PublicSpeeds()
			if len(got) != len(tt.wantPublic) || (len(got) > 0 && !reflect.DeepEqual(got, tt.wantPublic)) {
				t.Errorf("PublicSpeeds() = %v, want %v", got, tt.wantPublic)
			}
		})
	}

	t.Run("mutating PublicSpeeds does not change Speeds", func(t *testing.T) {
		var b match.ResourceBar
		b.RecordSpeed(20)
		b.RecordSpeed(15)
		pub := b.PublicSpeeds()
		pub[0] = 99
		if b.Speeds[0] != 20 {
			t.Errorf("Speeds aliased: %v", b.Speeds)
		}
	})
}
