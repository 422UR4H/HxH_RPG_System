package service_test

import (
	"testing"

	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/item"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

// reactionOf monta a reação de actorID já com os dados caídos — o estado que a sessão entrega.
func reactionOf(actorID uuid.UUID, kind action.ReactionKind) *action.Action {
	r := action.NewAction(actorID, nil, uuid.Nil, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil)
	r.ReactionKind = kind
	return r
}

// withReaction anexa (sem abrir) a reação ao ataque do turno.
func withReaction(tn *turn.Turn, r *action.Action) {
	a := tn.GetAction()
	r.ReactToID = a.GetID()
	tn.AddReaction(r)
}

func weakenedPhysical() statuseffect.Effect {
	return statuseffect.Effect{
		Kind: statuseffect.KindWeakened, Aptitude: statuseffect.AptitudePhysical,
		Params: statuseffect.Params{Disadvantage: true},
	}
}

func onlyResult(t *testing.T, res *service.TurnResolution) service.CharacterResult {
	t.Helper()
	if len(res.CharacterResults) != 1 {
		t.Fatalf("expected 1 character result, got %d (errors %v)", len(res.CharacterResults), res.Errors)
	}
	return res.CharacterResults[0]
}

func TestResolve_StatusEffectsWeighOnRolls(t *testing.T) {
	sword := enum.Sword
	actorID, targetID := uuid.New(), uuid.New()

	t.Run("weakened attacker hits with disadvantage", func(t *testing.T) {
		tn := attackTurn(actorID, targetID, []int{8, 6}, []int{1, 1}, &sword)
		act := tn.GetAction()
		in := resolveInput(t, actorID, targetID, tn)
		in.Effects = map[uuid.UUID][]statuseffect.Effect{actorID: {weakenedPhysical()}}

		// Sem o extra, a falta é pedida no acerto.
		res := service.TurnResolver{}.Resolve(in)
		if len(res.DiceShort) != 1 || res.DiceShort[0].Ref.Field != action.FieldHit ||
			res.DiceShort[0].Ref.ActionID != act.GetID() || res.DiceShort[0].Need != 1 {
			t.Fatalf("DiceShort = %+v, want one die on the hit", res.DiceShort)
		}

		act.Attack.Hit.Attempts.Extra = []int{2}
		in.Turn = turn.NewTurn(act)
		res = service.TurnResolver{}.Resolve(in)
		cr := onlyResult(t, res)
		if cr.Hit.Bias != -1 {
			t.Errorf("Hit.Bias = %d, want -1", cr.Hit.Bias)
		}
		// Desvantagem mantém os dois piores de [8 6 2]: 6 + 2.
		if cr.Hit.Total != 8 {
			t.Errorf("Hit.Total = %d, want 8", cr.Hit.Total)
		}
		if len(res.DiceShort) != 0 {
			t.Errorf("DiceShort = %+v, want none once the extra fell", res.DiceShort)
		}
	})

	t.Run("vision nullified target dodges with disadvantage", func(t *testing.T) {
		tn := attackTurn(actorID, targetID, []int{3, 3}, []int{1, 1}, &sword)
		r := reactionOf(targetID, action.ReactDodge)
		r.Dodge = &action.Dodge{RollCheck: action.RollCheck{
			SkillName: enum.Reflex.String(),
			Attempts:  action.RollAttempts{Primary: []int{5, 4}, Extra: []int{1}},
		}}
		withReaction(tn, r)
		in := resolveInput(t, actorID, targetID, tn)
		in.Effects = map[uuid.UUID][]statuseffect.Effect{targetID: {{
			Kind: statuseffect.KindSense, Sense: statuseffect.SenseVision,
			Params: statuseffect.Params{Level: statuseffect.SenseNullified},
		}}}

		cr := onlyResult(t, service.TurnResolver{}.Resolve(in))
		if cr.Dodge.Bias != -1 || cr.Dodge.Total != 5 {
			t.Errorf("Dodge = bias %d total %d, want bias -1 total 5 (4 + 1)", cr.Dodge.Bias, cr.Dodge.Total)
		}
		// O acerto do atacante não é tocado pelos status do alvo.
		if cr.Hit.Bias != 0 {
			t.Errorf("Hit.Bias = %d, want 0", cr.Hit.Bias)
		}
	})

	t.Run("prone target escape both tests disadvantaged", func(t *testing.T) {
		// No resolvedor, o teste de esquiva do escape lê a restrição "escape" do Caído (não a
		// "reflexDodge"): um nível só. O teste de movimento é derivado na sessão (Task 6).
		tn := attackTurn(actorID, targetID, []int{3, 3}, []int{1, 1}, &sword)
		r := reactionOf(targetID, action.ReactEscape)
		r.Dodge = &action.Dodge{RollCheck: action.RollCheck{
			SkillName: enum.Reflex.String(),
			Attempts:  action.RollAttempts{Primary: []int{5, 4}, Extra: []int{1}},
		}}
		withReaction(tn, r)
		in := resolveInput(t, actorID, targetID, tn)
		in.Effects = map[uuid.UUID][]statuseffect.Effect{targetID: {{Kind: statuseffect.KindProne}}}

		cr := onlyResult(t, service.TurnResolver{}.Resolve(in))
		if cr.Dodge.Bias != -1 || cr.Dodge.Total != 5 {
			t.Errorf("escape dodge = bias %d total %d, want bias -1 total 5", cr.Dodge.Bias, cr.Dodge.Total)
		}
	})

	t.Run("injured attack target repels with disadvantage", func(t *testing.T) {
		// R1: o repelir está em ataque E defesa — o Lesionado só no ataque pesa nele.
		repelWith := func(attempts action.RollAttempts, effects []statuseffect.Effect) (*service.TurnResolution, *action.Action) {
			tn := attackTurn(actorID, targetID, []int{3, 3}, []int{1, 1}, &sword)
			r := reactionOf(targetID, action.ReactRepel)
			r.Repel = &action.Repel{RollCheck: action.RollCheck{SkillName: enum.Repel.String(), Attempts: attempts}}
			withReaction(tn, r)
			in := resolveInput(t, actorID, targetID, tn)
			in.Effects = map[uuid.UUID][]statuseffect.Effect{targetID: effects}
			return service.TurnResolver{}.Resolve(in), r
		}
		injured := []statuseffect.Effect{{Kind: statuseffect.KindInjured, Params: statuseffect.Params{
			Injuries: map[statuseffect.Domain]statuseffect.InjuryLevel{statuseffect.DomainAttack: statuseffect.InjuryHurt},
		}}}

		res, r := repelWith(action.RollAttempts{Primary: []int{7, 4}}, injured)
		want := service.DiceShortage{Ref: service.DiceRef{ActionID: r.GetID(), Field: action.FieldRepel}, Need: 1}
		if len(res.DiceShort) != 1 || res.DiceShort[0] != want {
			t.Fatalf("DiceShort = %+v, want %+v", res.DiceShort, want)
		}

		plain, _ := repelWith(action.RollAttempts{Primary: []int{7, 4}}, nil)
		got, _ := repelWith(action.RollAttempts{Primary: []int{7, 4}, Extra: []int{1}}, injured)
		// Desvantagem mantém os dois piores de [7 4 1]: 4 + 1 = 5, contra 7 + 4 = 11 sem status.
		if d := onlyResult(t, plain).ReactionTotal - onlyResult(t, got).ReactionTotal; d != 6 {
			t.Errorf("repel total dropped by %d, want 6 (11 → 5)", d)
		}
	})

	t.Run("open guard default defense rolls", func(t *testing.T) {
		// Acerto 19 passa a esquiva passiva (11); a defesa padrão, só rolada, pede os dados.
		tn := attackTurn(actorID, targetID, []int{10, 9}, []int{1, 1}, &sword)
		r := reactionOf(targetID, action.ReactDefault)
		withReaction(tn, r)
		in := resolveInput(t, actorID, targetID, tn)
		in.Effects = map[uuid.UUID][]statuseffect.Effect{targetID: {{Kind: statuseffect.KindOpenGuard}}}

		res := service.TurnResolver{}.Resolve(in)
		want := service.DiceShortage{
			Ref:  service.DiceRef{ActionID: r.GetID(), Field: action.FieldDefense},
			Need: 2,
		}
		if len(res.DiceShort) != 1 || res.DiceShort[0] != want {
			t.Fatalf("DiceShort = %+v, want %+v", res.DiceShort, want)
		}

		r.DefaultDefense = &action.RollCheck{
			SkillName: enum.Defense.String(),
			Attempts:  action.RollAttempts{Primary: []int{3, 2}},
		}
		tn = attackTurn(actorID, targetID, []int{10, 9}, []int{1, 1}, &sword)
		withReaction(tn, r)
		in.Turn = tn
		res = service.TurnResolver{}.Resolve(in)
		cr := onlyResult(t, res)
		if cr.Defense.Passive || cr.Defense.Total != 5 {
			t.Errorf("Defense = passive %v total %d, want rolled 5", cr.Defense.Passive, cr.Defense.Total)
		}
		if cr.Defended {
			t.Error("a rolled 5 does not clear the CD of 9")
		}
		if len(res.DiceShort) != 0 {
			t.Errorf("DiceShort = %+v, want none", res.DiceShort)
		}
	})
}

func TestResolve_CriticalDamage(t *testing.T) {
	sword := enum.Sword
	actorID := uuid.New()

	chainOfTwo := func(t *testing.T, hitDice []int, damageMod int) *service.TurnResolution {
		t.Helper()
		a, b := uuid.New(), uuid.New()
		na := reactionOf(a, action.ReactNothing)
		nb := reactionOf(b, action.ReactNothing)
		tn := areaTurn(t, actorID, []uuid.UUID{a, b}, hitDice, []int{7, 3}, &sword,
			map[int]*action.Action{0: na, 1: nb}, []int{0, 1})
		if damageMod != 0 {
			act := tn.GetAction()
			act.Attack.Damage.Context.Condition = &action.RollCondition{Modifier: damageMod}
			tn = rebuildTurn(t, tn, act)
		}
		return service.TurnResolver{}.Resolve(service.ResolveInput{
			Turn: tn,
			Sheets: map[uuid.UUID]*csSheet.CharacterSheet{
				actorID: plainSheet(t), a: plainSheet(t), b: plainSheet(t),
			},
			Targets: charTargets{chars: map[uuid.UUID]bool{a: true, b: true}},
			Rules:   match.NewDefaultMatchRules(),
			Weapons: item.NewWeaponsManagerFactory().Build(),
		})
	}

	t.Run("critical hit doubles each target final damage", func(t *testing.T) {
		res := chainOfTwo(t, []int{10, 10}, 0)
		if len(res.CharacterResults) != 2 {
			t.Fatalf("expected 2 results, got %d", len(res.CharacterResults))
		}
		for i, cr := range res.CharacterResults {
			// 7 + 3 + 2 = 12 de dano bruto; sem armadura, o resíduo segue 12 para o segundo.
			if cr.RawDamage != 12 {
				t.Errorf("target %d RawDamage = %d, want 12 — the chain's residual is not doubled", i, cr.RawDamage)
			}
			if cr.EffectiveDamage != 24 {
				t.Errorf("target %d EffectiveDamage = %d, want 24", i, cr.EffectiveDamage)
			}
			if !cr.CriticalDamage {
				t.Errorf("target %d CriticalDamage = false, want true", i)
			}
		}
	})

	t.Run("critical on zero final damage stays zero", func(t *testing.T) {
		res := chainOfTwo(t, []int{10, 10}, -50)
		for i, cr := range res.CharacterResults {
			if cr.EffectiveDamage != 0 {
				t.Errorf("target %d EffectiveDamage = %d, want 0", i, cr.EffectiveDamage)
			}
			if cr.CriticalDamage {
				t.Errorf("target %d CriticalDamage = true, want false on zero damage", i)
			}
		}
	})

	t.Run("no critical keeps the final damage", func(t *testing.T) {
		res := chainOfTwo(t, []int{9, 10}, 0)
		for i, cr := range res.CharacterResults {
			if cr.EffectiveDamage != 12 || cr.CriticalDamage {
				t.Errorf("target %d = %d critical %v, want 12 and no critical", i, cr.EffectiveDamage, cr.CriticalDamage)
			}
		}
	})

	t.Run("critical failure on hit halves the total", func(t *testing.T) {
		res := chainOfTwo(t, []int{1, 1}, 0)
		cr := res.CharacterResults[0]
		if !cr.Hit.IsCriticalFailure || cr.Hit.Total != 1 {
			t.Errorf("Hit = critical failure %v total %d, want true and 1 (2 ÷ 2)", cr.Hit.IsCriticalFailure, cr.Hit.Total)
		}
		if cr.CriticalDamage {
			t.Error("a critical failure does not double the damage")
		}
		if cr.EffectiveDamage != 12 {
			t.Errorf("EffectiveDamage = %d, want 12 (nothing sent, the blow lands whole)", cr.EffectiveDamage)
		}
	})
}

// rebuildTurn refaz o turno com a ação editada, mantendo as reações e a ordem de abertura.
func rebuildTurn(t *testing.T, old *turn.Turn, a action.Action) *turn.Turn {
	t.Helper()
	tn := turn.NewTurn(a)
	for _, r := range old.GetReactions() {
		r := r
		tn.AddReaction(&r)
	}
	for _, id := range old.OpenedReactionIDs() {
		if !tn.OpenReaction(id) {
			t.Fatalf("reaction %s not attached", id)
		}
	}
	return tn
}

func TestResolve_Dies(t *testing.T) {
	sword := enum.Sword
	actorID, targetID := uuid.New(), uuid.New()

	// O alvo está com 10 de HP; o dano é ajustado pela condição do mestre sobre o dano bruto
	// da espada (1 + 1 + 2 = 4).
	resolveDamage := func(t *testing.T, damage int, effects []statuseffect.Effect) (service.CharacterResult, int) {
		t.Helper()
		tn := attackTurn(actorID, targetID, []int{9, 9}, []int{1, 1}, &sword)
		withReaction(tn, reactionOf(targetID, action.ReactNothing))
		act := tn.GetAction()
		act.Attack.Damage.Context.Condition = &action.RollCondition{Modifier: damage - 4}
		tn = rebuildTurn(t, tn, act)
		in := resolveInput(t, actorID, targetID, tn)
		bar := in.Sheets[targetID].GetAllStatusBar()[enum.Health]
		if err := bar.SetCurrent(10); err != nil {
			t.Fatalf("SetCurrent: %v", err)
		}
		in.Effects = map[uuid.UUID][]statuseffect.Effect{targetID: effects}
		cr := onlyResult(t, service.TurnResolver{}.Resolve(in))
		if bar.GetCurrent() != 10 {
			t.Fatalf("HP = %d after resolve, want 10 — a resolution is a dry run", bar.GetCurrent())
		}
		return cr, bar.GetMin()
	}

	t.Run("dies when final damage passes the negative max", func(t *testing.T) {
		_, minHP := resolveDamage(t, 1, nil)
		if minHP >= 0 {
			t.Fatalf("HP min = %d, want negative", minHP)
		}
		// Ficar exatamente no mínimo ainda é vivo; um ponto além é a morte.
		lives := 10 - minHP
		cr, _ := resolveDamage(t, lives, nil)
		if cr.EffectiveDamage != lives || cr.Dies {
			t.Errorf("damage %d (HP → %d) = dies %v, want alive", cr.EffectiveDamage, 10-cr.EffectiveDamage, cr.Dies)
		}
		cr, _ = resolveDamage(t, lives+1, nil)
		if cr.EffectiveDamage != lives+1 || !cr.Dies {
			t.Errorf("damage %d (HP → %d) = dies %v, want dies", cr.EffectiveDamage, 10-cr.EffectiveDamage, cr.Dies)
		}
	})

	t.Run("dead target is not reported as dying again", func(t *testing.T) {
		cr, _ := resolveDamage(t, 100, []statuseffect.Effect{{Kind: statuseffect.KindDead}})
		if cr.Dies {
			t.Error("a target already dead does not die again")
		}
	})
}

func TestResolve_WallCritical(t *testing.T) {
	sword := enum.Sword
	actorID, wallID := uuid.New(), uuid.New()
	wall := mapentity.WallSegment{ID: wallID.String(), HP: 40, MaxHP: 40, Resistance: 5}

	resolveWall := func(t *testing.T, hitDice []int) *service.TurnResolution {
		t.Helper()
		tn := attackTurn(actorID, wallID, hitDice, []int{7, 3}, &sword)
		return service.TurnResolver{}.Resolve(service.ResolveInput{
			Turn:    tn,
			Sheets:  map[uuid.UUID]*csSheet.CharacterSheet{actorID: plainSheet(t)},
			Targets: mockWallReader{wallID: wallID.String(), wall: wall},
			Rules:   match.NewDefaultMatchRules(),
			Weapons: item.NewWeaponsManagerFactory().Build(),
		})
	}

	t.Run("wall takes double on critical", func(t *testing.T) {
		// 12 de dano bruto − 5 de resistência = 7, dobrado = 14.
		res := resolveWall(t, []int{10, 10})
		if len(res.WallResults) != 1 {
			t.Fatalf("expected 1 wall result, got %d", len(res.WallResults))
		}
		wr := res.WallResults[0]
		if wr.EffectiveDamage != 14 || wr.UpdatedWall.HP != 26 {
			t.Errorf("wall = effective %d HP %d, want 14 and 26", wr.EffectiveDamage, wr.UpdatedWall.HP)
		}
		if wr.ReboundDamage != 5 {
			t.Errorf("ReboundDamage = %d, want 5 — the rebound is unchanged", wr.ReboundDamage)
		}
	})

	t.Run("wall without critical takes the plain damage", func(t *testing.T) {
		wr := resolveWall(t, []int{9, 10}).WallResults[0]
		if wr.EffectiveDamage != 7 || wr.UpdatedWall.HP != 33 {
			t.Errorf("wall = effective %d HP %d, want 7 and 33", wr.EffectiveDamage, wr.UpdatedWall.HP)
		}
	})

	t.Run("weakened attacker on a wall asks for the hit's extra die", func(t *testing.T) {
		tn := attackTurn(actorID, wallID, []int{10, 10}, []int{7, 3}, &sword)
		act := tn.GetAction()
		res := service.TurnResolver{}.Resolve(service.ResolveInput{
			Turn:    tn,
			Sheets:  map[uuid.UUID]*csSheet.CharacterSheet{actorID: plainSheet(t)},
			Effects: map[uuid.UUID][]statuseffect.Effect{actorID: {weakenedPhysical()}},
			Targets: mockWallReader{wallID: wallID.String(), wall: wall},
			Rules:   match.NewDefaultMatchRules(),
			Weapons: item.NewWeaponsManagerFactory().Build(),
		})
		want := service.DiceShortage{Ref: service.DiceRef{ActionID: act.GetID(), Field: action.FieldHit}, Need: 1}
		if len(res.DiceShort) != 1 || res.DiceShort[0] != want {
			t.Fatalf("DiceShort = %+v, want %+v", res.DiceShort, want)
		}
	})
}

// O Atordoado e o Morto bloqueiam (fase 1); a falha automática deles é só de sentido e não
// pode vazar para Accuracy, Reflex ou Defense.
func TestResolve_BlockingKindsLeaveNumbersAlone(t *testing.T) {
	sword := enum.Sword
	actorID, targetID := uuid.New(), uuid.New()

	resolve := func(t *testing.T, effects map[uuid.UUID][]statuseffect.Effect) service.CharacterResult {
		t.Helper()
		// Acerto 18 passa a esquiva passiva (11); a defesa padrão (11) é lida contra a CD 8.
		tn := attackTurn(actorID, targetID, []int{9, 9}, []int{1, 1}, &sword)
		withReaction(tn, reactionOf(targetID, action.ReactDefault))
		in := resolveInput(t, actorID, targetID, tn)
		in.Effects = effects
		res := service.TurnResolver{}.Resolve(in)
		if len(res.DiceShort) != 0 {
			t.Fatalf("DiceShort = %+v, want none", res.DiceShort)
		}
		return onlyResult(t, res)
	}

	for _, kind := range []statuseffect.Kind{statuseffect.KindStunned, statuseffect.KindDead} {
		name := map[statuseffect.Kind]string{
			statuseffect.KindStunned: "stunned has no effect on resolver numbers beyond blocks",
			statuseffect.KindDead:    "dead has no effect on resolver numbers beyond blocks",
		}[kind]
		t.Run(name, func(t *testing.T) {
			plain := resolve(t, nil)
			got := resolve(t, map[uuid.UUID][]statuseffect.Effect{
				actorID:  {{Kind: kind}},
				targetID: {{Kind: kind}},
			})
			if !plain.Defended || plain.Defense.Total == 0 {
				t.Fatalf("fixture: want the default defense read, got %+v", plain.Defense)
			}
			for _, c := range []struct {
				test      string
				got, want service.RollOutcome
			}{
				{"hit", got.Hit, plain.Hit},
				{"dodge", got.Dodge, plain.Dodge},
				{"defense", got.Defense, plain.Defense},
			} {
				if c.got.Total != c.want.Total || c.got.Bias != c.want.Bias || c.got.AutoFailed {
					t.Errorf("%s = total %d bias %d autoFailed %v, want total %d bias %d as with no effects",
						c.test, c.got.Total, c.got.Bias, c.got.AutoFailed, c.want.Total, c.want.Bias)
				}
			}
		})
	}
}
