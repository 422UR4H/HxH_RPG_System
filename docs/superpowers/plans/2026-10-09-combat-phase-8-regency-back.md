# Fase 8 — Regência — pacote de back — plano de implementação

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development
> (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** fazer a edição do mestre (`edit_action`) mudar de fato o desfecho em toda rolagem que a
resolução lê, trocar a perícia que mede o dano, mandar ao mestre a condição em vigor de cada
rolagem (sobrevivendo a recarregar), tratar a condição zerada como "sem condição", e escrever tudo
isso no contrato e no documento mestre.

**Architecture:** a condição continua morando dentro da action (`RollCheck.Context.Condition`); o
resolvedor passa a lê-la em todas as derivações. Dois campos novos no domínio, **fora do wire da
action**: `Action.DefaultDefense` (o portador da condição da defesa padrão de uma reação, com coluna
própria) e `Attack.DamageSkill` (a perícia do dano, dentro do JSON de `actions.attack`). A
`TurnResolution` ganha `DamageSkill` (público, persistido) e `Conditions` (só com turno aberto, só
mestre, não persistido) — e como o `match_full_state.resolution` do mestre já é `ResolveTurn(t)`,
a reconexão herda os dois sem código novo em `room.go`.

**Tech Stack:** Go 1.23, gorilla/websocket, pgx/v5, goose, `testing` padrão.

**Spec:** [`docs/superpowers/specs/2026-10-09-combat-phase-8-regency-back-design.md`](../specs/2026-10-09-combat-phase-8-regency-back-design.md)
— leia o spec inteiro antes da primeira tarefa. Cada tarefa cita a seção do spec que implementa.
Documento mestre: `docs/superpowers/specs/2026-09-20-front-combat-phases.md` §8, §6A.6 F7, §4.6,
§11.1. Contrato: `docs/dev/api/match-combat-ws.md`.

**Branch:** `feat/combat-phase-8-regency-back` (já criada a partir de `main` em `74fb5c0`).

## Global Constraints

- Go 1.23; `testing` padrão, table-driven com `t.Run`; mantenha o pacote de teste que cada arquivo já usa (`action_mapper_test.go` e `resolution_payload_test.go` são `package game`; os `*_e2e_test.go` são `package game_test`; `edit_action_test.go` é `package match_test`; `reaction_collision_test.go` e `turn_resolver_test.go` são `package service_test`; `actor_push_test.go` é `package service`). **TDD** — o teste vem antes.
- `room.go` é dono do lock (`r.mu`). **Nada que envia a cliente roda com `r.mu` preso.** Este plano não deveria precisar tocar `room.go`; se uma tarefa precisar, pare e diga por quê.
- `MatchSession` não tem lock: todo acesso à sessão é sob `r.mu` (os testes de `application/match` chamam a sessão direto, sem `Room`, e isso é o padrão deles).
- Wire em **camelCase**. Campo opcional novo de payload leva `omitempty`.
- **Nada novo vai ao wire da action** (`internal/app/wire/actionwire`). `DefaultDefense`, `DamageSkill` e `RollCheck.Context` **não** são mapeados lá — é isso que impede o jogador de ver a edição antes do fechamento (spec §3.2, §3.4, §5).
- **Nunca remover comentários `TODO`.** Comentários explicam o porquê, no tom e na densidade dos vizinhos (este repo comenta bastante; siga o vizinho).
- Todo item que muda o wire atualiza o contrato (`docs/dev/api/match-combat-ws.md` ou `match-history.md`) **no mesmo commit**. Migração nova entra em `docs/documentation-map.yaml` no mesmo commit.
- Verificação por tarefa: `go build ./...`, `go vet ./...`, `go vet -tags integration ./...`, `go vet -tags smoke ./...`, `go test ./internal/...`. Tarefa que toca `internal/app/game/`: também `go test -race ./internal/app/game/`. Tarefa que toca `internal/gateway/pg/`: também `go test -tags=integration -p 1 ./internal/gateway/pg/...` (banco em `TEST_DATABASE_URL`, padrão `postgres://postgres:postgres@localhost:5432/hxh_rpg_test?sslmode=disable`; aplique a migração nova no banco de teste antes — `goose -dir migrations postgres "$TEST_DATABASE_URL" up`).
- Commits terminam com a linha `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.
- Docs em PT-BR; nomes de código em inglês.
- **Nada que uma projeção esconde é descartado na gravação** (regra da Fase 7): o dado vai inteiro para o banco e é escondido só na leitura.
- **A edição muda o desfecho, nunca a economia.** Nenhuma tarefa toca barras, `Speeds`, ordem ou `deriveSpeeds`.

## Review Focus

1. **O jogador nunca vê a edição antes do fechamento** — nem no `resolution_updated`, nem no `match_full_state` ao reconectar (`openTurn.action`/`openTurn.reactions` não podem trazer `Grab`, a descrição da condição, nem um componente `defense` que não estava lá). Teste em T6 (e2e, jogador reconectando).
2. **O mestre que recarrega vê o que editou** — `match_full_state.resolution.conditions` e `damageSkill` iguais ao último `resolution_updated`. Teste em T6 (e2e, mestre reconectando).
3. **Editar de volta apaga a captura** — condição zerada (T1) e `damageSkill` de volta a `"Push"` (T5): `overridden_action_values` fica sem linha, e nada é gravado no fechamento. Testes em T1 e T5.
4. **Uma recusa não deixa meia edição** — viés no dano (T4) e `damageSkill` numa ação sem ataque (T5), mandados junto com uma condição válida no mesmo payload: nada muda, nada é capturado. Testes em T4 e T5.
5. **`defense` cai no lugar certo** — numa reação `dodge` vai para `DefaultDefense` e muda a defesa padrão; numa `repel` é recusado; numa action comum segue o `Defense` declarado, como hoje. Teste em T3.

---

### Task 0 (orquestrador, antes de despachar): o tipo de agente implementador

O effort de um subagente vem da definição do tipo de agente, não do despacho (documento mestre
§0.1). Crie `.claude/agents/implementer.md` **neste repo** (o front já tem o dele):

```markdown
---
name: implementer
description: Implementa UMA tarefa de um plano em docs/superpowers/plans/ do back (System_X_System), com TDD, e reporta o que fez.
model: sonnet
effort: medium
---

Você implementa exatamente uma tarefa de um plano em `docs/superpowers/plans/`. Leia a tarefa
inteira, o spec que o plano cita e os arquivos listados antes de editar. Siga TDD: teste
falhando, código mínimo, teste passando. Rode os comandos de verificação da seção "Global
Constraints" do plano. Não toque em arquivo fora da lista da tarefa sem dizer por quê no
relatório. Nunca remova comentários TODO. Commit ao fim, com a mensagem da tarefa. Relate: o que
mudou, a saída dos testes, e qualquer desvio do plano.
```

Commit: `chore: tipo de agente implementador para o back`. As tarefas marcadas **opus** são
despachadas com `model: opus` por cima do tipo.

---

### Task 1: condição zerada é "sem condição" (spec §3.6, decisão 4; contrato, decisão 5)

**Files:**
- Modify: `internal/domain/match/matchsession/match_session.go` (o laço `for _, edit := range ma.Conditions` que **aplica**, depois da passada de validação, em `ApplyMasterAction`, ~linha 296)
- Test: `internal/application/match/edit_action_test.go` (`TestOverrideCapture`)
- Docs: `docs/dev/api/match-combat-ws.md` (seção `edit_action`)

**Interfaces:**
- Consumes: `captureOverride(actionID, field, origin, masterUUID, current, incoming any)` e `valuesEqual` (`match_session.go`).
- Produces: depois de `ApplyMasterAction`, uma rolagem editada com `RollCondition{}` tem `Context.Condition == nil`.

- [ ] **Step 1: Write the failing tests** — dentro de `TestOverrideCapture`, depois do subteste `"editing back to the original erases the capture"`:

```go
	t.Run("a zeroed condition is no condition: editing back erases the capture", func(t *testing.T) {
		// The wire cannot say "no condition" — the closest it can send is
		// {bias: 0, modifier: 0, description: ""}. That has to read as the original nil, or
		// cancelling an edit would leave a row claiming the master changed something.
		f := newOpenAttackFixture(t)
		f.editHitModifier(t, 3)
		if got := len(f.session.PeekOverridesFor(f.openTurn())); got != 1 {
			t.Fatalf("captured %d values after the real edit, want 1", got)
		}

		f.editHitModifier(t, 0)

		if got := len(f.session.PeekOverridesFor(f.openTurn())); got != 0 {
			t.Fatalf("captured %d values after editing back to zero, want 0", got)
		}
		if c := f.openTurn().ActionRef().Attack.Hit.Context.Condition; c != nil {
			t.Fatalf("hit condition = %+v, want nil — a zeroed condition is no condition", *c)
		}
		if got := f.currentResolution(t).ActionResult.Total; got != f.primaryTotal {
			t.Fatalf("Total = %d, want the untouched %d", got, f.primaryTotal)
		}
	})

	t.Run("a zeroed first edit captures nothing and stores nothing", func(t *testing.T) {
		f := newOpenAttackFixture(t)
		f.editHitModifier(t, 0)

		if got := len(f.session.PeekOverridesFor(f.openTurn())); got != 0 {
			t.Fatalf("captured %d values for an edit that displaced nothing, want 0", got)
		}
		if c := f.openTurn().ActionRef().Attack.Hit.Context.Condition; c != nil {
			t.Fatalf("hit condition = %+v, want nil", *c)
		}
	})
```

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/application/match/ -run 'TestOverrideCapture' -v`
Expected: FAIL — o primeiro com `captured 1 values after editing back to zero, want 0`; o segundo com `captured 1 values for an edit that displaced nothing` (o `RollCondition{}` não é igual ao `nil` original).

- [ ] **Step 3: Implement** — no laço que aplica (não na passada de validação), troque o trecho de `cond := edit.Condition` até `rc.Context.Condition = &cond` por:

```go
		cond := edit.Condition
		// A zeroed condition IS "no condition". The wire cannot send nil — the closest a client
		// can say is {bias: 0, modifier: 0, description: ""} — and the original a first edit
		// captures is nil (the player sends no condition). Without this, cancelling an edit by
		// editing back would compare RollCondition{} to nil, keep the capture, and write a row
		// at the close claiming the master changed a test they had put back. incoming stays an
		// UNTYPED nil for the same reason current does above.
		var incoming any
		if cond != (action.RollCondition{}) {
			incoming = cond
		}
		s.captureOverride(target.GetID(), conditionFieldKey(edit), match.OriginPlayer, masterUUID,
			current, incoming)
		if incoming == nil {
			rc.Context.Condition = nil
		} else {
			rc.Context.Condition = &cond
		}
```

- [ ] **Step 4: Run to verify they pass**

Run: `go test ./internal/application/match/ -v -run 'TestOverrideCapture|TestEditActionRollCondition'` — PASS. Depois a verificação inteira da seção Global Constraints.

- [ ] **Step 5: Contrato** — em `docs/dev/api/match-combat-ws.md`, seção `edit_action`, troque o parágrafo de abertura:

> Edita a ação do turno aberto, ou uma das reações dela. Toda seção é opcional e independente
> — mande só o que muda. **Uma seção presente SUBSTITUI a lista inteira**: não há merge
> parcial, porque o wire não tem identidade por entrada.

por:

> Edita a ação do turno aberto, ou uma das reações dela. Toda seção é opcional e independente
> — mande só o que muda.
>
> - **`skills` e `targetIds` substituem a lista inteira**: não há merge parcial, porque o wire não
>   tem identidade por entrada.
> - **`conditions` se aplica por rolagem.** Cada entrada nomeia uma rolagem (`field` ou
>   `skillName`) e substitui a condição **daquela** rolagem; as rolagens que não aparecem ficam
>   como estão.
> - **Uma entrada zerada limpa.** `bias`, `modifier` e `description` ausentes ou zero/vazios
>   (`{ "field": "hit" }`) devolvem a rolagem a "sem condição" — e, se a edição anterior tinha sido
>   capturada, apagam a captura: é assim que o mestre **cancela** uma edição. Não há verbo de
>   confirmação nem de cancelamento.

- [ ] **Step 6: Commit**

```bash
git add internal/domain/match/matchsession/match_session.go internal/application/match/edit_action_test.go docs/dev/api/match-combat-ws.md
git commit -m "fix(match): condição zerada é sem condição, e editar de volta apaga a captura

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 2: as reações leem a condição do mestre — Reflexo, Evasion, aparo (spec §3.1, decisão 6)

**Files:**
- Modify: `internal/domain/match/service/reaction_collision.go` (`deriveReflex`, `deriveEvasion`, `resolveRepel`)
- Test: `internal/domain/match/service/reaction_collision_test.go`

**Interfaces:**
- Consumes: `RollInput.Condition` (`roll_calculator.go`) — `Derive` já soma `Bias`/`Modifier` quando não-nil; numa leitura passiva ignora o `Bias`.
- Produces: `ResolveReaction` passa a ler `in.Reaction.Dodge.Context.Condition`, a `Context.Condition` da entrada `Evasion` de `in.Reaction.Skills`, e `in.Reaction.Repel.Context.Condition`.

- [ ] **Step 1: Write the failing test** — no fim de `reaction_collision_test.go`:

```go
// TestResolveReaction_ReadsTheMastersCondition pins the bug AGENTS.md carried as "known": the
// master's edit_action on a reaction's dodge, Evasion or repel was accepted, captured, and
// never read. A plain sheet has every skill at 0, so a rolled test is just its dice plus the
// condition.
func TestResolveReaction_ReadsTheMastersCondition(t *testing.T) {
	t.Run("dodge: the modifier moves the reflex", func(t *testing.T) {
		r := reactionWith(action.ReactDodge, []int{5, 5}, nil, nil)
		r.Dodge.Context.Condition = &action.RollCondition{Modifier: 4}
		out := service.ResolveReaction(reactionInput(t, action.ReactDodge, r, 30))
		if out.Dodge.Total != 14 {
			t.Fatalf("Dodge.Total = %d, want 14 (dice 10 + modifier 4)", out.Dodge.Total)
		}
	})

	t.Run("dodge: advantage reads the secondary set that already fell", func(t *testing.T) {
		r := reactionWith(action.ReactDodge, []int{2, 2}, nil, nil)
		r.Dodge.Attempts.Secondary = []int{9, 9}
		r.Dodge.Context.Condition = &action.RollCondition{Bias: 1}
		out := service.ResolveReaction(reactionInput(t, action.ReactDodge, r, 30))
		if out.Dodge.Total != 18 {
			t.Fatalf("Dodge.Total = %d, want 18 (the better set)", out.Dodge.Total)
		}
	})

	t.Run("repel: the modifier moves the repel total", func(t *testing.T) {
		r := reactionWith(action.ReactRepel, nil, nil, []int{6, 6})
		r.Repel.Context.Condition = &action.RollCondition{Modifier: -3}
		out := service.ResolveReaction(reactionInput(t, action.ReactRepel, r, 30))
		if out.Repel.Total != 9 {
			t.Fatalf("Repel.Total = %d, want 9 (dice 12 − 3)", out.Repel.Total)
		}
	})

	t.Run("closed dodge: the Evasion condition moves Evasion, and the reserve with it", func(t *testing.T) {
		// Reflex 18, Evasion 10 + 3 = 13: the dodge is the worse (13) and the reserve is the
		// gap between the two READINGS — 5, not the 8 the bare dice would give (spec D4).
		r := reactionWith(action.ReactClosedDodge, []int{9, 9}, []int{5, 5}, nil)
		r.Skills[0].Context.Condition = &action.RollCondition{Modifier: 3}
		out := service.ResolveReaction(reactionInput(t, action.ReactClosedDodge, r, 30))
		if out.Evasion.Total != 13 {
			t.Fatalf("Evasion.Total = %d, want 13", out.Evasion.Total)
		}
		if out.Dodge.Total != 13 {
			t.Fatalf("Dodge.Total = %d, want 13 (the worse of 18 and 13)", out.Dodge.Total)
		}
		if len(out.Payouts) != 1 || out.Payouts[0].Amount != 5 {
			t.Fatalf("Payouts = %+v, want one reserve of 5", out.Payouts)
		}
	})
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/domain/match/service/ -run TestResolveReaction_ReadsTheMastersCondition -v`
Expected: FAIL nos quatro subtestes (os totais saem sem a condição: 10, 4, 12, 10).

- [ ] **Step 3: Implement** — em `reaction_collision.go`:

`deriveReflex`: troque o bloco de `attempts` e acrescente `Condition` ao `RollInput`:

```go
	passive := in.Reaction == nil
	var attempts action.RollAttempts
	var cond *action.RollCondition
	if !passive && in.Reaction.Dodge != nil {
		attempts = in.Reaction.Dodge.Attempts
		// The master's edit on this reaction's dodge (edit_action, field "dodge"). Nil reads
		// neutral. A passive dodge has no reaction and therefore nowhere to carry one — the
		// passive of a target that did not react is not editable yet (front-combat-phases.md).
		cond = in.Reaction.Dodge.Context.Condition
	}
	return calc.Derive(in.Rules, attempts, RollInput{
		SkillName:  enum.Reflex.String(),
		SkillValue: skillValueOf(in.Target, enum.Reflex.String()),
		Passive:    passive,
		Condition:  cond,
```

(o resto dos campos do `RollInput` fica como está).

`deriveEvasion`: no laço que acha a entrada `Evasion`, guarde também a condição e passe-a:

```go
	var attempts action.RollAttempts
	var cond *action.RollCondition
	if in.Reaction != nil {
		for _, s := range in.Reaction.Skills {
			if s.SkillName == enum.Evasion.String() {
				attempts = s.Attempts
				// edit_action's skillName "Evasion". The master's panel does not offer it
				// (front-combat-phases.md §8 — the test chain will redesign it), but the engine
				// reads every condition it accepts.
				cond = s.Context.Condition
				break
			}
		}
	}
```

e `Condition: cond,` no `RollInput` dele.

`resolveRepel`: idem:

```go
	var attempts action.RollAttempts
	var cond *action.RollCondition
	if in.Reaction != nil && in.Reaction.Repel != nil {
		attempts = in.Reaction.Repel.Attempts
		cond = in.Reaction.Repel.Context.Condition
	}
	out.Repel = calc.Derive(in.Rules, attempts, RollInput{
		SkillName:  enum.Repel.String(),
		SkillValue: skillValueOf(in.Target, enum.Repel.String()),
		Condition:  cond,
		AgainstID:  &in.AttackerID,
	})
```

- [ ] **Step 4: Run to verify it passes** — o teste novo e `go test ./internal/domain/match/...`; depois a verificação inteira.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/match/service/reaction_collision.go internal/domain/match/service/reaction_collision_test.go
git commit -m "fix(combat): a reação lê a condição do mestre no reflexo, na evasão e no aparo

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 3: o portador da defesa padrão — `Action.DefaultDefense` (spec §3.2, decisão 10)

**Files:**
- Modify: `internal/domain/match/entity/action/action.go` (struct `Action`)
- Modify: `internal/domain/match/matchsession/match_session.go` (`resolveRollCheck`, `case action.FieldDefense`)
- Modify: `internal/domain/match/service/reaction_collision.go` (`ResolveReaction`, a derivação da defesa padrão)
- Create: `migrations/20261009000000_actions_default_defense.sql`
- Modify: `internal/gateway/pg/round/persist_turn_close.go` (`insertAction`)
- Test: `internal/domain/match/service/reaction_collision_test.go`, `internal/application/match/edit_action_test.go`, `internal/gateway/pg/round/round_integration_test.go`
- Docs: `docs/documentation-map.yaml` (entrada da migração)

**Interfaces:**
- Consumes: `ReactionKind.KeepsDefault()` (`reaction_kind.go`), `session.AttachReaction(playerUUID, *action.Action)`, `session.OpenReaction(reactionID)`.
- Produces: `Action.DefaultDefense *RollCheck`. `resolveRollCheck(a, ConditionEdit{Field: FieldDefense})` numa reação com `ReactionKind != "" && KeepsDefault()` devolve `a.DefaultDefense`, criando `&RollCheck{SkillName: "Defense"}` se nil. `ResolveReaction` lê `in.Reaction.DefaultDefense.Context.Condition` na defesa padrão. Coluna `actions.default_defense JSONB NULL`. T6 lê `DefaultDefense` para listar as condições.

- [ ] **Step 1: Failing service test** — em `reaction_collision_test.go`:

```go
func TestResolveReaction_TheDefaultDefenseReadsItsCarrier(t *testing.T) {
	// A dodge that fails (2 vs a hit of 30) falls back on the default defense, which is
	// always passive: 0 skill + 11. The master's +4 on it lives in DefaultDefense.
	r := reactionWith(action.ReactDodge, []int{1, 1}, nil, nil)
	before := service.ResolveReaction(reactionInput(t, action.ReactDodge, r, 30)).Defense.Total

	r.DefaultDefense = &action.RollCheck{
		SkillName: enum.Defense.String(),
		Context:   action.RollContext{Condition: &action.RollCondition{Modifier: 4}},
	}
	after := service.ResolveReaction(reactionInput(t, action.ReactDodge, r, 30)).Defense.Total
	if after != before+4 {
		t.Fatalf("Defense.Total = %d, want %d (the passive %d + 4)", after, before+4, before)
	}
}
```

- [ ] **Step 2: Failing application tests** — em `edit_action_test.go`, um helper e um teste. O helper anexa uma reação da vítima ao turno aberto de `newOpenAttackFixture` (todos os personagens são de `f.playerUUID`; os 4 dados reservados do fixture cobrem o `Dodge` de uma reação `dodge` — **não** use tipos fechados aqui, eles rolariam Evasion além do roteiro):

```go
// attachReaction attaches a reaction of the given kind from the victim to the fixture's open
// attack and returns its ID. Free mode rolls no speed; a dodge rolls its 2D10 Dodge (4 faces —
// exactly the fixture's reserved budget), a repel its 2D10 Repel.
func (f *editFixture) attachReaction(t *testing.T, kind action.ReactionKind) uuid.UUID {
	t.Helper()
	r := action.NewAction(
		f.victimID, []uuid.UUID{f.attackerID}, f.actionID, nil,
		action.ActionSpeed{RollCheck: action.RollCheck{SkillName: enum.Legerity.String()}},
		nil, nil, nil, nil, nil, nil, nil,
	)
	r.ReactionKind = kind
	if kind == action.ReactRepel {
		r.Repel = &action.Repel{RollCheck: action.RollCheck{SkillName: enum.Repel.String()}}
	} else {
		r.Dodge = &action.Dodge{RollCheck: action.RollCheck{SkillName: enum.Reflex.String()}}
	}
	if _, err := f.session.AttachReaction(f.playerUUID, r); err != nil {
		t.Fatalf("AttachReaction(%s): %v", kind, err)
	}
	return r.GetID()
}

// reactionRef reads one reaction of the open turn by ID, or fails the test.
func (f *editFixture) reactionRef(t *testing.T, id uuid.UUID) *action.Action {
	t.Helper()
	r := f.openTurn().ReactionRef(id)
	if r == nil {
		t.Fatalf("reaction %v is not on the open turn", id)
	}
	return r
}

func TestEditActionDefaultDefense(t *testing.T) {
	editDefense := func(t *testing.T, f *editFixture, id uuid.UUID, modifier int) error {
		t.Helper()
		ma := action.NewMasterAction()
		ma.ActionID = id
		ma.Conditions = []action.ConditionEdit{{
			Field: action.FieldDefense, Condition: action.RollCondition{Modifier: modifier},
		}}
		_, err := match.NewEditActionUC().Execute(
			context.Background(), f.session, f.masterUUID, f.masterUUID, ma, nil)
		return err
	}

	t.Run("on a dodge reaction it lands on the default defense, and the resolution reads it", func(t *testing.T) {
		f := newOpenAttackFixture(t)
		id := f.attachReaction(t, action.ReactDodge)
		if _, _, err := f.session.OpenReaction(id); err != nil {
			t.Fatalf("OpenReaction: %v", err)
		}
		// Push the hit past the rolled dodge (6+7 = 13) so the default defense is reached.
		f.editHitModifier(t, 20)
		before := f.currentResolution(t).CharacterResults[0].Defense.Total

		if err := editDefense(t, f, id, 4); err != nil {
			t.Fatalf("Execute: %v", err)
		}

		r := f.reactionRef(t, id)
		if r.DefaultDefense == nil || r.DefaultDefense.Context.Condition == nil ||
			r.DefaultDefense.Context.Condition.Modifier != 4 {
			t.Fatalf("DefaultDefense = %+v, want the +4 condition", r.DefaultDefense)
		}
		if r.Defense != nil {
			t.Fatal("the edit created a declared Defense component — that one travels on the wire")
		}
		if got := f.currentResolution(t).CharacterResults[0].Defense.Total; got != before+4 {
			t.Fatalf("Defense.Total = %d, want %d", got, before+4)
		}
	})

	t.Run("on a repel reaction it is refused — a repel gives the default defense up", func(t *testing.T) {
		f := newOpenAttackFixture(t)
		id := f.attachReaction(t, action.ReactRepel)
		if err := editDefense(t, f, id, 4); !errors.Is(err, matchsession.ErrConditionTargetMissing) {
			t.Fatalf("err = %v, want ErrConditionTargetMissing", err)
		}
		if f.reactionRef(t, id).DefaultDefense != nil {
			t.Fatal("a refused edit left a DefaultDefense behind")
		}
	})

	t.Run("on the turn's own action it keeps today's meaning: the declared Defense", func(t *testing.T) {
		f := newOpenAttackFixture(t) // a plain attack: no Defense declared
		if err := editDefense(t, f, f.actionID, 4); !errors.Is(err, matchsession.ErrConditionTargetMissing) {
			t.Fatalf("err = %v, want ErrConditionTargetMissing", err)
		}
	})
}
```

Se `AttachReaction` exigir algo que o helper não manda (ver `match_session.go:1002`), ajuste o helper — não o teste.

- [ ] **Step 3: Run to verify they fail**

Run: `go test ./internal/domain/match/service/ -run TestResolveReaction_TheDefaultDefense -v` e `go test ./internal/application/match/ -run TestEditActionDefaultDefense -v`
Expected: falha de compilação (`DefaultDefense` não existe).

- [ ] **Step 4: Implement the domain**

`action.go`, no struct `Action`, logo depois de `Interact *Interact`:

```go
	// DefaultDefense carries the master's condition on the DEFAULT defense — the passive one
	// that stands behind a reaction whose kind keeps it (ReactionKind.KeepsDefault: dodge,
	// closedDodge, escapeGuard). It exists only on such a reaction, and only once the master
	// edited its "defense" (resolveRollCheck creates it). It has no dice: the default defense
	// is passive, so only the condition's Modifier ever moves it.
	//
	// It is NOT Defense. Defense is a component a player may DECLARE, the resolver never reads
	// it, and it travels on the action wire to everyone — a reaction that suddenly carried one
	// would tell the table the master touched its defense before the turn closed. This field is
	// mapped by no wire, the same way RollCheck.Context is not.
	DefaultDefense *RollCheck
```

`match_session.go`, `resolveRollCheck`, troque o `case action.FieldDefense:` por:

```go
	case action.FieldDefense:
		// On a reaction that keeps the default defense, "defense" names THAT one — the passive
		// the resolver actually reads — and its carrier is created on first edit. Everywhere
		// else it keeps naming the declared Defense component, as before.
		if a.ReactionKind != "" && a.ReactionKind.KeepsDefault() {
			if a.DefaultDefense == nil {
				a.DefaultDefense = &action.RollCheck{SkillName: enum.Defense.String()}
			}
			return a.DefaultDefense, nil
		}
		if a.Defense == nil {
			return nil, ErrConditionTargetMissing
		}
		return &a.Defense.RollCheck, nil
```

(A passada de validação chama `resolveRollCheck(&shadow, edit)`; `shadow` é cópia rasa, então criar o portador nela não toca a reação real — confira que o teste do repel continua sem `DefaultDefense`.)

`reaction_collision.go`, em `ResolveReaction`, troque a derivação da defesa padrão por:

```go
	// The master's condition on the default defense lives on the reaction's DefaultDefense
	// (edit_action, field "defense"). A target that did not react has no reaction to carry it
	// — that passive is not editable yet (front-combat-phases.md, pendências).
	var defenseCond *action.RollCondition
	if in.Reaction != nil && in.Reaction.DefaultDefense != nil {
		defenseCond = in.Reaction.DefaultDefense.Context.Condition
	}
	out.Defense = calc.Derive(in.Rules, action.RollAttempts{}, RollInput{
		SkillName:  enum.Defense.String(),
		SkillValue: skillValueOf(in.Target, enum.Defense.String()),
		Passive:    true,
		Condition:  defenseCond,
	})
```

- [ ] **Step 5: Run domain tests** — os dois testes novos passam; `go test ./internal/domain/... ./internal/application/...` verdes.

- [ ] **Step 6: Failing integration test** — em `round_integration_test.go`:

```go
func TestPersistTurnCloseWritesTheDefaultDefense(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	pgtest.TruncateAll(t, pool)
	repo := roundrepo.NewRepository(pool)
	fx := seedMatchAndSheets(t, pool)

	act := buildAttackAction(t, fx.attackerSheet, fx.victimSheet)
	tn := turnentity.NewTurn(*act)

	edited := action.NewAction(fx.victimSheet, nil, act.GetID(), nil, action.ActionSpeed{},
		nil, nil, nil, nil, nil, nil, nil)
	edited.ReactionKind = action.ReactDodge
	edited.DefaultDefense = &action.RollCheck{
		SkillName: "Defense",
		Context:   action.RollContext{Condition: &action.RollCondition{Modifier: 4, Description: "cansado"}},
	}
	tn.AddReaction(edited)
	tn.Close(time.Now())

	if err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
		Scene: fx.scene, Round: fx.round, Turn: tn, Action: act, MatchUUID: fx.matchUUID,
	}); err != nil {
		t.Fatalf("PersistTurnClose: %v", err)
	}

	var reactionCol, actionCol []byte
	if err := pool.QueryRow(ctx,
		`SELECT default_defense FROM actions WHERE uuid = $1`, edited.GetID()).Scan(&reactionCol); err != nil {
		t.Fatalf("read reaction default_defense: %v", err)
	}
	if !strings.Contains(string(reactionCol), "cansado") {
		t.Errorf("default_defense = %s, want the master's condition in it", reactionCol)
	}
	if err := pool.QueryRow(ctx,
		`SELECT default_defense FROM actions WHERE uuid = $1`, act.GetID()).Scan(&actionCol); err != nil {
		t.Fatalf("read action default_defense: %v", err)
	}
	if actionCol != nil {
		t.Errorf("default_defense = %s on an action with none, want SQL NULL", actionCol)
	}
}
```

(`strings` pode precisar entrar nos imports do arquivo.)

- [ ] **Step 7: Migration + insertAction**

`migrations/20261009000000_actions_default_defense.sql`:

```sql
-- +goose Up
-- +goose StatementBegin
BEGIN;

-- The master's condition on the DEFAULT defense of a reaction (Action.DefaultDefense, Phase 8):
-- the passive defense that stands behind a dodge, closedDodge or escapeGuard. It is its own
-- column, not the defense one, because that one is a component a player declares and the wire
-- carries; this one is the master's, and no surface shows it. NULL means the master did not
-- edit it — every plain action, every other reaction, and every row from before this column.
ALTER TABLE actions ADD COLUMN IF NOT EXISTS default_defense JSONB;

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

ALTER TABLE actions DROP COLUMN IF EXISTS default_defense;

COMMIT;
-- +goose StatementEnd
```

`persist_turn_close.go`, `insertAction`: depois do bloco `interactJSON`:

```go
	defaultDefenseJSON, err := marshalNullablePtr(act.DefaultDefense)
	if err != nil {
		return fmt.Errorf("marshal default defense: %w", err)
	}
```

e acrescente a coluna ao `INSERT` (vira `$22`):

```go
		`INSERT INTO actions
		 (uuid, turn_uuid, actor_uuid, react_to_uuid, target_ids, type,
		  speed, skills, move, attack, defense, dodge, repel, feint, trigger,
		  interact, system_bias, reaction_kind, created_at, move_views, consumed_action_ids,
		  default_defense)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)`,
		act.GetID(), turnID, act.GetActorID(), reactToUUID,
		targetIDs, deriveActionType(act),
		speedJSON, skillsJSON, moveJSON, attackJSON,
		defenseJSON, dodgeJSON, repelJSON, feintJSON, triggerJSON,
		interactJSON, act.SystemBias,
		reactionKind, createdAt, moveViewsJSON, consumedOrNil(act.ConsumedActionIDs),
		defaultDefenseJSON,
	)
```

O histórico **não** lê a coluna de volta (spec §3.2): nada a projeta.

`docs/documentation-map.yaml`, logo depois da entrada de `migrations/20261005000000_actions_consumed_action_ids.sql`:

```yaml
  - code_path: migrations/20261009000000_actions_default_defense.sql
    dev_docs:
      - path: docs/dev/match/combat-engine.md
        confidence: directly_affected
      - path: docs/dev/api/match-combat-ws.md
        confidence: possibly_affected
    notes: >-
      actions.default_defense JSONB NULL — Action.DefaultDefense, o portador da condição do mestre
      na defesa padrão de uma reação que a mantém (dodge, closedDodge, escapeGuard); criado no
      primeiro edit_action com field "defense" nessa reação. Fora de toda superfície de wire
      (não é o componente Defense declarado, que viaja); o histórico não o lê de volta. NULL = o
      mestre não editou. Fase 8, spec 2026-10-09 §3.2.
```

- [ ] **Step 8: Run** — aplique a migração no banco de teste e rode `go test -tags=integration -p 1 ./internal/gateway/pg/...`; depois a verificação inteira.

- [ ] **Step 9: Commit**

```bash
git add internal/domain/match/entity/action/action.go internal/domain/match/matchsession/match_session.go internal/domain/match/service/reaction_collision.go internal/domain/match/service/reaction_collision_test.go internal/application/match/edit_action_test.go migrations/20261009000000_actions_default_defense.sql internal/gateway/pg/round/persist_turn_close.go internal/gateway/pg/round/round_integration_test.go docs/documentation-map.yaml
git commit -m "feat(combat): a condição do mestre na defesa padrão de uma reação

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 4: o dano lê o ajuste do mestre, e recusa viés (spec §3.3, decisão 7)

**Files:**
- Modify: `internal/domain/match/service/turn_resolver.go` (`seedChain`, o ramo `TargetKindWallSegment` de `Resolve`, helper novo `rawDamage`)
- Modify: `internal/domain/match/matchsession/error.go`, `internal/domain/match/matchsession/match_session.go` (passada de validação de `ApplyMasterAction`)
- Test: `internal/application/match/edit_action_test.go`
- Docs: `docs/dev/api/match-combat-ws.md` (`edit_action` → erros; tabela de erros do fim, linha do `game_error`/`edit_action` se houver)

**Interfaces:**
- Consumes: `RawDamage(dice, weapon, catalogue, push)` (`damage.go`), `floorZero` (`attack_chain.go`, mesmo pacote), `tr.actorPush(in, a)`.
- Produces: `func (tr TurnResolver) rawDamage(in ResolveInput, a action.Action) (int, error)` — T5 troca o `actorPush` de dentro dele. `matchsession.ErrDamageHasNoAdvantage`.

- [ ] **Step 1: Failing tests** — em `edit_action_test.go`:

```go
func TestEditActionDamageCondition(t *testing.T) {
	editDamage := func(t *testing.T, f *editFixture, cond action.RollCondition, extra ...action.ConditionEdit) error {
		t.Helper()
		ma := action.NewMasterAction()
		ma.ActionID = f.actionID
		ma.Conditions = append(extra, action.ConditionEdit{Field: action.FieldDamage, Condition: cond})
		_, err := match.NewEditActionUC().Execute(
			context.Background(), f.session, f.masterUUID, f.masterUUID, ma, nil)
		return err
	}
	// The fixture's hit (7) does not beat the victim's passive reflex (11): the blow would be
	// avoided and RawDamage would read 0 regardless. Lift the hit first.
	landed := func(t *testing.T) *editFixture {
		t.Helper()
		f := newOpenAttackFixture(t)
		f.editHitModifier(t, 20)
		return f
	}

	t.Run("the modifier moves the raw damage", func(t *testing.T) {
		f := landed(t)
		before := f.currentResolution(t).CharacterResults[0].RawDamage
		if err := editDamage(t, f, action.RollCondition{Modifier: 5}); err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if got := f.currentResolution(t).CharacterResults[0].RawDamage; got != before+5 {
			t.Fatalf("RawDamage = %d, want %d", got, before+5)
		}
	})

	t.Run("the raw damage floors at zero", func(t *testing.T) {
		f := landed(t)
		if err := editDamage(t, f, action.RollCondition{Modifier: -1000}); err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if got := f.currentResolution(t).CharacterResults[0].RawDamage; got != 0 {
			t.Fatalf("RawDamage = %d, want 0", got)
		}
	})

	t.Run("bias on the damage is refused, and nothing in the payload lands", func(t *testing.T) {
		f := newOpenAttackFixture(t)
		hit := action.ConditionEdit{Field: action.FieldHit, Condition: action.RollCondition{Modifier: 3}}
		err := editDamage(t, f, action.RollCondition{Bias: 1}, hit)
		if !errors.Is(err, matchsession.ErrDamageHasNoAdvantage) {
			t.Fatalf("err = %v, want ErrDamageHasNoAdvantage", err)
		}
		a := f.openTurn().ActionRef()
		if a.Attack.Damage.Context.Condition != nil || a.Attack.Hit.Context.Condition != nil {
			t.Fatal("a refused edit left a condition behind")
		}
		if got := len(f.session.PeekOverridesFor(f.openTurn())); got != 0 {
			t.Fatalf("captured %d values for a refused edit, want 0", got)
		}
	})
}
```

- [ ] **Step 2: Run to verify they fail** — `go test ./internal/application/match/ -run TestEditActionDamageCondition -v`: compilação falha (`ErrDamageHasNoAdvantage`).

- [ ] **Step 3: Implement**

`error.go`, junto de `ErrAmbiguousConditionEdit`:

```go
	// ErrDamageHasNoAdvantage means a condition edit set a bias on the damage. Damage rolls a
	// single set of dice (rollActionDice: "damage has no advantage"), so there is no second set
	// for a bias to choose — accepting it would store something no reading can ever use. The
	// flat modifier is accepted.
	ErrDamageHasNoAdvantage = errors.New("damage has no advantage: a damage condition takes no bias")
```

`match_session.go`, `ApplyMasterAction`, no laço de validação (`for _, edit := range ma.Conditions { if _, err := resolveRollCheck(&shadow, edit) ...`), **antes** de `resolveRollCheck`:

```go
		if edit.Field == action.FieldDamage && edit.Condition.Bias != 0 {
			return nil, ErrDamageHasNoAdvantage
		}
```

`turn_resolver.go`: helper novo, logo depois de `actorPush`:

```go
// rawDamage is the attack's raw damage as the chain and a wall read it: RawDamage (the
// weapon's dice, its flat bonus and the skill that measures damage) plus the master's flat
// adjustment on the damage (edit_action, field "damage"). Only the Modifier: damage rolls one
// set of dice, so a bias has nothing to choose, and the session refuses one.
//
// Floored at zero. A cut bigger than the blow would otherwise seed the chain with a negative
// residual; EffectiveDamage and the chain's own floorZero already floor every later step, and
// this is the same floor one step earlier.
func (tr TurnResolver) rawDamage(in ResolveInput, a action.Action) (int, error) {
	raw, err := RawDamage(a.Attack.Damage.Attempts.Primary, a.Attack.Weapon, in.Weapons, tr.actorPush(in, a))
	if err != nil {
		return 0, err
	}
	if c := a.Attack.Damage.Context.Condition; c != nil {
		raw = floorZero(raw + c.Modifier)
	}
	return raw, nil
}
```

Troque as duas chamadas diretas: em `seedChain`, `raw, err := tr.rawDamage(in, a)`; no ramo de parede de `Resolve`, `raw, err := tr.rawDamage(in, a)` (o `if err != nil { raw = 0 }` fica).

- [ ] **Step 4: Run** — o teste passa; `go test ./internal/domain/match/... ./internal/application/...`; verificação inteira.

- [ ] **Step 5: Contrato** — `match-combat-ws.md`, `edit_action`: na tabela de campos, linha de `bias`, acrescente: "**Recusado no dano** (`field: "damage"`): o dano rola um conjunto só de dados, não há o que escolher — `game_error` `damage has no advantage: a damage condition takes no bias`. O `modifier` no dano soma ao dano bruto (piso zero) e vale para toda a cadeia de alvos." Na lista **Erros** da seção, acrescente esse `game_error`.

- [ ] **Step 6: Commit**

```bash
git add internal/domain/match/service/turn_resolver.go internal/domain/match/matchsession/error.go internal/domain/match/matchsession/match_session.go internal/application/match/edit_action_test.go docs/dev/api/match-combat-ws.md
git commit -m "fix(combat): o dano lê o ajuste do mestre e recusa viés

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 5: a perícia do dano — `edit_action.damageSkill` (spec §3.4, decisão 2)

**Files:**
- Modify: `internal/domain/match/entity/action/attack.go` (campo `DamageSkill`, método `EffectiveDamageSkill`)
- Modify: `internal/domain/match/entity/action/master_action.go` (campo `DamageSkill`)
- Modify: `internal/domain/match/service/turn_resolver.go` (`actorPush` → `actorDamageSkill`; `TurnResolution.DamageSkill`; `Resolve` o preenche)
- Modify: `internal/domain/match/service/actor_push_test.go` (o rename)
- Modify: `internal/domain/match/matchsession/error.go`, `match_session.go` (`ApplyMasterAction`)
- Modify: `internal/application/match/edit_action.go` (`hasRollEdits`)
- Modify: `internal/app/game/message.go` (`EditActionPayload.DamageSkill`), `internal/app/game/action_mapper.go` (`buildEditAction`)
- Test: `internal/application/match/edit_action_test.go`, `internal/app/game/action_mapper_test.go`, `internal/domain/match/service/actor_push_test.go`
- Docs: `docs/dev/api/match-combat-ws.md` (`edit_action`; a linha de `attack.damage.skillName` ~200; erros)

**Interfaces:**
- Consumes: `enum.SkillNameFrom(string) (enum.SkillName, error)`, `match.OriginSystem`, `captureOverride`, `rawDamage` (T4).
- Produces: `Attack.DamageSkill enum.SkillName`; `func (a Attack) EffectiveDamageSkill() enum.SkillName`; `MasterAction.DamageSkill *enum.SkillName`; `TurnResolution.DamageSkill string`; `matchsession.ErrNoDamageToMeasure`; `EditActionPayload.DamageSkill *string` (`json:"damageSkill,omitempty"`). T6 põe `TurnResolution.DamageSkill` no wire.

- [ ] **Step 1: Failing tests**

`actor_push_test.go` (`package service`): troque toda chamada `actorPush(` por `actorDamageSkill(` e acrescente:

```go
// TestActorDamageSkill_ReadsTheMastersChoice: with the master's Grab on the attack, the damage
// is measured by the sheet's Grab, not its Push.
func TestActorDamageSkill_ReadsTheMastersChoice(t *testing.T) {
	actorID := uuid.New()
	cs, _ := pushedSheet(t)
	if err := cs.IncreaseExpForSkill(experience.NewUpgradeCascade(900), enum.Grab); err != nil {
		t.Fatalf("IncreaseExpForSkill(Grab): %v", err)
	}
	grab, err := cs.GetValueForTestOfSkill(enum.Grab)
	if err != nil {
		t.Fatalf("GetValueForTestOfSkill(Grab): %v", err)
	}
	a := action.NewAction(actorID, nil, uuid.Nil, nil, action.ActionSpeed{},
		nil, nil, &action.Attack{DamageSkill: enum.Grab}, nil, nil, nil, nil)
	in := ResolveInput{Sheets: map[uuid.UUID]*csSheet.CharacterSheet{actorID: cs}}

	if got := (TurnResolver{}).actorDamageSkill(in, *a); got != grab {
		t.Fatalf("actorDamageSkill() = %d, want the sheet's Grab %d", got, grab)
	}
}
```

`action_mapper_test.go` (`package game`):

```go
func TestBuildEditActionDamageSkill(t *testing.T) {
	grab := "Grab"
	ma, err := buildEditAction(EditActionPayload{DamageSkill: &grab})
	if err != nil {
		t.Fatalf("buildEditAction: %v", err)
	}
	if ma.DamageSkill == nil || *ma.DamageSkill != enum.Grab {
		t.Fatalf("DamageSkill = %v, want Grab", ma.DamageSkill)
	}

	unknown := "Telekinesis"
	if _, err := buildEditAction(EditActionPayload{DamageSkill: &unknown}); err == nil {
		t.Fatal("an unknown skill name was accepted")
	}

	ma, err = buildEditAction(EditActionPayload{})
	if err != nil {
		t.Fatalf("buildEditAction: %v", err)
	}
	if ma.DamageSkill != nil {
		t.Fatal("an absent damageSkill must stay nil — untouched")
	}
}
```

`edit_action_test.go` (acrescente o import `".../internal/domain/entity/character_sheet/experience"`):

```go
func TestEditActionDamageSkill(t *testing.T) {
	editSkill := func(t *testing.T, f *editFixture, id uuid.UUID, name enum.SkillName, extra ...action.ConditionEdit) error {
		t.Helper()
		ma := action.NewMasterAction()
		ma.ActionID = id
		ma.DamageSkill = &name
		ma.Conditions = extra
		_, err := match.NewEditActionUC().Execute(
			context.Background(), f.session, f.masterUUID, f.masterUUID, ma, nil)
		return err
	}
	// grabbing raises the attacker's Grab and lifts the hit past the passive dodge, so the
	// blow lands and RawDamage is read. Raising Grab can also move Push (they share an
	// attribute), so the test reads both off the sheet instead of assuming either.
	grabbing := func(t *testing.T) (f *editFixture, push, grab int) {
		t.Helper()
		f = newOpenAttackFixture(t)
		cs, err := f.session.GetCharSheet(f.attackerID)
		if err != nil {
			t.Fatalf("GetCharSheet: %v", err)
		}
		if err := cs.IncreaseExpForSkill(experience.NewUpgradeCascade(900), enum.Grab); err != nil {
			t.Fatalf("IncreaseExpForSkill(Grab): %v", err)
		}
		push, _ = cs.GetValueForTestOfSkill(enum.Push)
		grab, _ = cs.GetValueForTestOfSkill(enum.Grab)
		if grab == push {
			t.Fatal("fixture: Grab equals Push, the swap would prove nothing")
		}
		f.editHitModifier(t, 20)
		return f, push, grab
	}

	t.Run("an untouched attack is measured by Push", func(t *testing.T) {
		f := newOpenAttackFixture(t)
		if got := f.currentResolution(t).DamageSkill; got != enum.Push.String() {
			t.Fatalf("DamageSkill = %q, want Push", got)
		}
	})

	t.Run("Grab measures the damage, and the resolution says so", func(t *testing.T) {
		f, push, grab := grabbing(t)
		before := f.currentResolution(t).CharacterResults[0].RawDamage
		if err := editSkill(t, f, f.actionID, enum.Grab); err != nil {
			t.Fatalf("Execute: %v", err)
		}
		res := f.currentResolution(t)
		if res.DamageSkill != enum.Grab.String() {
			t.Fatalf("DamageSkill = %q, want Grab", res.DamageSkill)
		}
		if got, want := res.CharacterResults[0].RawDamage, before-push+grab; got != want {
			t.Fatalf("RawDamage = %d, want %d (Push %d swapped for Grab %d)", got, want, push, grab)
		}
	})

	t.Run("the capture keeps the original Push, from the system", func(t *testing.T) {
		f, _, _ := grabbing(t)
		before := len(f.session.PeekOverridesFor(f.openTurn())) // the hit edit's row
		if err := editSkill(t, f, f.actionID, enum.Grab); err != nil {
			t.Fatalf("Execute: %v", err)
		}
		var row *matchDomain.OverriddenValue
		for _, o := range f.session.PeekOverridesFor(f.openTurn()) {
			if o.Field == "damageSkill" {
				o := o
				row = &o
			}
		}
		if row == nil {
			t.Fatal("no damageSkill row was captured")
		}
		if row.Origin != matchDomain.OriginSystem {
			t.Fatalf("Origin = %q, want system — the engine chose Push, not the player", row.Origin)
		}
		if row.Original != enum.Push {
			t.Fatalf("Original = %#v, want Push", row.Original)
		}

		if err := editSkill(t, f, f.actionID, enum.Push); err != nil {
			t.Fatalf("Execute (back to Push): %v", err)
		}
		if got := len(f.session.PeekOverridesFor(f.openTurn())); got != before {
			t.Fatalf("captured %d values after editing back to Push, want %d", got, before)
		}
	})

	t.Run("an action with no attack is refused, and nothing in the payload lands", func(t *testing.T) {
		f := newOpenMoveFixture(t)
		speed := action.ConditionEdit{Field: action.FieldSpeed, Condition: action.RollCondition{Modifier: 2}}
		err := editSkill(t, f, f.actionID, enum.Grab, speed)
		if !errors.Is(err, matchsession.ErrNoDamageToMeasure) {
			t.Fatalf("err = %v, want ErrNoDamageToMeasure", err)
		}
		if f.openTurn().ActionRef().Speed.Context.Condition != nil {
			t.Fatal("a refused edit left the speed condition behind")
		}
	})
}
```

(`matchDomain.OverriddenValue` é o tipo que `PeekOverridesFor` devolve — confira o nome em `internal/domain/match/override.go` e ajuste.)

- [ ] **Step 2: Run to verify they fail** — compilação falha (`DamageSkill`, `actorDamageSkill`, `ErrNoDamageToMeasure`).

- [ ] **Step 3: Implement**

`attack.go`, no struct `Attack`, depois de `Damage RollCheck`:

```go
	// DamageSkill is the skill that measures this attack's damage. The player does not choose
	// it — the weapon deals the damage, and Push measures it (front-combat-phases.md §4.6) —
	// but the master may swap it (edit_action damageSkill: Grab, or another). Zero means Push:
	// every attack, and every row persisted before the field existed.
	//
	// Deliberately NOT Damage.SkillName. That is the field the player sends and the server
	// discards, and the action wire carries it to everyone: storing the master's choice there
	// would let a player choose, and would show a player who reconnects mid-turn the master's
	// edit before the turn closes. No wire maps this field.
	DamageSkill enum.SkillName
```

e, no mesmo arquivo:

```go
// EffectiveDamageSkill is DamageSkill with its zero value read as Push.
func (a Attack) EffectiveDamageSkill() enum.SkillName {
	if a.DamageSkill == "" {
		return enum.Push
	}
	return a.DamageSkill
}
```

`master_action.go`: importe `enum` e acrescente ao struct `MasterAction`, depois de `Conditions`:

```go
	// DamageSkill swaps the skill that measures the damage of the action's attack (Push by
	// default). nil = not sent, untouched.
	DamageSkill *enum.SkillName
```

`turn_resolver.go`: renomeie `actorPush` para `actorDamageSkill` e troque o corpo:

```go
// actorDamageSkill reads the attacker's skill that measures damage — Push, unless the master
// swapped it (Attack.DamageSkill, edit_action damageSkill).
//
// It nil-guards on purpose: the wall branch is NOT behind actorSheetMissing, so an action
// whose actor sheet never reached the resolver arrives here with nothing. Zero is the honest
// answer there, and the missing sheet is already reported as a ResolutionError by the
// character branch when it applies.
func (tr TurnResolver) actorDamageSkill(in ResolveInput, a action.Action) int {
	cs, ok := in.Sheets[a.GetActorID()]
	if !ok || cs == nil {
		return 0
	}
	skill := enum.Push
	if a.Attack != nil {
		skill = a.Attack.EffectiveDamageSkill()
	}
	return skillValueOf(cs, skill.String())
}
```

Atualize as referências a `actorPush` (o `rawDamage` de T4; o comentário de `actorWeaponProficiency` cita `actorPush` — troque o nome no texto). `TurnResolution`, depois de `IsSettled bool`:

```go
	// DamageSkill names the skill that measured the attack's damage — "Push" unless the master
	// swapped it. Empty when the action has no attack. Public, like rawDamage: once settled,
	// whoever reads the damage reads what measured it.
	DamageSkill string
```

Em `Resolve`, logo depois de `a := in.Turn.GetAction()`:

```go
	if a.Attack != nil {
		res.DamageSkill = a.Attack.EffectiveDamageSkill().String()
	}
```

`error.go`:

```go
	// ErrNoDamageToMeasure means a damageSkill edit named an action that carries no attack —
	// there is no damage for a skill to measure.
	ErrNoDamageToMeasure = errors.New("damageSkill edit targets an action with no attack")
```

`match_session.go`, `ApplyMasterAction`: na validação (antes do laço `for _, edit := range ma.Conditions` de validação):

```go
	if ma.DamageSkill != nil && target.Attack == nil {
		return nil, ErrNoDamageToMeasure
	}
```

e na aplicação, depois do bloco `if ma.Skills != nil { ... }`:

```go
	if ma.DamageSkill != nil {
		// The ORIGINAL is the effective skill — Push when the field was never set — and its
		// origin is the system: the engine chose Push, the player never did. Editing back to
		// Push erases the capture, like any other edit-back.
		s.captureOverride(target.GetID(), "damageSkill", match.OriginSystem, masterUUID,
			target.Attack.EffectiveDamageSkill(), *ma.DamageSkill)
		target.Attack.DamageSkill = *ma.DamageSkill
	}
```

`edit_action.go`, `hasRollEdits`:

```go
	return ma != nil && (len(ma.Conditions) > 0 || ma.Skills != nil || ma.TargetID != nil ||
		ma.DamageSkill != nil)
```

`message.go`, `EditActionPayload`, depois de `TargetIDs`:

```go
	// DamageSkill swaps the skill that measures the damage of the action's attack (Push by
	// default): any valid skill name. Absent = untouched; "Push" puts it back.
	DamageSkill *string `json:"damageSkill,omitempty"`
```

`action_mapper.go`, `buildEditAction`, antes do `return ma, nil`:

```go
	if p.DamageSkill != nil {
		name, err := enum.SkillNameFrom(*p.DamageSkill)
		if err != nil {
			return nil, err
		}
		ma.DamageSkill = &name
	}
```

- [ ] **Step 4: Run** — os testes novos passam; verificação inteira, com `-race` em `internal/app/game/`.

- [ ] **Step 5: Contrato** — `match-combat-ws.md`:
  - `edit_action`: na tabela de campos, linha nova `damageSkill` — "A perícia que mede o dano do ataque da ação (padrão **`Push`**). Qualquer perícia válida do enum (perícia desconhecida → `invalid_action`). Ausente = não mexe; `"Push"` devolve ao padrão e apaga a captura. Só numa ação com ataque — senão `game_error` `damageSkill edit targets an action with no attack`. O original vai para `overridden_action_values` com origem `system`." Um exemplo: `{ "type": "edit_action", "payload": { "damageSkill": "Grab" } }`. Acrescente o erro à lista **Erros**.
  - A linha de `attack.damage.skillName` (~200): troque "Trocar `Push` por `Grab` é prerrogativa do mestre, ainda não implementada." por "Trocar `Push` por outra perícia é prerrogativa do mestre: [`edit_action`](#edit_action) `damageSkill`. A escolha dele **não** aparece aqui — este campo continua sendo o do jogador, descartado."

- [ ] **Step 6: Commit**

```bash
git add internal/domain/match/entity/action/attack.go internal/domain/match/entity/action/master_action.go internal/domain/match/service/turn_resolver.go internal/domain/match/service/actor_push_test.go internal/domain/match/matchsession/error.go internal/domain/match/matchsession/match_session.go internal/application/match/edit_action.go internal/application/match/edit_action_test.go internal/app/game/message.go internal/app/game/action_mapper.go internal/app/game/action_mapper_test.go docs/dev/api/match-combat-ws.md
git commit -m "feat(combat): o mestre troca a perícia que mede o dano (edit_action.damageSkill)

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 6: a resolução leva `damageSkill` e as condições em vigor ao mestre — **opus** (spec §3.4, §3.5, §4, §5)

**Files:**
- Create: `internal/domain/match/service/turn_conditions.go`, `internal/domain/match/service/turn_conditions_test.go`
- Modify: `internal/domain/match/service/turn_resolver.go` (`TurnResolution.Conditions`; `Resolve` o preenche)
- Modify: `internal/domain/match/service/projection.go` (`ProjectResolution`)
- Modify: `internal/app/game/message.go` (`ResolutionUpdatedPayload`, `ConditionPayload`, `newResolutionUpdatedPayload`)
- Modify: `internal/gateway/pg/round/resolution_record.go` (`resolutionRecord.DamageSkill`, `encodeResolution`, `DecodeResolution`)
- Modify: `internal/app/api/match/get_match_history.go` (`TurnResolutionResponse.DamageSkill`, `toTurnResolutionResponse`)
- Create: `internal/app/game/regency_e2e_test.go`
- Test: `internal/app/game/resolution_payload_test.go`, a projeção (`internal/domain/match/service/projection_test.go`, ou o arquivo de teste que já cobre `ProjectResolution` — procure por `ProjectResolution(` nos `_test.go`), `internal/gateway/pg/round/round_integration_test.go`
- Docs: `docs/dev/api/match-combat-ws.md` (`resolution_updated`, `match_full_state`), `docs/dev/api/match-history.md`

**Interfaces:**
- Consumes: `Action.DefaultDefense` (T3), `TurnResolution.DamageSkill` (T5), `turn.Turn.GetAction()`, `GetReactions()`.
- Produces: `service.CheckCondition{ActionID uuid.UUID; Field string; SkillName string; Condition action.RollCondition}`; `TurnResolution.Conditions []CheckCondition`; `game.ConditionPayload`; `ResolutionUpdatedPayload.DamageSkill`/`.Conditions`.

- [ ] **Step 1: Failing domain test** — `turn_conditions_test.go` (`package service_test`):

```go
package service_test

import (
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

func TestResolve_ListsTheConditionsInForceWhileTheTurnIsOpen(t *testing.T) {
	build := func() (*turn.Turn, *action.Action, *action.Action) {
		a := action.NewAction(uuid.New(), nil, uuid.Nil, nil, action.ActionSpeed{},
			nil, nil, &action.Attack{}, nil, nil, nil, nil)
		a.Attack.Hit.Context.Condition = &action.RollCondition{Bias: -1, Modifier: -2, Description: "escuridao"}
		tn := turn.NewTurn(*a)
		r := reactionWith(action.ReactDodge, []int{5, 5}, nil, nil)
		r.Dodge.Context.Condition = &action.RollCondition{Bias: 1}
		r.DefaultDefense = &action.RollCheck{
			Context: action.RollContext{Condition: &action.RollCondition{Modifier: 2}},
		}
		tn.AddReaction(r)
		return tn, a, r
	}

	t.Run("open: every check with a condition, action first, then each reaction", func(t *testing.T) {
		tn, a, r := build()
		res := service.TurnResolver{}.Resolve(resolveWith(tn, noopTargetReader{}))
		want := []service.CheckCondition{
			{ActionID: a.GetID(), Field: "hit", Condition: action.RollCondition{Bias: -1, Modifier: -2, Description: "escuridao"}},
			{ActionID: r.GetID(), Field: "dodge", Condition: action.RollCondition{Bias: 1}},
			{ActionID: r.GetID(), Field: "defense", Condition: action.RollCondition{Modifier: 2}},
		}
		if len(res.Conditions) != len(want) {
			t.Fatalf("Conditions = %+v, want %+v", res.Conditions, want)
		}
		for i := range want {
			if res.Conditions[i] != want[i] {
				t.Errorf("Conditions[%d] = %+v, want %+v", i, res.Conditions[i], want[i])
			}
		}
	})

	t.Run("settled: none — the conditions live in the actions table, not in the record", func(t *testing.T) {
		tn, _, _ := build()
		tn.Close(time.Now())
		res := service.TurnResolver{}.Resolve(resolveWith(tn, noopTargetReader{}))
		if res.Conditions != nil {
			t.Fatalf("Conditions = %+v on a settled turn, want nil", res.Conditions)
		}
	})
}
```

(Confira o nome do método que fecha um turno — o teste de integração usa `tn.Close(time.Now())` — e o que acrescenta reação — `tn.AddReaction(r)` recebe `*action.Action`.)

E, no teste de projeção (o arquivo que já chama `ProjectResolution`):

```go
func TestProjectResolution_TheConditionsAreTheMasters(t *testing.T) {
	res := &service.TurnResolution{
		DamageSkill: "Grab",
		Conditions: []service.CheckCondition{
			{ActionID: uuid.New(), Field: "hit", Condition: action.RollCondition{Modifier: 2}},
		},
	}
	if got := service.ProjectResolution(res, service.Viewer{IsMaster: true}); len(got.Conditions) != 1 {
		t.Fatalf("the master lost the conditions: %+v", got.Conditions)
	}
	player := service.ProjectResolution(res, service.Viewer{})
	if player.Conditions != nil {
		t.Fatalf("a player received the master's conditions: %+v", player.Conditions)
	}
	if player.DamageSkill != "Grab" {
		t.Fatalf("DamageSkill = %q for a player, want Grab — it is public, like rawDamage", player.DamageSkill)
	}
	if len(res.Conditions) != 1 {
		t.Fatal("ProjectResolution mutated the master's copy")
	}
}
```

- [ ] **Step 2: Run to verify they fail** — compilação falha (`CheckCondition`, `Conditions`).

- [ ] **Step 3: Implement the domain**

`turn_conditions.go`:

```go
package service

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/google/uuid"
)

// CheckCondition is one master condition in force on one check of the open turn — what the
// master's editor needs to show what is in force after a reload (front-combat-phases.md §0.2).
// Field and SkillName are alternatives, exactly as in edit_action: Field names a fixed check,
// SkillName an entry of Skills.
type CheckCondition struct {
	ActionID  uuid.UUID
	Field     string
	SkillName string
	Condition action.RollCondition
}

// turnConditions lists every check of the turn that carries a master condition: the turn's own
// action first, then each attached reaction (opened or not) in the turn's order. Inside one
// action the order is fixed — speed, feint, hit, damage, dodge, defense, repel, moveSpeed, then
// each skill — so two reads of the same turn list the same thing in the same order.
//
// It reads the live action, which is where the condition lives ("the edited action IS the
// action", combat-engine.md); there is no second copy to keep in sync.
func turnConditions(t *turn.Turn) []CheckCondition {
	var out []CheckCondition
	out = appendActionConditions(out, t.GetAction())
	for _, r := range t.GetReactions() {
		out = appendActionConditions(out, r)
	}
	return out
}

func appendActionConditions(out []CheckCondition, a action.Action) []CheckCondition {
	add := func(field action.ConditionField, rc *action.RollCheck) {
		if rc == nil || rc.Context.Condition == nil {
			return
		}
		out = append(out, CheckCondition{
			ActionID: a.GetID(), Field: string(field), Condition: *rc.Context.Condition,
		})
	}
	add(action.FieldSpeed, &a.Speed.RollCheck)
	add(action.FieldFeint, a.Feint)
	if a.Attack != nil {
		add(action.FieldHit, &a.Attack.Hit)
		add(action.FieldDamage, &a.Attack.Damage)
	}
	if a.Dodge != nil {
		add(action.FieldDodge, &a.Dodge.RollCheck)
	}
	// "defense" names the declared Defense on a plain action and the DefaultDefense on a
	// reaction that keeps it (resolveRollCheck) — never both editable on the same action.
	if a.Defense != nil {
		add(action.FieldDefense, &a.Defense.RollCheck)
	}
	add(action.FieldDefense, a.DefaultDefense)
	if a.Repel != nil {
		add(action.FieldRepel, &a.Repel.RollCheck)
	}
	if a.Move != nil {
		add(action.FieldMoveSpeed, a.Move.Speed)
	}
	for _, s := range a.Skills {
		if s.Context.Condition == nil {
			continue
		}
		out = append(out, CheckCondition{
			ActionID: a.GetID(), SkillName: s.SkillName, Condition: *s.Context.Condition,
		})
	}
	return out
}
```

(Se `t.GetReactions()` devolver `[]*action.Action` em vez de `[]action.Action`, passe `*r`.)

`turn_resolver.go`, `TurnResolution`, depois de `DamageSkill`:

```go
	// Conditions is every master condition in force on the turn — filled only while the turn
	// is OPEN, when the whole resolution is the master's. It is how a master who reloads
	// mid-edit finds what they had set (match_full_state.resolution is a fresh Resolve). Not
	// persisted: the conditions already live in the actions table, inside each RollCheck, and a
	// second copy in turns.resolution would be one that can diverge. ProjectResolution strips
	// it for anyone but the master.
	Conditions []CheckCondition
```

Em `Resolve`, no fim, antes do `return res`:

```go
	if !res.IsSettled {
		res.Conditions = turnConditions(in.Turn)
	}
```

`projection.go`, dentro do `if !v.IsMaster {`:

```go
		// The master's conditions — what they edited, and why — are theirs alone, open turn or
		// not. The numbers they produced are the table's once the turn settles; the reasons
		// are not.
		out.Conditions = nil
```

- [ ] **Step 4: Run domain tests** — passam.

- [ ] **Step 5: Failing wire tests** — `resolution_payload_test.go` (`package game`):

```go
func TestResolutionUpdatedPayloadCarriesTheRegency(t *testing.T) {
	actionID := uuid.New()
	res := &service.TurnResolution{
		DamageSkill: "Grab",
		Conditions: []service.CheckCondition{
			{ActionID: actionID, Field: "hit", Condition: action.RollCondition{Bias: -1, Modifier: -2, Description: "escuridao"}},
			{ActionID: actionID, SkillName: "Evasion", Condition: action.RollCondition{Bias: 1}},
		},
	}
	raw, err := json.Marshal(newResolutionUpdatedPayload(uuid.New(), res))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got struct {
		DamageSkill string           `json:"damageSkill"`
		Conditions  []map[string]any `json:"conditions"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.DamageSkill != "Grab" {
		t.Errorf("damageSkill = %q, want Grab", got.DamageSkill)
	}
	if len(got.Conditions) != 2 {
		t.Fatalf("conditions = %v, want 2 entries", got.Conditions)
	}
	first := got.Conditions[0]
	if first["actionId"] != actionID.String() || first["field"] != "hit" ||
		first["bias"] != float64(-1) || first["modifier"] != float64(-2) || first["description"] != "escuridao" {
		t.Errorf("conditions[0] = %v", first)
	}
	if _, ok := first["skillName"]; ok {
		t.Error("conditions[0] carries skillName — field and skillName are alternatives")
	}
	if got.Conditions[1]["skillName"] != "Evasion" {
		t.Errorf("conditions[1] = %v, want skillName Evasion", got.Conditions[1])
	}

	empty, _ := json.Marshal(newResolutionUpdatedPayload(uuid.New(), &service.TurnResolution{}))
	if strings.Contains(string(empty), "conditions") || strings.Contains(string(empty), "damageSkill") {
		t.Errorf("an empty resolution carries the keys anyway: %s", empty)
	}
}
```

- [ ] **Step 6: Implement the wire** — `message.go`, `ResolutionUpdatedPayload`, depois de `Errors`:

```go
	// DamageSkill names the skill that measured the attack's damage — "Push" unless the master
	// swapped it (edit_action damageSkill). Absent when the action has no attack. Public on a
	// settled resolution, like rawDamage.
	DamageSkill string `json:"damageSkill,omitempty"`
	// Conditions is every master condition in force on the open turn — what the master's
	// editor shows after a reload. MASTER-ONLY and OPEN-TURN-ONLY: service.ProjectResolution
	// strips it for anyone else, and the resolver does not fill it once the turn is settled.
	// Absent = nothing edited.
	Conditions []ConditionPayload `json:"conditions,omitempty"`
```

e o tipo:

```go
// ConditionPayload is one master condition in force — the shape of an edit_action conditions
// entry plus the action it lives on. ActionID is always the real ID (the turn's own action
// included, never zero). Field and SkillName are alternatives.
type ConditionPayload struct {
	ActionID    uuid.UUID `json:"actionId"`
	Field       string    `json:"field,omitempty"`
	SkillName   string    `json:"skillName,omitempty"`
	Bias        int       `json:"bias"`
	Modifier    int       `json:"modifier"`
	Description string    `json:"description,omitempty"`
}
```

Em `newResolutionUpdatedPayload`, antes do `return p`:

```go
	p.DamageSkill = res.DamageSkill
	for _, c := range res.Conditions {
		p.Conditions = append(p.Conditions, ConditionPayload{
			ActionID: c.ActionID, Field: c.Field, SkillName: c.SkillName,
			Bias: c.Condition.Bias, Modifier: c.Condition.Modifier, Description: c.Condition.Description,
		})
	}
```

`resolution_record.go`: em `resolutionRecord`, depois de `IsSettled`:

```go
	// DamageSkill is what measured the damage ("Push" unless the master swapped it). Absent
	// on rows from before Phase 8 — all of those were Push — and on a turn with no attack.
	DamageSkill string `json:"damageSkill,omitempty"`
```

`encodeResolution`: `DamageSkill: res.DamageSkill,` no literal de `rec`. `DecodeResolution`: copie `rec.DamageSkill` para o `DamageSkill` da `TurnResolution` que ele monta. **Não** grave `Conditions` (spec D5).

`get_match_history.go`: em `TurnResolutionResponse`, depois de `Targets`:

```go
	// DamageSkill names the skill that measured the damage ("Push" unless the master swapped
	// it). Absent on a turn with no attack and on turns recorded before Phase 8 (all Push).
	DamageSkill string `json:"damageSkill,omitempty"`
```

e `DamageSkill: res.DamageSkill,` em `toTurnResolutionResponse`.

- [ ] **Step 7: Failing e2e** — `regency_e2e_test.go` (`package game_test`). Use os helpers existentes (`newCombatFixture`, `connectWS`, `readMessage`, `newCollector`, `sendWS`, `findMessage`, `awaitCount`, `lastResolutionUpdated`); o padrão de reconexão é o de `TestMatchFullStateOnConnectCarriesTheCombatState` (`combat_e2e_test.go`): o mestre segura a sala enquanto o jogador reconecta, e o jogador segura enquanto o mestre reconecta.

```go
package game_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/google/uuid"
)

// TestE2E_Regency drives the master's edit over a real socket, against a real Room: the
// conditions in force and the damage skill reach the master — live and after a reload — and
// nothing of the edit reaches a player before the turn closes (spec 2026-10-09 §4, §5).
func TestE2E_Regency(t *testing.T) {
	f := newCombatFixture(t)

	master := connectWS(t, f.server.URL, f.masterUUID, f.matchUUID)
	readMessage(t, master) // room_state
	masterMsgs := newCollector(master)
	player := connectWS(t, f.server.URL, f.playerUUID, f.matchUUID)
	readMessage(t, player) // room_state
	playerMsgs := newCollector(player)

	f.enqueueAttack(t, player)
	if !masterMsgs.await(game.MsgTypeActionQueued, 2*time.Second) {
		t.Fatal("the master was never told an action was queued")
	}
	sendWS(t, master, "open_next_action", map[string]any{})
	if !masterMsgs.await(game.MsgTypeResolutionUpdate, 2*time.Second) {
		t.Fatal("no resolution_updated for the opened turn")
	}
	playerResolutionsBefore := playerMsgs.count(game.MsgTypeResolutionUpdate)

	before := masterMsgs.count(game.MsgTypeResolutionUpdate)
	sendWS(t, master, "edit_action", map[string]any{
		"conditions":  []map[string]any{{"field": "hit", "bias": -1, "modifier": -2, "description": "escuridao"}},
		"damageSkill": "Grab",
	})
	if !masterMsgs.await(game.MsgTypeActionEdited, 2*time.Second) {
		t.Fatal("the master never received action_edited")
	}
	if !awaitCount(masterMsgs, game.MsgTypeResolutionUpdate, before+1, 2*time.Second) {
		t.Fatal("the master never received the recomputed resolution_updated")
	}

	assertRegency := func(t *testing.T, p game.ResolutionUpdatedPayload, where string) {
		t.Helper()
		if p.DamageSkill != "Grab" {
			t.Errorf("%s: damageSkill = %q, want Grab", where, p.DamageSkill)
		}
		if len(p.Conditions) != 1 {
			t.Fatalf("%s: conditions = %+v, want the one hit condition", where, p.Conditions)
		}
		c := p.Conditions[0]
		if c.ActionID == uuid.Nil || c.Field != "hit" || c.Bias != -1 || c.Modifier != -2 || c.Description != "escuridao" {
			t.Errorf("%s: conditions[0] = %+v", where, c)
		}
	}

	t.Run("the master sees what is in force", func(t *testing.T) {
		assertRegency(t, lastResolutionUpdated(t, masterMsgs), "resolution_updated")
	})

	t.Run("a player receives nothing of the edit while the turn is open", func(t *testing.T) {
		time.Sleep(100 * time.Millisecond) // let anything misrouted arrive
		if n := playerMsgs.count(game.MsgTypeResolutionUpdate); n != playerResolutionsBefore {
			t.Errorf("the player received %d resolution_updated mid-turn", n-playerResolutionsBefore)
		}
		if n := playerMsgs.count(game.MsgTypeActionEdited); n != 0 {
			t.Errorf("the player received action_edited")
		}
	})

	t.Run("a player who reconnects mid-turn sees nothing of the edit", func(t *testing.T) {
		late := connectWS(t, f.server.URL, f.playerUUID, f.matchUUID)
		defer late.Close() //nolint:errcheck
		lateMsgs := newCollector(late)
		if !lateMsgs.await(game.MsgTypeMatchFullState, 2*time.Second) {
			t.Fatal("the reconnecting player never received match_full_state")
		}
		raw := string(findMessage(t, lateMsgs.snapshotMessages(), game.MsgTypeMatchFullState).Payload)
		for _, leak := range []string{"Grab", "escuridao", "damageSkill", "conditions", "defaultDefense"} {
			if strings.Contains(raw, leak) {
				t.Errorf("match_full_state for a player carries %q: %s", leak, raw)
			}
		}
	})

	t.Run("the master who reloads finds what they had set", func(t *testing.T) {
		master.Close() //nolint:errcheck // the player keeps the room alive
		reloaded := connectWS(t, f.server.URL, f.masterUUID, f.matchUUID)
		defer reloaded.Close() //nolint:errcheck
		reloadedMsgs := newCollector(reloaded)
		if !reloadedMsgs.await(game.MsgTypeMatchFullState, 2*time.Second) {
			t.Fatal("the reloading master never received match_full_state")
		}
		var full game.MatchFullStatePayload
		if err := json.Unmarshal(
			findMessage(t, reloadedMsgs.snapshotMessages(), game.MsgTypeMatchFullState).Payload, &full,
		); err != nil {
			t.Fatalf("unmarshal match_full_state: %v", err)
		}
		if full.Resolution == nil {
			t.Fatal("the master reloaded into an open turn and lost the calculation")
		}
		assertRegency(t, *full.Resolution, "match_full_state.resolution")

		// Closing the turn: the settled resolution reaches the player with the damage skill
		// and without the master's conditions.
		settledBefore := playerMsgs.count(game.MsgTypeResolutionUpdate)
		sendWS(t, reloaded, "close_turn", map[string]any{})
		if !awaitCount(playerMsgs, game.MsgTypeResolutionUpdate, settledBefore+1, 3*time.Second) {
			t.Fatal("the player never received the settled resolution")
		}
		settled := lastResolutionUpdated(t, playerMsgs)
		if !settled.IsSettled {
			t.Fatal("the resolution the player received is not the settled one")
		}
		if settled.DamageSkill != "Grab" {
			t.Errorf("settled damageSkill = %q, want Grab", settled.DamageSkill)
		}
		if settled.Conditions != nil {
			t.Errorf("settled conditions reached a player: %+v", settled.Conditions)
		}
	})
}
```

(`f.server`, `f.masterUUID`, `f.playerUUID`, `f.matchUUID` vêm do `combatFixture`; se `MatchFullStatePayload.Resolution` não for `*ResolutionUpdatedPayload`, ajuste o tipo. Se o `close_turn` sem reações pendentes exigir `confirm`, mande `{"confirm": true}`.)

- [ ] **Step 8: Failing integration test** — `round_integration_test.go`:

```go
func TestPersistTurnCloseWritesTheDamageSkill(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.SetupTestDB(t)
	pgtest.TruncateAll(t, pool)
	repo := roundrepo.NewRepository(pool)
	fx := seedMatchAndSheets(t, pool)

	act := buildAttackAction(t, fx.attackerSheet, fx.victimSheet)
	act.Attack.DamageSkill = enum.Grab
	tn := turnentity.NewTurn(*act)
	tn.Close(time.Now())
	res := &service.TurnResolution{
		IsSettled:   true,
		DamageSkill: "Grab",
		// A resolution handed in with conditions must not write them: they live in actions.
		Conditions: []service.CheckCondition{{ActionID: act.GetID(), Field: "hit",
			Condition: action.RollCondition{Modifier: 2, Description: "nao-gravar"}}},
	}

	if err := repo.PersistTurnClose(ctx, appmatch.TurnCloseData{
		Scene: fx.scene, Round: fx.round, Turn: tn, Action: act, MatchUUID: fx.matchUUID,
		Resolution: res,
	}); err != nil {
		t.Fatalf("PersistTurnClose: %v", err)
	}

	var stored []byte
	if err := pool.QueryRow(ctx, `SELECT resolution FROM turns WHERE uuid = $1`, tn.GetID()).Scan(&stored); err != nil {
		t.Fatalf("read back resolution: %v", err)
	}
	if !strings.Contains(string(stored), `"damageSkill":"Grab"`) {
		t.Errorf("turns.resolution = %s, want damageSkill Grab", stored)
	}
	if strings.Contains(string(stored), "nao-gravar") {
		t.Errorf("turns.resolution carries the master's conditions: %s", stored)
	}
	if got := roundrepo.DecodeResolution(stored); got == nil || got.DamageSkill != "Grab" {
		t.Errorf("DecodeResolution lost damageSkill: %+v", got)
	}

	scenes, err := repo.FindMatchHistory(ctx, fx.matchUUID)
	if err != nil {
		t.Fatalf("FindMatchHistory: %v", err)
	}
	got := scenes[0].Rounds[0].Turns[0]
	if got.Action.Attack == nil || got.Action.Attack.DamageSkill != enum.Grab {
		t.Errorf("the attack read back without the damage skill: %+v", got.Action.Attack)
	}
}
```

(Confira o nome do campo da resolução em `appmatch.TurnCloseData` — `TestPersistTurnCloseWritesTheSettledResolution` mostra como passar uma — e ajuste.)

- [ ] **Step 9: Run** — `go test ./internal/... -run 'Regency|Conditions|DamageSkill|ProjectResolution' -v`, `go test -race ./internal/app/game/`, integração; verificação inteira.

- [ ] **Step 10: Contrato** —
  - `match-combat-ws.md`, `resolution_updated`: no exemplo JSON, `"damageSkill": "Push"` no topo e um `"conditions": [ { "actionId": "3333…", "field": "hit", "bias": -1, "modifier": -2, "description": "escuridao" } ]`. Na tabela de campos, duas linhas:
    - `damageSkill` — "A perícia que mediu o dano: `Push`, ou a que o mestre escolheu ([`edit_action`](#edit_action) `damageSkill`). **Ausente** sem ataque. Segue o eixo de `rawDamage`: com o turno aberto, só o mestre; fechado, todos."
    - `conditions` — "As condições do mestre **em vigor** no turno aberto: uma entrada por rolagem com condição, no formato de uma entrada de `edit_action.conditions` mais `actionId` (sempre o ID real, inclusive da ação do turno). **Só o mestre, só com turno aberto** — nunca no payload liquidado, nunca no histórico (a condição fica gravada dentro da ação, na tabela `actions`). É o que o editor do mestre mostra depois de recarregar. Ausente = nada editado. `field: "defense"` numa reação é a defesa padrão dela."
  - `match_full_state`, linha de `resolution`: acrescente "Traz `conditions` e `damageSkill` como o `resolution_updated` — é o que faz a edição do mestre sobreviver a recarregar."
  - `match-history.md`: em `resolution`, o campo `damageSkill` ("ausente em turno sem ataque e em turnos gravados antes da Fase 8, que foram todos `Push`"); diga que `conditions` **não** existe no histórico.

- [ ] **Step 11: Commit**

```bash
git add internal/domain/match/service/turn_conditions.go internal/domain/match/service/turn_conditions_test.go internal/domain/match/service/turn_resolver.go internal/domain/match/service/projection.go internal/app/game/message.go internal/app/game/resolution_payload_test.go internal/app/game/regency_e2e_test.go internal/gateway/pg/round/resolution_record.go internal/gateway/pg/round/round_integration_test.go internal/app/api/match/get_match_history.go docs/dev/api/match-combat-ws.md docs/dev/api/match-history.md
git add -u internal/domain/match/service   # o arquivo de teste da projeção
git commit -m "feat(combat): a resolução leva ao mestre as condições em vigor e a perícia do dano

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 7: contrato consolidado, motor, documento mestre, AGENTS.md (spec §3.7)

**Files:**
- Modify: `docs/dev/api/match-combat-ws.md` (`edit_action`: a tabela "o que cada rolagem muda")
- Modify: `docs/dev/match/combat-engine.md` ("A edição do mestre")
- Modify: `docs/superpowers/specs/2026-09-20-front-combat-phases.md` (§0, §4.6, §6A.6 F7, §8, §11.1, pendências)
- Modify: `AGENTS.md` (Known Issues — o bug da condição em dodge/defense/repel)
- Modify: `docs/documentation-map.yaml` (`notes` de `reaction_collision.go`, `turn_resolver.go`)

Só documentação: nenhum teste. Leia o código já mergeado nas tarefas 1–6 e escreva o que ele faz.

- [ ] **Step 1: `match-combat-ws.md`, `edit_action`** — depois da tabela de campos, uma subseção:

> **O que cada rolagem muda com o turno aberto.**
>
> | `field` / `skillName` | Onde | Muda o desfecho? | Leitura |
> |---|---|---|---|
> | `hit` | a ação | sim — o golpe é um só, vale para todos os alvos | rolada |
> | `damage` | a ação | sim — só o `modifier` (soma ao dano bruto, piso zero, vale para a cadeia); `bias` é recusado | — |
> | `dodge` | reação de esquiva/fuga | sim | rolada |
> | `defense` | reação `dodge`, `closedDodge`, `escapeGuard` | sim — é a **defesa padrão** atrás da reação | **passiva**: só o `modifier` move |
> | `repel` | reação `repel` | sim | rolada |
> | `moveSpeed` | reação de fuga | sim — decide se a fuga escapa | rolada no `escape`/`escapeGuard` (Dash); **passiva** no `closedEscape` (Shift) |
> | `skillName: "Evasion"` | reação fechada | sim — entra na esquiva fechada e na reserva | rolada |
> | `speed` | qualquer | **não** — move a economia, que não se refaz | passiva fora do regime Race |
> | `moveSpeed` | a ação | **não** — a peça já andou na abertura, e nada testa contra o movimento de uma ação | — |
> | `feint` | a ação | **não** — a resolução da finta não existe ainda | — |
> | outro `skillName` | qualquer | **não** — ninguém lê o resultado das perícias (corrente de testes) | — |
>
> Tudo é aceito e guardado — a regra pode crescer; a tela é que esconde o que não muda nada. Viés
> numa leitura **passiva** é aceito e não tem efeito (não há dado para escolher). A **esquiva e a
> defesa passivas de um alvo que não reagiu** não são editáveis: não há reação onde guardar a
> condição.

Confira a tabela de erros do fim do documento (`game_error`): os dois erros novos (T4, T5) aparecem
nela ou na lista da seção; e a tabela de `invalid_action` cita "perícia desconhecida" — `damageSkill`
cai nela.

- [ ] **Step 2: `combat-engine.md`**, seção "A edição do mestre", depois de "Os valores deslocados": subseção nova `#### O que cada condição move, e onde ela mora`, com: (a) a tabela acima, resumida; (b) **`DefaultDefense`** — por que não `Defense` (vai ao wire; contaria a edição antes do fechamento) e que é criado no primeiro edit; (c) **a perícia do dano** — `Attack.DamageSkill`, zero = `Push`, por que não `Damage.SkillName`, captura com origem `system`; (d) **condição zerada é sem condição** — o wire não manda `nil`, e é isso que deixa "cancelar é editar de volta" funcionar para a condição; (e) **a reserva fechada lê a condição** (spec D4). Procure também o trecho de "O escape: esquiva e movimento" que documenta o bug de `dodge`/`defense`/`repel` sem condição e troque pela descrição do comportamento atual.

- [ ] **Step 3: documento mestre** (`2026-09-20-front-combat-phases.md`):
  - §0, tabela: linha 8 → `| **8** | Regência — a edição do mestre (§8) | back feito (este PR); front **próximo** |` (e a linha 7, se ainda diz "front próximo", confira no git log se o front da 7 já entrou — PR #70/#71 do front; não invente: se não souber, deixe como está).
  - §4.6, no fim: "✅ **Feito no pacote de back da Fase 8:** `edit_action.damageSkill` aceita qualquer perícia do enum; a tela oferece Push e Grab. A resolução diz qual mediu (`damageSkill`)."
  - §6A.6 F7, troque o parágrafo que começa "> **Por que ele nasce só de leitura:**" por:

    > **Por que ele nasce só de leitura:** os botões que agem sobre ele pertencem a fases que ainda
    > não chegaram. **Ao fim da Fase 8 o painel está completo**, com: **edição de rolagem** (viés,
    > ajuste, motivo), **troca da perícia do dano**, **escolha de onde cai o escape que falhou**
    > (F14) e **dar a palavra às reações** (Fase 7). **Editar perícias fica fora** até existir a
    > corrente de testes (§11.1): hoje ninguém lê o resultado delas, e um controle que não muda nada
    > é pior que controle nenhum. A condição na **Evasion das reações fechadas** também fica fora do
    > painel — seria uma tela especial para uma perícia só, que a corrente vai redesenhar. (O motor
    > lê as duas; é a tela que não as oferece.)

  - §8, troque o bloco **Escopo** por:

    > **Escopo:**
    > - Edição do mestre: `edit_action` / `action_edited`. **Edição de rolagem** — viés, ajuste e
    >   motivo — em toda rolagem que muda o desfecho do turno aberto (o contrato tem a tabela); a tela
    >   esconde o viés onde ele não faz sentido (leitura passiva) e não oferece as rolagens que não
    >   mudam nada. **Troca da perícia do dano**, `Push` → `Grab` (§4.6).
    > - Os botões de **editar** no cálculo do turno aberto, no card da ação (F7, §6A.6). Com eles, o
    >   cálculo fica completo: edição de rolagem, troca da perícia do dano, onde cai o escape que
    >   falhou e dar a palavra às reações.
    > - **Fora:** editar **perícias** (até existir a corrente de testes, §11.1) e a condição na
    >   **Evasion** das reações fechadas.
    > - O pacote de back desta fase ([spec](2026-10-09-combat-phase-8-regency-back-design.md)) fez a
    >   resolução ler toda condição que aceita, criou o portador da defesa padrão, a perícia do dano,
    >   e mandou ao mestre as condições em vigor (sobrevivem a recarregar).

  - §11.1, no fim: "Por isso também **o painel do mestre não edita perícias** (Fase 8, §8): o motor guarda e lê o que receber, mas uma edição que não muda nada não ganha botão."
  - Pendências: em §0, no parágrafo "Depois da Fase 8, o próximo passo é enriquecer a mecânica de combate —", acrescente à lista: "**editar uma ação na fila antes de ela agir** (velocidade e o resto — o `edit_action` só aceita o turno aberto) e **a edição da esquiva e da defesa passivas de um alvo que não reagiu** (não há reação onde guardar a condição)".

- [ ] **Step 4: `AGENTS.md`** — troque o bullet "⚠️ **Bug conhecido, achado na Task 12 ...**" por:

  > - **A condição do mestre é lida em toda rolagem que a resolução usa** (Fase 8, spec
  >   2026-10-09): acerto, dano (só o ajuste; viés recusado), esquiva, Evasion das fechadas, aparo,
  >   defesa padrão (portador `Action.DefaultDefense`, fora do wire) e o `moveSpeed` das fugas.
  >   `speed`, o `moveSpeed` da própria ação e `feint` são aceitos e não mudam o desfecho do turno
  >   aberto. Condição zerada é "sem condição". Ver `docs/dev/match/combat-engine.md` ("A edição do
  >   mestre").

- [ ] **Step 5: `documentation-map.yaml`** — nas entradas de `internal/domain/match/service/reaction_collision.go` e de `turn_resolver.go`, acrescente às `notes` uma frase: a condição do mestre é lida em todas as derivações (Fase 8); `TurnResolution.Conditions` (só aberto) e `DamageSkill`. Se `turn_conditions.go` não tiver entrada, crie uma apontando para `match-combat-ws.md` (`resolution_updated`) e `combat-engine.md`.

- [ ] **Step 6: Commit**

```bash
git add docs/ AGENTS.md
git commit -m "docs: Fase 8 — contrato, motor e documento mestre da regência

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 8: verificação final e PR

- [ ] **Step 1: Suíte inteira**

```bash
go build ./... && go vet ./... && go vet -tags integration ./... && go vet -tags smoke ./...
go test ./internal/... && go test -race ./internal/app/game/
go test -tags=integration -p 1 ./internal/gateway/pg/...
golangci-lint run ./...
```

Todos verdes. Cole a saída resumida no PR.

- [ ] **Step 2: Revisão do contrato contra o código** — leia de ponta a ponta `edit_action`, `resolution_updated`, `match_full_state` (linha `resolution`), a tabela de erros de `match-combat-ws.md` e o trecho mudado de `match-history.md`. Cada campo citado existe com esse nome JSON no código; cada destino bate com o envio real.

- [ ] **Step 3: Revisão da branch inteira** (subagent-driven-development: o revisor final).

- [ ] **Step 4: PR** — `gh pr create` para `main`, título "feat: Fase 8 — pacote de back da regência". Corpo em PT-BR:
  - o que entrou (decisões 2–7, 10, 11 e a documentação), apontando para o spec;
  - **o que foi verificado**: a suíte, `-race`, integração, lint; o e2e `TestE2E_Regency` contra a `Room` real por websocket (mestre ao vivo, mestre recarregando, jogador ao vivo e reconectando, fechamento);
  - **o que não foi, e por quê**: nenhum smoke manual por REST+WS — o e2e contra a `Room` real cobre exatamente o caminho, e montar o cenário à mão é desproporcional a um pacote sem tela (substituição registrada como manda o CLAUDE.md da raiz); a verificação de ponta a ponta, com as três contas, é do PR de front da Fase 8, contra este back mergeado. Por isso **não** rodar o `dev-checkout.sh` aqui;
  - as decisões D1–D6 do spec §8, para o revisor confirmar; e o item "observado, não tratado" do spec §6 (edição de `targetIds` visível a quem reconecta);
  - termine com `🤖 Generated with [Claude Code](https://claude.com/claude-code)`.
