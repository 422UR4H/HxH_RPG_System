package matchsession_test

import (
	"testing"

	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

// Os testes que a SESSÃO deriva (velocidades, moveSpeed e skills[]) leem os status de quem
// testa, na chegada e a cada mudança de status (Task 6).

type rollsFixture struct {
	s                *matchsession.MatchSession
	src              *countingSource
	playerA, playerB uuid.UUID
	a, b             uuid.UUID
	sheetA           *csSheet.CharacterSheet
}

func newRollsFixture(t *testing.T) *rollsFixture {
	t.Helper()
	matchUUID := uuid.New()
	playerA, playerB := uuid.New(), uuid.New()
	pA, pB := makeParticipant(matchUUID, &playerA), makeParticipant(matchUUID, &playerB)
	a, b := pA.Sheet.UUID, pB.Sheet.UUID
	sheetA := buildPlainSheet(t)
	s := matchsession.NewMatchSession(matchUUID,
		map[uuid.UUID]*csSheet.CharacterSheet{a: sheetA, b: buildPlainSheet(t)},
		[]*match.Participant{pA, pB})
	src := &countingSource{face: 5}
	s.SetRollSource(src)
	return &rollsFixture{s: s, src: src, playerA: playerA, playerB: playerB, a: a, b: b, sheetA: sheetA}
}

// changeStatus é o caminho da sala numa mudança de status: troca o conjunto e o aplica ao que
// o personagem tem pendente.
func changeStatus(s *matchsession.MatchSession, charID uuid.UUID, es ...statuseffect.Effect) {
	s.ReplaceStatusEffects(charID, es)
	_, _ = s.DropRefusedByStatusWithReactions(charID)
}

func dashBy(actor uuid.UUID) *action.Action {
	return action.NewAction(actor, nil, uuid.Nil, nil, action.ActionSpeed{}, nil,
		&action.Move{Category: enum.Dash, Position: [3]int{2, 2, 0}}, nil, nil, nil, nil, nil)
}

func withSkills(a *action.Action, names ...enum.SkillName) *action.Action {
	for _, n := range names {
		a.Skills = append(a.Skills, action.Skill{SkillName: n.String()})
	}
	return a
}

func skillEntry(t *testing.T, a *action.Action, name enum.SkillName) action.Skill {
	t.Helper()
	for _, sk := range a.Skills {
		if sk.SkillName == name.String() {
			return sk
		}
	}
	t.Fatalf("no skill entry %s", name)
	return action.Skill{}
}

func slowedOf(charID uuid.UUID, penalty int) statuseffect.Effect {
	return effectOf(charID, statuseffect.KindSlowed, statuseffect.Params{Penalty: penalty})
}

func TestStatusEffectsOnSessionRolls(t *testing.T) {
	t.Run("slowed lowers queued move speed and reorders", func(t *testing.T) {
		f := newRollsFixture(t)
		first := mustEnqueue(t, f.s, f.playerA, dashBy(f.a))
		second := mustEnqueue(t, f.s, f.playerB, dashBy(f.b))
		before := first.Move.FinalSpeed
		if before != second.Move.FinalSpeed {
			t.Fatalf("speeds = %d and %d, want a tie (the earlier wins)", before, second.Move.FinalSpeed)
		}

		changeStatus(f.s, f.a, slowedOf(f.a, 3))

		if first.Move.FinalSpeed != before-3 {
			t.Errorf("FinalSpeed = %d, want %d (Lento 3)", first.Move.FinalSpeed, before-3)
		}
		if second.Move.FinalSpeed != before {
			t.Errorf("the other character's FinalSpeed = %d, want %d untouched", second.Move.FinalSpeed, before)
		}
		opened := mustOpen(t, f.s)
		if act := opened.GetAction(); act.GetID() != second.GetID() {
			t.Fatal("the slowed move must fall behind: the second dash opens first")
		}
	})

	t.Run("removing slowed restores the speed", func(t *testing.T) {
		f := newRollsFixture(t)
		a := mustEnqueue(t, f.s, f.playerA, dashBy(f.a))
		before, rolls := a.Move.FinalSpeed, f.src.rolls

		changeStatus(f.s, f.a, slowedOf(f.a, 2))
		changeStatus(f.s, f.a)

		if a.Move.FinalSpeed != before {
			t.Errorf("FinalSpeed = %d, want %d back", a.Move.FinalSpeed, before)
		}
		if f.src.rolls != rolls {
			t.Errorf("dice rolled = %d, want %d: a penalty draws nothing", f.src.rolls, rolls)
		}
	})

	t.Run("weakened lowers action speed", func(t *testing.T) {
		f := newRollsFixture(t)
		a := mustEnqueue(t, f.s, f.playerA, attackBy(f.a, f.b))
		before := a.Speed.Result

		weak := effectOf(f.a, statuseffect.KindWeakened, statuseffect.Params{Penalty: 2})
		weak.Aptitude = statuseffect.AptitudePhysical
		changeStatus(f.s, f.a, weak)

		if a.Speed.Result != before-2 {
			t.Errorf("Speed.Result = %d, want %d (Enfraquecido físico 2 pesa na Legerity)", a.Speed.Result, before-2)
		}
	})

	t.Run("vision nullified auto fails a vision skill entry", func(t *testing.T) {
		f := newRollsFixture(t)
		a := mustEnqueue(t, f.s, f.playerA, withSkills(attackBy(f.a, f.b), enum.Vision, enum.Hearing))
		hearing := skillEntry(t, a, enum.Hearing).Result
		if v := skillEntry(t, a, enum.Vision).Result; v == 0 {
			t.Fatalf("Vision entry = 0 before the status, want a derived total")
		}

		blind := effectOf(f.a, statuseffect.KindSense, statuseffect.Params{Level: statuseffect.SenseNullified})
		blind.Sense = statuseffect.SenseVision
		changeStatus(f.s, f.a, blind)

		if v := skillEntry(t, a, enum.Vision).Result; v != 0 {
			t.Errorf("Vision entry = %d, want 0 (falha automática)", v)
		}
		if h := skillEntry(t, a, enum.Hearing).Result; h != hearing {
			t.Errorf("Hearing entry = %d, want %d untouched", h, hearing)
		}
	})

	t.Run("skill entry total carries the master condition", func(t *testing.T) {
		f := newRollsFixture(t)
		a := mustEnqueue(t, f.s, f.playerA, withSkills(attackBy(f.a, f.b), enum.Vision))
		want := 10 + skillValue(t, f.sheetA, enum.Vision)
		if got := skillEntry(t, a, enum.Vision).Result; got != want {
			t.Fatalf("Vision entry = %d, want %d (5 + 5 + value)", got, want)
		}

		ma := action.NewMasterAction()
		ma.ActionID = a.GetID()
		ma.Conditions = []action.ConditionEdit{{SkillName: enum.Vision.String(), Condition: action.RollCondition{Modifier: 4}}}
		if _, err := f.s.ApplyMasterAction(ma, uuid.New()); err != nil {
			t.Fatalf("ApplyMasterAction: %v", err)
		}

		if got := skillEntry(t, a, enum.Vision).Result; got != want+4 {
			t.Errorf("Vision entry = %d, want %d (+4 do mestre)", got, want+4)
		}
	})

	t.Run("skill entry tact reads the touch face", func(t *testing.T) {
		tactOf := func(charID uuid.UUID, face statuseffect.TouchFace) statuseffect.Effect {
			e := effectOf(charID, statuseffect.KindSense, statuseffect.Params{
				Level: statuseffect.SenseImpaired, Faces: []statuseffect.TouchFace{face}, Penalty: 2,
			})
			e.Sense = statuseffect.SenseTact
			return e
		}
		f := newRollsFixture(t)
		want := 10 + skillValue(t, f.sheetA, enum.Tact)

		// Na chegada: só a antecipação afetada não alcança um Tato declarado.
		setEffects(f.s, f.a, tactOf(f.a, statuseffect.FacePreemption))
		a := mustEnqueue(t, f.s, f.playerA, withSkills(attackBy(f.a, f.b), enum.Tact))
		if got := skillEntry(t, a, enum.Tact).Result; got != want {
			t.Errorf("Tact entry under preemption = %d, want %d untouched", got, want)
		}

		changeStatus(f.s, f.a, tactOf(f.a, statuseffect.FaceTouch))
		if got := skillEntry(t, a, enum.Tact).Result; got != want-2 {
			t.Errorf("Tact entry under touch = %d, want %d (o toque, −2)", got, want-2)
		}
	})

	t.Run("prone escape move speed has disadvantage", func(t *testing.T) {
		f := newRollsFixture(t)
		mustEnqueue(t, f.s, f.playerA, attackBy(f.a, f.b))
		opened := mustOpen(t, f.s)

		escape := reactionBy(f.b, opened.ActionRef().GetID(), action.ReactEscape)
		// Os dados já caídos: 9 e 8 no movimento. O extra que a desvantagem pede cai do monte
		// (countingSource, face 5), e o par mantido é o pior: 8 + 5.
		escape.Move.Speed = &action.RollCheck{Attempts: action.RollAttempts{Primary: []int{9, 8}}}
		if _, err := f.s.AttachReaction(f.playerB, escape); err != nil {
			t.Fatalf("AttachReaction: %v", err)
		}
		live := opened.ReactionRef(escape.GetID())
		accelerate := skillValue(t, f.sheetA, enum.Accelerate)
		if live.Move.FinalSpeed != 17+accelerate {
			t.Fatalf("FinalSpeed = %d, want %d before the status", live.Move.FinalSpeed, 17+accelerate)
		}

		changeStatus(f.s, f.b, proneOf(f.b))

		live = opened.ReactionRef(escape.GetID())
		if live.Move.FinalSpeed != 13+accelerate {
			t.Errorf("FinalSpeed = %d, want %d (Caído: o pior par de 9, 8, 5)", live.Move.FinalSpeed, 13+accelerate)
		}
	})
}
