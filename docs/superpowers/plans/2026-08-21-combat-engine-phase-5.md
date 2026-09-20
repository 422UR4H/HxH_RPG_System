# Combat Engine — Phase 5 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development
> (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** The table works. The master ends a turn on purpose instead of as a side effect,
edits the action when the scene calls for it, and every person in the room receives the
version of the turn that is theirs — with a history that is readable afterwards.

**Architecture:** Three seams, and nothing invents a fourth. (1) Turn close stops being a side
effect of opening: `close_turn` is its own message, and it routes through the *same*
`closeOpenTurn` that already resolves, applies damage and advances the ledgers — the private
path becomes the only path. (2) Visibility becomes a pure domain projection, `service.Project*`,
applied through `Room.dispatchPerPlayer`, the mechanism the fog of war already uses. (3) The
master's edit lands **on the action itself** — there is no parallel version to merge — and what
it displaced is captured in memory, keyed by field, and written in the transaction that already
closes the turn.

**Tech Stack:** Go 1.23, standard `testing` (table-driven, `t.Run`, external test packages),
goose migrations on PostgreSQL, huma v2 + chi for REST, gorilla-style WebSocket delivery in
`internal/app/game/`.

**Spec:** `docs/superpowers/specs/2026-08-21-combat-engine-phase-5.md` — self-contained and
binding. The **rules** it points at live in `docs/dev/match/combat-engine.md` §§ *A edição do
mestre*, *Os fluxos, e quais confirmações existem de verdade*, *A política: público por
omissão, deny-list explícita, mesa inteira*, *O caminho de escrita estava quebrado desde a Fase
2*, *Quantos dados cada reação consome*. Player-facing narrative:
`docs/game/combate/o-mestre.md`, `docs/game/combate/reacoes.md`.

**Branch:** `feat/combat-engine-phase-5`, from `main`.

> **`git pull` first.** Phases 1–4 and the write-path fix (PR #69) are on `main`.
> `actions.actor_uuid` already points at `character_sheets`, reactions are already persisted,
> and `reaction_kind` / `repel` already have columns. Do not re-do any of it.

---

## Global Constraints

- **Go 1.23**, module `github.com/422UR4H/HxH_RPG_System`. No test frameworks — standard
  `testing` only, table-driven with `t.Run()`, external test packages (`package foo_test`).
- **NEVER remove a TODO comment.** They are intentional markers left by the repo owner. That
  includes `buildMasterAction`'s two — `// TODO: map Move fully once frontend contract is
  finalized` and `// TODO: map Attack once frontend contract is finalized` — which stay exactly
  where they are. This phase does **not** finish them (see Task 8).
- **Layering:** `entity ← domain ← app`, `entity ← gateway`. `service` imports `action`;
  **`action` must never import `service`**, and never `match` (the `Modifier` package).
- **Domain services are stateless structs.** `service.TurnResolver{}`, `service.RoundScheduler{}`,
  `service.BarEconomy{}`, `service.RoundOrchestrator{}` keep working as zero values. New
  projection functions are **free functions**, not methods on state.
- **`MatchSession` has no lock of its own.** `room.go`'s `r.mu` is the only serialization. Every
  new call into the session holds it **write-locked** whenever the session mutates — which now
  includes closing a turn explicitly and applying a master edit.
- **Wire format is camelCase** on both sides, via manual struct tags.
- **The master never re-rolls a player's die.** `Derive` derives; only `Roll` rolls, and only
  for a test that did not exist before.
- **A test of 2D10 consumes FOUR faces, not two.** `RollCalculator.Roll` always rolls both
  `Primary` and `Secondary`, even with no advantage. Only weapon damage rolls a single set.
  Every scripted `RollSource` in this plan is budgeted at 4 faces per match-set test.
- **The edit changes the outcome, never the economy.** Charged bars, recorded `Speeds` and the
  order already played are never redone.
- **`PersistTurnClose` swallows its error by design** (`room.go` logs and carries on — the turn
  already closed in memory and the table cannot stop). A test that wants to prove persistence
  must look at the database, never at the operation's return value.
- Commits include `Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>`.
- After every task touching `internal/`: `go vet -tags=integration ./internal/...` **and**
  `go vet -tags=smoke ./internal/...` — the build-tagged files are invisible to a plain `go vet`.

---

## The rules, restated

Read this section once before Task 1. Every task below assumes it.

### The three recipient classes

| Class | Sees |
|---|---|
| **Master** | everything |
| **Owner** of the action or reaction | everything that is theirs |
| **Everyone else** | everything minus the deny-list |

**The target is not privileged.** A feint against you does not tell you it was a feint.

### The deny-list

| Hidden from third parties | Why |
|---|---|
| HP | the damage is public, the HP is not |
| `Feint` | a revealed feint is not a feint |
| `Trigger` | same, until it fires |
| the `Evasion` entry of a closed dodge, and the reserve it generates | the opponent deduces |
| **the `ReactionKind` itself, on the closed variants** | the label is the leak |

A closed dodge reaches a third party **indistinguishable from a plain dodge** (`dodge`); a
closed escape, from an escape (`escape`). Deducing from the public bar is legitimate; being
told is not.

### The time gate, and why it is separate from the class

`combat-engine.md` § *Visibilidade* says it twice: *"o cálculo é só do mestre até o turno
encerrar"*. So the projection has two axes, not one:

- **Turn still open** (`TurnResolution.IsSettled == false`) → **master only**, exactly as
  today. Nothing changes for players while the master is still deliberating.
- **Turn settled** (`IsSettled == true`) → projected per recipient, by the three classes above.

`IsSettled` already exists on `TurnResolution` and is already set from
`Turn.GetFinishedAt() != nil`. Do not add a second flag.

### The master's edit

- **The edited action IS the action.** No parallel version, no merge on read. `RollCondition`
  already lives in `RollContext`, inside `RollCheck`, inside `Action`.
- **Adding a skill rolls new dice** — not a re-roll, the first roll of a test that did not
  exist.
- **Removing a skill keeps its dice in memory** while the turn is open, so taking it out and
  putting it back is not a free re-roll.
- **There is no confirmation verb.** Passing the baton is the confirmation. `close_turn`'s
  refusal is a different thing entirely — it is about someone losing the moment to narrate.
- **The audit stores the value the edit displaced**, one row per field, and reverting to the
  original **erases the capture**.

### What this phase does not prove, and must say so in the PR

`match_session.go` rolls every `Skill` of an action and **nobody reads the result**. The chain
of tests (margin crossing from one test to the next, missing by 10+ killing the chain) is
written in `docs/game/combate/acoes.md` and **does not exist in code**. This phase delivers the
edit surface and the audit; adding or removing a skill changes the list, and the list does not
decide anything yet. Say that in the PR — do not pretend it decides.

---

## File Structure

**Created:**

| File | Responsibility |
|---|---|
| `internal/domain/match/service/projection.go` | `Viewer`, `ProjectResolution`, `ProjectAction` — the deny-list as pure functions |
| `internal/domain/match/service/projection_test.go` | the deny-list, rung by rung |
| `internal/domain/match/override.go` | `OverriddenValue`, `OverrideOrigin` — what an edit displaced |
| `internal/domain/match/override_test.go` | equality/revert semantics |
| `internal/application/match/close_turn.go` | `CloseTurnUC` — explicit close, with the confirmation gate |
| `internal/application/match/close_turn_test.go` | refusal, acceptance, damage applied |
| `internal/application/match/edit_action.go` | `EditActionUC` — applies a `MasterAction` onto the open turn |
| `internal/application/match/edit_action_test.go` | conditions, skills, targets, revert |
| `internal/application/match/get_match_history.go` | `GetMatchHistoryUC` — read + authorize |
| `internal/application/match/get_match_history_test.go` | authorization and shape |
| `migrations/20260822000000_turn_resolution.sql` | `turns.resolution JSONB` |
| `migrations/20260822000001_overridden_action_values.sql` | the audit table |
| `internal/gateway/pg/round/resolution_record.go` | the persisted shape of a settled resolution, and its round trip |
| `internal/gateway/pg/round/find_match_history.go` | the nested read query |
| `internal/app/api/match/get_match_history.go` | REST handler + response DTOs |
| `internal/app/api/match/get_match_history_test.go` | humatest coverage |
| `internal/app/game/visibility_e2e_test.go` | the two-client done-criterion |
| `docs/dev/api/match-history.md` | the REST contract (source of truth for the front) |

**Modified:**

| File | Change |
|---|---|
| `internal/domain/match/entity/turn/turn.go` | `UnopenedReactions()` |
| `internal/domain/match/matchsession/match_session.go` | `CloseOpenTurn()` replaces the broken `CloseTurn()`; override capture; `ApplyMasterAction` |
| `internal/domain/match/entity/action/master_action.go` | `ActionID`, `Conditions` |
| `internal/domain/match/modifier.go` | `Scope.Kind()` / `Scope.ID()` accessors |
| `internal/application/match/i_repository.go` | `PersistTurnClose` takes a struct; `FindMatchHistory` |
| `internal/gateway/pg/round/persist_turn_close.go` | the struct signature, the resolution, the overrides |
| `internal/app/game/message.go` | `close_turn`, `turn_closed`, `close_turn_refused`, `action_queued`, projected `resolution_updated` |
| `internal/app/game/room.go` | the new routes; `dispatchPerPlayer` for resolutions |
| `internal/app/game/handler.go`, `hub.go`, `cmd/game/main.go` | wiring for two new use cases |
| `internal/app/api/match/routes.go` | `GET /matches/{uuid}/history` |
| `docs/dev/match/combat-engine.md` | § *O que a Fase 5 fixou no motor* |
| `docs/documentation-map.yaml` | the new code paths |

---

## Task 1: The turn closes on purpose — the session seam

The private `closeOpenTurn` is the only correct close: it resolves, applies damage, and
advances the ledgers. The public `CloseTurn()` skips all three. This task makes the correct one
public and deletes the trap.

**Files:**
- Modify: `internal/domain/match/entity/turn/turn.go`
- Modify: `internal/domain/match/matchsession/match_session.go:590-592`
- Test: `internal/domain/match/entity/turn/turn_test.go`
- Test: `internal/domain/match/matchsession/match_session_test.go`

**Interfaces:**
- Produces: `(*turn.Turn).UnopenedReactions() []action.Action`;
  `(*matchsession.MatchSession).CloseOpenTurn() (*TurnTransition, error)`.
- Consumes: nothing new.

- [ ] **Step 1: Write the failing test for `UnopenedReactions`**

In `internal/domain/match/entity/turn/turn_test.go`:

```go
func TestUnopenedReactions(t *testing.T) {
	newReaction := func(actor uuid.UUID) *action.Action {
		a := action.NewAction(actor, nil, uuid.New(), nil, action.ActionSpeed{},
			nil, nil, nil, nil, nil, nil, nil)
		a.ReactionKind = action.ReactDodge
		return a
	}

	t.Run("every attached reaction is unopened before the master opens any", func(t *testing.T) {
		tn := turn.NewTurn(*action.NewAction(uuid.New(), nil, uuid.Nil, nil,
			action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil))
		r1, r2 := newReaction(uuid.New()), newReaction(uuid.New())
		tn.AddReaction(r1)
		tn.AddReaction(r2)

		if got := len(tn.UnopenedReactions()); got != 2 {
			t.Fatalf("UnopenedReactions() = %d, want 2", got)
		}
	})

	t.Run("opening one removes it, and only it", func(t *testing.T) {
		tn := turn.NewTurn(*action.NewAction(uuid.New(), nil, uuid.Nil, nil,
			action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil))
		r1, r2 := newReaction(uuid.New()), newReaction(uuid.New())
		tn.AddReaction(r1)
		tn.AddReaction(r2)
		tn.OpenReaction(r1.GetID())

		left := tn.UnopenedReactions()
		if len(left) != 1 {
			t.Fatalf("UnopenedReactions() = %d, want 1", len(left))
		}
		if left[0].GetID() != r2.GetID() {
			t.Fatalf("the wrong reaction stayed unopened: %v", left[0].GetID())
		}
	})
}
```

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./internal/domain/match/entity/turn/ -run TestUnopenedReactions -v`
Expected: FAIL — `tn.UnopenedReactions undefined`.

- [ ] **Step 3: Implement `UnopenedReactions`**

In `internal/domain/match/entity/turn/turn.go`, after `OpenedReactionIDs`:

```go
// UnopenedReactions is every attached reaction the master has not yet given the floor to.
//
// These are exactly the ones close_turn warns about: they ARE in the calculation — the chain
// walks the opened ones first and then everyone left over — so nobody is punished
// mechanically. What they lose is the moment to narrate, and that is what the master is being
// asked to confirm away.
func (t *Turn) UnopenedReactions() []action.Action {
	out := make([]action.Action, 0, len(t.reactions))
	for i := range t.reactions {
		if !slices.Contains(t.openedReactions, t.reactions[i].GetID()) {
			out = append(out, t.reactions[i])
		}
	}
	return out
}
```

- [ ] **Step 4: Run it and watch it pass**

Run: `go test ./internal/domain/match/entity/turn/ -run TestUnopenedReactions -v`
Expected: PASS.

- [ ] **Step 5: Write the failing test for `CloseOpenTurn`**

In `internal/domain/match/matchsession/match_session_test.go`. This is the one that catches the
silent failure: the old `CloseTurn` returned a turn with damage never applied.

```go
func TestCloseOpenTurnAppliesDamage(t *testing.T) {
	t.Run("an explicit close applies the damage the implicit one would have", func(t *testing.T) {
		f := newAttackFixture(t) // existing helper: attacker, victim, one enqueued attack
		before := f.victimHealth()

		if _, err := f.session.OpenNextAction(); err != nil {
			t.Fatalf("OpenNextAction: %v", err)
		}
		if f.victimHealth() != before {
			t.Fatalf("HP moved while the turn was open: %d → %d", before, f.victimHealth())
		}

		tr, err := f.session.CloseOpenTurn()
		if err != nil {
			t.Fatalf("CloseOpenTurn: %v", err)
		}
		if tr.Closed == nil {
			t.Fatal("CloseOpenTurn returned no closed turn")
		}
		if tr.ClosedResolution == nil {
			t.Fatal("CloseOpenTurn returned no settled resolution — it skipped closeOpenTurn")
		}
		if len(tr.Damaged) == 0 {
			t.Fatal("CloseOpenTurn applied no damage — it skipped closeOpenTurn")
		}
		if f.victimHealth() >= before {
			t.Fatalf("HP did not move on close: %d → %d", before, f.victimHealth())
		}
	})

	t.Run("closing with no open turn is an error, not a silent no-op", func(t *testing.T) {
		f := newAttackFixture(t)
		if _, err := f.session.CloseOpenTurn(); !errors.Is(err, matchsession.ErrNoOpenTurn) {
			t.Fatalf("CloseOpenTurn with nothing open = %v, want ErrNoOpenTurn", err)
		}
	})
}
```

> If `newAttackFixture`/`victimHealth` do not exist in this test file under those names, build
> them from `internal/app/game/combat_e2e_test.go`'s `newCombatFixture` — a session with two
> sheets, `SetRollSource(topFaceSource{})`, and one enqueued attack. Do not weaken the
> assertions to fit a fixture.

- [ ] **Step 6: Run it and watch it fail**

Run: `go test ./internal/domain/match/matchsession/ -run TestCloseOpenTurn -v`
Expected: FAIL — `CloseOpenTurn undefined`, `ErrNoOpenTurn undefined`.

- [ ] **Step 7: Replace `CloseTurn` with `CloseOpenTurn`**

In `internal/domain/match/matchsession/error.go`, add:

```go
// ErrNoOpenTurn is close_turn with nothing under the baton. It is an error rather than a
// no-op because the master pressed a button that describes an action they believe is
// happening; answering silently would leave them believing it happened.
var ErrNoOpenTurn = errors.New("no open turn to close")
```

In `internal/domain/match/matchsession/match_session.go`, **delete** the old method:

```go
func (s *MatchSession) CloseTurn() (*turn.Turn, error) {
	return s.roundOrch.CloseTurnErr(s.activeRound, time.Now())
}
```

and put in its place:

```go
// CloseOpenTurn ends the turn under the baton explicitly, and it is the SAME path the two
// baton operations take: it resolves one last time, applies what that resolution says, and
// advances the ledgers.
//
// It exists because the old CloseTurn() did none of those three. It called
// roundOrch.CloseTurnErr directly, so the turn ended, the damage evaporated and nothing
// reported an error — the worst shape a bug can have. There is now one way to close a turn,
// and this is it.
func (s *MatchSession) CloseOpenTurn() (*TurnTransition, error) {
	if !s.activeRound.HasOpenTurn() {
		return nil, ErrNoOpenTurn
	}
	return s.closeOpenTurn(), nil
}

// UnopenedReactions is the open turn's attached-but-not-opened reactions, for the confirmation
// gate in CloseTurnUC. Empty when there is no open turn — the caller's error for that case is
// ErrNoOpenTurn, raised by CloseOpenTurn, not by this reader.
func (s *MatchSession) UnopenedReactions() []action.Action {
	t := s.activeRound.CurrentTurn()
	if t == nil || t.GetFinishedAt() != nil {
		return nil
	}
	return t.UnopenedReactions()
}
```

- [ ] **Step 8: Fix every caller of the deleted method**

Run: `rtk proxy grep -rn '\.CloseTurn()' --include=*.go internal/ cmd/`
Expected: no production caller (the spec says nothing calls it). Update any test that does.
If `roundOrch.CloseTurnErr` now has no caller at all, **leave it** — it is
`RoundOrchestrator`'s API, not dead code this task owns.

- [ ] **Step 9: Run the tests**

Run: `go test ./internal/domain/... && go vet -tags=integration ./internal/...`
Expected: PASS.

- [ ] **Step 10: Commit**

```bash
git add internal/domain/match/entity/turn/ internal/domain/match/matchsession/
git commit -m "feat(match): one way to close a turn, and it applies the damage

MatchSession.CloseTurn skipped closeOpenTurn: it ended the turn without
resolving it, without applying damage and without advancing the ledgers,
and reported no error while doing it. Replaced by CloseOpenTurn, which
routes through the same private path both baton operations already take.

Turn.UnopenedReactions is the list close_turn will warn about.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 2: `close_turn` — the use case, the message, the refusal

**Files:**
- Create: `internal/application/match/close_turn.go`
- Create: `internal/application/match/close_turn_test.go`
- Modify: `internal/app/game/message.go`, `room.go`, `handler.go`, `hub.go`, `cmd/game/main.go`
- Test: `internal/app/game/handler_test.go`

**Interfaces:**
- Consumes: `MatchSession.CloseOpenTurn`, `MatchSession.UnopenedReactions` (Task 1).
- Produces:
  ```go
  type CloseTurnResult struct {
      ClosedTurn *turn.Turn
      Resolution *service.TurnResolution
      Damaged    []matchsession.DamagedCharacter
      // Refused is the unopened reactions that blocked the close. Non-empty means nothing
      // was closed and the master must resend with Confirm.
      Refused []action.Action
  }
  type ICloseTurn interface {
      Execute(ctx context.Context, session *matchsession.MatchSession,
          masterUUID, callerUUID uuid.UUID, confirm bool) (*CloseTurnResult, error)
  }
  ```

- [ ] **Step 1: Write the failing use-case test**

`internal/application/match/close_turn_test.go`:

```go
func TestCloseTurnUC(t *testing.T) {
	t.Run("refuses when a reaction was attached and never opened", func(t *testing.T) {
		f := newTurnFixture(t)          // session with one open turn
		f.attachReaction(t)             // one reaction, not opened

		uc := match.NewCloseTurnUC(&noopStatusWriter{})
		res, err := uc.Execute(context.Background(), f.session, f.masterUUID, f.masterUUID, false)
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if len(res.Refused) != 1 {
			t.Fatalf("Refused = %d, want 1", len(res.Refused))
		}
		if res.ClosedTurn != nil {
			t.Fatal("the turn closed despite the refusal")
		}
	})

	t.Run("closes with confirm, and the unopened reaction still counts", func(t *testing.T) {
		f := newTurnFixture(t)
		f.attachReaction(t)

		uc := match.NewCloseTurnUC(&noopStatusWriter{})
		res, err := uc.Execute(context.Background(), f.session, f.masterUUID, f.masterUUID, true)
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if res.ClosedTurn == nil {
			t.Fatal("confirm:true did not close the turn")
		}
		if res.Resolution == nil || len(res.Resolution.CharacterResults) == 0 {
			t.Fatal("the unopened reaction's target produced no character result")
		}
	})

	t.Run("closes without confirm when nothing is pending", func(t *testing.T) {
		f := newTurnFixture(t)
		uc := match.NewCloseTurnUC(&noopStatusWriter{})
		res, err := uc.Execute(context.Background(), f.session, f.masterUUID, f.masterUUID, false)
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if res.ClosedTurn == nil {
			t.Fatal("a clean close was refused")
		}
	})

	t.Run("only the master may close", func(t *testing.T) {
		f := newTurnFixture(t)
		uc := match.NewCloseTurnUC(&noopStatusWriter{})
		_, err := uc.Execute(context.Background(), f.session, f.masterUUID, uuid.New(), true)
		if !errors.Is(err, match.ErrNotMatchMaster) {
			t.Fatalf("err = %v, want ErrNotMatchMaster", err)
		}
	})
}
```

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./internal/application/match/ -run TestCloseTurnUC -v`
Expected: FAIL — `NewCloseTurnUC undefined`.

- [ ] **Step 3: Implement the use case**

`internal/application/match/close_turn.go`:

```go
package match

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

type CloseTurnResult struct {
	ClosedTurn *turn.Turn
	Resolution *service.TurnResolution
	Damaged    []matchsession.DamagedCharacter
	// Refused is the unopened reactions that blocked the close. Non-empty means nothing was
	// closed: the master has to send it again with Confirm.
	Refused []action.Action
}

type ICloseTurn interface {
	Execute(ctx context.Context, session *matchsession.MatchSession,
		masterUUID, callerUUID uuid.UUID, confirm bool) (*CloseTurnResult, error)
}

type CloseTurnUC struct {
	statusWriter ISheetStatusWriter
}

func NewCloseTurnUC(statusWriter ISheetStatusWriter) *CloseTurnUC {
	return &CloseTurnUC{statusWriter: statusWriter}
}

// Execute ends the open turn on purpose.
//
// The confirmation is the SERVER's, not the front's: refusing here is what makes the
// criterion verifiable without a browser. What is being confirmed away is not the
// calculation — an unopened reaction is in the chain either way — it is the moment to narrate.
//
// Closing a turn does NOT close the round. Exhaustion stays detected in exactly one place,
// OpenNextActionUC, where the scheduling happens. Two detection points is how two versions of
// one rule are born.
func (uc *CloseTurnUC) Execute(
	ctx context.Context,
	session *matchsession.MatchSession,
	masterUUID, callerUUID uuid.UUID,
	confirm bool,
) (*CloseTurnResult, error) {
	if callerUUID != masterUUID {
		return nil, ErrNotMatchMaster
	}
	if !confirm {
		if pending := session.UnopenedReactions(); len(pending) > 0 {
			return &CloseTurnResult{Refused: pending}, nil
		}
	}
	tr, err := session.CloseOpenTurn()
	if err != nil {
		return nil, err
	}
	// Same policy as OpenNextActionUC: persist before anything can bail out. The damage is
	// already applied in memory and the turn is already closed.
	persistDamage(ctx, uc.statusWriter, tr.Damaged)
	return &CloseTurnResult{
		ClosedTurn: tr.Closed,
		Resolution: tr.ClosedResolution,
		Damaged:    tr.Damaged,
	}, nil
}

var _ ICloseTurn = (*CloseTurnUC)(nil)
```

- [ ] **Step 4: Run it and watch it pass**

Run: `go test ./internal/application/match/ -run TestCloseTurnUC -v`
Expected: PASS.

- [ ] **Step 5: Add the wire types**

In `internal/app/game/message.go`, in the client-to-server block:

```go
	MsgTypeCloseTurn MessageType = "close_turn"
```

in the server-to-client block:

```go
	MsgTypeTurnClosed        MessageType = "turn_closed"
	MsgTypeCloseTurnRefused  MessageType = "close_turn_refused"
```

and the payloads:

```go
// CloseTurnPayload ends the open turn. Master only.
//
// Confirm is the master answering the refusal below. It is not a general "yes": a clean close
// never needs it, so a client that always sends true is not cheating anything — there is
// nothing to confirm when nothing is pending.
type CloseTurnPayload struct {
	Confirm bool `json:"confirm,omitempty"`
}

// TurnClosedPayload announces that the baton was put down. BROADCAST — that a turn ended is
// table state. The numbers travel separately, in the projected resolution_updated that
// follows.
type TurnClosedPayload struct {
	TurnID uuid.UUID `json:"turnId"`
}

// CloseTurnRefusedPayload is the confirmation dialog's content, computed by the server.
//
// Master-only, because it names reactions the table has not been shown yet. The front draws
// the dialog from this; the back is what decides the dialog is needed, which is why the
// criterion is verifiable with no front at all.
type CloseTurnRefusedPayload struct {
	TurnID           uuid.UUID                `json:"turnId"`
	PendingReactions []PendingReactionPayload `json:"pendingReactions"`
}
```

- [ ] **Step 6: Route it in `room.go`**

Add the field `closeTurnUC ICloseTurn` to `Room` and to `NewRoom`'s parameters (after
`openReactionUC`, keeping the existing order stable), and the same to `Handler`/`NewHandler`
and `Hub`'s `NewRoom` call in `hub.go`, and to `cmd/game/main.go`. Declare the interface
alongside the others in `room.go`:

```go
type ICloseTurn = appmatch.ICloseTurn
```

Then the case, next to `MsgTypeOpenReaction`:

```go
	case MsgTypeCloseTurn:
		if !r.IsMaster(client.userUUID) {
			client.SendMessage(NewErrorMessage("forbidden", ErrNotMaster.Error()))
			return
		}
		var payload CloseTurnPayload
		if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
			client.SendMessage(NewErrorMessage("invalid_payload", "invalid close_turn payload"))
			return
		}
		// Write lock across Execute — closing resolves the turn, applies damage to the target
		// sheets and advances every ledger. Exactly the same surface open_next_action mutates.
		r.mu.Lock()
		session := r.session
		var result *appmatch.CloseTurnResult
		var err error
		var turnID uuid.UUID
		if session != nil {
			turnID = currentTurnID(session)
			result, err = r.closeTurnUC.Execute(
				context.Background(), session, r.masterUUID, client.userUUID, payload.Confirm)
		}
		r.mu.Unlock()
		if session == nil {
			client.SendMessage(NewErrorMessage("match_not_started", "match session not initialized"))
			return
		}
		if err != nil {
			client.SendMessage(NewErrorMessage("game_error", err.Error()))
			return
		}
		if len(result.Refused) > 0 {
			pending := make([]PendingReactionPayload, 0, len(result.Refused))
			for i := range result.Refused {
				pending = append(pending, PendingReactionPayload{
					ReactionID: result.Refused[i].GetID(),
					ActorID:    result.Refused[i].GetActorID(),
					Kind:       string(result.Refused[i].ReactionKind),
				})
			}
			client.SendMessage(NewServerMessage(MsgTypeCloseTurnRefused,
				CloseTurnRefusedPayload{TurnID: turnID, PendingReactions: pending}))
			return
		}

		closedTurn := result.ClosedTurn
		closedAct := closedTurn.GetAction()
		r.mu.RLock()
		activeScene := session.GetActiveScene()
		activeRound := session.GetActiveRound()
		matchUUID := session.GetMatchUUID()
		r.mu.RUnlock()
		if err2 := r.roundRepo.PersistTurnClose(context.Background(), appmatch.TurnCloseData{
			Scene: activeScene, Round: activeRound, Turn: closedTurn,
			Action: &closedAct, MatchUUID: matchUUID, Resolution: result.Resolution,
		}); err2 != nil {
			log.Printf("PersistTurnClose error: %v", err2)
		} else {
			r.mu.Lock()
			session.MarkRoundPersisted()
			r.mu.Unlock()
		}

		out := NewServerMessage(MsgTypeTurnClosed, TurnClosedPayload{TurnID: closedTurn.GetID()})
		data, _ := json.Marshal(out)
		go func() { r.broadcast <- data }()
		r.publishResolution(closedTurn.GetID(), result.Resolution)
		r.broadcastBars(session)
```

> `TurnCloseData` and `publishResolution` do not exist yet — Tasks 6 and 4 build them. Until
> then, call `PersistTurnClose` positionally and `r.sendToMaster(NewServerMessage(
> MsgTypeResolutionUpdate, newResolutionUpdatedPayload(...)))`, and come back. Note the
> temporary shape in the commit message so the next task's author knows to finish it.

- [ ] **Step 7: Write the delivery test**

In `internal/app/game/handler_test.go`, add a case driving `close_turn` over the real socket:
a non-master gets `forbidden`; a master with one unopened reaction gets `close_turn_refused`
carrying that reaction's ID; the same master re-sending with `{"confirm":true}` gets
`turn_closed`.

- [ ] **Step 8: Run everything**

Run: `go test ./internal/... && go vet -tags=integration ./internal/... && go vet -tags=smoke ./internal/...`
Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add internal/application/match/close_turn.go internal/application/match/close_turn_test.go \
        internal/app/game/ cmd/game/main.go
git commit -m "feat(match): close_turn, and a refusal the server computes

The turn stops closing as a side effect of opening the next one. The
confirmation gate is the back's: close_turn refuses while a reaction is
attached and unopened, naming them, and accepts with confirm:true. The
reaction was in the calculation either way — what it loses is the moment
to narrate.

Closing a turn does not close the round: exhaustion stays detected in
OpenNextActionUC alone.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 3: The deny-list as a pure function

The rule is a game rule, so it lives in the domain. The wire layer projects an already-filtered
resolution and knows nothing about what is secret.

**Files:**
- Create: `internal/domain/match/service/projection.go`
- Create: `internal/domain/match/service/projection_test.go`

**Interfaces:**
- Produces:
  ```go
  type Viewer struct {
      IsMaster bool
      // Owns is the set of character sheet UUIDs this viewer controls. A master's is
      // irrelevant — IsMaster short-circuits everything.
      Owns map[uuid.UUID]bool
  }
  func (v Viewer) SeesAllOf(charID uuid.UUID) bool
  func ProjectResolution(res *TurnResolution, v Viewer) *TurnResolution
  func ProjectAction(a action.Action, v Viewer) action.Action
  ```

- [ ] **Step 1: Write the failing tests**

`internal/domain/match/service/projection_test.go`:

```go
func TestProjectResolution(t *testing.T) {
	owner, third := uuid.New(), uuid.New()

	base := func() *service.TurnResolution {
		return &service.TurnResolution{
			IsSettled:    true,
			ActionResult: service.RollResult{SkillName: "Legerity", Total: 18, DiceRolled: []int{9, 9}},
			CharacterResults: []service.CharacterResult{{
				TargetID:        owner,
				ReactionKind:    string(action.ReactClosedDodge),
				ReactionTotal:   15,
				ProjectedDamage: 0,
				RawDamage:       7,
				EffectiveDamage: 0,
				Payouts: []match.Modifier{{
					Amount: 4, Applies: match.DimDodge, Source: match.SourceSystem,
					Against: match.ScopeAllBut(uuid.New()), ExpiresAt: match.LifetimeEndOfRound,
					Reason: "closed dodge reserve",
				}},
			}},
			PendingReactions: []service.PendingReaction{{ReactionID: uuid.New(), ActorID: owner, Kind: "dodge"}},
		}
	}

	t.Run("the master sees the closed dodge as a closed dodge", func(t *testing.T) {
		got := service.ProjectResolution(base(), service.Viewer{IsMaster: true})
		if got.CharacterResults[0].ReactionKind != string(action.ReactClosedDodge) {
			t.Fatalf("ReactionKind = %q, want closedDodge", got.CharacterResults[0].ReactionKind)
		}
		if len(got.CharacterResults[0].Payouts) != 1 {
			t.Fatal("the master lost the reserve")
		}
		if len(got.PendingReactions) != 1 {
			t.Fatal("the master lost their own to-do list")
		}
	})

	t.Run("the owner sees their own closed dodge", func(t *testing.T) {
		v := service.Viewer{Owns: map[uuid.UUID]bool{owner: true}}
		got := service.ProjectResolution(base(), v)
		if got.CharacterResults[0].ReactionKind != string(action.ReactClosedDodge) {
			t.Fatalf("the owner was lied to about their own reaction: %q",
				got.CharacterResults[0].ReactionKind)
		}
		if len(got.CharacterResults[0].Payouts) != 1 {
			t.Fatal("the owner lost their own reserve")
		}
	})

	t.Run("a third party sees a plain dodge, and no reserve", func(t *testing.T) {
		v := service.Viewer{Owns: map[uuid.UUID]bool{third: true}}
		got := service.ProjectResolution(base(), v)
		if got.CharacterResults[0].ReactionKind != string(action.ReactDodge) {
			t.Fatalf("ReactionKind = %q, want dodge — the label is the leak",
				got.CharacterResults[0].ReactionKind)
		}
		if len(got.CharacterResults[0].Payouts) != 0 {
			t.Fatal("a third party can see the closed dodge's reserve")
		}
		if len(got.PendingReactions) != 0 {
			t.Fatal("a third party can read the master's to-do list")
		}
	})

	t.Run("the numbers stay public — deduction needs them", func(t *testing.T) {
		v := service.Viewer{Owns: map[uuid.UUID]bool{third: true}}
		got := service.ProjectResolution(base(), v)
		if got.ActionResult.Total != 18 || len(got.ActionResult.DiceRolled) != 2 {
			t.Fatal("the attack's numbers were hidden; the opponent has nothing to deduce from")
		}
		if got.CharacterResults[0].RawDamage != 7 {
			t.Fatal("damage is public")
		}
	})

	t.Run("closed escape reaches a third party as escape", func(t *testing.T) {
		res := base()
		res.CharacterResults[0].ReactionKind = string(action.ReactClosedEscape)
		v := service.Viewer{Owns: map[uuid.UUID]bool{third: true}}
		got := service.ProjectResolution(res, v)
		if got.CharacterResults[0].ReactionKind != string(action.ReactEscape) {
			t.Fatalf("ReactionKind = %q, want escape", got.CharacterResults[0].ReactionKind)
		}
	})

	t.Run("projecting does not mutate the original", func(t *testing.T) {
		res := base()
		v := service.Viewer{Owns: map[uuid.UUID]bool{third: true}}
		_ = service.ProjectResolution(res, v)
		if res.CharacterResults[0].ReactionKind != string(action.ReactClosedDodge) {
			t.Fatal("ProjectResolution mutated its input — the master's copy is now a lie")
		}
	})
}

func TestProjectAction(t *testing.T) {
	owner, third := uuid.New(), uuid.New()
	mk := func() action.Action {
		a := action.NewAction(owner, nil, uuid.New(),
			[]action.Skill{
				{SkillName: enum.Evasion.String()},
				{SkillName: enum.Legerity.String()},
			},
			action.ActionSpeed{}, &action.RollCheck{SkillName: enum.Legerity.String()},
			nil, nil, nil, &action.Dodge{}, nil, nil)
		a.ReactionKind = action.ReactClosedDodge
		return *a
	}

	t.Run("a third party sees neither the feint nor the evasion entry", func(t *testing.T) {
		got := service.ProjectAction(mk(), service.Viewer{Owns: map[uuid.UUID]bool{third: true}})
		if got.Feint != nil {
			t.Fatal("a revealed feint is not a feint")
		}
		for _, s := range got.Skills {
			if s.SkillName == enum.Evasion.String() {
				t.Fatal("the evasion entry leaked")
			}
		}
		if got.ReactionKind != action.ReactDodge {
			t.Fatalf("ReactionKind = %q, want dodge", got.ReactionKind)
		}
	})

	t.Run("the owner keeps everything", func(t *testing.T) {
		got := service.ProjectAction(mk(), service.Viewer{Owns: map[uuid.UUID]bool{owner: true}})
		if got.Feint == nil || len(got.Skills) != 2 || got.ReactionKind != action.ReactClosedDodge {
			t.Fatal("the owner was projected away from their own action")
		}
	})
}
```

- [ ] **Step 2: Run and watch it fail**

Run: `go test ./internal/domain/match/service/ -run 'TestProject' -v`
Expected: FAIL — `service.Viewer undefined`.

- [ ] **Step 3: Implement the projection**

`internal/domain/match/service/projection.go`:

```go
package service

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/google/uuid"
)

// Viewer is one recipient's standing, and there are exactly THREE classes, not four: the
// master, the owner of a given action or reaction, and everyone else.
//
// The target is deliberately NOT a class. A feint against you does not tell you it was a
// feint — privileging the target would delete the feint from the game.
type Viewer struct {
	IsMaster bool
	// Owns is the set of character sheet UUIDs this viewer controls.
	Owns map[uuid.UUID]bool
}

// SeesAllOf reports whether this viewer is entitled to the unprojected truth about one
// character's action or reaction.
func (v Viewer) SeesAllOf(charID uuid.UUID) bool {
	return v.IsMaster || v.Owns[charID]
}

// publicKind demotes the closed variants to the open ones they must be indistinguishable
// from.
//
// The LABEL is the leak. If closedDodge arrives public, nobody has to deduce anything — the
// name already said there was an Evasion folded in. Deducing from the public bar is
// legitimate (a closed escape charges one bar where the standard charges two, and
// bars_updated is public); being told is not.
func publicKind(k string) string {
	switch action.ReactionKind(k) {
	case action.ReactClosedDodge:
		return string(action.ReactDodge)
	case action.ReactClosedEscape:
		return string(action.ReactEscape)
	default:
		return k
	}
}

// ProjectResolution returns the copy of res this viewer is entitled to.
//
// Public by omission: the numbers travel, because "the opponent has to deduce from the
// numbers" is impossible without them. What is withheld is a closed list.
//
// It NEVER mutates res: the master holds the same pointer, and a projection that edited in
// place would turn their copy into a lie one recipient at a time.
func ProjectResolution(res *TurnResolution, v Viewer) *TurnResolution {
	if res == nil {
		return nil
	}
	out := *res
	out.CharacterResults = make([]CharacterResult, 0, len(res.CharacterResults))
	for _, cr := range res.CharacterResults {
		if !v.SeesAllOf(cr.TargetID) {
			cr.ReactionKind = publicKind(cr.ReactionKind)
			// The closed dodge's reserve is the other half of the same secret: the size of
			// the dodge that was not spent says how much Evasion was folded in.
			cr.Payouts = nil
		}
		out.CharacterResults = append(out.CharacterResults, cr)
	}
	// PendingReactions is the master's own to-do list — attached, not yet given the floor.
	// Nobody else is owed the knowledge that an answer is waiting.
	if !v.IsMaster {
		out.PendingReactions = nil
	}
	// Blows carry no numbers (see battle.Blow) and are not projected; ReactionResults are
	// rolls already reflected in CharacterResults and travel as-is.
	return &out
}

// ProjectAction returns the copy of an action this viewer is entitled to. Used by the Action
// History, where the same policy applies to the DECLARATION rather than to the arithmetic.
func ProjectAction(a action.Action, v Viewer) action.Action {
	if v.SeesAllOf(a.GetActorID()) {
		return a
	}
	out := a
	// A revealed feint is not a feint; a revealed trigger is a trigger nobody can be caught by.
	out.Feint = nil
	out.Trigger = nil
	out.ReactionKind = action.ReactionKind(publicKind(string(a.ReactionKind)))
	if len(a.Skills) > 0 {
		kept := make([]action.Skill, 0, len(a.Skills))
		for _, s := range a.Skills {
			if s.SkillName == enum.Evasion.String() {
				continue
			}
			kept = append(kept, s)
		}
		out.Skills = kept
	}
	return out
}
```

- [ ] **Step 4: Run and watch it pass**

Run: `go test ./internal/domain/match/service/ -run 'TestProject' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/match/service/projection.go internal/domain/match/service/projection_test.go
git commit -m "feat(match): the deny-list, as a pure projection

Three recipient classes, not four — the target is not privileged, or a
feint against you would announce itself. Public by omission: the numbers
travel, because deducing from them is the game. What is withheld is
closed: the reserve of a closed dodge, the master's pending list, and the
LABEL itself — closedDodge reaches a third party as dodge.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 4: `resolution_updated` stops being master-only

**Files:**
- Modify: `internal/app/game/room.go` (all four emit sites + a new helper)
- Modify: `internal/app/game/message.go`
- Create: `internal/app/game/visibility_e2e_test.go`

**Interfaces:**
- Consumes: `service.ProjectResolution`, `service.Viewer` (Task 3);
  `MatchSession.GetCharToPlayer() map[string]uuid.UUID`.
- Produces: `(*Room).publishResolution(turnID uuid.UUID, res *service.TurnResolution)`.

- [ ] **Step 1: Write the failing e2e test — the phase's headline criterion**

`internal/app/game/visibility_e2e_test.go`. Build the fixture on
`reaction_chain_e2e_test.go`'s (three characters, three players) — the shape needed is: an
attacker owned by player A, a defender owned by player B who answers with a **closed dodge**,
and a bystander owned by player C.

```go
// TestE2E_AClosedDodgeReachesAThirdPartyAsAPlainDodge is the phase's headline criterion:
// two clients connected as different players receive DIFFERENT payloads for the SAME turn.
//
// The dice are scripted. Budget four faces per 2D10 test — RollCalculator.Roll always rolls
// Primary AND Secondary — plus two for the sword's damage set.
func TestE2E_AClosedDodgeReachesAThirdPartyAsAPlainDodge(t *testing.T) {
	f := newVisibilityFixture(t)
	master, defenderConn, bystanderConn := f.connect(t)
	defenderMsgs := newCollector(defenderConn)
	bystanderMsgs := newCollector(bystanderConn)

	f.enqueueAttack(t, f.attackerConn)
	sendWS(t, master, "open_next_action", struct{}{})
	if !newCollector(master).await(game.MsgTypeTurnOpened, 2*time.Second) {
		t.Fatal("the turn never opened")
	}

	f.attachClosedDodge(t, defenderConn)
	sendWS(t, master, "close_turn", map[string]any{"confirm": true})

	if !defenderMsgs.await(game.MsgTypeResolutionUpdate, 2*time.Second) {
		t.Fatal("the defender never received the settled resolution")
	}
	if !bystanderMsgs.await(game.MsgTypeResolutionUpdate, 2*time.Second) {
		t.Fatal("the bystander never received the settled resolution")
	}

	defKind := kindOfFirstTarget(t, defenderMsgs)
	bysKind := kindOfFirstTarget(t, bystanderMsgs)

	if defKind != "closedDodge" {
		t.Fatalf("the owner sees %q, want closedDodge — they were lied to about their own reaction", defKind)
	}
	if bysKind != "dodge" {
		t.Fatalf("the bystander sees %q, want dodge — the label is the leak", bysKind)
	}
	if defKind == bysKind {
		t.Fatal("both clients received the same payload; nothing was projected")
	}
}

// TestE2E_TheCalculationIsTheMastersUntilTheTurnCloses guards the time axis: while the turn
// is open, no player receives a resolution at all.
func TestE2E_TheCalculationIsTheMastersUntilTheTurnCloses(t *testing.T) {
	f := newVisibilityFixture(t)
	master, defenderConn, _ := f.connect(t)
	defenderMsgs := newCollector(defenderConn)
	masterMsgs := newCollector(master)

	f.enqueueAttack(t, f.attackerConn)
	sendWS(t, master, "open_next_action", struct{}{})

	if !masterMsgs.await(game.MsgTypeResolutionUpdate, 2*time.Second) {
		t.Fatal("the master did not receive the open turn's projection")
	}
	// Deliberately not a timeout-based negative: the collector is drained in the background,
	// so a blocking read is never needed and the connection stays usable.
	time.Sleep(300 * time.Millisecond)
	if n := defenderMsgs.count(game.MsgTypeResolutionUpdate); n != 0 {
		t.Fatalf("a player received %d resolution_updated while the turn was still open", n)
	}
}
```

Add the small reader:

```go
func kindOfFirstTarget(t *testing.T, c *collector) string {
	t.Helper()
	for _, m := range c.snapshotMessages() {
		if m.Type != game.MsgTypeResolutionUpdate {
			continue
		}
		var p game.ResolutionUpdatedPayload
		if err := json.Unmarshal(m.Payload, &p); err != nil {
			t.Fatalf("unmarshal resolution_updated: %v", err)
		}
		if !p.IsSettled || len(p.Targets) == 0 || p.Targets[0].Reaction == nil {
			continue
		}
		return p.Targets[0].Reaction.Kind
	}
	t.Fatal("no settled resolution with a reaction arrived")
	return ""
}
```

- [ ] **Step 2: Run and watch it fail**

Run: `go test ./internal/app/game/ -run TestE2E_AClosedDodge -v`
Expected: FAIL — the bystander receives nothing (still master-only).

- [ ] **Step 3: Implement `publishResolution`**

In `internal/app/game/room.go`, next to `broadcastBars`:

```go
// publishResolution sends one turn's resolution to everyone entitled to a version of it.
//
// TWO axes, not one:
//
//   - TIME: while the turn is open the calculation belongs to the master alone
//     (combat-engine.md § Visibilidade). Only a SETTLED resolution reaches the table.
//   - CLASS: master / owner / everyone else, applied by service.ProjectResolution.
//
// It reuses dispatchPerPlayer, which is the mechanism the fog of war already uses for exactly
// this. Do not grow a second one.
func (r *Room) publishResolution(turnID uuid.UUID, res *service.TurnResolution) {
	if res == nil {
		return
	}
	if !res.IsSettled {
		r.sendToMaster(NewServerMessage(
			MsgTypeResolutionUpdate, newResolutionUpdatedPayload(turnID, res)))
		return
	}
	r.mu.RLock()
	charToPlayer := map[string]uuid.UUID{}
	if r.session != nil {
		charToPlayer = r.session.GetCharToPlayer()
	}
	r.mu.RUnlock()

	owned := make(map[uuid.UUID]map[uuid.UUID]bool, len(charToPlayer))
	for charStr, playerID := range charToPlayer {
		charID, err := uuid.Parse(charStr)
		if err != nil {
			continue
		}
		if owned[playerID] == nil {
			owned[playerID] = map[uuid.UUID]bool{}
		}
		owned[playerID][charID] = true
	}

	r.dispatchPerPlayer(func(playerID uuid.UUID, isMaster bool) *Message {
		v := service.Viewer{IsMaster: isMaster, Owns: owned[playerID]}
		msg := NewServerMessage(
			MsgTypeResolutionUpdate,
			newResolutionUpdatedPayload(turnID, service.ProjectResolution(res, v)),
		)
		return &msg
	})
}
```

- [ ] **Step 4: Replace every emit site**

There are four. Swap each `r.sendToMaster(NewServerMessage(MsgTypeResolutionUpdate,
newResolutionUpdatedPayload(X, Y)))` for `r.publishResolution(X, Y)`:

- `MsgTypeOpenNextAction`, the closed resolution (`room.go` ~517)
- `MsgTypeOpenNextAction`, the opened projection (~554)
- `MsgTypePullAction`, both of the same (~645, ~668)
- `MsgTypeOpenReaction` (~810)
- `MsgTypeCloseTurn` (Task 2's temporary line)

The open-turn ones stay master-only automatically — `IsSettled` is false for them, and
`publishResolution` short-circuits. **Do not add an `if` at the call site**; one place decides.

Also update `ResolutionUpdatedPayload`'s doc comment: it is no longer master-only, and
`PendingReactions` is the field that still is.

- [ ] **Step 5: Run the tests**

Run: `go test ./internal/app/game/ -run 'TestE2E' -race -v`
Expected: PASS, including the pre-existing combat and reaction-chain e2e tests. If
`combat_e2e_test.go`'s "the player receives no resolution_updated at all" assertion now fails,
that is **correct and expected** — it encoded the old rule. Update it to assert the new one:
no resolution while the turn is open, a projected one after it closes.

- [ ] **Step 6: Commit**

```bash
git add internal/app/game/
git commit -m "feat(match): resolution_updated is projected per recipient

Two axes. Time: while the turn is open the calculation is the master's,
so only a settled resolution reaches the table. Class: master, owner,
everyone else — via dispatchPerPlayer, the same mechanism the fog already
uses, and via the domain's deny-list.

combat_e2e's 'the player receives nothing' assertion encoded the old rule
and now asserts the new one.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 5: The queued action is the master's news — and it carries the ID

`pull_action` takes an `actionId` the master has no way to learn. Same hole `PendingReactions`
closed for `open_reaction`; same fix.

**Files:**
- Modify: `internal/app/game/message.go`, `internal/app/game/room.go`
- Test: `internal/app/game/handler_test.go`

**Interfaces:**
- Produces: `MsgTypeActionQueued` = `"action_queued"` with `ActionQueuedPayload`.

- [ ] **Step 1: Write the failing test**

In `internal/app/game/handler_test.go`:

```go
func TestActionQueuedReachesOnlyTheMaster(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	masterMsgs := newCollector(master)
	playerMsgs := newCollector(player)

	f.enqueueAttack(t, player)

	if !masterMsgs.await(game.MsgTypeActionQueued, 2*time.Second) {
		t.Fatal("the master was never told an action was queued")
	}
	var p game.ActionQueuedPayload
	for _, m := range masterMsgs.snapshotMessages() {
		if m.Type == game.MsgTypeActionQueued {
			if err := json.Unmarshal(m.Payload, &p); err != nil {
				t.Fatalf("unmarshal action_queued: %v", err)
			}
		}
	}
	if p.ActionID == uuid.Nil {
		t.Fatal("action_queued carried no action ID — pull_action stays unreachable")
	}
	if p.ActorID != f.attackerID {
		t.Fatalf("ActorID = %v, want the attacking character %v", p.ActorID, f.attackerID)
	}
	if n := playerMsgs.count(game.MsgTypeActionQueued); n != 0 {
		t.Fatalf("a player received %d action_queued; the queue is secret", n)
	}

	// And the ID is usable: pull_action with it opens that exact turn.
	sendWS(t, master, "pull_action", map[string]any{"actionId": p.ActionID})
	if !masterMsgs.await(game.MsgTypeTurnOpened, 2*time.Second) {
		t.Fatal("pull_action with the advertised ID opened nothing")
	}
}
```

- [ ] **Step 2: Run and watch it fail**

Run: `go test ./internal/app/game/ -run TestActionQueuedReachesOnlyTheMaster -v`
Expected: FAIL — `game.ActionQueuedPayload undefined`.

- [ ] **Step 3: Add the message**

In `internal/app/game/message.go`:

```go
	MsgTypeActionQueued MessageType = "action_queued"
```

```go
// ActionQueuedPayload tells the MASTER that something landed in the queue, and names it.
//
// Master-only, and that is the whole point: the queue is secret (combat-engine.md § As barras
// são públicas — "a fila é secreta; a barra e a ordem são públicas"), so a player learning
// what is pending would read the table's intentions off the wire.
//
// ActionID is not decoration. pull_action takes an action ID, and until this payload existed
// the master had no legitimate way to learn one — the same hole PendingReactions closed for
// open_reaction, with the same consequence: an ID a client cannot learn is an operation a
// client cannot invoke.
//
// Nothing here describes the action's CONTENT. Weapon, target, skill and dice stay the
// player's until the master opens the turn.
type ActionQueuedPayload struct {
	ActionID uuid.UUID `json:"actionId"`
	ActorID  uuid.UUID `json:"actorId"`
	// Bars is which clocks it will charge — the master's scheduling surface, and already
	// derivable from the public bars_updated order. Nothing new is disclosed by naming it here.
	Bars []string `json:"bars"`
}
```

- [ ] **Step 4: Emit it**

In `room.go`, in `MsgTypeEnqueueAction`, after the existing ack
`client.SendMessage(NewServerMessage(MsgTypeActionEnqueued, struct{}{}))`:

```go
		// The sender's own ack stays as it is — it says "we got it", to the person who sent it.
		// This is different news, for a different recipient: the master is the one who has to
		// decide when it opens, and they need the ID to be able to pull it.
		bars := make([]string, 0, 2)
		for _, b := range a.Bars() {
			bars = append(bars, string(b))
		}
		r.sendToMaster(NewServerMessage(MsgTypeActionQueued, ActionQueuedPayload{
			ActionID: a.GetID(), ActorID: a.GetActorID(), Bars: bars,
		}))
		r.broadcastBars(session)
```

- [ ] **Step 5: Run the tests**

Run: `go test ./internal/app/game/ -race`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/app/game/
git commit -m "feat(match): action_queued, master-only, and pull_action becomes reachable

pull_action takes an actionId nobody could learn. Same hole
PendingReactions closed for open_reaction, same fix: the ID travels in a
payload the master receives. The queue stays secret — the payload names
who and which bars, never the weapon, target or dice.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 6: The settled resolution is persisted

The declaration is stored; the collision is not. Recomputing later is impossible — the
`ModifierLedger` of that instant is gone.

**Files:**
- Create: `migrations/20260822000000_turn_resolution.sql`
- Create: `internal/gateway/pg/round/resolution_record.go`
- Modify: `internal/domain/match/modifier.go` (Scope accessors)
- Modify: `internal/application/match/i_repository.go`
- Modify: `internal/gateway/pg/round/persist_turn_close.go`
- Modify: `internal/app/game/room.go` (three `PersistTurnClose` call sites)
- Test: `internal/gateway/pg/round/round_integration_test.go`

**Interfaces:**
- Produces:
  ```go
  // internal/application/match
  type TurnCloseData struct {
      Scene      *sceneentity.Scene
      Round      *roundentity.Round
      Turn       *turnentity.Turn
      Action     *action.Action
      MatchUUID  uuid.UUID
      Resolution *service.TurnResolution // settled; nil is allowed and writes SQL NULL
      Overrides  []match.OverriddenValue // Task 9 fills this; nil until then
  }
  PersistTurnClose(ctx context.Context, d TurnCloseData) error
  ```
  ```go
  // internal/domain/match
  func (s Scope) Kind() string
  func (s Scope) ID() uuid.UUID
  ```

- [ ] **Step 1: Write the migration**

`migrations/20260822000000_turn_resolution.sql`:

```sql
-- +goose Up
-- +goose StatementBegin
BEGIN;

-- The declaration of a turn was persisted; the COLLISION never was. Margin, damage, ladder
-- rung and chain state existed only in memory, and recomputing them afterwards is impossible:
-- the ModifierLedger of that instant is gone with the process.
--
-- JSONB rather than tables because this is the snapshot of a calculation, not a queryable
-- entity — nobody is going to ask "every turn with damage over 10" in the MVP — and because
-- it is one more write in a transaction that already exists.
--
-- The resolution written here is the SETTLED one, the one whose damage was applied. The
-- master's edits recompute the projection many times; only the close is persisted.
ALTER TABLE turns ADD COLUMN IF NOT EXISTS resolution JSONB;

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

ALTER TABLE turns DROP COLUMN IF EXISTS resolution;

COMMIT;
-- +goose StatementEnd
```

- [ ] **Step 2: Add the `Scope` accessors and their test**

`match.Scope` keeps `kind` and `id` private, so a `Modifier` cannot round-trip through JSON —
and `CharacterResult.Payouts` is a `[]match.Modifier`. Add to
`internal/domain/match/modifier.go`, after the three constructors:

```go
// Kind and ID expose a Scope for serialization only. They are read-only on purpose: the
// three constructors stay the only way to BUILD one, so an unset ID can still never be
// mistaken for "anyone".
func (s Scope) Kind() string  { return string(s.kind) }
func (s Scope) ID() uuid.UUID { return s.id }

// ScopeFrom rebuilds a Scope read back from storage. It is the inverse of Kind/ID and
// nothing else should call it: an unknown kind reads as "anyone", which is the safe answer
// for a modifier whose scope was lost.
func ScopeFrom(kind string, id uuid.UUID) Scope {
	switch ScopeKind(kind) {
	case scopeOnly:
		return ScopeOnly(id)
	case scopeAllBut:
		return ScopeAllBut(id)
	default:
		return ScopeAnyone()
	}
}
```

Add a round-trip test in `internal/domain/match/modifier_test.go` covering all three forms.

- [ ] **Step 3: Write the failing integration test**

In `internal/gateway/pg/round/round_integration_test.go` (build tag `integration`):

```go
func TestPersistTurnCloseWritesTheSettledResolution(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.NewPool(t)
	repo := round.NewRepository(pool)
	fx := seedMatchAndSheets(t, pool) // existing helper style: a match, a master, two sheets

	act := buildAttackAction(t, fx.attackerSheet, fx.victimSheet)
	tn := turnentity.NewTurn(*act)
	tn.Close(time.Now())

	res := &service.TurnResolution{
		IsSettled:    true,
		ActionResult: service.RollResult{SkillName: "Legerity", Total: 19, DiceRolled: []int{10, 9}},
		CharacterResults: []service.CharacterResult{{
			TargetID: fx.victimSheet, RawDamage: 11, DefenseApplied: 3, EffectiveDamage: 8,
			ReactionKind: string(action.ReactRepel),
			Ladder:       service.LadderOutcome{Rung: service.RungNearMiss, Margin: -4, Difference: 4},
			Payouts: []match.Modifier{{
				Amount: -4, Applies: match.DimActionSpeed, Source: match.SourceSystem,
				Against: match.ScopeAnyone(), ExpiresAt: match.LifetimeEndOfRound,
				Reason: "parry penalty",
			}},
		}},
	}

	err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
		Scene: fx.scene, Round: fx.round, Turn: tn, Action: act,
		MatchUUID: fx.matchUUID, Resolution: res,
	})
	if err != nil {
		t.Fatalf("PersistTurnClose: %v", err)
	}

	var raw []byte
	if err := pool.QueryRow(ctx,
		`SELECT resolution FROM turns WHERE uuid = $1`, tn.GetID()).Scan(&raw); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("turns.resolution is NULL — the collision was not persisted")
	}

	got := round.DecodeResolution(raw)
	if got == nil || len(got.CharacterResults) != 1 {
		t.Fatalf("round trip lost the character results: %+v", got)
	}
	cr := got.CharacterResults[0]
	if cr.EffectiveDamage != 8 || cr.RawDamage != 11 {
		t.Fatalf("damage did not survive: raw=%d effective=%d", cr.RawDamage, cr.EffectiveDamage)
	}
	if cr.Ladder.Rung != service.RungNearMiss || cr.Ladder.Difference != 4 {
		t.Fatalf("the ladder did not survive: %+v", cr.Ladder)
	}
	if len(cr.Payouts) != 1 || cr.Payouts[0].Against.Kind() != match.ScopeAnyone().Kind() {
		t.Fatalf("the payout's scope did not survive: %+v", cr.Payouts)
	}
}

func TestPersistTurnCloseAcceptsANilResolution(t *testing.T) {
	// A turn with nothing resolvable still closes. NULL, not an error, and not a zero-value
	// record that would read back as "a collision that produced nothing".
	// ... same shape, Resolution: nil, assert the column IS NULL.
}
```

- [ ] **Step 4: Run and watch it fail**

Run: `make migrate-up && go test -tags=integration ./internal/gateway/pg/round/ -run TestPersistTurnCloseWrites -v`
Expected: FAIL — `TurnCloseData` and `DecodeResolution` undefined.

- [ ] **Step 5: Implement the record**

`internal/gateway/pg/round/resolution_record.go`:

```go
package round

import (
	"encoding/json"
	"log"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

// The persisted shape of a settled resolution.
//
// It is an EXPLICIT record rather than json.Marshal on service.TurnResolution, for two
// reasons that are not style:
//
//   - battle.Blow has none of its fields exported, so it would serialize as {} — a silent
//     hole. It is also derived and carries no numbers (see its own doc), so it is dropped
//     here on purpose rather than half-written.
//   - match.Scope keeps kind and id private, so a Payout's Against would be lost the same
//     silent way. It travels through Scope.Kind()/ID() and comes back through ScopeFrom.
//
// Tags are camelCase, like every other wire shape in this repo.
type resolutionRecord struct {
	IsSettled    bool                    `json:"isSettled"`
	ActionResult rollResultRecord        `json:"actionResult"`
	Characters   []characterResultRecord `json:"characters"`
}

type rollResultRecord struct {
	SkillName         string `json:"skillName"`
	SkillValue        int    `json:"skillValue"`
	DiceRolled        []int  `json:"diceRolled"`
	Total             int    `json:"total"`
	IsCritical        bool   `json:"isCritical"`
	IsCriticalFailure bool   `json:"isCriticalFailure"`
	Margin            *int   `json:"margin,omitempty"`
}

type characterResultRecord struct {
	TargetID            uuid.UUID        `json:"targetId"`
	ReactionID          uuid.UUID        `json:"reactionId,omitempty"`
	ReactionKind        string           `json:"reactionKind,omitempty"`
	ReactionTotal       int              `json:"reactionTotal,omitempty"`
	ReactionStopsAttack bool             `json:"reactionStopsAttack,omitempty"`
	AttackStopped       bool             `json:"attackStopped,omitempty"`
	Avoided             bool             `json:"avoided"`
	Defended            bool             `json:"defended"`
	Ladder              ladderRecord     `json:"ladder"`
	DamageDice          []int            `json:"damageDice,omitempty"`
	RawDamage           int              `json:"rawDamage"`
	DefenseApplied      int              `json:"defenseApplied"`
	EffectiveDamage     int              `json:"effectiveDamage"`
	Payouts             []modifierRecord `json:"payouts,omitempty"`
}

type ladderRecord struct {
	Rung       string `json:"rung,omitempty"`
	Margin     int    `json:"margin,omitempty"`
	Difference int    `json:"difference,omitempty"`
}

type modifierRecord struct {
	Amount      int       `json:"amount"`
	Bias        int       `json:"bias"`
	Applies     string    `json:"applies"`
	Source      string    `json:"source"`
	AgainstKind string    `json:"againstKind"`
	AgainstID   uuid.UUID `json:"againstId,omitempty"`
	ExpiresAt   string    `json:"expiresAt"`
	Reason      string    `json:"reason,omitempty"`
}

// encodeResolution returns nil (SQL NULL) for a nil resolution. A turn that resolved nothing
// stores nothing — a zero-value record would read back as "a collision that produced zero",
// which is a different claim.
func encodeResolution(res *service.TurnResolution) ([]byte, error) {
	if res == nil {
		return nil, nil
	}
	rec := resolutionRecord{
		IsSettled:    res.IsSettled,
		ActionResult: rollResultRecord(res.ActionResult),
		Characters:   make([]characterResultRecord, 0, len(res.CharacterResults)),
	}
	for _, cr := range res.CharacterResults {
		out := characterResultRecord{
			TargetID: cr.TargetID, ReactionID: cr.ReactionID,
			ReactionKind: cr.ReactionKind, ReactionTotal: cr.ReactionTotal,
			ReactionStopsAttack: cr.ReactionStopsAttack, AttackStopped: cr.AttackStopped,
			Avoided: cr.Avoided, Defended: cr.Defended,
			Ladder: ladderRecord{
				Rung: string(cr.Ladder.Rung), Margin: cr.Ladder.Margin,
				Difference: cr.Ladder.Difference,
			},
			DamageDice: cr.DamageDice, RawDamage: cr.RawDamage,
			DefenseApplied: cr.DefenseApplied, EffectiveDamage: cr.EffectiveDamage,
		}
		for _, m := range cr.Payouts {
			out.Payouts = append(out.Payouts, modifierRecord{
				Amount: m.Amount, Bias: m.Bias, Applies: string(m.Applies),
				Source: string(m.Source), AgainstKind: m.Against.Kind(),
				AgainstID: m.Against.ID(), ExpiresAt: string(m.ExpiresAt), Reason: m.Reason,
			})
		}
		rec.Characters = append(rec.Characters, out)
	}
	return json.Marshal(rec)
}

// DecodeResolution rebuilds a settled resolution read back from turns.resolution. Exported
// because the history read path in this package and its tests both need it.
//
// A row that will not decode returns nil and logs: a turn whose stored collision is
// unreadable is still a turn that happened, and the history must show the declaration rather
// than fail the whole match's history over one bad row.
func DecodeResolution(raw []byte) *service.TurnResolution {
	if len(raw) == 0 {
		return nil
	}
	var rec resolutionRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		log.Printf("DecodeResolution: %v", err)
		return nil
	}
	out := &service.TurnResolution{
		IsSettled:        rec.IsSettled,
		ActionResult:     service.RollResult(rec.ActionResult),
		CharacterResults: make([]service.CharacterResult, 0, len(rec.Characters)),
	}
	for _, c := range rec.Characters {
		cr := service.CharacterResult{
			TargetID: c.TargetID, ReactionID: c.ReactionID,
			ReactionKind: c.ReactionKind, ReactionTotal: c.ReactionTotal,
			ReactionStopsAttack: c.ReactionStopsAttack, AttackStopped: c.AttackStopped,
			Avoided: c.Avoided, Defended: c.Defended,
			Ladder: service.LadderOutcome{
				Rung: service.LadderRung(c.Ladder.Rung), Margin: c.Ladder.Margin,
				Difference: c.Ladder.Difference,
			},
			DamageDice: c.DamageDice, RawDamage: c.RawDamage,
			DefenseApplied: c.DefenseApplied, EffectiveDamage: c.EffectiveDamage,
		}
		for _, m := range c.Payouts {
			cr.Payouts = append(cr.Payouts, match.Modifier{
				Amount: m.Amount, Bias: m.Bias, Applies: match.Dimension(m.Applies),
				Source: match.Source(m.Source), Against: match.ScopeFrom(m.AgainstKind, m.AgainstID),
				ExpiresAt: match.Lifetime(m.ExpiresAt), Reason: m.Reason,
			})
		}
		out.CharacterResults = append(out.CharacterResults, cr)
	}
	return out
}
```

> `rollResultRecord(res.ActionResult)` compiles only if the field sets match exactly. If
> `service.RollResult` gains a field, that conversion breaks at compile time — which is the
> point. Do not replace it with a field-by-field copy that would silently drop the new one.

- [ ] **Step 6: Change `PersistTurnClose`'s signature**

In `internal/application/match/i_repository.go`, define `TurnCloseData` (see **Interfaces**)
and change the interface method to `PersistTurnClose(ctx context.Context, d TurnCloseData) error`.

In `internal/gateway/pg/round/persist_turn_close.go`, take the struct, keep the body, and add
the resolution to the turn insert:

```go
	resolutionJSON, err := encodeResolution(d.Resolution)
	if err != nil {
		return fmt.Errorf("PersistTurnClose marshal resolution: %w", err)
	}
	_, err = tx.Exec(ctx,
		`INSERT INTO turns (uuid, round_uuid, created_at, finished_at, resolution)
		 VALUES ($1, $2, $3, $4, $5)`,
		d.Turn.GetID(), d.Round.GetID(), now, finishedAt, resolutionJSON,
	)
```

- [ ] **Step 7: Update the call sites and the mocks**

Three in `room.go` (`open_next_action`, `pull_action`, `close_turn`), each passing
`Resolution: result.ClosedResolution`. Plus every `mockRoundRepo*` across
`internal/app/game/*_test.go` and `internal/application/match/*_test.go`.

Run: `rtk proxy grep -rn 'PersistTurnClose' --include=*.go internal/`

- [ ] **Step 8: Run everything**

Run: `go test ./internal/... && go test -tags=integration ./internal/gateway/pg/round/ -v`
Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add migrations/20260822000000_turn_resolution.sql internal/gateway/pg/round/ \
        internal/domain/match/modifier.go internal/domain/match/modifier_test.go \
        internal/application/match/i_repository.go internal/app/game/
git commit -m "feat(match): persist the settled resolution on turns.resolution

The declaration was stored; the collision was not, and recomputing it
later is impossible — the ledger of that instant is gone. JSONB, written
in the transaction that already closes the turn, holding the SETTLED
resolution: the master's edits recompute the projection many times, only
the close is persisted.

An explicit record, not json.Marshal on the domain struct: battle.Blow and
match.Scope both keep their fields private and would have serialized to
silence. Scope gains read-only accessors and ScopeFrom for the trip back.

PersistTurnClose takes a struct — the parameter list had grown to seven.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 7: The master edits how a test is read

The first of the two edit surfaces: `RollCondition` — bias, flat modifier, reason — recomputed
by `Derive`, with no die re-rolled.

**Files:**
- Modify: `internal/domain/match/entity/action/master_action.go`
- Modify: `internal/domain/match/matchsession/match_session.go`
- Create: `internal/application/match/edit_action.go`
- Create: `internal/application/match/edit_action_test.go`
- Modify: `internal/app/game/message.go`, `action_mapper.go`, `room.go`, `handler.go`, `hub.go`, `cmd/game/main.go`

**Interfaces:**
- Produces:
  ```go
  // internal/domain/match/entity/action
  type ConditionField string
  const (
      FieldSpeed    ConditionField = "speed"
      FieldHit      ConditionField = "hit"
      FieldDamage   ConditionField = "damage"
      FieldDodge    ConditionField = "dodge"
      FieldDefense  ConditionField = "defense"
      FieldRepel    ConditionField = "repel"
      FieldFeint    ConditionField = "feint"
      FieldMoveSpeed ConditionField = "moveSpeed"
  )
  type ConditionEdit struct {
      Field     ConditionField
      SkillName string // set instead of Field to target one entry of Skills
      Condition RollCondition
  }
  // MasterAction gains: ActionID uuid.UUID; Conditions []ConditionEdit
  ```
  ```go
  // internal/domain/match/matchsession
  // masterUUID is carried from the first version, unused until Task 9 stamps it onto each
  // captured override. Threading it now costs one parameter; adding it in Task 9 would churn
  // the use case, the room and every test that calls either.
  func (s *MatchSession) ApplyMasterAction(
      ma *action.MasterAction, masterUUID uuid.UUID,
  ) (*service.TurnResolution, error)
  ```
  ```go
  // internal/application/match
  type EditActionResult struct{ Resolution *service.TurnResolution; TurnID uuid.UUID }
  type IEditAction interface {
      Execute(ctx context.Context, session *matchsession.MatchSession,
          masterUUID, callerUUID uuid.UUID, ma *action.MasterAction) (*EditActionResult, error)
  }
  ```

- [ ] **Step 1: Build the fixture the edit tests need, then write the failing test**

`newOpenAttackFixture` does not exist. Build it in `edit_action_test.go` on the shape of
`combat_e2e_test.go`'s `newCombatFixture`, minus the HTTP server: a session with an attacker,
a victim and a third character, a `scriptedFaces` roll source, one enqueued attack, and
`OpenNextAction` already called. It needs three accessors the existing helpers do not have:

```go
// consumed is how many faces the scripted source has handed out. It is the only way to prove
// "no die was re-rolled" — asserting on the numbers cannot distinguish a re-roll that
// happened to land the same.
//
// The existing scriptedFaces in combat_e2e_test.go tracks this in its unexported i; add the
// accessor there rather than a second scripted source.
func (s *scriptedFaces) consumed() int { return s.i }

// currentResolution re-resolves the open turn without mutating it.
func (f *editFixture) currentResolution(t *testing.T) *service.TurnResolution {
	t.Helper()
	return f.session.ResolveTurn(f.openTurn())
}

// skillOn reads one entry of the open action's skill list.
func (f *editFixture) skillOn(t *testing.T, name string) action.Skill {
	t.Helper()
	for _, s := range f.openTurn().GetAction().Skills {
		if s.SkillName == name {
			return s
		}
	}
	t.Fatalf("no skill %q on the open action", name)
	return action.Skill{}
}
```

Script `Secondary` strictly higher than `Primary` on the hit check — the advantage test below
depends on it, and `f.primaryTotal` is the total the fixture asserts before any edit.

Then `internal/application/match/edit_action_test.go`:

```go
func TestEditActionRollCondition(t *testing.T) {
	t.Run("a flat modifier moves the total without touching the dice", func(t *testing.T) {
		f := newOpenAttackFixture(t) // one open turn, attack vs one target, scripted dice
		before := f.currentResolution(t)
		diceBefore := append([]int(nil), before.ActionResult.DiceRolled...)

		uc := match.NewEditActionUC()
		ma := action.NewMasterAction()
		ma.ActionID = f.actionID
		ma.Conditions = []action.ConditionEdit{{
			Field:     action.FieldHit,
			Condition: action.RollCondition{Modifier: 3, Description: "creative positioning"},
		}}

		res, err := uc.Execute(context.Background(), f.session, f.masterUUID, f.masterUUID, ma)
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if got, want := res.Resolution.ActionResult.Total, before.ActionResult.Total+3; got != want {
			t.Fatalf("Total = %d, want %d", got, want)
		}
		if !slices.Equal(res.Resolution.ActionResult.DiceRolled, diceBefore) {
			t.Fatalf("the dice changed: %v → %v", diceBefore, res.Resolution.ActionResult.DiceRolled)
		}
	})

	t.Run("advantage reads the other set that was already rolled", func(t *testing.T) {
		f := newOpenAttackFixture(t)
		uc := match.NewEditActionUC()
		ma := action.NewMasterAction()
		ma.ActionID = f.actionID
		ma.Conditions = []action.ConditionEdit{{
			Field: action.FieldHit, Condition: action.RollCondition{Bias: 1},
		}}

		res, err := uc.Execute(context.Background(), f.session, f.masterUUID, f.masterUUID, ma)
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		// The fixture scripts Secondary strictly higher than Primary, so advantage must
		// switch which set is read — proving RollAttempts is why the edit needs no re-roll.
		if res.Resolution.ActionResult.Total <= f.primaryTotal {
			t.Fatalf("advantage did not switch to the better set (total %d)",
				res.Resolution.ActionResult.Total)
		}
		if f.rollSource.consumed() != f.facesAtOpen {
			t.Fatalf("the edit consumed %d extra faces — a die was re-rolled",
				f.rollSource.consumed()-f.facesAtOpen)
		}
	})

	t.Run("the edit never moves the economy", func(t *testing.T) {
		f := newOpenAttackFixture(t)
		balanceBefore, speedsBefore := f.session.BarState(f.attackerID, action.BarAction)

		uc := match.NewEditActionUC()
		ma := action.NewMasterAction()
		ma.ActionID = f.actionID
		ma.Conditions = []action.ConditionEdit{{
			Field: action.FieldSpeed, Condition: action.RollCondition{Modifier: 12},
		}}
		if _, err := uc.Execute(context.Background(), f.session, f.masterUUID, f.masterUUID, ma); err != nil {
			t.Fatalf("Execute: %v", err)
		}

		balanceAfter, speedsAfter := f.session.BarState(f.attackerID, action.BarAction)
		if balanceAfter != balanceBefore || !slices.Equal(speedsAfter, speedsBefore) {
			t.Fatalf("the edit rewrote the economy: %v/%v → %v/%v",
				balanceBefore, speedsBefore, balanceAfter, speedsAfter)
		}
	})

	t.Run("only the master edits", func(t *testing.T) {
		f := newOpenAttackFixture(t)
		uc := match.NewEditActionUC()
		_, err := uc.Execute(context.Background(), f.session, f.masterUUID, uuid.New(),
			action.NewMasterAction())
		if !errors.Is(err, match.ErrNotMatchMaster) {
			t.Fatalf("err = %v, want ErrNotMatchMaster", err)
		}
	})
}
```

- [ ] **Step 2: Run and watch it fail**

Run: `go test ./internal/application/match/ -run TestEditActionRollCondition -v`
Expected: FAIL — `NewEditActionUC undefined`.

- [ ] **Step 3: Grow `MasterAction`**

In `internal/domain/match/entity/action/master_action.go`, keeping every existing field and
comment:

```go
// ConditionField names which RollCheck inside an Action a condition edit lands on.
//
// A path rather than a pointer because the edit arrives over the wire, from a client that
// holds no Go references — and because naming the field is what the audit stores.
type ConditionField string

const (
	FieldSpeed     ConditionField = "speed"
	FieldHit       ConditionField = "hit"
	FieldDamage    ConditionField = "damage"
	FieldDodge     ConditionField = "dodge"
	FieldDefense   ConditionField = "defense"
	FieldRepel     ConditionField = "repel"
	FieldFeint     ConditionField = "feint"
	FieldMoveSpeed ConditionField = "moveSpeed"
)

// ConditionEdit is the master changing HOW one test is read: the dice bias, a flat
// adjustment, and the reason both are surfaced by.
//
// Field and SkillName are alternatives: SkillName targets one entry of Skills (each is a test
// with its own DC), Field targets one of the action's fixed checks. Setting both is a client
// bug and is refused at the boundary.
type ConditionEdit struct {
	Field     ConditionField
	SkillName string
	Condition RollCondition
}
```

and on the struct itself:

```go
type MasterAction struct {
	// ActionID names which action of the OPEN turn this lands on — the turn's own action, or
	// one of its reactions. Zero means the turn's action.
	ActionID    uuid.UUID
	TargetID    []uuid.UUID
	Skills      []Skill
	// Conditions is the master changing how existing tests are read. Skills and TargetID
	// change WHICH tests exist; this changes how they are read. The two surfaces are
	// deliberately separate — see combat-engine.md § A edição do mestre.
	Conditions  []ConditionEdit
	Move        *Move
	Attack      *Attack
	ActionSpeed *RollCheck
	Interact    *Interact
	happenedAt  time.Time
	// Initiative *Initiative ?
	// Penalidade *Penalty ?
}
```

- [ ] **Step 4: Implement `ApplyMasterAction` on the session**

In `internal/domain/match/matchsession/match_session.go`:

```go
// ApplyMasterAction lands the master's edit ON the action and recomputes.
//
// There is no parallel version to merge on read: the edited action IS the action, which is
// the shape the code already had — RollCondition lives in RollContext, inside RollCheck,
// inside Action. The price of that model is that the original is destroyed in the live
// object, which is exactly why the override capture exists (see CaptureOverride).
//
// It NEVER rolls for a condition edit: Derive reads the two sets RollAttempts has held since
// the action arrived, and a late advantage changes WHICH set is read, never what fell.
//
// It never touches the economy either. Charged bars, recorded Speeds and the order already
// played stay as they are — bars_updated has been on the wire since before the edit, and
// redoing the price would reorder what has already been played.
func (s *MatchSession) ApplyMasterAction(
	ma *action.MasterAction, masterUUID uuid.UUID,
) (*service.TurnResolution, error) {
	t := s.activeRound.CurrentTurn()
	if t == nil || t.GetFinishedAt() != nil {
		return nil, ErrNoActiveTurn
	}
	target, err := s.actionOnTurn(t, ma.ActionID)
	if err != nil {
		return nil, err
	}
	for _, edit := range ma.Conditions {
		rc, err := resolveRollCheck(target, edit)
		if err != nil {
			return nil, err
		}
		cond := edit.Condition
		rc.Context.Condition = &cond
	}
	// Re-derive the speeds so a condition on speed or moveSpeed reads through. systemBias 0:
	// the disadvantage of an action→reaction conversion was applied once, at attach, and is
	// not re-imposed by an unrelated edit.
	s.deriveSpeeds(target, 0)
	ma.SetHappenedAt(time.Now())
	t.AddMasterAction(*ma)
	return s.ResolveTurn(t), nil
}

// actionOnTurn finds the turn's action or one of its reactions by ID. The zero UUID means the
// turn's own action, which is what a client editing "the action" sends.
func (s *MatchSession) actionOnTurn(t *turn.Turn, id uuid.UUID) (*action.Action, error) {
	a := t.ActionRef()
	if id == uuid.Nil || id == a.GetID() {
		return a, nil
	}
	if r := t.ReactionRef(id); r != nil {
		return r, nil
	}
	return nil, ErrActionNotOnTurn
}

// resolveRollCheck maps an edit's path to the RollCheck it names. A path that names nothing
// present is an error rather than a silent no-op: the master pressed a control describing a
// test they believe exists, and answering silently leaves them believing they changed it.
func resolveRollCheck(a *action.Action, e action.ConditionEdit) (*action.RollCheck, error) {
	if e.SkillName != "" {
		if e.Field != "" {
			return nil, ErrAmbiguousConditionEdit
		}
		for i := range a.Skills {
			if a.Skills[i].SkillName == e.SkillName {
				return &a.Skills[i].RollCheck, nil
			}
		}
		return nil, ErrConditionTargetMissing
	}
	switch e.Field {
	case action.FieldSpeed:
		return &a.Speed.RollCheck, nil
	case action.FieldFeint:
		if a.Feint == nil {
			return nil, ErrConditionTargetMissing
		}
		return a.Feint, nil
	case action.FieldHit:
		if a.Attack == nil {
			return nil, ErrConditionTargetMissing
		}
		return &a.Attack.Hit, nil
	case action.FieldDamage:
		if a.Attack == nil {
			return nil, ErrConditionTargetMissing
		}
		return &a.Attack.Damage, nil
	case action.FieldDodge:
		if a.Dodge == nil {
			return nil, ErrConditionTargetMissing
		}
		return &a.Dodge.RollCheck, nil
	case action.FieldDefense:
		if a.Defense == nil {
			return nil, ErrConditionTargetMissing
		}
		return &a.Defense.RollCheck, nil
	case action.FieldRepel:
		if a.Repel == nil {
			return nil, ErrConditionTargetMissing
		}
		return &a.Repel.RollCheck, nil
	case action.FieldMoveSpeed:
		if a.Move == nil || a.Move.Speed == nil {
			return nil, ErrConditionTargetMissing
		}
		return a.Move.Speed, nil
	default:
		return nil, ErrConditionTargetMissing
	}
}
```

Add the three errors to `internal/domain/match/matchsession/error.go`, and to
`internal/domain/match/entity/turn/turn.go` the two references the session needs:

```go
// ActionRef and ReactionRef hand out POINTERS, unlike GetAction/GetReactions which copy.
//
// The copies exist so a reader cannot mutate the turn by accident. The master's edit is the
// one caller that must mutate it — the edited action IS the action — so it gets the real
// thing, deliberately and narrowly, rather than by turning the safe accessors unsafe.
func (t *Turn) ActionRef() *action.Action { return &t.action }

func (t *Turn) ReactionRef(id uuid.UUID) *action.Action {
	for i := range t.reactions {
		if t.reactions[i].GetID() == id {
			return &t.reactions[i]
		}
	}
	return nil
}
```

- [ ] **Step 5: Implement the use case**

`internal/application/match/edit_action.go` — authorization, then delegate:

```go
package match

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

type EditActionResult struct {
	Resolution *service.TurnResolution
	TurnID     uuid.UUID
}

type IEditAction interface {
	Execute(ctx context.Context, session *matchsession.MatchSession,
		masterUUID, callerUUID uuid.UUID, ma *action.MasterAction) (*EditActionResult, error)
}

type EditActionUC struct{}

func NewEditActionUC() *EditActionUC { return &EditActionUC{} }

// Execute applies the master's edit and recomputes. There is deliberately NO confirmation
// verb: the master edits, the resolution recomputes on the spot, and passing the baton — open
// the next action, open the next reaction, close the turn — is the confirmation. What a
// confirm button would really offer is cancel, and cancelling is editing back to the original,
// which the master already has in hand.
func (uc *EditActionUC) Execute(
	ctx context.Context,
	session *matchsession.MatchSession,
	masterUUID, callerUUID uuid.UUID,
	ma *action.MasterAction,
) (*EditActionResult, error) {
	if callerUUID != masterUUID {
		return nil, ErrNotMatchMaster
	}
	res, err := session.ApplyMasterAction(ma, masterUUID)
	if err != nil {
		return nil, err
	}
	return &EditActionResult{Resolution: res, TurnID: session.CurrentTurnID()}, nil
}

var _ IEditAction = (*EditActionUC)(nil)
```

Add `MatchSession.CurrentTurnID()` if it does not exist — `room.go`'s package-level
`currentTurnID(session)` helper already does this and should be replaced by the method so both
callers read one implementation.

- [ ] **Step 6: Wire the message**

`internal/app/game/message.go`:

```go
	MsgTypeEditAction MessageType = "edit_action"
	MsgTypeActionEdited MessageType = "action_edited"
```

```go
// EditActionPayload is the master editing the open turn's action, or one of its reactions.
// Master only.
//
// Every section is optional and independent: send only what changes. A section that is
// present REPLACES its list wholesale — partial list merging would need a per-entry identity
// the wire does not have.
type EditActionPayload struct {
	// ActionID names the target. Omitted or zero means the turn's own action.
	ActionID   uuid.UUID               `json:"actionId,omitempty"`
	Conditions []ConditionEditPayload  `json:"conditions,omitempty"`
	Skills     *[]ActionSkillPayload   `json:"skills,omitempty"`
	TargetIDs  *[]uuid.UUID            `json:"targetIds,omitempty"`
}

// ConditionEditPayload changes how one test is read. Field and skillName are alternatives:
// one names a fixed check on the action, the other names an entry of skills.
type ConditionEditPayload struct {
	Field       string `json:"field,omitempty"`
	SkillName   string `json:"skillName,omitempty"`
	Bias        int    `json:"bias,omitempty"`
	Modifier    int    `json:"modifier,omitempty"`
	Description string `json:"description,omitempty"`
}

// ActionEditedPayload confirms the edit to the MASTER. It carries no numbers — the recomputed
// resolution travels in the resolution_updated that follows, projected like every other.
type ActionEditedPayload struct {
	TurnID   uuid.UUID `json:"turnId"`
	ActionID uuid.UUID `json:"actionId"`
}
```

Add `buildEditAction(p EditActionPayload) (*action.MasterAction, error)` to
`action_mapper.go`, crossing the `string → ConditionField` boundary and reusing `buildSkills`
for the skills section (Task 8 uses that half).

Add the route in `room.go` next to `MsgTypeOpenReaction`, master-gated, **write-locked across
Execute** (the edit mutates the action and re-derives), answering with `action_edited` to the
sender and then `r.publishResolution(result.TurnID, result.Resolution)`.

- [ ] **Step 7: Run everything**

Run: `go test ./internal/... -race && go vet -tags=integration ./internal/...`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/domain/match/ internal/application/match/edit_action.go \
        internal/application/match/edit_action_test.go internal/app/game/ cmd/game/main.go
git commit -m "feat(match): the master edits how a test is read

RollCondition lands on the action itself — there is no parallel version to
merge — and Derive recomputes without a single new die. That is the caller
RollAttempts was designed for in phase 1: a late advantage changes WHICH
set is read, never what fell.

The edit never moves the economy: charged bars, recorded Speeds and the
order already played stay put, because bars_updated already went out.

No confirmation verb. Passing the baton is the confirmation.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 8: The master edits which tests exist

The second surface: `Skills` and `TargetID`. Adding a skill rolls new dice; removing one keeps
its dice in memory so putting it back is not a free re-roll.

**Files:**
- Modify: `internal/domain/match/matchsession/match_session.go`
- Modify: `internal/application/match/edit_action_test.go`
- Modify: `internal/app/game/action_mapper.go`

**Interfaces:**
- Consumes: `MatchSession.ApplyMasterAction` (Task 7), `rollActionDice`.
- Produces: nothing new on the public surface — `ApplyMasterAction` grows two branches.

- [ ] **Step 1: Write the failing tests**

Append to `internal/application/match/edit_action_test.go`:

```go
func TestEditActionSkills(t *testing.T) {
	t.Run("adding a skill rolls dice for it — a first roll, not a re-roll", func(t *testing.T) {
		f := newOpenAttackFixture(t)
		facesBefore := f.rollSource.consumed()

		uc := match.NewEditActionUC()
		ma := action.NewMasterAction()
		ma.ActionID = f.actionID
		ma.Skills = []action.Skill{{SkillName: enum.Acrobatics.String()}}

		if _, err := uc.Execute(context.Background(), f.session, f.masterUUID, f.masterUUID, ma); err != nil {
			t.Fatalf("Execute: %v", err)
		}
		// One 2D10 test: four faces, because Roll always rolls Primary AND Secondary.
		if got := f.rollSource.consumed() - facesBefore; got != 4 {
			t.Fatalf("added skill consumed %d faces, want 4", got)
		}
		added := f.skillOn(t, enum.Acrobatics.String())
		if added.Attempts.IsEmpty() {
			t.Fatal("the added skill has no dice")
		}
	})

	t.Run("removing and re-adding a skill reuses its dice", func(t *testing.T) {
		f := newOpenAttackFixtureWithSkill(t, enum.Acrobatics.String())
		original := f.skillOn(t, enum.Acrobatics.String()).Attempts

		uc := match.NewEditActionUC()
		strip := action.NewMasterAction()
		strip.ActionID = f.actionID
		strip.Skills = []action.Skill{} // present and empty: remove them all
		if _, err := uc.Execute(context.Background(), f.session, f.masterUUID, f.masterUUID, strip); err != nil {
			t.Fatalf("strip: %v", err)
		}
		facesAfterStrip := f.rollSource.consumed()

		restore := action.NewMasterAction()
		restore.ActionID = f.actionID
		restore.Skills = []action.Skill{{SkillName: enum.Acrobatics.String()}}
		if _, err := uc.Execute(context.Background(), f.session, f.masterUUID, f.masterUUID, restore); err != nil {
			t.Fatalf("restore: %v", err)
		}

		if f.rollSource.consumed() != facesAfterStrip {
			t.Fatal("re-adding a removed skill rolled again — that is a free re-roll")
		}
		back := f.skillOn(t, enum.Acrobatics.String()).Attempts
		if !slices.Equal(back.Primary, original.Primary) ||
			!slices.Equal(back.Secondary, original.Secondary) {
			t.Fatalf("the dice came back different: %v → %v", original, back)
		}
	})

	t.Run("adding a skill to a pure move does not start charging the action bar", func(t *testing.T) {
		f := newOpenMoveFixture(t) // a move-only action, opened, charging BarMove alone
		_, speedsBefore := f.session.BarState(f.actorID, action.BarAction)

		uc := match.NewEditActionUC()
		ma := action.NewMasterAction()
		ma.ActionID = f.actionID
		ma.Skills = []action.Skill{{SkillName: enum.Acrobatics.String()}}
		if _, err := uc.Execute(context.Background(), f.session, f.masterUUID, f.masterUUID, ma); err != nil {
			t.Fatalf("Execute: %v", err)
		}

		_, speedsAfter := f.session.BarState(f.actorID, action.BarAction)
		if !slices.Equal(speedsAfter, speedsBefore) {
			t.Fatal("the edit re-priced the action bar; the order already played would move")
		}
	})
}

func TestEditActionTargets(t *testing.T) {
	t.Run("replacing the target list re-resolves against the new targets", func(t *testing.T) {
		f := newOpenAttackFixture(t)
		other := f.thirdCharacterID

		uc := match.NewEditActionUC()
		ma := action.NewMasterAction()
		ma.ActionID = f.actionID
		ma.TargetID = []uuid.UUID{other}

		res, err := uc.Execute(context.Background(), f.session, f.masterUUID, f.masterUUID, ma)
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if len(res.Resolution.CharacterResults) != 1 ||
			res.Resolution.CharacterResults[0].TargetID != other {
			t.Fatalf("the resolution still names the old target: %+v", res.Resolution.CharacterResults)
		}
	})
}
```

- [ ] **Step 2: Run and watch it fail**

Run: `go test ./internal/application/match/ -run 'TestEditActionSkills|TestEditActionTargets' -v`
Expected: FAIL — the skills and targets branches do not exist.

- [ ] **Step 3: Implement the two branches**

In `ApplyMasterAction`, before the conditions loop:

```go
	if ma.TargetID != nil {
		target.TargetID = append([]uuid.UUID(nil), ma.TargetID...)
	}
	if ma.Skills != nil {
		s.applySkillEdit(target, ma.Skills)
	}
```

and the helper, with the dice memory:

```go
// applySkillEdit replaces the action's skill list, and it is where the two asymmetric rules of
// combat-engine.md § A edição do mestre live.
//
// ADDING rolls new dice. That is not a re-roll and does not break "the master never re-rolls a
// player's die": it is the FIRST roll of a test that did not exist a moment ago.
//
// REMOVING keeps the dice. They go into removedSkillDice, keyed by action and skill name, and
// a later re-add reads them back. Without that, taking a skill out and putting it back would
// be a free re-roll — the master would only have to dislike a number to be given another one.
//
// The list changes and the list does not yet DECIDE anything: nobody reads a Skill's result
// (combat-engine.md § A corrente de testes). That is stated, not hidden.
func (s *MatchSession) applySkillEdit(a *action.Action, want []action.Skill) {
	if s.removedSkillDice == nil {
		s.removedSkillDice = map[uuid.UUID]map[string]action.RollAttempts{}
	}
	memory := s.removedSkillDice[a.GetID()]
	if memory == nil {
		memory = map[string]action.RollAttempts{}
		s.removedSkillDice[a.GetID()] = memory
	}

	held := make(map[string]action.RollAttempts, len(a.Skills))
	for _, prev := range a.Skills {
		held[prev.SkillName] = prev.Attempts
	}

	next := make([]action.Skill, 0, len(want))
	for _, sk := range want {
		switch {
		case !held[sk.SkillName].IsEmpty():
			sk.Attempts = held[sk.SkillName] // untouched: it was already there
		case !memory[sk.SkillName].IsEmpty():
			sk.Attempts = memory[sk.SkillName] // put back: same dice as before
			delete(memory, sk.SkillName)
		}
		if sk.RollCheck.SkillName == "" {
			sk.RollCheck.SkillName = sk.SkillName
		}
		next = append(next, sk)
	}
	// Whatever left the list parks its dice, so a re-add is not a re-roll.
	for name, attempts := range held {
		stillThere := false
		for _, sk := range next {
			if sk.SkillName == name {
				stillThere = true
				break
			}
		}
		if !stillThere && !attempts.IsEmpty() {
			memory[name] = attempts
		}
	}
	a.Skills = next
	// rollActionDice leaves alone every RollCheck whose dice already fell, so this rolls for
	// exactly the entries that are genuinely new.
	s.rollActionDice(a)
}
```

Add the field to `MatchSession`:

```go
	// removedSkillDice parks the dice of skills the master took off an action, keyed by action
	// then skill name, so putting one back is not a free re-roll. It lives for as long as the
	// session does; a turn's entries are drained into the audit when it closes.
	removedSkillDice map[uuid.UUID]map[string]action.RollAttempts
```

> ⚠️ Note what is deliberately absent: nothing here re-runs `recordActed`, `chargeReactionBars`
> or `FreezePrices`. Adding a skill to a pure move changes what `Bars()` answers, and the
> economy still must not move. **Do not "fix" that** — the third test above exists to keep
> someone from doing it.

- [ ] **Step 4: Map the wire half**

In `action_mapper.go`, `buildEditAction` maps `Skills *[]ActionSkillPayload` through the
existing `buildSkills` (so an unknown skill name is still refused at the boundary) and
distinguishes *absent* from *present and empty* — the pointer is what carries "remove them
all".

- [ ] **Step 5: Run everything**

Run: `go test ./internal/... -race`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/domain/match/matchsession/ internal/application/match/ internal/app/game/
git commit -m "feat(match): the master edits which tests exist

Skills and targets. Adding a skill rolls new dice — the first roll of a
test that did not exist, not a re-roll. Removing one parks its dice, so
taking it out and putting it back is not a free re-roll for a master who
disliked a number.

The edit still does not move the economy, even when adding a skill to a
pure move changes what Bars() answers. There is a test whose whole job is
to stop someone from 'fixing' that.

The list changes; the list does not yet decide anything — nobody reads a
Skill's result. Said, not hidden.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 9: Capturing what the edit displaced

**Files:**
- Create: `internal/domain/match/override.go`
- Create: `internal/domain/match/override_test.go`
- Modify: `internal/domain/match/matchsession/match_session.go`

**Interfaces:**
- Produces:
  ```go
  // internal/domain/match
  type OverrideOrigin string
  const (
      OriginPlayer OverrideOrigin = "player"
      OriginSystem OverrideOrigin = "system"
  )
  type OverriddenValue struct {
      ActionID   uuid.UUID
      Field      string
      Origin     OverrideOrigin
      MasterUUID uuid.UUID
      At         time.Time
      Original   any
  }
  ```
  ```go
  // internal/domain/match/matchsession
  func (s *MatchSession) TakeOverridesFor(t *turn.Turn) []match.OverriddenValue
  ```

- [ ] **Step 1: Write the failing test**

`internal/domain/match/matchsession/match_session_test.go`:

```go
func TestOverrideCapture(t *testing.T) {
	t.Run("the first edit captures the ORIGINAL, not the edit", func(t *testing.T) {
		f := newOpenAttackFixture(t)
		f.editHitModifier(t, 3)

		got := f.session.TakeOverridesFor(f.openTurn())
		if len(got) != 1 {
			t.Fatalf("captured %d values, want 1", len(got))
		}
		if got[0].Field != "hit.condition" {
			t.Fatalf("Field = %q", got[0].Field)
		}
		if got[0].Original != nil {
			t.Fatalf("Original = %v, want nil — the player sent no condition", got[0].Original)
		}
	})

	t.Run("a second edit of the same field does not add a row", func(t *testing.T) {
		f := newOpenAttackFixture(t)
		f.editHitModifier(t, 3)
		f.editHitModifier(t, 5)

		if got := len(f.session.TakeOverridesFor(f.openTurn())); got != 1 {
			t.Fatalf("captured %d values, want 1 — one row per FIELD, not per edit", got)
		}
	})

	t.Run("editing back to the original erases the capture", func(t *testing.T) {
		f := newOpenAttackFixtureWithSkill(t, enum.Acrobatics.String())
		before := f.skillNames(t)

		f.editSkills(t, []string{}) // strip
		if len(f.session.PeekOverridesFor(f.openTurn())) != 1 {
			t.Fatal("the strip did not capture the original skill list")
		}
		f.editSkills(t, before) // put it back exactly

		if got := len(f.session.PeekOverridesFor(f.openTurn())); got != 0 {
			t.Fatalf("captured %d values after a revert, want 0", got)
		}
	})

	t.Run("the removed skill's dice ride along in the captured list", func(t *testing.T) {
		f := newOpenAttackFixtureWithSkill(t, enum.Acrobatics.String())
		f.editSkills(t, []string{})

		got := f.session.TakeOverridesFor(f.openTurn())
		skills, ok := got[0].Original.([]action.Skill)
		if !ok || len(skills) != 1 {
			t.Fatalf("Original = %#v, want the original []action.Skill", got[0].Original)
		}
		if skills[0].Attempts.IsEmpty() {
			t.Fatal("the removed skill's dice were dropped; a test that happened left no trace")
		}
	})
}
```

- [ ] **Step 2: Run and watch it fail**

Run: `go test ./internal/domain/match/matchsession/ -run TestOverrideCapture -v`
Expected: FAIL — `TakeOverridesFor` undefined.

- [ ] **Step 3: Implement the value type**

`internal/domain/match/override.go`:

```go
package match

import (
	"time"

	"github.com/google/uuid"
)

// OverrideOrigin says where the displaced value came from. Two origins, one overwriter —
// the master. There is no Source field naming the overwriter, because if only the master
// overrides, it would have nothing to discriminate. The bias the SYSTEM applies is a
// Modifier in the ledger and was never an override.
type OverrideOrigin string

const (
	OriginPlayer OverrideOrigin = "player" // the value the player sent
	OriginSystem OverrideOrigin = "system" // the value the engine computed
)

// OverriddenValue is what a master's edit DISPLACED — never the edit itself.
//
// The action already carries the new value; storing both would be duplication that diverges.
// The purpose is not to log what the master did, it is to not lose what the player sent and
// what the system calculated, so the original can be reconstructed by reading backwards from
// the corresponding row of actions.
//
// ONE ROW PER FIELD, holding the ORIGINAL — not one per edit. Two good properties fall out:
// reverting is free (edit back and nothing is stored at all), and a mistake plus its fix
// leave no noise behind.
type OverriddenValue struct {
	ActionID   uuid.UUID
	Field      string
	Origin     OverrideOrigin
	MasterUUID uuid.UUID
	At         time.Time
	// Original is the displaced value in its domain shape — an int, a []action.Skill, a set
	// of target UUIDs. The gateway marshals it to JSONB; the format genuinely varies and
	// nobody is going to query inside it.
	Original any
}
```

- [ ] **Step 4: Implement the capture on the session**

Add to `MatchSession`:

```go
	// overrides is what the master's edits displaced, keyed by action then field. In memory
	// while the turn is open — the master edits and unedits freely — and drained into the
	// close transaction by TakeOverridesFor. A revert deletes its entry rather than adding
	// a second one.
	overrides map[uuid.UUID]map[string]match.OverriddenValue
```

and, called from `ApplyMasterAction` **before** each mutation:

```go
// captureOverride records the value about to be displaced, once per field, and erases the
// record when an edit puts the original back.
//
// current is what is there right now; incoming is what the master is about to write. Both
// are compared with reflect.DeepEqual because the shapes are heterogeneous by design (an int,
// a slice of skills, a set of UUIDs) — the alternative is a comparison function per field,
// which is three places to forget to update.
func (s *MatchSession) captureOverride(
	actionID uuid.UUID, field string, origin match.OverrideOrigin,
	masterUUID uuid.UUID, current, incoming any,
) {
	if s.overrides == nil {
		s.overrides = map[uuid.UUID]map[string]match.OverriddenValue{}
	}
	byField := s.overrides[actionID]
	if byField == nil {
		byField = map[string]match.OverriddenValue{}
		s.overrides[actionID] = byField
	}
	if existing, ok := byField[field]; ok {
		// Back to where it started: nothing was displaced after all. This is what makes a
		// cancel verb unnecessary — cancelling IS editing back.
		if reflect.DeepEqual(existing.Original, incoming) {
			delete(byField, field)
		}
		return // one row per field; the intermediate values are neither the player's nor the system's
	}
	if reflect.DeepEqual(current, incoming) {
		return // an edit that changes nothing displaces nothing
	}
	byField[field] = match.OverriddenValue{
		ActionID: actionID, Field: field, Origin: origin,
		MasterUUID: masterUUID, At: time.Now(), Original: current,
	}
}

// turnActionIDs is every action ID a turn owns: its own, plus each attached reaction. Both
// can be edited, so both can have captures.
func turnActionIDs(t *turn.Turn) []uuid.UUID {
	a := t.GetAction()
	out := []uuid.UUID{a.GetID()}
	for _, r := range t.GetReactions() {
		out = append(out, r.GetID())
	}
	return out
}

// PeekOverridesFor reads the captures belonging to one turn without draining them — for
// tests, and for a master view of what they have displaced so far.
func (s *MatchSession) PeekOverridesFor(t *turn.Turn) []match.OverriddenValue {
	if t == nil {
		return nil
	}
	var out []match.OverriddenValue
	for _, id := range turnActionIDs(t) {
		for _, ov := range s.overrides[id] {
			out = append(out, ov)
		}
	}
	return out
}

// TakeOverridesFor drains the captures belonging to one turn's action and reactions, for the
// close transaction.
//
// Draining, not copying: they are written exactly once, and a turn that has closed cannot be
// edited again. Leaving them behind would mean a later turn's close re-inserting them — which
// the unique constraint would swallow silently, so the bug would only ever show up as rows
// that never appeared.
//
// The removed-skill dice parked for those actions go with them: their only remaining purpose
// was to survive a re-add, and there is nothing left to re-add to.
func (s *MatchSession) TakeOverridesFor(t *turn.Turn) []match.OverriddenValue {
	if t == nil {
		return nil
	}
	var out []match.OverriddenValue
	for _, id := range turnActionIDs(t) {
		for _, ov := range s.overrides[id] {
			out = append(out, ov)
		}
		delete(s.overrides, id)
		delete(s.removedSkillDice, id)
	}
	// Stable order, so a failing integration test names the same row twice in a row.
	sort.Slice(out, func(i, j int) bool {
		if out[i].ActionID != out[j].ActionID {
			return out[i].ActionID.String() < out[j].ActionID.String()
		}
		return out[i].Field < out[j].Field
	})
	return out
}
```

`ApplyMasterAction` captures with these field keys, using the `masterUUID` it already takes:

| Edit | Field key | Origin |
|---|---|---|
| condition on a fixed check | `"<field>.condition"`, e.g. `"hit.condition"` | `player` |
| condition on a skill entry | `"skill:<name>.condition"` | `player` |
| skill list replaced | `"skills"` | `player` |
| target list replaced | `"targetIds"` | `player` |

> The removed skill's dice need no key of their own: the captured `"skills"` original is the
> whole prior `[]action.Skill`, `Attempts` and all. That satisfies *"a removed skill's dice are
> stored, never shown"* in one move — the history reads `actions.skills`, which holds the
> current list, and never this table.

- [ ] **Step 5: Run and commit**

Run: `go test ./internal/domain/... ./internal/application/... -race`
Expected: PASS.

```bash
git add internal/domain/match/override.go internal/domain/match/override_test.go \
        internal/domain/match/matchsession/ internal/application/match/
git commit -m "feat(match): capture what an edit displaced, one row per field

Not the edit — the value the edit overwrote. The action already carries
the new value; storing both is duplication that diverges. One row per
FIELD holding the ORIGINAL, so a mistake and its fix leave nothing behind
and reverting needs no verb.

A removed skill's dice ride along inside the captured skill list, which is
why they are stored and never shown: the history reads actions.skills.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 10: `overridden_action_values` — the table and the write

**Files:**
- Create: `migrations/20260822000001_overridden_action_values.sql`
- Modify: `internal/gateway/pg/round/persist_turn_close.go`
- Modify: `internal/app/game/room.go` (three call sites pass `Overrides`)
- Test: `internal/gateway/pg/round/round_integration_test.go`

- [ ] **Step 1: Write the migration**

```sql
-- +goose Up
-- +goose StatementBegin
BEGIN;

-- What a master's edit DISPLACED — never the edit itself. The action already carries the new
-- value; storing both would be duplication that diverges, and the original is reconstructed by
-- reading this row against the corresponding row of actions.
--
-- The name refuses things, which is the whole reason it is not called SystemData: a generic
-- name cannot turn anything away and becomes a dumping ground. Wall state cannot be put in
-- overridden_action_values without the name being visibly false.
--
-- ONE ROW PER FIELD, holding the ORIGINAL — hence the unique constraint. Not one row per edit:
-- the master's intermediate values are neither what the player sent nor what the system
-- calculated, and those two are the only things this table exists to preserve.
--
-- No source column. If only the master overrides, there is nothing for it to discriminate; the
-- bias the SYSTEM applies is a Modifier in the ledger and was never an override. `origin` is a
-- different question: where the DISPLACED value came from.
CREATE TABLE IF NOT EXISTS overridden_action_values (
    id             SERIAL PRIMARY KEY,
    action_uuid    UUID NOT NULL REFERENCES actions(uuid) ON DELETE CASCADE,
    field          VARCHAR(64) NOT NULL,
    origin         VARCHAR(16) NOT NULL,
    master_uuid    UUID NOT NULL REFERENCES users(uuid),
    overridden_at  TIMESTAMP NOT NULL,
    original_value JSONB,
    UNIQUE (action_uuid, field)
);
CREATE INDEX IF NOT EXISTS idx_overridden_action_values_action_uuid
    ON overridden_action_values(action_uuid);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP TABLE IF EXISTS overridden_action_values;

COMMIT;
-- +goose StatementEnd
```

> `original_value` is nullable on purpose: the commonest capture is a `RollCondition` the
> player never sent, whose original is genuinely nothing. NULL says "there was no value";
> `'null'::jsonb` would say "there was a value, and it was null".

- [ ] **Step 2: Write the failing integration test**

```go
func TestPersistTurnCloseWritesOverriddenValues(t *testing.T) {
	// ... build a closed turn whose action was edited twice on the same field
	err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
		/* ... */ Overrides: []match.OverriddenValue{{
			ActionID: act.GetID(), Field: "skills", Origin: match.OriginPlayer,
			MasterUUID: fx.masterUUID, At: time.Now(),
			Original: []action.Skill{{SkillName: "Acrobatics"}},
		}},
	})
	if err != nil {
		t.Fatalf("PersistTurnClose: %v", err)
	}

	var n int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM overridden_action_values WHERE action_uuid = $1`,
		act.GetID()).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Fatalf("wrote %d rows, want 1 — one row per field", n)
	}
}

func TestPersistTurnCloseWritesNoRowForAnUneditedTurn(t *testing.T) {
	// Overrides: nil. Assert COUNT(*) == 0 — a revert leaves no trace, and neither does a
	// turn the master never touched.
}
```

- [ ] **Step 3: Implement the write, inside the same transaction**

In `persist_turn_close.go`, after the reactions loop and before `tx.Commit`:

```go
	// The overrides go in the SAME transaction as the actions they point at: the FK requires
	// the action row to exist, and a capture that outlived its action would be unreadable.
	for _, ov := range d.Overrides {
		original, err := marshalNullableAny(ov.Original)
		if err != nil {
			return fmt.Errorf("PersistTurnClose marshal override %s: %w", ov.Field, err)
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO overridden_action_values
			 (action_uuid, field, origin, master_uuid, overridden_at, original_value)
			 VALUES ($1,$2,$3,$4,$5,$6)
			 ON CONFLICT (action_uuid, field) DO NOTHING`,
			ov.ActionID, ov.Field, string(ov.Origin), ov.MasterUUID, ov.At, original,
		)
		if err != nil {
			return fmt.Errorf("PersistTurnClose insert override: %w", err)
		}
	}
```

```go
// marshalNullableAny returns nil (SQL NULL) for a nil value. NULL says "there was no value"
// — which is the honest answer for a RollCondition the player never sent — where 'null'::jsonb
// would claim there was one and it was null.
func marshalNullableAny(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}
```

- [ ] **Step 4: Pass the captures from `room.go`**

At all three `PersistTurnClose` call sites, under the write lock that already reads the scene
and round: `Overrides: session.TakeOverridesFor(closedTurn)`.

- [ ] **Step 5: Run and commit**

Run: `make migrate-up && go test -tags=integration ./internal/gateway/pg/round/ -v && go test ./internal/...`

```bash
git add migrations/20260822000001_overridden_action_values.sql internal/gateway/pg/round/ internal/app/game/
git commit -m "feat(match): overridden_action_values, written with the turn

One row per field holding the original, in the transaction that already
writes the action it points at. The name refuses things, which is why it
is not SystemData: a generic name cannot turn anything away and becomes a
dumping ground.

original_value is nullable — the commonest capture is a RollCondition the
player never sent, and NULL says 'there was no value' where 'null'::jsonb
would claim there was one.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 11: Reading the history back

**Files:**
- Create: `internal/gateway/pg/round/find_match_history.go`
- Modify: `internal/application/match/i_repository.go`
- Test: `internal/gateway/pg/round/round_integration_test.go`

**Interfaces:**
- Produces:
  ```go
  // internal/application/match — the nested shape, not a flat list
  type HistoryScene struct {
      UUID       uuid.UUID
      Category   string
      BriefDesc  string
      CreatedAt  time.Time
      FinishedAt *time.Time
      Rounds     []HistoryRound
  }
  type HistoryRound struct {
      UUID       uuid.UUID
      Mode       string
      CreatedAt  time.Time
      FinishedAt *time.Time
      Turns      []HistoryTurn
  }
  type HistoryTurn struct {
      UUID       uuid.UUID
      CreatedAt  time.Time
      FinishedAt time.Time
      Action     action.Action
      Reactions  []action.Action
      Resolution *service.TurnResolution // nil for a turn that resolved nothing
  }
  // on IRoundRepository:
  FindMatchHistory(ctx context.Context, matchUUID uuid.UUID) ([]HistoryScene, error)
  ```

- [ ] **Step 1: Write the failing integration test**

```go
func TestFindMatchHistoryIsNested(t *testing.T) {
	// Persist two turns in one round of one scene, one of them with a reaction and a
	// resolution. Then:
	scenes, err := repo.FindMatchHistory(ctx, fx.matchUUID)
	if err != nil {
		t.Fatalf("FindMatchHistory: %v", err)
	}
	if len(scenes) != 1 || len(scenes[0].Rounds) != 1 || len(scenes[0].Rounds[0].Turns) != 2 {
		t.Fatalf("the tree is wrong: %d scenes", len(scenes))
	}
	turns := scenes[0].Rounds[0].Turns
	if turns[0].FinishedAt.After(turns[1].FinishedAt) {
		t.Fatal("turns came back out of order")
	}
	withReaction := turns[1]
	if len(withReaction.Reactions) != 1 {
		t.Fatalf("reactions = %d, want 1 — they are persisted since PR #69",
			len(withReaction.Reactions))
	}
	if withReaction.Reactions[0].ReactionKind == "" {
		t.Fatal("reaction_kind did not come back")
	}
	if withReaction.Resolution == nil || len(withReaction.Resolution.CharacterResults) == 0 {
		t.Fatal("the settled resolution did not come back")
	}
}

func TestFindMatchHistoryOfAMatchWithNoTurns(t *testing.T) {
	// An empty slice, not an error and not nil-that-marshals-to-null.
}
```

- [ ] **Step 2: Run and watch it fail**

Run: `go test -tags=integration ./internal/gateway/pg/round/ -run TestFindMatchHistory -v`
Expected: FAIL — `FindMatchHistory` undefined.

- [ ] **Step 3: Implement the query**

`internal/gateway/pg/round/find_match_history.go`: one query, ordered, assembled in Go.

```go
// FindMatchHistory returns the match's closed turns as the TREE the domain already is —
// Scene → Round → Turn → Action — not a flat list.
//
// The hierarchy is not decoration: the front renders action cards INSIDE the scope of each
// scene, because scenes are the logical blocks the match is organised into. Flattening here
// would push the regrouping onto every consumer.
//
// One query, ordered, assembled in one pass. Reactions come back in the same result set,
// discriminated by react_to_uuid IS NOT NULL, so there is no N+1 over turns.
func (r *Repository) FindMatchHistory(
	ctx context.Context, matchUUID uuid.UUID,
) ([]appmatch.HistoryScene, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT s.uuid, s.category, s.brief_initial_description, s.created_at, s.finished_at,
		        ro.uuid, ro.mode, ro.created_at, ro.finished_at,
		        t.uuid, t.created_at, t.finished_at, t.resolution,
		        a.uuid, a.actor_uuid, a.react_to_uuid, a.target_ids, a.type, a.reaction_kind,
		        a.speed, a.skills, a.move, a.attack, a.defense, a.dodge, a.repel, a.feint, a.trigger
		 FROM scenes s
		 JOIN rounds ro ON ro.scene_uuid = s.uuid
		 JOIN turns  t  ON t.round_uuid = ro.uuid
		 JOIN actions a ON a.turn_uuid = t.uuid
		 WHERE s.match_uuid = $1
		 ORDER BY s.created_at, ro.created_at, t.finished_at,
		          (a.react_to_uuid IS NOT NULL), a.created_at`,
		matchUUID,
	)
	// ... scan, unmarshal each JSONB column back into its action sub-struct, group by
	// scene/round/turn in one pass, and DecodeResolution(t.resolution) once per turn.
}
```

> The `ORDER BY (a.react_to_uuid IS NOT NULL)` clause puts the turn's own action before its
> reactions within each turn, so the assembly loop can take the first row of a turn as the
> action and the rest as reactions without a second sort.
>
> ⚠️ `actions` has no column for the master's actions — `t.GetMasterActions()` is deliberately
> not persisted (spec § *A edição do mestre*). The history shows the edited action, which is
> the action; what the edit displaced lives in `overridden_action_values` and is **not** part
> of this read.

- [ ] **Step 4: Run and commit**

Run: `go test -tags=integration ./internal/gateway/pg/round/ -v`

```bash
git add internal/gateway/pg/round/find_match_history.go internal/application/match/i_repository.go \
        internal/gateway/pg/round/round_integration_test.go
git commit -m "feat(match): read the history back as the tree it is

Scene -> Round -> Turn -> Action, in one query, assembled in one pass. The
hierarchy is not decoration: the front renders action cards inside the
scope of each scene, so flattening here would push the regrouping onto
every consumer.

Reactions ride the same result set, ordered after the action they answer.
Master actions are deliberately not persisted and are not read.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 12: The Action History endpoint, projected

**Files:**
- Create: `internal/application/match/get_match_history.go`, `get_match_history_test.go`
- Create: `internal/app/api/match/get_match_history.go`, `get_match_history_test.go`
- Modify: `internal/app/api/match/routes.go`, `internal/app/api/api.go` (or wherever `Api` is built)
- Create: `docs/dev/api/match-history.md`
- Modify: `docs/documentation-map.yaml`

- [ ] **Step 1: Write the failing use-case test**

```go
func TestGetMatchHistoryUC(t *testing.T) {
	t.Run("a non-participant of a private match is refused", func(t *testing.T) { /* auth.ErrInsufficientPermissions */ })
	t.Run("the master's viewer sees everything", func(t *testing.T) { /* Viewer.IsMaster true */ })
	t.Run("a player's viewer owns exactly their own characters", func(t *testing.T) {
		// Assert the Viewer handed to the projection carries the caller's sheets and no others.
	})
	t.Run("a third party's turn arrives with the closed dodge demoted", func(t *testing.T) {
		// The done-criterion, at the REST layer: the reaction reads "dodge", and the Evasion
		// entry is gone from its skills.
	})
}
```

- [ ] **Step 2: Implement the use case**

`internal/application/match/get_match_history.go`, following `GetMatchParticipantsUC` exactly
for authorization (match public, or master, or a campaign participant), then building the
`service.Viewer` from `ListParticipantsByMatchUUID` — the caller's own sheets — and running
every action and resolution in the tree through `service.ProjectAction` /
`service.ProjectResolution`.

> The **same** projection functions as the WS path. Two implementations of one deny-list is
> how a field ends up public in one surface and hidden in the other.

- [ ] **Step 3: Implement the REST handler**

`internal/app/api/match/get_match_history.go` — request `{uuid}` from the path, response DTOs
in camelCase mirroring the nested tree, error mapping in the house style
(404 / 403 / 500), and registration in `routes.go`:

```go
	huma.Register(api, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/matches/{uuid}/history",
		Description: "Action history of a match, nested by scene, projected per viewer",
		Tags:        []string{"matches"},
		Errors: []int{
			http.StatusNotFound,
			http.StatusForbidden,
			http.StatusUnauthorized,
			http.StatusInternalServerError,
		},
	}, a.GetMatchHistoryHandler)
```

- [ ] **Step 4: Write the humatest coverage**

`internal/app/api/match/get_match_history_test.go`, in the style of
`get_match_participants_test.go`: table-driven over the use case's outcomes, asserting status
codes and — for the 200 case — that a third-party viewer's JSON carries `"reactionKind":
"dodge"` and no `Evasion` entry.

- [ ] **Step 5: Write the contract**

`docs/dev/api/match-history.md`, in the shape of `docs/dev/api/match.md`: method, path, auth,
the nested response with a full example, the error table, and — this one matters for the front
— an explicit section saying **the response is already projected**: the same turn returns
different JSON to different viewers, so nothing client-side should assume two players see
equal payloads, and no client-side filtering is needed or wanted.

- [ ] **Step 6: Register in the documentation map**

In `docs/documentation-map.yaml`, add entries for `internal/app/api/match/get_match_history.go`
→ `docs/dev/api/match-history.md`, and for `internal/domain/match/service/projection.go` →
`docs/dev/match/combat-engine.md`.

- [ ] **Step 7: Run and commit**

Run: `go test ./internal/... && go vet -tags=integration ./internal/... && go vet -tags=smoke ./internal/...`

```bash
git add internal/application/match/get_match_history*.go internal/app/api/match/ \
        docs/dev/api/match-history.md docs/documentation-map.yaml
git commit -m "feat(api): GET /matches/{uuid}/history, projected per viewer

Nested by scene, because the front renders action cards inside the scope
of each scene and the domain hierarchy is already that shape.

It runs the SAME service.Project* the WebSocket path runs. Two
implementations of one deny-list is how a field ends up public in one
surface and hidden in the other.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 13: The six criteria, and the durable doc

Nothing new is built here. This task proves the phase and writes down what it fixed.

- [ ] **Step 1: Walk the criteria, one command each**

| Criterion | Command |
|---|---|
| two clients, different payloads, same turn | `go test ./internal/app/game/ -run TestE2E_AClosedDodge -race -v` |
| a closed dodge reaches a third party as `dodge` | same test |
| `close_turn` refuses, then accepts, and damage lands both ways | `go test ./internal/application/match/ -run TestCloseTurnUC -v` and the handler test |
| the history returns a closed turn, nested, with the hidden fields respected | `go test -tags=integration ./internal/gateway/pg/round/ -run TestFindMatchHistory -v` and `go test ./internal/app/api/match/ -run TestGetMatchHistory -v` |
| an override appears with the ORIGINAL, and a revert leaves no row | `go test -tags=integration ./internal/gateway/pg/round/ -run TestPersistTurnCloseWritesOverridden -v` and `go test ./internal/domain/match/matchsession/ -run TestOverrideCapture -v` |
| `pull_action` is invocable from an ID the master received | `go test ./internal/app/game/ -run TestActionQueuedReachesOnlyTheMaster -v` |

- [ ] **Step 2: Full suite, all three tag sets**

```bash
go test ./... -race
go vet -tags=integration ./internal/...
go vet -tags=smoke ./internal/...
go test -tags=integration ./internal/gateway/pg/...
```

- [ ] **Step 3: Smoke-test the REST endpoint against a running server**

Per `System_X_System_Project/CLAUDE.md` § *Entrega*: `make run-dev`, then

```bash
curl -s -H "Authorization: Bearer $TOKEN" localhost:5000/matches/$MATCH/history | jq '.scenes[0].rounds[0].turns[0]'
```

with two different tokens — the master's and a non-participating player's — and diff the two
responses. **That diff is the phase.** If curling the real scenario needs reverse-engineering
undocumented flows (campaign → scenario → match → enrollment) out of proportion to the change,
substitute the equivalent automated evidence against the real HTTP handler (`humatest`, full
suite) and **record the substitution and why** in the PR, as the delivery rule allows.

- [ ] **Step 4: Write § *O que a Fase 5 fixou no motor***

Append to `docs/dev/match/combat-engine.md`, after § *O que a Fase 4 fixou no motor*, in the
same voice as its neighbours — one `###` per thing that is now durably true, and say where
each diverged from this plan:

- `CloseOpenTurn` is the only way to close a turn, and the old `CloseTurn` was a silent
  damage-eater.
- The refusal is the server's; `confirm: true` is the answer.
- The projection has **two axes** — time (`IsSettled`) and class — and both live in one place,
  `Room.publishResolution` + `service.Project*`.
- The label is the leak: `publicKind`, and why a bystander deducing from `bars_updated` is
  fine.
- `turns.resolution` holds the **settled** resolution, and why an explicit record beats
  `json.Marshal` on the domain struct (`battle.Blow` and `match.Scope` both serialize to
  silence).
- The edit lands on the action; `Derive` is the caller `RollAttempts` was built for; the
  economy never moves.
- Removed skills park their dice; `overridden_action_values` stores the ORIGINAL, one row per
  field, and a revert stores nothing.
- `action_queued` closes the `pull_action` hole.
- ⛔ **Still open, and this phase deliberately did not close it:** nobody reads a `Skill`'s
  result. The chain of tests is written in `docs/game/combate/acoes.md` and does not exist in
  code, so adding and removing skills changes a list that decides nothing.

Then update § *Pendências estruturais*, marking `Tabela SystemData` as ✅ Fase 5 under its real
name, and the Action History read path likewise.

- [ ] **Step 5: Commit the doc**

```bash
git add docs/dev/match/combat-engine.md docs/documentation-map.yaml
git commit -m "docs(match): what phase 5 fixed in the engine

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

- [ ] **Step 6: Open the PR**

Say explicitly, per `System_X_System_Project/CLAUDE.md` § *Entrega*:

- what was verified end-to-end and how;
- **what was not**, and why — at minimum: the front draws no confirmation dialog and no
  history yet (Phase 6), and the skill list the master edits **decides nothing** because the
  chain of tests is not in code;
- whether `./dev-checkout.sh feat/combat-engine-phase-5` is needed for manual validation. It
  **is**, if anything from step 3 could not be curled for real.

---

## Out of scope — do not build these here

- **NPC rostering.** Both done-criteria are reachable with two player clients. It is its own
  slice, before **Phase 6** — the front is what cannot draw a table without NPCs.
- **The chain of tests in code.** Written in `docs/game/combate/acoes.md`, absent from the
  engine, and explicitly not this phase's.
- **The override of the area chain's outcome.** Game rule not yet written;
  `combat-engine.md` marks the spot without describing it.
- **`buildMasterAction`'s `Move` and `Attack` TODOs.** Pending the front contract. The TODO
  comments stay.
- **The percept exception** at the start of battle. Blocked — the mental sub-attributes do not
  exist.
- **Stances; the armour model** (`ChainState.Reduce` keeps `const armour = 0` on purpose).
- **Any front work.**
