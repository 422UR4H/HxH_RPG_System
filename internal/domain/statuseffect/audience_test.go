package statuseffect_test

import (
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

func TestListed(t *testing.T) {
	owner, other := uuid.New(), uuid.New()
	pc := statuseffect.Subject{OwnerID: owner}
	npc := statuseffect.Subject{OwnerID: owner, IsNPC: true}
	master := statuseffect.Viewer{UserID: uuid.New(), IsMaster: true}
	ownerV := statuseffect.Viewer{UserID: owner}
	otherV := statuseffect.Viewer{UserID: other}

	pub := statuseffect.Effect{Kind: statuseffect.KindProne}
	hidden := statuseffect.Effect{Kind: statuseffect.KindBleeding}
	with := func(e statuseffect.Effect, a *statuseffect.Audience) statuseffect.Effect {
		e.Audience = a
		return e
	}

	tests := []struct {
		name     string
		e        statuseffect.Effect
		v        statuseffect.Viewer
		s        statuseffect.Subject
		pcPublic bool
		want     bool
	}{
		{"master sees empty list", with(hidden, &statuseffect.Audience{Players: []uuid.UUID{}}), master, pc, false, true},
		{"master sees hidden default on npc", hidden, master, npc, false, true},
		{"public kind: owner", pub, ownerV, pc, false, true},
		{"public kind: other", pub, otherV, pc, false, true},
		{"public kind on npc: other", pub, otherV, npc, false, true},
		{"hidden kind on pc: owner", hidden, ownerV, pc, false, true},
		{"hidden kind on pc: other", hidden, otherV, pc, false, false},
		{"hidden kind on npc: owner", hidden, ownerV, npc, false, false},
		{"hidden kind on npc: other", hidden, otherV, npc, false, false},
		{"flag on: other sees pc hidden kind", hidden, otherV, pc, true, true},
		{"flag on: other does not see npc hidden kind", hidden, otherV, npc, true, false},
		{"everyone: other", with(hidden, &statuseffect.Audience{Everyone: true}), otherV, pc, false, true},
		{"empty list: owner not included", with(pub, &statuseffect.Audience{Players: []uuid.UUID{}}), ownerV, pc, false, false},
		{"list [other]: other", with(hidden, &statuseffect.Audience{Players: []uuid.UUID{other}}), otherV, pc, false, true},
		{"list [other]: owner not included", with(hidden, &statuseffect.Audience{Players: []uuid.UUID{other}}), ownerV, pc, false, false},
		{"list [owner] on public kind: other", with(pub, &statuseffect.Audience{Players: []uuid.UUID{owner}}), otherV, pc, false, false},
		{"explicit list ignores flag", with(hidden, &statuseffect.Audience{Players: []uuid.UUID{}}), otherV, pc, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := statuseffect.Listed(tt.e, tt.v, tt.s, tt.pcPublic); got != tt.want {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}
}

func TestAudienceValidate(t *testing.T) {
	a := uuid.New()
	tests := []struct {
		name    string
		aud     statuseffect.Audience
		wantErr bool
	}{
		{"everyone alone", statuseffect.Audience{Everyone: true}, false},
		{"empty list is master only", statuseffect.Audience{Players: []uuid.UUID{}}, false},
		{"nil list", statuseffect.Audience{}, false},
		{"one player", statuseffect.Audience{Players: []uuid.UUID{a}}, false},
		{"everyone with players", statuseffect.Audience{Everyone: true, Players: []uuid.UUID{a}}, true},
		{"duplicate player", statuseffect.Audience{Players: []uuid.UUID{a, a}}, true},
		{"nil uuid", statuseffect.Audience{Players: []uuid.UUID{uuid.Nil}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.aud.Validate()
			if tt.wantErr != errors.Is(err, statuseffect.ErrInvalidAudience) {
				t.Fatalf("err=%v wantErr=%v", err, tt.wantErr)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected err %v", err)
			}
		})
	}
}
