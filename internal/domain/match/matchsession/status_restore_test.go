package matchsession_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

// Devolver o que um status desfeito no turno cancelou (D3): a sessão revalida cada ação contra
// o estado de agora e só devolve o que ainda cabe — no mesmo lugar da fila, com o id, os dados e
// as edições do mestre de antes.

func rollsOf(a *action.Action) string {
	out := fmt.Sprintf("speed=%+v", a.Speed)
	if a.Attack != nil {
		out += fmt.Sprintf(" hit=%+v", a.Attack.Hit)
	}
	return out
}

func refusedIDs(rs []matchsession.RestoreRefusal) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(rs))
	for _, r := range rs {
		out = append(out, r.ActionID)
	}
	return out
}

// attackMoveBy é um ataque que anda junto: o domínio de ataque o recusa, e ele se empilha.
func attackMoveBy(actor, target uuid.UUID) *action.Action {
	a := attackBy(actor, target)
	a.Move = &action.Move{Category: enum.Dash, Position: [3]int{4, 4, 0}}
	return a
}

func TestRestoreCancelledByStatus_RestoresOriginalIDAndOrder(t *testing.T) {
	f := newDefaultFixture(t)
	bAttack := mustEnqueue(t, f.session, f.playerB, attackBy(f.b, f.c))
	cAttack := mustEnqueue(t, f.session, f.playerC, attackBy(f.c, f.b))
	bMove := mustEnqueue(t, f.session, f.playerB, moveBy(f.b))
	rolls := map[uuid.UUID]string{bAttack.GetID(): rollsOf(bAttack), bMove.GetID(): rollsOf(bMove)}

	setEffects(f.session, f.b, kindOnly(f.b, statuseffect.KindStunned))
	drops := f.session.DropRefusedByStatus(f.b)
	if seated := f.turn.SeatedFor(f.b); seated == nil || seated.ReactionKind != action.ReactNothing {
		t.Fatalf("stunned B seated = %+v, want a nothing", seated)
	}

	setEffects(f.session, f.b)
	restored, refused := f.session.RestoreCancelledByStatus(drops)
	if len(refused) != 0 {
		t.Fatalf("refused = %+v, want none", refused)
	}
	if got, want := idsOf(restored), []uuid.UUID{bAttack.GetID(), bMove.GetID()}; !slices.Equal(got, want) {
		t.Fatalf("restored = %v, want %v", got, want)
	}
	if got, want := pendingIDsOf(f.session), []uuid.UUID{bAttack.GetID(), cAttack.GetID(), bMove.GetID()}; !slices.Equal(got, want) {
		t.Fatalf("queue = %v, want the original order %v", got, want)
	}
	for _, a := range restored {
		if rollsOf(a) != rolls[a.GetID()] {
			t.Errorf("%v came back with %s, want the rolls it had: %s", a.GetID(), rollsOf(a), rolls[a.GetID()])
		}
	}
	// A reação sentada volta a seguir o conjunto de agora.
	if seated := f.turn.SeatedFor(f.b); seated == nil || seated.ReactionKind != action.ReactDefault {
		t.Fatalf("B seated after the restore = %+v, want the default", seated)
	}
}

// Empate de velocidade: quem chegou primeiro sai primeiro. A ação devolvida mantém o lugar
// contra as que chegaram antes e depois dela — inclusive as que chegaram depois do cancelamento.
func TestRestoreCancelledByStatus_KeepsItsPlaceInTies(t *testing.T) {
	f := newDefaultFixture(t)
	before := mustEnqueue(t, f.session, f.playerC, attackBy(f.c, f.b))
	bAttack := mustEnqueue(t, f.session, f.playerB, attackBy(f.b, f.c))
	afterIt := mustEnqueue(t, f.session, f.playerC, attackBy(f.c, f.b))
	if before.OrderSpeed() != bAttack.OrderSpeed() || bAttack.OrderSpeed() != afterIt.OrderSpeed() {
		t.Fatalf("setup: speeds %d/%d/%d, want a tie", before.OrderSpeed(), bAttack.OrderSpeed(), afterIt.OrderSpeed())
	}
	setEffects(f.session, f.b, kindOnly(f.b, statuseffect.KindStunned))
	drops := f.session.DropRefusedByStatus(f.b)
	afterCancel := mustEnqueue(t, f.session, f.playerC, attackBy(f.c, f.b))

	setEffects(f.session, f.b)
	if restored, refused := f.session.RestoreCancelledByStatus(drops); len(restored) != 1 || len(refused) != 0 {
		t.Fatalf("restore = %v / %+v, want the attack back", idsOf(restored), refused)
	}
	want := []uuid.UUID{before.GetID(), bAttack.GetID(), afterIt.GetID(), afterCancel.GetID()}
	if got := pendingIDsOf(f.session); !slices.Equal(got, want) {
		t.Fatalf("queue = %v, want %v", got, want)
	}
	// O desempate de verdade: fechado o turno aberto, a fila abre na ordem de chegada.
	if _, err := f.session.CloseOpenTurn(); err != nil {
		t.Fatalf("CloseOpenTurn: %v", err)
	}
	for _, id := range want[:2] {
		// O ataque de C em B fecha antes do de B: B volta ao HP cheio para o golpe não o matar
		// (o Morto tiraria o ataque dele da fila, e o desempate não seria provado).
		healToMax(t, f.session, f.b)
		if got := openedID(t, f.session); got != id {
			t.Fatalf("opened %v, want %v", got, id)
		}
	}
}

func TestRestoreCancelledByStatus_BringsBackTheMastersEdits(t *testing.T) {
	f := newDefaultFixture(t)
	bAttack := mustEnqueue(t, f.session, f.playerB, attackBy(f.b, f.c))
	for _, ma := range []*action.MasterAction{
		conditionOn(bAttack.GetID(), action.FieldHit, action.RollCondition{Modifier: 4}),
		conditionOn(bAttack.GetID(), action.FieldSpeed, action.RollCondition{Modifier: 5}),
	} {
		if _, err := f.session.ApplyMasterAction(ma, f.master); err != nil {
			t.Fatalf("ApplyMasterAction: %v", err)
		}
	}
	editedSpeed := bAttack.Speed.Result

	setEffects(f.session, f.b, kindOnly(f.b, statuseffect.KindStunned))
	drops := f.session.DropRefusedByStatus(f.b)
	if c := bAttack.Attack.Hit.Context.Condition; c != nil {
		t.Fatalf("setup: the cancel kept the hit condition %+v", c)
	}
	if bAttack.Speed.Result == editedSpeed {
		t.Fatal("setup: the cancel kept the edited speed")
	}

	setEffects(f.session, f.b)
	if restored, _ := f.session.RestoreCancelledByStatus(drops); len(restored) != 1 {
		t.Fatalf("restored = %v, want the attack", idsOf(restored))
	}
	if c := bAttack.Attack.Hit.Context.Condition; c == nil || c.Modifier != 4 {
		t.Errorf("hit condition = %+v, want the master's +4 back", c)
	}
	if bAttack.Speed.Result != editedSpeed {
		t.Errorf("speed = %d, want the edited %d back", bAttack.Speed.Result, editedSpeed)
	}
	// As capturas voltaram também: editar de volta ao original as apaga, como antes do cancelamento.
	if _, err := f.session.ApplyMasterAction(conditionOn(bAttack.GetID(), action.FieldSpeed, action.RollCondition{}), f.master); err != nil {
		t.Fatalf("ApplyMasterAction(back): %v", err)
	}
	if bAttack.Speed.Result != editedSpeed-5 {
		t.Errorf("speed after editing back = %d, want %d", bAttack.Speed.Result, editedSpeed-5)
	}
}

func TestRestoreCancelledByStatus_RefusesAlreadyQueued(t *testing.T) {
	f := newDefaultFixture(t)
	bAttack := mustEnqueue(t, f.session, f.playerB, attackBy(f.b, f.c))
	setEffects(f.session, f.b, kindOnly(f.b, statuseffect.KindStunned))
	drops := f.session.DropRefusedByStatus(f.b)
	setEffects(f.session, f.b)
	if restored, _ := f.session.RestoreCancelledByStatus(drops); len(restored) != 1 {
		t.Fatalf("first restore gave back %v, want the attack", idsOf(restored))
	}

	restored, refused := f.session.RestoreCancelledByStatus(drops)
	if len(restored) != 0 {
		t.Fatalf("second restore gave back %v, want nothing", idsOf(restored))
	}
	if got := refusedIDs(refused); !slices.Equal(got, []uuid.UUID{bAttack.GetID()}) || refused[0].Reason == "" {
		t.Fatalf("refused = %+v, want the attack with a reason", refused)
	}
	if refused[0].BlockedBy != nil {
		t.Errorf("BlockedBy = %v, want nil (not a status)", refused[0].BlockedBy)
	}
	if got := pendingIDsOf(f.session); len(got) != 1 {
		t.Fatalf("queue = %v, want the attack once", got)
	}
}

func TestRestoreCancelledByStatus_RefusesWhenAnotherStatusStillBlocks(t *testing.T) {
	f := newDefaultFixture(t)
	bAttack := mustEnqueue(t, f.session, f.playerB, attackBy(f.b, f.c))
	bMove := mustEnqueue(t, f.session, f.playerB, moveBy(f.b))
	setEffects(f.session, f.b, kindOnly(f.b, statuseffect.KindStunned), paralyzedIn(f.b, statuseffect.DomainAttack))
	drops := f.session.DropRefusedByStatus(f.b)

	// Sai o Atordoado; o Paralisado no ataque continua.
	setEffects(f.session, f.b, paralyzedIn(f.b, statuseffect.DomainAttack))
	restored, refused := f.session.RestoreCancelledByStatus(drops)
	if got := idsOf(restored); !slices.Equal(got, []uuid.UUID{bMove.GetID()}) {
		t.Fatalf("restored = %v, want only the move", got)
	}
	if len(refused) != 1 || refused[0].ActionID != bAttack.GetID() || refused[0].Drop.Action != bAttack {
		t.Fatalf("refused = %+v, want the attack and its drop", refused)
	}
	if r := refused[0].Reason; !strings.HasSuffix(r, " não pode atacar porque está paralisado") {
		t.Errorf("reason = %q, want a clause without a final period", r)
	}
	if !slices.Equal(refused[0].BlockedBy, []statuseffect.Kind{statuseffect.KindParalyzed}) {
		t.Errorf("BlockedBy = %v, want [paralyzed]", refused[0].BlockedBy)
	}
	if got := pendingIDsOf(f.session); !slices.Equal(got, []uuid.UUID{bMove.GetID()}) {
		t.Fatalf("queue = %v, want only the move", got)
	}
	// O drop recusado pode ser devolvido depois, quando o Paralisado sair.
	setEffects(f.session, f.b)
	if restored, _ := f.session.RestoreCancelledByStatus([]matchsession.StatusDrop{refused[0].Drop}); len(restored) != 1 {
		t.Fatalf("second try gave back %v, want the attack", idsOf(restored))
	}
	if got := pendingIDsOf(f.session); !slices.Equal(got, []uuid.UUID{bAttack.GetID(), bMove.GetID()}) {
		t.Fatalf("queue = %v, want the attack back in front of the move", got)
	}
}

func TestRestoreCancelledByStatus_RestoresStackedChainWhole(t *testing.T) {
	f := newDefaultFixture(t)
	first := mustEnqueue(t, f.session, f.playerB, moveBy(f.b))
	second := mustEnqueue(t, f.session, f.playerB, moveBy(f.b))
	if second.Move.StackedOn != first.GetID() {
		t.Fatalf("setup: second.StackedOn = %v, want %v", second.Move.StackedOn, first.GetID())
	}
	setEffects(f.session, f.b, kindOnly(f.b, statuseffect.KindStunned))
	drops := f.session.DropRefusedByStatus(f.b)
	setEffects(f.session, f.b)

	restored, refused := f.session.RestoreCancelledByStatus(drops)
	if len(refused) != 0 {
		t.Fatalf("refused = %+v, want none", refused)
	}
	if got := idsOf(restored); !slices.Equal(got, []uuid.UUID{first.GetID(), second.GetID()}) {
		t.Fatalf("restored = %v, want the chain in order", got)
	}
	if second.Move.StackedOn != first.GetID() {
		t.Errorf("second.StackedOn = %v, want still the first", second.Move.StackedOn)
	}
	if tail := f.session.MoveChainTail(f.b); tail == nil || tail.GetID() != second.GetID() {
		t.Errorf("chain tail = %v, want the second move", tail)
	}
}

func TestRestoreCancelledByStatus_RefusesStackedWithoutBase(t *testing.T) {
	t.Run("the base is not given back", func(t *testing.T) {
		f := newDefaultFixture(t)
		mustEnqueue(t, f.session, f.playerB, moveBy(f.b))
		second := mustEnqueue(t, f.session, f.playerB, moveBy(f.b))
		setEffects(f.session, f.b, kindOnly(f.b, statuseffect.KindStunned))
		drops := f.session.DropRefusedByStatus(f.b)
		setEffects(f.session, f.b)

		restored, refused := f.session.RestoreCancelledByStatus(drops[1:])
		if len(restored) != 0 {
			t.Fatalf("restored = %v, want nothing (half a chain)", idsOf(restored))
		}
		if len(refused) != 1 || refused[0].ActionID != second.GetID() || refused[0].Reason == "" {
			t.Fatalf("refused = %+v, want the stacked move with a reason", refused)
		}
		if got := pendingIDsOf(f.session); len(got) != 0 {
			t.Fatalf("queue = %v, want empty", got)
		}
	})

	t.Run("the chain moved on without it", func(t *testing.T) {
		f := newDefaultFixture(t)
		first := mustEnqueue(t, f.session, f.playerB, moveBy(f.b))
		second := mustEnqueue(t, f.session, f.playerB, moveBy(f.b))
		setEffects(f.session, f.b, kindOnly(f.b, statuseffect.KindStunned))
		drops := f.session.DropRefusedByStatus(f.b)
		setEffects(f.session, f.b)
		// Um movimento novo chegou no meio: é ele a cabeça da cadeia agora.
		fresh := mustEnqueue(t, f.session, f.playerB, moveBy(f.b))

		restored, refused := f.session.RestoreCancelledByStatus(drops)
		if len(restored) != 0 {
			t.Fatalf("restored = %v, want nothing (a fork in the chain)", idsOf(restored))
		}
		if got := refusedIDs(refused); !slices.Equal(got, []uuid.UUID{first.GetID(), second.GetID()}) {
			t.Fatalf("refused = %v, want both moves", got)
		}
		if got := pendingIDsOf(f.session); !slices.Equal(got, []uuid.UUID{fresh.GetID()}) {
			t.Fatalf("queue = %v, want only the fresh move", got)
		}
	})

	// A base estava na fila na saída e o jogador a cancelou depois: o empilhado não volta sozinho.
	t.Run("the base was cancelled after the drop", func(t *testing.T) {
		f := newDefaultFixture(t)
		base := mustEnqueue(t, f.session, f.playerB, moveBy(f.b))
		stacked := mustEnqueue(t, f.session, f.playerB, attackMoveBy(f.b, f.c))
		if stacked.Move.StackedOn != base.GetID() {
			t.Fatalf("setup: stacked.StackedOn = %v, want %v", stacked.Move.StackedOn, base.GetID())
		}
		setEffects(f.session, f.b, paralyzedIn(f.b, statuseffect.DomainAttack))
		drops := f.session.DropRefusedByStatus(f.b)
		if got := idsOf(matchsession.DroppedActions(drops)); !slices.Equal(got, []uuid.UUID{stacked.GetID()}) {
			t.Fatalf("dropped = %v, want only the attack that moves", got)
		}
		if _, err := f.session.CancelAction(f.playerB, base.GetID()); err != nil {
			t.Fatalf("CancelAction: %v", err)
		}
		setEffects(f.session, f.b)
		restored, refused := f.session.RestoreCancelledByStatus(drops)
		if len(restored) != 0 || len(refused) != 1 || !strings.Contains(refused[0].Reason, "não está mais na fila") {
			t.Fatalf("restore = %v / %+v, want the stacked move refused: its base left", idsOf(restored), refused)
		}
	})
}

// A base de um empilhado já tinha saído da fila (abriu) antes do status: ele era cabeça de fato,
// e volta como cabeça — a não ser que outro movimento tenha chegado no meio.
func TestRestoreCancelledByStatus_AStackedMoveWhoseBaseOpenedIsAHead(t *testing.T) {
	setup := func(t *testing.T) (*defaultFixture, *action.Action, []matchsession.StatusDrop) {
		t.Helper()
		f := newDefaultFixture(t)
		base := mustEnqueue(t, f.session, f.playerB, moveBy(f.b))
		stacked := mustEnqueue(t, f.session, f.playerB, moveBy(f.b))
		follower := mustEnqueue(t, f.session, f.playerB, moveBy(f.b))
		if _, err := f.session.CloseOpenTurn(); err != nil {
			t.Fatalf("CloseOpenTurn: %v", err)
		}
		if _, err := f.session.PullAction(base.GetID()); err != nil {
			t.Fatalf("PullAction(base): %v", err)
		}
		setEffects(f.session, f.b, kindOnly(f.b, statuseffect.KindStunned))
		drops := f.session.DropRefusedByStatus(f.b)
		if got, want := idsOf(matchsession.DroppedActions(drops)), []uuid.UUID{stacked.GetID(), follower.GetID()}; !slices.Equal(got, want) {
			t.Fatalf("dropped = %v, want %v", got, want)
		}
		setEffects(f.session, f.b)
		return f, stacked, drops
	}

	t.Run("comes back with the chain behind it", func(t *testing.T) {
		f, stacked, drops := setup(t)
		restored, refused := f.session.RestoreCancelledByStatus(drops)
		if len(refused) != 0 || len(restored) != 2 {
			t.Fatalf("restore = %v / %+v, want both back", idsOf(restored), refused)
		}
		if got := pendingIDsOf(f.session); !slices.Equal(got, idsOf(restored)) || got[0] != stacked.GetID() {
			t.Fatalf("queue = %v, want the stacked move then its follower", got)
		}
	})

	t.Run("refused when another move arrived in the meantime", func(t *testing.T) {
		f, _, drops := setup(t)
		fresh := mustEnqueue(t, f.session, f.playerB, moveBy(f.b))
		restored, refused := f.session.RestoreCancelledByStatus(drops)
		if len(restored) != 0 || len(refused) != 2 {
			t.Fatalf("restore = %v / %+v, want both refused", idsOf(restored), refused)
		}
		if got := pendingIDsOf(f.session); !slices.Equal(got, []uuid.UUID{fresh.GetID()}) {
			t.Fatalf("queue = %v, want only the fresh move", got)
		}
	})
}

// A velocidade volta como estava no instante antes do cancelamento, não derivada de novo: um
// turno que fechou entre o enqueue e a saída expirou um modificador de actionSpeed do ator, e a
// saída (discardEdits → deriveSpeeds) leu o ledger sem ele.
func TestRestoreCancelledByStatus_GivesBackTheSpeedItHadBeforeTheCancel(t *testing.T) {
	f := newDefaultFixture(t)
	status, err := f.session.GetCharacterStatus(f.b)
	if err != nil {
		t.Fatalf("GetCharacterStatus: %v", err)
	}
	status.Ledger.Add(match.Modifier{
		Amount: 5, Applies: match.DimActionSpeed, Source: match.SourceSystem,
		Against: match.ScopeAnyone(), ExpiresAt: match.LifetimeEndOfTurn, Reason: "bônus de teste",
	})
	bAttack := mustEnqueue(t, f.session, f.playerB, attackBy(f.b, f.c))
	before := rollsOf(bAttack)
	if _, err := f.session.CloseOpenTurn(); err != nil {
		t.Fatalf("CloseOpenTurn: %v", err)
	}
	if got := status.Ledger.TotalAmount(match.DimActionSpeed, nil); got != 0 {
		t.Fatalf("setup: the close left %d on the ledger, want the bonus expired", got)
	}

	setEffects(f.session, f.b, kindOnly(f.b, statuseffect.KindStunned))
	drops := f.session.DropRefusedByStatus(f.b)
	if rollsOf(bAttack) == before {
		t.Fatal("setup: the cancel did not re-derive the speed — the test would prove nothing")
	}
	setEffects(f.session, f.b)
	if restored, _ := f.session.RestoreCancelledByStatus(drops); len(restored) != 1 {
		t.Fatalf("restored = %v, want the attack", idsOf(restored))
	}
	if got := rollsOf(bAttack); got != before {
		t.Fatalf("restored with %s, want the speed it had before the cancel: %s", got, before)
	}
}

// Um movimento empilhado numa base recusada por outro status segue a base: a recusa dele leva os
// mesmos tipos (BlockedBy), e os dois voltam juntos quando esse status sai.
func TestRestoreCancelledByStatus_AFollowerOfARefusedBaseFollowsItsBlocker(t *testing.T) {
	f := newDefaultFixture(t)
	base := mustEnqueue(t, f.session, f.playerB, attackMoveBy(f.b, f.c))
	follower := mustEnqueue(t, f.session, f.playerB, moveBy(f.b))
	if follower.Move.StackedOn != base.GetID() {
		t.Fatalf("setup: follower.StackedOn = %v, want the base %v", follower.Move.StackedOn, base.GetID())
	}
	setEffects(f.session, f.b, kindOnly(f.b, statuseffect.KindStunned), paralyzedIn(f.b, statuseffect.DomainAttack))
	drops := f.session.DropRefusedByStatus(f.b)

	setEffects(f.session, f.b, paralyzedIn(f.b, statuseffect.DomainAttack))
	restored, refused := f.session.RestoreCancelledByStatus(drops)
	if len(restored) != 0 || len(refused) != 2 {
		t.Fatalf("restore = %v / %+v, want both refused", idsOf(restored), refused)
	}
	paralyzed := []statuseffect.Kind{statuseffect.KindParalyzed}
	for _, r := range refused {
		if !slices.Equal(r.BlockedBy, paralyzed) {
			t.Errorf("%v BlockedBy = %v, want %v (the follower follows its base)", r.ActionID, r.BlockedBy, paralyzed)
		}
	}

	setEffects(f.session, f.b)
	again := []matchsession.StatusDrop{refused[0].Drop, refused[1].Drop}
	if restored, refused := f.session.RestoreCancelledByStatus(again); len(restored) != 2 || len(refused) != 0 {
		t.Fatalf("second restore = %v / %+v, want both back", idsOf(restored), refused)
	}
}
