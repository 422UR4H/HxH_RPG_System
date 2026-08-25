package game

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
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
		// actionSpeed.SkillName is ALWAYS Legerity, whatever the payload sent
		if a.Speed.SkillName != enum.Legerity.String() {
			t.Errorf("Speed.SkillName = %q, want Legerity", a.Speed.SkillName)
		}
		// Bar is deliberately NOT mapped: which bar an action pays from is derived from its
		// content by Action.Bars(), never trusted from the client. The payload above sends
		// Bar: 1 precisely so this assertion fails if someone starts honouring it again.
		if a.Speed.Bar != 0 {
			t.Errorf("Speed.Bar = %d, want 0 — the client's bar must be ignored", a.Speed.Bar)
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
		Dodge:     &DodgePayload{Category: "closedDodge", RollCheck: &RollCheckPayload{SkillName: enum.Reflex.String()}},
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
	if a.Dodge == nil || a.Dodge.SkillName != enum.Reflex.String() {
		t.Errorf("Dodge = %+v, want its roll check mapped", a.Dodge)
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

func TestBuildAction_SpeedSkills(t *testing.T) {
	actor := uuid.New()

	t.Run("actionSpeed is always Legerity, whatever the client asked for", func(t *testing.T) {
		a, err := buildAction(actor, ActionPayload{
			ActorID: actor,
			Speed:   &ActionSpeedPayload{RollCheck: &RollCheckPayload{SkillName: enum.Accuracy.String()}},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if a.Speed.SkillName != enum.Legerity.String() {
			t.Errorf("actionSpeed skill = %q, want %q — the player never picks it",
				a.Speed.SkillName, enum.Legerity)
		}
	})

	t.Run("actionSpeed is Legerity even when the payload omits speed entirely", func(t *testing.T) {
		a, err := buildAction(actor, ActionPayload{ActorID: actor})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if a.Speed.SkillName != enum.Legerity.String() {
			t.Errorf("actionSpeed skill = %q, want %q", a.Speed.SkillName, enum.Legerity)
		}
	})

	t.Run("a Dash rolls Accelerate", func(t *testing.T) {
		a, err := buildAction(actor, ActionPayload{
			ActorID: actor,
			Move:    &MovePayload{Category: string(enum.Dash), Position: [3]int{1, 1, 0}},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if a.Move.Speed == nil {
			t.Fatal("a move must always carry a speed check")
		}
		if a.Move.Speed.SkillName != enum.Accelerate.String() {
			t.Errorf("move skill = %q, want %q", a.Move.Speed.SkillName, enum.Accelerate)
		}
	})

	t.Run("a Shift uses Brake, and the client's choice is overwritten", func(t *testing.T) {
		a, err := buildAction(actor, ActionPayload{
			ActorID: actor,
			Move: &MovePayload{
				Category: string(enum.Shift),
				Position: [3]int{1, 1, 0},
				Speed:    &RollCheckPayload{SkillName: enum.Accelerate.String()},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if a.Move.Speed.SkillName != enum.Brake.String() {
			t.Errorf("move skill = %q, want %q — the category picks the skill, not the client",
				a.Move.Speed.SkillName, enum.Brake)
		}
	})

	t.Run("the five unmapped categories are refused, not guessed", func(t *testing.T) {
		for _, cat := range []enum.MoveCategory{enum.Back, enum.Roll, enum.Slide, enum.Jump, enum.FlatJump} {
			t.Run(string(cat), func(t *testing.T) {
				_, err := buildAction(actor, ActionPayload{
					ActorID: actor,
					Move:    &MovePayload{Category: string(cat), Position: [3]int{1, 1, 0}},
				})
				if err == nil {
					t.Errorf("category %q must be refused: its skill is defined in the movement slice, and mapping it by analogy would be silently wrong", cat)
				}
			})
		}
	})

	t.Run("an unknown category string is refused too", func(t *testing.T) {
		if _, err := buildAction(actor, ActionPayload{
			ActorID: actor,
			Move:    &MovePayload{Category: "Teleport", Position: [3]int{1, 1, 0}},
		}); err == nil {
			t.Error("an unknown move category must be an error at the boundary")
		}
	})
}

func TestBuildAction_Reactions(t *testing.T) {
	actorID := uuid.New()

	t.Run("maps the declared kind onto the action", func(t *testing.T) {
		a, err := buildAction(actorID, ActionPayload{
			ActorID:      actorID,
			ReactToID:    uuid.New(),
			ReactionKind: "repel",
			Repel:        &RepelPayload{RollCheck: RollCheckPayload{SkillName: "Repel"}},
		})
		if err != nil {
			t.Fatalf("buildAction: %v", err)
		}
		if a.ReactionKind != action.ReactRepel {
			t.Errorf("ReactionKind = %q, want repel", a.ReactionKind)
		}
		if a.Repel == nil || a.Repel.SkillName != "Repel" {
			t.Fatal("the repel component did not survive the mapping")
		}
	})

	t.Run("an unknown kind is refused at the boundary", func(t *testing.T) {
		_, err := buildAction(actorID, ActionPayload{
			ActorID: actorID, ReactToID: uuid.New(), ReactionKind: "parry",
		})
		if err == nil {
			t.Fatal("an unknown kind must not reach the domain")
		}
	})

	t.Run("an escape without a move is refused — an escape that does not displace is a dodge", func(t *testing.T) {
		_, err := buildAction(actorID, ActionPayload{
			ActorID: actorID, ReactToID: uuid.New(), ReactionKind: "closedEscape",
		})
		if err == nil {
			t.Fatal("a displacing reaction with no Move must be refused, never defaulted")
		}
	})

	t.Run("an action carries no reaction kind", func(t *testing.T) {
		a, err := buildAction(actorID, ActionPayload{
			ActorID: actorID, Attack: &AttackPayload{},
		})
		if err != nil {
			t.Fatalf("buildAction: %v", err)
		}
		if a.ReactionKind != "" {
			t.Errorf("ReactionKind = %q, want empty on a plain action", a.ReactionKind)
		}
	})

	t.Run("nothing requires no component and is accepted bare", func(t *testing.T) {
		a, err := buildAction(actorID, ActionPayload{
			ActorID: actorID, ReactToID: uuid.New(), ReactionKind: "nothing",
		})
		if err != nil {
			t.Fatalf("buildAction: %v", err)
		}
		if a.ReactionKind != action.ReactNothing {
			t.Errorf("ReactionKind = %q, want nothing", a.ReactionKind)
		}
	})

	t.Run("dodge without a dodge component is refused", func(t *testing.T) {
		_, err := buildAction(actorID, ActionPayload{
			ActorID: actorID, ReactToID: uuid.New(), ReactionKind: "dodge",
		})
		if err == nil {
			t.Fatal("a dodge reaction with no Dodge payload must be refused, never derived against an empty roll")
		}
	})

	t.Run("dodge with a dodge component is accepted", func(t *testing.T) {
		a, err := buildAction(actorID, ActionPayload{
			ActorID: actorID, ReactToID: uuid.New(), ReactionKind: "dodge",
			Dodge: &DodgePayload{RollCheck: &RollCheckPayload{SkillName: enum.Reflex.String()}},
		})
		if err != nil {
			t.Fatalf("buildAction: %v", err)
		}
		if a.Dodge == nil {
			t.Fatal("the dodge component did not survive the mapping")
		}
	})

	t.Run("closedDodge without a dodge component is refused", func(t *testing.T) {
		_, err := buildAction(actorID, ActionPayload{
			ActorID: actorID, ReactToID: uuid.New(), ReactionKind: "closedDodge",
		})
		if err == nil {
			t.Fatal("a closedDodge reaction with no Dodge payload must be refused")
		}
	})

	// Without this, a closedDodge accepted with a Dodge but no Evasion entry derives Evasion
	// against an empty RollCheck (skillValue + 0, Passive: false) — strictly worse than a
	// plain dodge, and dodgeAndReserve still banks a reserve off the bogus gap. See
	// ReactionKind.RequiresEvasionSkill.
	t.Run("closedDodge with a dodge but no evasion skill entry is refused", func(t *testing.T) {
		_, err := buildAction(actorID, ActionPayload{
			ActorID: actorID, ReactToID: uuid.New(), ReactionKind: "closedDodge",
			Dodge: &DodgePayload{RollCheck: &RollCheckPayload{SkillName: enum.Reflex.String()}},
		})
		if err == nil {
			t.Fatal("a closedDodge with no Evasion entry must be refused, never derived against an empty roll")
		}
	})

	t.Run("closedDodge with a dodge and an evasion skill entry is accepted", func(t *testing.T) {
		a, err := buildAction(actorID, ActionPayload{
			ActorID: actorID, ReactToID: uuid.New(), ReactionKind: "closedDodge",
			Dodge:  &DodgePayload{RollCheck: &RollCheckPayload{SkillName: enum.Reflex.String()}},
			Skills: []ActionSkillPayload{{SkillName: enum.Evasion.String()}},
		})
		if err != nil {
			t.Fatalf("buildAction: %v", err)
		}
		if a.Dodge == nil {
			t.Fatal("the dodge component did not survive the mapping")
		}
		found := false
		for _, s := range a.Skills {
			if s.SkillName == enum.Evasion.String() {
				found = true
			}
		}
		if !found {
			t.Fatal("the evasion skill entry did not survive the mapping")
		}
	})

	t.Run("closedEscape with a dodge and a move but no evasion skill entry is refused", func(t *testing.T) {
		_, err := buildAction(actorID, ActionPayload{
			ActorID: actorID, ReactToID: uuid.New(), ReactionKind: "closedEscape",
			Dodge: &DodgePayload{RollCheck: &RollCheckPayload{SkillName: enum.Reflex.String()}},
			Move:  &MovePayload{Category: string(enum.Dash), Position: [3]int{1, 1, 0}},
		})
		if err == nil {
			t.Fatal("a closedEscape with no Evasion entry must be refused just like closedDodge")
		}
	})

	t.Run("escapeGuard with a dodge but no move is refused — displacing is still required", func(t *testing.T) {
		_, err := buildAction(actorID, ActionPayload{
			ActorID: actorID, ReactToID: uuid.New(), ReactionKind: "escapeGuard",
			Dodge: &DodgePayload{RollCheck: &RollCheckPayload{SkillName: enum.Reflex.String()}},
		})
		if err == nil {
			t.Fatal("an escape with a Dodge but no Move must still be refused — it forces the dodge BY displacing")
		}
	})

	t.Run("escapeGuard with a move but no dodge is refused — the old Displaces() check alone was not enough", func(t *testing.T) {
		_, err := buildAction(actorID, ActionPayload{
			ActorID: actorID, ReactToID: uuid.New(), ReactionKind: "escapeGuard",
			Move: &MovePayload{Category: string(enum.Dash), Position: [3]int{1, 1, 0}},
		})
		if err == nil {
			t.Fatal("an escape with a Move but no Dodge must be refused too")
		}
	})

	t.Run("escapeGuard with both a dodge and a move is accepted", func(t *testing.T) {
		a, err := buildAction(actorID, ActionPayload{
			ActorID: actorID, ReactToID: uuid.New(), ReactionKind: "escapeGuard",
			Dodge: &DodgePayload{RollCheck: &RollCheckPayload{SkillName: enum.Reflex.String()}},
			Move:  &MovePayload{Category: string(enum.Dash), Position: [3]int{1, 1, 0}},
		})
		if err != nil {
			t.Fatalf("buildAction: %v", err)
		}
		if a.Dodge == nil || a.Move == nil {
			t.Fatal("both components must survive the mapping")
		}
	})

	t.Run("repel without a repel component is refused", func(t *testing.T) {
		_, err := buildAction(actorID, ActionPayload{
			ActorID: actorID, ReactToID: uuid.New(), ReactionKind: "repel",
		})
		if err == nil {
			t.Fatal("a repel reaction with no Repel payload must be refused, never derived against an empty roll into RungFailure")
		}
	})
}

func TestBuildEditAction_RejectsAnUnknownField(t *testing.T) {
	p := EditActionPayload{
		Conditions: []ConditionEditPayload{{Field: "luck", Modifier: 3}},
	}
	if _, err := buildEditAction(p); err == nil {
		t.Error("expected an unrecognized condition field to be rejected at the boundary")
	}
}

func TestBuildEditAction_MapsKnownField(t *testing.T) {
	p := EditActionPayload{
		Conditions: []ConditionEditPayload{{Field: "hit", Bias: 1, Modifier: 3, Description: "flanked"}},
	}
	ma, err := buildEditAction(p)
	if err != nil {
		t.Fatalf("buildEditAction: %v", err)
	}
	if len(ma.Conditions) != 1 {
		t.Fatalf("Conditions = %+v, want exactly one entry", ma.Conditions)
	}
	got := ma.Conditions[0]
	if got.Field != action.FieldHit || got.Condition.Bias != 1 || got.Condition.Modifier != 3 ||
		got.Condition.Description != "flanked" {
		t.Errorf("Conditions[0] = %+v, want Field=hit Bias=1 Modifier=3 Description=flanked", got)
	}
}
