package game

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/google/uuid"
)

func TestBuildAction_MapsTheWholePayload(t *testing.T) {
	actorCharID := uuid.New()
	targetID := uuid.New()
	weapon := "Sword"

	p := ActionPayload{
		ActorID:  actorCharID,
		TargetID: []uuid.UUID{targetID},
		Skills: []ActionSkillPayload{
			{SkillName: enum.Acrobatics.String()},
		},
		Speed: &ActionSpeedPayload{
			Bar:       1,
			RollCheck: &RollCheckPayload{SkillName: enum.Legerity.String()},
		},
		Feint: &RollCheckPayload{SkillName: enum.Feint.String()},
		Attack: &AttackPayload{
			Weapon: &weapon,
			Hit:    RollCheckPayload{SkillName: enum.Accuracy.String()},
			Damage: RollCheckPayload{SkillName: enum.Push.String()},
			Charge: &RollCheckPayload{SkillName: enum.Brake.String()},
		},
		Defense: &DefensePayload{
			RollCheck: RollCheckPayload{SkillName: enum.Defense.String()},
		},
	}

	a, err := buildAction(actorCharID, p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("the actor is the character, not the player", func(t *testing.T) {
		if a.GetActorID() != actorCharID {
			t.Errorf("actorID = %v, want %v", a.GetActorID(), actorCharID)
		}
	})

	t.Run("the attack survives", func(t *testing.T) {
		if a.Attack == nil {
			t.Fatal("Attack was dropped")
		}
		if a.Attack.Weapon == nil || *a.Attack.Weapon != enum.Sword {
			t.Errorf("Weapon = %v, want Sword", a.Attack.Weapon)
		}
		if a.Attack.Hit.SkillName != enum.Accuracy.String() {
			t.Errorf("Hit.SkillName = %q, want Accuracy", a.Attack.Hit.SkillName)
		}
		if a.Attack.Charge == nil {
			t.Error("Charge was dropped")
		}
	})

	t.Run("speed, skills, feint and defense survive", func(t *testing.T) {
		if a.Speed.Bar != 1 || a.Speed.SkillName != enum.Legerity.String() {
			t.Errorf("Speed = %+v, want bar 1 and Legerity", a.Speed)
		}
		if len(a.Skills) != 1 || a.Skills[0].SkillName != enum.Acrobatics.String() {
			t.Errorf("Skills = %+v, want one Acrobatics", a.Skills)
		}
		if a.Feint == nil {
			t.Error("Feint was dropped")
		}
		if a.Defense == nil || a.Defense.SkillName != enum.Defense.String() {
			t.Errorf("Defense = %+v, want a Defense check", a.Defense)
		}
	})

	t.Run("no dice have fallen yet — that is the session's job", func(t *testing.T) {
		if !a.Attack.Hit.Attempts.IsEmpty() {
			t.Error("the mapper must not roll; the session rolls on arrival")
		}
	})
}

func TestBuildAction_RejectsAnUnknownSkillName(t *testing.T) {
	p := ActionPayload{
		ActorID: uuid.New(),
		Attack: &AttackPayload{
			Hit:    RollCheckPayload{SkillName: "Kamehameha"},
			Damage: RollCheckPayload{SkillName: enum.Push.String()},
		},
	}
	if _, err := buildAction(p.ActorID, p); err == nil {
		t.Error("expected an unknown skill name to be rejected at the boundary")
	}
}

func TestBuildAction_RejectsAnUnknownWeapon(t *testing.T) {
	bogus := "Excalibur"
	p := ActionPayload{
		ActorID: uuid.New(),
		Attack: &AttackPayload{
			Weapon: &bogus,
			Hit:    RollCheckPayload{SkillName: enum.Accuracy.String()},
			Damage: RollCheckPayload{SkillName: enum.Push.String()},
		},
	}
	if _, err := buildAction(p.ActorID, p); err == nil {
		t.Error("expected a weapon outside the catalogue to be rejected at the boundary")
	}
}

func TestBuildAction_KeepsMappingWhatItAlreadyMapped(t *testing.T) {
	actorCharID := uuid.New()
	p := ActionPayload{
		ActorID: actorCharID,
		Move: &MovePayload{
			Category: string(enum.Dash),
			From:     [3]int{1, 1, 0},
			Position: [3]int{2, 1, 0},
			Speed:    &RollCheckPayload{SkillName: enum.Accelerate.String()},
		},
		Dodge:     &DodgePayload{Category: string(enum.Evasive), RollCheck: &RollCheckPayload{SkillName: enum.Reflex.String()}},
		Interact:  &InteractPayload{Kind: "open"},
		ReactToID: uuid.New(),
	}
	a, err := buildAction(actorCharID, p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Move == nil || a.Move.Position != [3]int{2, 1, 0} {
		t.Errorf("Move = %+v, want the mapped position", a.Move)
	}
	if a.Move.Speed == nil || a.Move.Speed.SkillName != enum.Accelerate.String() {
		t.Error("Move.Speed was dropped")
	}
	if a.Dodge == nil || a.Dodge.Category != enum.Evasive {
		t.Errorf("Dodge = %+v, want an Evasive dodge", a.Dodge)
	}
	if a.Interact == nil {
		t.Error("Interact was dropped")
	}
	if a.ReactToID != p.ReactToID {
		t.Errorf("ReactToID = %v, want %v", a.ReactToID, p.ReactToID)
	}
}

func TestBuildAction_EmptyPayloadIsValid(t *testing.T) {
	// A bare action — no attack, no move — is legal; the session still rolls its speed.
	actorCharID := uuid.New()
	a, err := buildAction(actorCharID, ActionPayload{ActorID: actorCharID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Attack != nil || a.Move != nil || a.Defense != nil {
		t.Errorf("expected an empty action, got %+v", a)
	}
}
