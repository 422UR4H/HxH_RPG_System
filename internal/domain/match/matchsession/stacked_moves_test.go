package matchsession_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

// moveTo é um Dash puro até (col, row) que entrou na rodada com essa moveSpeed. Sem ficha na
// sessão, deriveSpeeds não roda e a velocidade fica a que o teste pôs.
func moveTo(actor uuid.UUID, col, row, speed int) *action.Action {
	return action.NewAction(actor, nil, uuid.Nil, nil, action.ActionSpeed{},
		nil, &action.Move{Category: enum.Dash, Position: [3]int{col, row, 0}, FinalSpeed: speed},
		nil, nil, nil, nil, nil)
}

func mustEnqueue(t *testing.T, s *matchsession.MatchSession, player uuid.UUID, a *action.Action) *action.Action {
	t.Helper()
	if err := s.EnqueueAction(player, a); err != nil {
		t.Fatalf("EnqueueAction: %v", err)
	}
	return a
}

func pendingIDsOf(s *matchsession.MatchSession) []uuid.UUID {
	var ids []uuid.UUID
	for _, a := range s.PendingActions() {
		ids = append(ids, a.GetID())
	}
	return ids
}

func idsOf(as []*action.Action) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(as))
	for _, a := range as {
		out = append(out, a.GetID())
	}
	return out
}

func openedID(t *testing.T, s *matchsession.MatchSession) uuid.UUID {
	t.Helper()
	tr := mustOpenNext(t, s)
	if tr.Opened == nil {
		t.Fatal("OpenNextAction opened nothing")
	}
	a := tr.Opened.GetAction()
	return a.GetID()
}

// D3: um movimento se empilha no último movimento pendente do personagem.
func TestStackedMoves_EnqueueChainsOnTheLastPendingMove(t *testing.T) {
	player, other := uuid.New(), uuid.New()
	s, chars := sessionWithParticipants(player, other)
	me := chars[0]

	first := moveTo(me, 2, 2, 10)
	from := [3]int{0, 0, 0}
	first.Move.From = &from // o que a Room põe: a casa da peça
	mustEnqueue(t, s, player, first)
	attack := mustEnqueue(t, s, player, makeAction(me))
	theirs := mustEnqueue(t, s, other, moveTo(chars[1], 7, 7, 10))
	second := mustEnqueue(t, s, player, moveTo(me, 4, 2, 10))
	third := mustEnqueue(t, s, player, moveTo(me, 4, 5, 10))

	if first.Move.StackedOn != uuid.Nil || first.Move.From == nil || *first.Move.From != from {
		t.Errorf("first move: StackedOn %v From %v — the head keeps the piece's origin", first.Move.StackedOn, first.Move.From)
	}
	if second.Move.StackedOn != first.GetID() {
		t.Errorf("second.StackedOn = %v, want the first move %v (the attack in between does not break the chain)", second.Move.StackedOn, first.GetID())
	}
	if second.Move.From == nil || *second.Move.From != first.Move.Position {
		t.Errorf("second.From = %v, want the first move's destination %v", second.Move.From, first.Move.Position)
	}
	if third.Move.StackedOn != second.GetID() || third.Move.From == nil || *third.Move.From != second.Move.Position {
		t.Errorf("third: StackedOn %v From %v, want stacked on the second, from %v", third.Move.StackedOn, third.Move.From, second.Move.Position)
	}
	if attack.Move != nil {
		t.Error("a non-move action never stacks")
	}
	if theirs.Move.StackedOn != uuid.Nil {
		t.Error("another character's move never stacks on mine")
	}
	// O From é uma cópia: mexer no destino do anterior não arrasta a origem do seguinte.
	first.Move.Position[0] = 99
	if second.Move.From[0] == 99 {
		t.Error("second.From aliases the first move's Position")
	}
}

// D3: um movimento só se empilha num que ainda está na fila.
func TestStackedMoves_AnOpenedMoveIsNoLongerTheTail(t *testing.T) {
	player := uuid.New()
	s, chars := sessionWithParticipants(player)
	first := mustEnqueue(t, s, player, moveTo(chars[0], 2, 2, 10))
	mustOpen(t, s)
	next := mustEnqueue(t, s, player, moveTo(chars[0], 3, 3, 10))
	if next.Move.StackedOn != uuid.Nil {
		t.Errorf("StackedOn = %v, want Nil: %v already opened", next.Move.StackedOn, first.GetID())
	}
}

// D4: o empilhado nunca abre antes do anterior; o atraso é o preço que o primeiro paga.
func TestStackedMoves_OpenInChainOrder(t *testing.T) {
	t.Run("Race: a faster stacked move opens after its head, and the projection agrees", func(t *testing.T) {
		p1, p2, p3 := uuid.New(), uuid.New(), uuid.New()
		s, chars := sessionWithParticipants(p1, p2, p3)
		s.GetActiveRound().SetMode(enum.Race)
		head := mustEnqueue(t, s, p1, moveTo(chars[0], 2, 2, 20))
		second := mustEnqueue(t, s, p1, moveTo(chars[0], 3, 3, 30))
		slow := mustEnqueue(t, s, p2, moveTo(chars[1], 5, 5, 9))
		mid := mustEnqueue(t, s, p3, moveTo(chars[2], 6, 6, 22))

		q := action.NewActionPriorityQueue(func() *[]*action.Action { p := s.PendingActions(); return &p }())
		proj := service.RoundScheduler{}.ProjectRound(service.ScheduleInput{Queue: &q, Round: s.GetActiveRound(), Bars: s})
		var projected []uuid.UUID
		for _, slot := range proj.Order {
			projected = append(projected, slot.ActorID)
		}
		if want := []uuid.UUID{chars[2], chars[0], chars[0], chars[1]}; !slices.Equal(projected, want) {
			t.Fatalf("projected actors = %v, want %v", projected, want)
		}

		var opened []uuid.UUID
		for range 4 {
			opened = append(opened, openedID(t, s))
		}
		if want := idsOf([]*action.Action{mid, head, second, slow}); !slices.Equal(opened, want) {
			t.Errorf("opened = %v, want %v (the 30 waits for the 20)", opened, want)
		}
	})

	t.Run("Free: same rule", func(t *testing.T) {
		p1, p2 := uuid.New(), uuid.New()
		s, chars := sessionWithParticipants(p1, p2)
		head := mustEnqueue(t, s, p1, moveTo(chars[0], 2, 2, 10))
		second := mustEnqueue(t, s, p1, moveTo(chars[0], 3, 3, 30))
		other := mustEnqueue(t, s, p2, moveTo(chars[1], 5, 5, 15))
		var opened []uuid.UUID
		for range 3 {
			opened = append(opened, openedID(t, s))
		}
		if want := idsOf([]*action.Action{other, head, second}); !slices.Equal(opened, want) {
			t.Errorf("opened = %v, want %v", opened, want)
		}
	})
}

// D4: puxar um movimento empilhado é recusado, e nada muda.
func TestStackedMoves_PullingAStackedMoveIsRefused(t *testing.T) {
	p1, p2 := uuid.New(), uuid.New()
	s, chars := sessionWithParticipants(p1, p2)
	s.GetActiveRound().SetMode(enum.Race)
	mustEnqueue(t, s, p2, moveTo(chars[1], 9, 9, 15))
	openTurn := mustOpen(t, s)
	head := mustEnqueue(t, s, p1, moveTo(chars[0], 2, 2, 10))
	second := mustEnqueue(t, s, p1, moveTo(chars[0], 3, 3, 30))
	before := pendingIDsOf(s)
	pricesBefore := s.RoundPrices()

	tr, err := s.PullAction(second.GetID())
	if !errors.Is(err, matchsession.ErrStackedOnPending) {
		t.Fatalf("err = %v, want ErrStackedOnPending", err)
	}
	if err.Error() != "action is stacked on a pending move" {
		t.Errorf("message = %q", err.Error())
	}
	if tr == nil || tr.Closed != nil || tr.Opened != nil {
		t.Errorf("transition = %+v, want empty: nothing closes, nothing opens", tr)
	}
	if got := pendingIDsOf(s); !slices.Equal(got, before) {
		t.Errorf("queue = %v, want unchanged %v", got, before)
	}
	if openTurn.GetFinishedAt() != nil {
		t.Error("the open turn was closed by a refused pull")
	}
	if got := s.RoundPrices(); len(got) != len(pricesBefore) {
		t.Errorf("prices = %v, want unchanged %v", got, pricesBefore)
	}
	// A cabeça pode ser puxada; aí o empilhado vira candidato normal.
	if _, err := s.PullAction(head.GetID()); err != nil {
		t.Fatalf("pulling the head: %v", err)
	}
	if _, err := s.PullAction(second.GetID()); err != nil {
		t.Fatalf("pulling the second once its head opened: %v", err)
	}
}

// D5: cancelar um movimento cancela o que está empilhado nele, em ordem de cadeia.
func TestStackedMoves_CancelTakesTheChainAfterIt(t *testing.T) {
	setup := func(t *testing.T) (*matchsession.MatchSession, uuid.UUID, []*action.Action, *action.Action) {
		player := uuid.New()
		s, chars := sessionWithParticipants(player)
		s.GetActiveRound().SetMode(enum.Race)
		a := mustEnqueue(t, s, player, moveTo(chars[0], 1, 1, 10))
		attack := mustEnqueue(t, s, player, makeAction(chars[0]))
		b := mustEnqueue(t, s, player, moveTo(chars[0], 2, 2, 10))
		c := mustEnqueue(t, s, player, moveTo(chars[0], 3, 3, 10))
		return s, player, []*action.Action{a, b, c}, attack
	}

	t.Run("cancelling the head returns all three, in chain order", func(t *testing.T) {
		s, player, chain, attack := setup(t)
		got, err := s.CancelAction(player, chain[0].GetID())
		if err != nil {
			t.Fatalf("CancelAction: %v", err)
		}
		if !slices.Equal(idsOf(got), idsOf(chain)) {
			t.Errorf("cancelled = %v, want %v", idsOf(got), idsOf(chain))
		}
		if pending := pendingIDsOf(s); !slices.Equal(pending, []uuid.UUID{attack.GetID()}) {
			t.Errorf("pending = %v, want only the attack", pending)
		}
	})

	t.Run("cancelling the middle one takes the third and keeps the first", func(t *testing.T) {
		s, player, chain, attack := setup(t)
		got, err := s.CancelAction(player, chain[1].GetID())
		if err != nil {
			t.Fatalf("CancelAction: %v", err)
		}
		if !slices.Equal(idsOf(got), idsOf(chain[1:])) {
			t.Errorf("cancelled = %v, want %v", idsOf(got), idsOf(chain[1:]))
		}
		if pending := pendingIDsOf(s); !slices.Equal(pending, []uuid.UUID{chain[0].GetID(), attack.GetID()}) {
			t.Errorf("pending = %v, want the first move and the attack", pending)
		}
	})

	t.Run("cancelling the tail takes only itself", func(t *testing.T) {
		s, player, chain, _ := setup(t)
		got, err := s.CancelAction(player, chain[2].GetID())
		if err != nil {
			t.Fatalf("CancelAction: %v", err)
		}
		if !slices.Equal(idsOf(got), idsOf(chain[2:])) {
			t.Errorf("cancelled = %v, want only the tail", idsOf(got))
		}
	})
}

// D6: a reação consome o primeiro movimento da cadeia; o resto sobrevive só se a fuga vai
// para onde aquele movimento ia.
func TestStackedMoves_AReactionSpendsTheHead(t *testing.T) {
	setup := func(t *testing.T) (*matchsession.MatchSession, uuid.UUID, uuid.UUID, uuid.UUID, *action.Action, *action.Action) {
		s, playerB, b, actID := swapSetup(t, dashAction) // a cabeça vai para (2,2)
		heads := s.PendingActions()
		head := heads[len(heads)-1]
		second := mustEnqueue(t, s, playerB, action.NewAction(b, nil, uuid.Nil, nil, action.ActionSpeed{},
			nil, &action.Move{Category: enum.Dash, Position: [3]int{3, 3, 0}}, nil, nil, nil, nil, nil))
		if second.Move.StackedOn != head.GetID() {
			t.Fatalf("setup: second not stacked on the head")
		}
		return s, playerB, b, actID, head, second
	}

	t.Run("escape to the head's destination: head consumed, the second becomes the head", func(t *testing.T) {
		s, playerB, b, actID, head, second := setup(t)
		r := escapeTo(b, actID, 2, 2)
		_, cancelled, err := s.AttachReactionWithCancels(playerB, r)
		if err != nil {
			t.Fatalf("AttachReaction: %v", err)
		}
		if !slices.Equal(r.ConsumedActionIDs, []uuid.UUID{head.GetID()}) {
			t.Errorf("ConsumedActionIDs = %v, want only the head", r.ConsumedActionIDs)
		}
		if len(cancelled) != 0 {
			t.Errorf("cancelled = %v, want none", idsOf(cancelled))
		}
		if second.Move.StackedOn != uuid.Nil {
			t.Errorf("second.StackedOn = %v, want Nil: it is the head now", second.Move.StackedOn)
		}
		if !slices.Contains(pendingIDsOf(s), second.GetID()) {
			t.Error("the second move must still be queued")
		}
	})

	t.Run("escape elsewhere: head consumed, the stacked move cancelled", func(t *testing.T) {
		s, playerB, b, actID, head, second := setup(t)
		r := escapeTo(b, actID, 5, 5)
		_, cancelled, err := s.AttachReactionWithCancels(playerB, r)
		if err != nil {
			t.Fatalf("AttachReaction: %v", err)
		}
		if !slices.Equal(r.ConsumedActionIDs, []uuid.UUID{head.GetID()}) {
			t.Errorf("ConsumedActionIDs = %v, want only the head", r.ConsumedActionIDs)
		}
		if !slices.Equal(idsOf(cancelled), []uuid.UUID{second.GetID()}) {
			t.Errorf("cancelled = %v, want the stacked move", idsOf(cancelled))
		}
		if slices.Contains(pendingIDsOf(s), second.GetID()) {
			t.Error("the stacked move must have left the queue")
		}
	})

	t.Run("an escape spends ONE move of the chain, even charging both bars", func(t *testing.T) {
		// A cabeça é um combinado (Dash + golpe): o escape a consome pela barra de ação. O
		// movimento empilhado sobrevive (mesmo destino) e não é consumido pela barra de
		// movimento da mesma reação — ela gasta um movimento da cadeia, não a cadeia.
		combined := func(actor uuid.UUID) *action.Action {
			a := dashAction(actor)
			sword := enum.Sword
			a.Attack = &action.Attack{Weapon: &sword}
			a.TargetID = []uuid.UUID{uuid.New()}
			return a
		}
		s, playerB, b, actID := swapSetup(t, combined)
		heads := s.PendingActions()
		head := heads[len(heads)-1]
		second := mustEnqueue(t, s, playerB, action.NewAction(b, nil, uuid.Nil, nil, action.ActionSpeed{},
			nil, &action.Move{Category: enum.Dash, Position: [3]int{3, 3, 0}}, nil, nil, nil, nil, nil))
		r := escapeTo(b, actID, 2, 2)
		_, cancelled, err := s.AttachReactionWithCancels(playerB, r)
		if err != nil {
			t.Fatalf("AttachReaction: %v", err)
		}
		if !slices.Equal(r.ConsumedActionIDs, []uuid.UUID{head.GetID()}) {
			t.Errorf("ConsumedActionIDs = %v, want only the combined head", r.ConsumedActionIDs)
		}
		if len(cancelled) != 0 || !slices.Contains(pendingIDsOf(s), second.GetID()) || second.Move.StackedOn != uuid.Nil {
			t.Errorf("the stacked move must survive as the new head (cancelled %v, StackedOn %v)", idsOf(cancelled), second.Move.StackedOn)
		}
	})
}
