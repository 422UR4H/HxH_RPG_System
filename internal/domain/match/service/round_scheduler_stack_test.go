package service_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

// moveStackedAt é um movimento que entrou na rodada com essa velocidade, empilhado em on (D3).
func moveStackedAt(actorID uuid.UUID, speed int, on *action.Action) *action.Action {
	a := moveAt(actorID, speed)
	a.Move.StackedOn = on.GetID()
	return a
}

// D4: um movimento empilhado não é candidato enquanto o anterior está na fila.
func TestRoundScheduler_StackedMoveWaitsForItsPredecessor(t *testing.T) {
	sch := service.RoundScheduler{}

	t.Run("SelectNext never picks it before the head, even faster", func(t *testing.T) {
		p1 := uuid.New()
		head := moveAt(p1, 12)
		second := moveStackedAt(p1, 20, head)
		q := action.NewActionPriorityQueue(&[]*action.Action{head, second})
		in := service.ScheduleInput{Queue: &q, Round: round.NewRound(enum.Race), Bars: newFakeBars()}
		in.Provisional = sch.ProvisionalPrices(in)

		if got := sch.SelectNext(in); got != head {
			t.Fatalf("SelectNext = %v, want the head (12): the 20 is stacked on it", got)
		}
	})

	t.Run("ProjectRound lists it after the head, keyed with the head already acted", func(t *testing.T) {
		// Preço 9 (o movimento de p2). Sem a regra, o 30 abriria primeiro; empilhado, espera o 20
		// e então tem chave (20+30)/2 − 9 = 16 — o preço do primeiro já contado (D4).
		p1, p2, p3 := uuid.New(), uuid.New(), uuid.New()
		head := moveAt(p1, 20)
		second := moveStackedAt(p1, 30, head)
		slow := moveAt(p2, 9)
		mid := moveAt(p3, 22)
		q := action.NewActionPriorityQueue(&[]*action.Action{head, second, slow, mid})
		in := service.ScheduleInput{Queue: &q, Round: round.NewRound(enum.Race), Bars: newFakeBars()}

		proj := sch.ProjectRound(in)
		wantActors := []uuid.UUID{p3, p1, p1, p2}
		wantKeys := []float64{22, 20, 16, 9}
		if len(proj.Order) != len(wantActors) {
			t.Fatalf("Order = %d slots, want %d (Deferred %d)", len(proj.Order), len(wantActors), len(proj.Deferred))
		}
		for i := range wantActors {
			if proj.Order[i].ActorID != wantActors[i] || !closeTo(proj.Order[i].Key, wantKeys[i]) {
				t.Errorf("Order[%d] = %v key %v, want %v key %v", i, proj.Order[i].ActorID, proj.Order[i].Key, wantActors[i], wantKeys[i])
			}
		}
	})

	t.Run("a deferred head defers what is stacked on it", func(t *testing.T) {
		p1, p2 := uuid.New(), uuid.New()
		bars := newFakeBars()
		bars.setCarry(p1, action.BarMove, -20) // a cabeça não paga o preço desta rodada
		r := round.NewRound(enum.Race)
		r.FreezePrice(action.BarMove, 10)
		head := moveAt(p1, 11)
		second := moveStackedAt(p1, 40, head)
		other := moveAt(p2, 15)
		q := action.NewActionPriorityQueue(&[]*action.Action{head, second, other})
		in := service.ScheduleInput{Queue: &q, Round: r, Bars: bars}

		proj := sch.ProjectRound(in)
		if len(proj.Order) != 1 || proj.Order[0].ActorID != p2 {
			t.Fatalf("Order = %+v, want only p2", proj.Order)
		}
		if len(proj.Deferred) != 2 {
			t.Fatalf("Deferred = %d, want the head and the move stacked on it", len(proj.Deferred))
		}
		// O que fica no fim da rodada é o que a mesa veria: nada mais pode abrir.
		bars.act(other)
		q2 := action.NewActionPriorityQueue(&[]*action.Action{head, second})
		in.Queue = &q2
		if sch.AnyEligible(in) {
			t.Error("AnyEligible: the stacked 40 must not keep the round alive while its head is deferred")
		}
	})
}

// D6: a reação consome o primeiro movimento da cadeia, não o empilhado mais rápido.
func TestRoundScheduler_BestPendingForSkipsStackedMoves(t *testing.T) {
	sch := service.RoundScheduler{}
	actor := uuid.New()
	head := moveAt(actor, 8)
	second := moveStackedAt(actor, 25, head)
	q := action.NewActionPriorityQueue(&[]*action.Action{head, second})
	in := service.ScheduleInput{Queue: &q, Round: round.NewRound(enum.Race), Bars: newFakeBars()}

	if got := sch.BestPendingFor(in, actor, action.BarMove); got != head {
		t.Fatalf("BestPendingFor = %v, want the head: the faster one is stacked on it", got)
	}

	t.Run("falls back to any pending action on the bar when none is unblocked", func(t *testing.T) {
		// Um combinado (movimento+ataque) empilhado num movimento puro: na barra de ação ele é o
		// único candidato, e está bloqueado. Cai no fallback, como hoje.
		comb := chargeAt(actor, 14, 9)
		comb.Move.StackedOn = head.GetID()
		q := action.NewActionPriorityQueue(&[]*action.Action{head, comb})
		in := service.ScheduleInput{Queue: &q, Round: round.NewRound(enum.Race), Bars: newFakeBars()}
		if got := sch.BestPendingFor(in, actor, action.BarAction); got != comb {
			t.Fatalf("BestPendingFor(action) = %v, want the blocked combined action as fallback", got)
		}
	})
}
