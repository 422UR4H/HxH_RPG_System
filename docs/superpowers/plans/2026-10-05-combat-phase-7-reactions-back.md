# Fase 7 — Reações — pacote de back — plano de implementação

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development
> (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** dar à reação as mesmas superfícies de wire que a ação ganhou no fechamento da Fase 6 —
resposta a quem reagiu, mecânica pública ao abrir, reconexão, histórico como foi visto ao vivo —,
nomear a ação que a reação consome, recusar o segundo attach, derivar as perícias das reações no
servidor e escrever tudo isso no contrato e no documento mestre.

**Architecture:** nada de corte novo nem de portão de fog novo: o `reaction_opened` e o
`match_full_state.openTurn.reactions` passam pelo mesmo `turnActionWireLocked` do `turn_opened`;
o veredito de fog da abertura da reação é guardado em `turnWrites` e gravado no `move_views` da
linha da reação, como o da ação. O consumo vira um campo da reação (`ConsumedActionIDs`), projetado
como segredo de mestre e dono e persistido numa coluna nova. Uma mensagem nova
(`reaction_attached`) vai a quem reagiu e ao mestre.

**Tech Stack:** Go 1.23, gorilla/websocket, pgx/v5, goose, `testing` padrão.

**Spec:** [`docs/superpowers/specs/2026-10-05-combat-phase-7-reactions-back-design.md`](../specs/2026-10-05-combat-phase-7-reactions-back-design.md)
— leia o spec inteiro antes da primeira tarefa. Cada tarefa cita a seção do spec que implementa.
Documento mestre: `docs/superpowers/specs/2026-09-20-front-combat-phases.md` §7, §11.4, §12.
Contrato: `docs/dev/api/match-combat-ws.md`.

**Branch:** `feat/combat-phase-7-reactions-back` (já criada a partir de `main` em `754366c`).

## Global Constraints

- Go 1.23; `testing` padrão, table-driven com `t.Run`; mantenha o pacote de teste que cada arquivo já usa (`action_mapper_test.go` é `package game`; os `*_e2e_test.go` são `package game_test`; `match_session_test.go` é `package matchsession_test`). **TDD** — o teste vem antes.
- `room.go` é dono do lock (`r.mu`). **Nada que envia a cliente roda com `r.mu` preso.** `dispatchPerPlayer`, `sendToMaster`, `buildMatchFullState` pegam o lock sozinhos — nunca chame com ele preso. Dentro do `build` de `dispatchPerPlayer`, pegue `r.mu.RLock()` você mesmo se ler sessão ou tabuleiro (como o `turn_opened` faz).
- `MatchSession` não tem lock: todo acesso à sessão é sob `r.mu`. Um ponteiro devolvido pela sessão (`OpenReactionResult.Opened`) aponta para memória viva do turno — **copie o valor sob o lock**.
- Wire em **camelCase**; tipo de mensagem em snake_case; listas que o contrato diz "sempre presentes" nunca saem `null` (inicialize com `[]uuid.UUID{}`).
- **Nunca remover comentários `TODO`.** Comentários explicam o porquê, no tom e na densidade dos vizinhos (este repo comenta bastante; siga o vizinho).
- Todo item que muda o wire atualiza o contrato (`docs/dev/api/match-combat-ws.md` ou `match-history.md`) **no mesmo commit**. Migração nova entra em `docs/documentation-map.yaml` no mesmo commit.
- Verificação por tarefa: `go build ./...`, `go vet ./...`, `go vet -tags integration ./...`, `go vet -tags smoke ./...`, `go test ./internal/...`. Tarefa que toca `room.go`: também `go test -race ./internal/app/game/`. Tarefa que toca `internal/gateway/pg/`: também `go test -tags=integration -p 1 ./internal/gateway/pg/...` (banco em `TEST_DATABASE_URL`, padrão `postgres://postgres:postgres@localhost:5432/hxh_rpg_test?sslmode=disable`; aplique a migração nova no banco de teste antes).
- Commits terminam com a linha `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.
- Docs em PT-BR; nomes de código em inglês.
- **Nada que uma projeção esconde é descartado na gravação** (spec §4.6.1): o dado vai inteiro para o banco e é escondido só na leitura. Ao fim da partida, os participantes vão poder ver tudo — e isso só é possível se o banco tiver tudo.

## Review Focus

1. **A ação consumida reconectando depois do fechamento** — o jogador perde o `reaction_attached`, o turno fecha, ele reconecta: o histórico tem que trazer `consumedActionIds` na reação **para ele** e não para um terceiro. Teste em T6 (use case do histórico) e T6 (integração).
2. **Segundo attach recusado sem cobrar nada** — a fila e as barras ficam como estavam (o primeiro attach já cobrou; o segundo não pode consumir de novo). Teste em T2.
3. **`reaction_opened` não vaza o rótulo fechado nem o destino escondido** — um terceiro recebe `escape` para um `closedEscape`, sem a entrada `Evasion`, sem `consumedActionIds`, e sem `move.position` quando não vê a casa. Teste em T4.
4. **Ordem `reaction_opened` → `resolution_updated` no mestre** — a pista mudou (`r.broadcast` → direta); o mestre tem que receber o anúncio antes do cálculo recomputado. Teste em T4.
5. **O mestre com um NPC alvo e um jogador alvo no mesmo turno** — `ownReactions` do mestre traz só as reações dos NPCs; a do jogador fica no `ownReactions` dele. Teste em T5.

---

### Task 1: as perícias das reações são derivadas no servidor (spec §4.7, item 10)

**Files:**
- Modify: `internal/app/game/action_mapper.go` (blocos `dodge`, `repel` e o `if kind.RequiresEvasionSkill()` em `buildAction`)
- Test: `internal/app/game/action_mapper_test.go` (`package game`)
- Docs: `docs/dev/api/match-combat-ws.md` (seção `attach_reaction`)

**Interfaces:**
- Consumes: `enum.Reflex`, `enum.Repel`, `enum.Evasion` (`internal/domain/entity/enum/skill_name.go`).
- Produces: `buildAction` sempre devolve `Dodge.RollCheck.SkillName == "Reflex"`, `Repel.RollCheck.SkillName == "Repel"`, e uma entrada `Evasion` em `Skills` para `closedDodge`/`closedEscape`. A recusa `must carry an evasion skill entry` deixa de existir.

- [ ] **Step 1: Write the failing test**

Acrescente em `action_mapper_test.go`:

```go
func TestBuildAction_DerivesReactionSkillNames(t *testing.T) {
	actor, reactTo := uuid.New(), uuid.New()

	t.Run("dodge reads Reflex whatever the payload named", func(t *testing.T) {
		for _, sent := range []string{"", "Legerity"} {
			a, err := buildAction(actor, ActionPayload{
				ActorID: actor, ReactToID: reactTo, ReactionKind: "dodge",
				Dodge: &DodgePayload{RollCheck: &RollCheckPayload{SkillName: sent}},
			})
			if err != nil {
				t.Fatalf("sent %q: %v", sent, err)
			}
			if got := a.Dodge.RollCheck.SkillName; got != enum.Reflex.String() {
				t.Errorf("sent %q: dodge skill = %q, want Reflex", sent, got)
			}
		}
	})

	t.Run("an empty dodge object is enough", func(t *testing.T) {
		a, err := buildAction(actor, ActionPayload{
			ActorID: actor, ReactToID: reactTo, ReactionKind: "dodge", Dodge: &DodgePayload{},
		})
		if err != nil {
			t.Fatalf("dodge: {}: %v", err)
		}
		if got := a.Dodge.RollCheck.SkillName; got != enum.Reflex.String() {
			t.Errorf("dodge skill = %q, want Reflex", got)
		}
	})

	t.Run("an unknown dodge skill name is still refused at the door", func(t *testing.T) {
		_, err := buildAction(actor, ActionPayload{
			ActorID: actor, ReactToID: reactTo, ReactionKind: "dodge",
			Dodge: &DodgePayload{RollCheck: &RollCheckPayload{SkillName: "NotASkill"}},
		})
		if err == nil {
			t.Fatal("an unknown skill name must stay a client bug, refused here")
		}
	})

	t.Run("repel reads Repel whatever the payload named", func(t *testing.T) {
		a, err := buildAction(actor, ActionPayload{
			ActorID: actor, ReactToID: reactTo, ReactionKind: "repel",
			Repel: &RepelPayload{RollCheck: RollCheckPayload{SkillName: "Defense"}},
		})
		if err != nil {
			t.Fatalf("repel: %v", err)
		}
		if got := a.Repel.RollCheck.SkillName; got != enum.Repel.String() {
			t.Errorf("repel skill = %q, want Repel", got)
		}
	})

	t.Run("a closed dodge gets its Evasion entry from the server, once", func(t *testing.T) {
		for _, sent := range [][]ActionSkillPayload{nil, {{SkillName: enum.Evasion.String()}}} {
			a, err := buildAction(actor, ActionPayload{
				ActorID: actor, ReactToID: reactTo, ReactionKind: "closedDodge",
				Dodge: &DodgePayload{}, Skills: sent,
			})
			if err != nil {
				t.Fatalf("skills %v: %v", sent, err)
			}
			n := 0
			for _, s := range a.Skills {
				if s.SkillName == enum.Evasion.String() {
					n++
				}
			}
			if n != 1 {
				t.Errorf("skills %v: %d Evasion entries, want exactly 1", sent, n)
			}
		}
	})

	t.Run("a closed escape gets its Evasion entry from the server", func(t *testing.T) {
		a, err := buildAction(actor, ActionPayload{
			ActorID: actor, ReactToID: reactTo, ReactionKind: "closedEscape",
			Dodge: &DodgePayload{},
			Move:  &MovePayload{Category: string(enum.Shift), Position: [3]int{2, 2, 0}},
		})
		if err != nil {
			t.Fatalf("closedEscape: %v", err)
		}
		found := false
		for _, s := range a.Skills {
			found = found || s.SkillName == enum.Evasion.String()
		}
		if !found {
			t.Fatal("closedEscape without an Evasion entry in the payload must still read Evasion")
		}
	})
}
```

> Os nomes dos tipos de payload (`DodgePayload`, `RepelPayload`, `MovePayload`,
> `RollCheckPayload`, `ActionSkillPayload`) e a forma de cada campo (ponteiro ou valor) são os de
> `message.go`. Se algum diferir, ajuste o teste à forma real — o comportamento pedido é o mesmo.

Nos subtestes existentes que esperam a recusa da Evasão (procure por
`must carry an evasion skill entry` em `action_mapper_test.go`, hoje nas linhas ~430 e ~463):
troque "is refused" por "is accepted, with the Evasion entry added by the server" e asserte que a
entrada existe — não apague o subteste, ele vira a prova da injeção.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/app/game/ -run 'TestBuildAction_DerivesReactionSkillNames|TestBuildAction' -v`
Expected: FAIL — dodge skill = "" (ou "Legerity"), repel skill = "Defense", closed sem Evasão recusada.

- [ ] **Step 3: Write minimal implementation**

Em `buildAction`:

```go
	var dodge *action.Dodge
	if p.Dodge != nil {
		rc, err := buildRollCheck(p.Dodge.RollCheck)
		if err != nil {
			return nil, err
		}
		dodge = &action.Dodge{}
		if rc != nil {
			dodge.RollCheck = *rc
		}
		// Derived, like the hit's Accuracy: the dodge is ALWAYS read on Reflex (deriveReflex,
		// reaction_collision.go, reads exactly that name). The payload's name was validated above
		// — an unknown one is still a client bug — and is replaced here, so the front never has
		// to write a skill name to react (front-combat-phases.md §7, item 10).
		dodge.RollCheck.SkillName = enum.Reflex.String()
	}
```

```go
		repel = &action.Repel{Weapon: weapon, RollCheck: *rc}
		// Derived for the same reason as the dodge's Reflex: resolveRepel reads Repel by name.
		repel.RollCheck.SkillName = enum.Repel.String()
```

E troque o bloco da Evasão por injeção:

```go
		// Evasion is not a ReactionComponent — it names an entry inside Skills — so the loop
		// above does not cover it. The closed variants need it: without it they derive against
		// an empty RollCheck and end up strictly worse than a plain dodge. The KIND already says
		// the player wants it, so the server adds the entry instead of refusing a payload that
		// forgot it (item 10: the server derives the reactions' skill names). See
		// ReactionKind.RequiresEvasionSkill.
		if kind.RequiresEvasionSkill() {
			hasEvasion := false
			for _, s := range skills {
				if s.SkillName == enum.Evasion.String() {
					hasEvasion = true
					break
				}
			}
			if !hasEvasion {
				ev := enum.Evasion.String()
				skills = append(skills, action.Skill{SkillName: ev, RollCheck: action.RollCheck{SkillName: ev}})
			}
		}
```

Atualize também o comentário de `RequiresEvasionSkill` em
`internal/domain/match/entity/action/reaction_kind.go`: "Enforced at the WS boundary, refused…"
→ "Supplied at the WS boundary (`buildAction` adds the entry when the payload lacks it)".

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/app/game/ -run TestBuildAction -v` → PASS. Depois `go test ./internal/...` (o e2e de escape fechado e o de reação continuam passando: eles mandam `Evasion`, que agora é aceita e não duplicada).

- [ ] **Step 5: Contrato**

Em `match-combat-ws.md`, seção `attach_reaction`:
- Logo depois da tabela de `reactionKind`, um parágrafo **"As perícias da reação são derivadas pelo servidor"**: `dodge.rollCheck.skillName` é sempre `Reflex`, `repel.rollCheck.skillName` sempre `Repel`, e `closedDodge`/`closedEscape` ganham a entrada `{skillName: "Evasion"}` em `skills` quando ela falta. O que o payload mandar nesses campos é validado (nome desconhecido é recusado) e substituído — o mesmo estado do `attack.hit`. O front não escreve nome de perícia nenhum para reagir.
- A tabela do **payload mínimo** do spec §4.7, copiada.
- Os dois exemplos JSON passam a mostrar o payload mínimo (`"dodge": {}` sem `skills` no `closedDodge`; `"repel": { "weapon": "Sword" }`).
- Na tabela de erros, tire `an evasion skill entry` da lista de `invalid_action`.

- [ ] **Step 6: Commit**

```bash
git add internal/app/game/action_mapper.go internal/app/game/action_mapper_test.go \
  internal/domain/match/entity/action/reaction_kind.go docs/dev/api/match-combat-ws.md
git commit -m "feat(game): o servidor deriva as perícias das reações (Reflex, Repel, Evasion)

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 2: a reação guarda o que consumiu, e o segundo attach é recusado (spec §4.2, §4.3)

**Files:**
- Modify: `internal/domain/match/entity/action/action.go` (campo novo)
- Modify: `internal/domain/match/matchsession/match_session.go` (`AttachReaction`, `consumePendingFor`)
- Modify: `internal/domain/match/matchsession/error.go`
- Modify: `internal/domain/match/service/projection.go` (`ProjectAction`)
- Modify: `internal/app/wire/actionwire/action.go`, `from.go`
- Test: `internal/domain/match/matchsession/match_session_test.go`, `internal/domain/match/service/projection_test.go` (ou o arquivo de teste que já cobre `ProjectAction` — `grep -rn "func TestProjectAction" internal/domain/match/service`), `internal/app/wire/actionwire/from_test.go`

**Interfaces:**
- Produces:
  - `action.Action.ConsumedActionIDs []uuid.UUID` — nil quando nada foi consumido.
  - `matchsession.ErrReactorAlreadyReacted = errors.New("this character already reacted to the open action")`.
  - `func (s *MatchSession) consumePendingFor(r *action.Action) []uuid.UUID` (não exportada; antes devolvia `bool`).
  - `actionwire.Action.ConsumedActionIDs []uuid.UUID` com tag `json:"consumedActionIds,omitempty"`, preenchido por `From` em todos os níveis.
  - `ProjectAction` zera `ConsumedActionIDs` para quem não é mestre nem dono.

- [ ] **Step 1: Write the failing tests**

Em `match_session_test.go`, dentro de `TestMatchSession_ReactionCost` (usa o `setup(t)` dela, que devolve `s, chars, _, playerB, act`):

```go
	t.Run("a charged reaction names the action it consumed", func(t *testing.T) {
		s, chars, _, playerB, act := setup(t)
		pending := makeActionWithSpeed(chars[1], 30)
		if err := s.EnqueueAction(playerB, pending); err != nil {
			t.Fatalf("EnqueueAction: %v", err)
		}
		r := makeReactionTo(chars[1], act.GetID())
		r.ReactionKind = action.ReactRepel
		if _, err := s.AttachReaction(playerB, r); err != nil {
			t.Fatalf("AttachReaction: %v", err)
		}
		if len(r.ConsumedActionIDs) != 1 || r.ConsumedActionIDs[0] != pending.GetID() {
			t.Fatalf("ConsumedActionIDs = %v, want [%v]", r.ConsumedActionIDs, pending.GetID())
		}
		// The copy the turn holds says the same: the persistence and the history read it there.
		reacts := s.GetActiveRound().CurrentTurn().GetReactions()
		if len(reacts) != 1 || len(reacts[0].ConsumedActionIDs) != 1 {
			t.Fatalf("the turn's copy of the reaction lost ConsumedActionIDs: %+v", reacts)
		}
	})

	t.Run("a free reaction consumes nothing and names nothing", func(t *testing.T) {
		s, chars, _, playerB, act := setup(t)
		s.EnqueueAction(playerB, makeActionWithSpeed(chars[1], 30)) //nolint:errcheck
		r := makeReactionTo(chars[1], act.GetID())
		r.ReactionKind = action.ReactDodge
		if _, err := s.AttachReaction(playerB, r); err != nil {
			t.Fatalf("AttachReaction: %v", err)
		}
		if r.ConsumedActionIDs != nil {
			t.Fatalf("a free reaction consumed %v", r.ConsumedActionIDs)
		}
	})
```

No subteste já existente `"a two-bar reaction consumes its one combined pending action exactly once"`,
acrescente no fim: `if len(r.ConsumedActionIDs) != 1 || r.ConsumedActionIDs[0] != combined.GetID() { t.Fatalf(...) }`
(o nome da variável da reação é o do subteste; leia-o antes).

Em `TestMatchSession_AttachReaction`, um subteste novo:

```go
	t.Run("a second reaction by the same character is refused, charging nothing", func(t *testing.T) {
		playerA, playerB := uuid.New(), uuid.New()
		s, chars := sessionWithParticipants(playerA, playerB)
		s.GetActiveRound().SetMode(enum.Race)
		a := makeActionWithSpeed(chars[0], 10)
		a.TargetID = []uuid.UUID{chars[1]}
		if err := s.EnqueueAction(playerA, a); err != nil {
			t.Fatalf("EnqueueAction: %v", err)
		}
		opened := mustOpen(t, s)
		act := opened.GetAction()

		first := makeReactionTo(chars[1], act.GetID())
		first.ReactionKind = action.ReactDodge
		if _, err := s.AttachReaction(playerB, first); err != nil {
			t.Fatalf("first attach: %v", err)
		}
		// Something the second one could consume, if it were (wrongly) let through.
		queued := makeActionWithSpeed(chars[1], 30)
		if err := s.EnqueueAction(playerB, queued); err != nil {
			t.Fatalf("EnqueueAction(queued): %v", err)
		}
		pendingBefore := len(s.PendingActions())

		second := makeReactionTo(chars[1], act.GetID())
		second.ReactionKind = action.ReactRepel
		if _, err := s.AttachReaction(playerB, second); !errors.Is(err, matchsession.ErrReactorAlreadyReacted) {
			t.Fatalf("second attach: err = %v, want ErrReactorAlreadyReacted", err)
		}
		if got := len(s.PendingActions()); got != pendingBefore {
			t.Fatalf("the refused attach consumed from the queue: %d -> %d", pendingBefore, got)
		}
		if got := len(opened.GetReactions()); got != 1 {
			t.Fatalf("the turn holds %d reactions, want 1", got)
		}
	})
```

Em `projection_test.go` (ou onde `ProjectAction` é testado), table-driven:

```go
func TestProjectAction_ConsumedActionIDsAreTheOwnersAndTheMasters(t *testing.T) {
	reactor := uuid.New()
	consumed := []uuid.UUID{uuid.New()}
	r := action.NewAction(reactor, nil, uuid.New(), nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil)
	r.ReactionKind = action.ReactRepel
	r.ConsumedActionIDs = consumed

	tests := []struct {
		name   string
		viewer service.Viewer
		want   int
	}{
		{"master", service.Viewer{IsMaster: true}, 1},
		{"owner", service.Viewer{Owns: map[uuid.UUID]bool{reactor: true}}, 1},
		{"third party", service.Viewer{Owns: map[uuid.UUID]bool{}}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.ProjectAction(*r, tt.viewer, false)
			if len(got.ConsumedActionIDs) != tt.want {
				t.Fatalf("ConsumedActionIDs = %v, want %d entries", got.ConsumedActionIDs, tt.want)
			}
		})
	}
}
```

> `service.Viewer` é `{IsMaster bool; Owns map[uuid.UUID]bool}` (`projection.go:14`) — o "dono"
> é quem tem o reator em `Owns`.

Em `from_test.go`: `From` com `ConsumedActionIDs = [id]` devolve `ConsumedActionIDs == [id]` em `Full`, `Opened` e `Declaration`; com nil, o JSON não tem a chave `consumedActionIds`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/domain/match/... ./internal/app/wire/... -run 'ReactionCost|AttachReaction|ConsumedActionIDs|From' -v`
Expected: FAIL de compilação (`ConsumedActionIDs`, `ErrReactorAlreadyReacted` indefinidos).

- [ ] **Step 3: Implement**

`action.go`, logo depois de `SystemBias int` (e antes dos campos não exportados):

```go
	// ConsumedActionIDs is what this REACTION took off the queue when it was attached — one
	// pending action of the reactor per bar its kind charges, the best-keyed one
	// (scheduler.BestPendingFor), a combined action counted once. nil on a free reaction, on a
	// charged one that found nothing queued, and on every plain action. Set only by
	// MatchSession.AttachReaction. It is the queue's secret, so service.ProjectAction keeps it
	// for the master and the reactor's owner only; the owner needs it to tell "consumed" apart
	// from "lost" when reconciling what they had declared (front-combat-phases.md §7, item 3).
	ConsumedActionIDs []uuid.UUID
```

`error.go`:

```go
	// ErrReactorAlreadyReacted means this character already has a reaction attached to the open
	// action. One reaction per character per action: the chain counts a character once
	// (buildChainOrder), so a second one would be charged and then ignored.
	ErrReactorAlreadyReacted = errors.New("this character already reacted to the open action")
```

`match_session.go`, em `AttachReaction`, logo depois da checagem `act.GetID() != r.ReactToID`
(e antes de `s.rollActionDice(r)`):

```go
	// One reaction per character per action, refused before a die falls or a bar is charged —
	// the same "validate before mutating" rule as the checks above. A second one was accepted,
	// charged and then silently skipped by the chain (buildChainOrder counts a character once).
	for _, existing := range t.GetReactions() {
		if existing.GetActorID() == r.GetActorID() {
			return nil, ErrReactorAlreadyReacted
		}
	}
```

E troque o bloco do consumo:

```go
	if !r.ReactionKind.IsFree() {
		r.ConsumedActionIDs = s.consumePendingFor(r)
		systemBias := 0
		if len(r.ConsumedActionIDs) > 0 {
			systemBias = -1
		}
		s.deriveSpeeds(r, systemBias)
		s.chargeReactionBars(r)
	}
```

(mantenha os comentários que já existem ali). `consumePendingFor` passa a devolver os IDs:

```go
// consumePendingFor pulls this character's about-to-open action off the queue, once per bar the
// reaction charges, and returns what it took — in the order of ReactionKind.Bars(), nil when
// nothing was there.
//
// A combined action sits on both bars and is counted once — it leaves on the first bar that
// finds it and is simply not there for the second.
func (s *MatchSession) consumePendingFor(r *action.Action) []uuid.UUID {
	var consumed []uuid.UUID
	for _, bar := range r.ReactionKind.Bars() {
		victim := s.scheduler.BestPendingFor(s.scheduleInput(), r.GetActorID(), bar)
		if victim == nil {
			continue
		}
		s.activeQueue.ExtractByID(victim.GetID())
		consumed = append(consumed, victim.GetID())
	}
	return consumed
}
```

`ConsumedActionIDs` é atribuído **antes** de `s.roundOrch.AttachReaction`, que copia a reação
para dentro do turno — por isso a cópia do turno também o tem.

`projection.go`, em `ProjectAction`, junto do `out.Trigger = nil`:

```go
	// What the reaction took off the queue is the queue's secret (the queue is the master's;
	// the owner knows their own) — see action.Action.ConsumedActionIDs.
	out.ConsumedActionIDs = nil
```

`actionwire/action.go`, depois de `SystemBias`:

```go
	// ConsumedActionIDs is what a charged reaction took off the reactor's queue
	// (action.Action.ConsumedActionIDs). Not cut by Level: who may see it is
	// service.ProjectAction's decision, already taken upstream (it nils it for anyone but the
	// master and the owner). omitempty keeps it off every action and every free reaction.
	ConsumedActionIDs []uuid.UUID `json:"consumedActionIds,omitempty"`
```

`from.go`, no literal de `out`: `ConsumedActionIDs: a.ConsumedActionIDs,`.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/...` → PASS. Se o golden test do histórico REST (`get_match_history_golden_test.go`) quebrar, é porque alguma fixture tem reação cobrada com fila: atualize o golden só se a diferença for exatamente a chave `consumedActionIds`.

- [ ] **Step 5: Contrato**

Em `match-combat-ws.md`, `attach_reaction`:
- No parágrafo "Uma reação **cobrada** consome a ação enfileirada…": acrescente **qual** — "por barra que cobra, a de **melhor chave** naquela barra (`BestPendingFor`, a mesma escolha do escalonador); uma ação combinada, que está nas duas barras, sai uma vez só. Os IDs consumidos vão ao dono e ao mestre em [`reaction_attached`](#reaction_attached)" (a seção nasce na Task 3; o link já pode ficar).
- Na tabela de erros, em `game_error`, acrescente `this character already reacted to the open action` — "segundo `attach_reaction` do mesmo personagem na mesma ação, de qualquer tipo, aberta ou não. Recusado antes de rolar ou cobrar: a fila e as barras não mudam".
- §7 (catálogo de erros): nenhuma linha nova (é `game_error`), mas cite a mensagem na linha de `game_error` se a lista estiver lá.

- [ ] **Step 6: Commit**

```bash
git add internal/domain internal/app/wire docs/dev/api/match-combat-ws.md
git commit -m "feat(match): a reação guarda as ações que consumiu e o segundo attach é recusado

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 3: `reaction_attached` — a resposta a quem reagiu, e ao mestre (spec §4.2, §4.3, §4.4)

**Files:**
- Modify: `internal/app/game/message.go` (tipo e payload novos)
- Modify: `internal/app/game/room.go` (`handleReaction`)
- Test: `internal/app/game/reaction_attached_e2e_test.go` (novo, `package game_test`)
- Docs: `docs/dev/api/match-combat-ws.md`

**Interfaces:**
- Consumes: `action.Action.ConsumedActionIDs` (Task 2).
- Produces:
  - `MsgTypeReactionAttached MessageType = "reaction_attached"`.
  - `type ReactionAttachedPayload struct { TurnID uuid.UUID \`json:"turnId"\`; ReactionID uuid.UUID \`json:"reactionId"\`; ActorID uuid.UUID \`json:"actorId"\`; ConsumedActionIDs []uuid.UUID \`json:"consumedActionIds"\` }` — `ConsumedActionIDs` nunca nil.

- [ ] **Step 1: Write the failing e2e test**

Crie `internal/app/game/reaction_attached_e2e_test.go`:

```go
package game_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/google/uuid"
)

// This file is the Phase 7 back package's reaction_attached (spec §4.2, §4.3, §4.4): the
// answer to whoever attached a reaction — and to the master, who is told which queued actions
// a charged reaction consumed. The table hears nothing: that someone reacted is not table news
// until the master opens it.

func lastReactionAttached(t *testing.T, c *collector) game.ReactionAttachedPayload {
	t.Helper()
	msgs := c.snapshotMessages()
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Type != game.MsgTypeReactionAttached {
			continue
		}
		var p game.ReactionAttachedPayload
		if err := json.Unmarshal(msgs[i].Payload, &p); err != nil {
			t.Fatalf("unmarshal reaction_attached: %v", err)
		}
		return p
	}
	t.Fatal("no reaction_attached in the collected messages")
	return game.ReactionAttachedPayload{}
}

func TestE2E_ReactionAttachedAnswersTheReactorAndTheMasterOnly(t *testing.T) {
	f := newCombatFixture(t, withBystander)
	master, player, blind, mc, pc, bc := f.connectTable(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	defer blind.Close()  //nolint:errcheck

	actionID := f.openAttackOn(t, player, master, mc)
	sendWS(t, player, string(game.MsgTypeAttachReaction), game.ActionPayload{
		ActorID: f.victimID, ReactToID: actionID, ReactionKind: "dodge", Dodge: &game.DodgePayload{},
	})
	if !pc.await(game.MsgTypeReactionAttached, 2*time.Second) {
		t.Fatalf("the reactor never got reaction_attached; got %v", messageTypes(pc.snapshotMessages()))
	}
	if !mc.await(game.MsgTypeReactionAttached, 2*time.Second) {
		t.Fatal("the master never got reaction_attached")
	}
	got := lastReactionAttached(t, pc)
	if got.ActorID != f.victimID || got.ReactionID == uuid.Nil || got.TurnID == uuid.Nil {
		t.Fatalf("reaction_attached = %+v", got)
	}
	toMaster := lastReactionAttached(t, mc)
	if toMaster.ActorID != got.ActorID || toMaster.ReactionID != got.ReactionID || toMaster.TurnID != got.TurnID {
		t.Fatalf("the master's copy differs: %+v vs %+v", toMaster, got)
	}
	// consumedActionIds is ALWAYS a list — [] on a free reaction, never null or absent.
	var raw map[string]json.RawMessage
	for _, m := range pc.snapshotMessages() {
		if m.Type == game.MsgTypeReactionAttached {
			if err := json.Unmarshal(m.Payload, &raw); err != nil {
				t.Fatal(err)
			}
		}
	}
	if string(raw["consumedActionIds"]) != "[]" {
		t.Fatalf("consumedActionIds = %s, want []", raw["consumedActionIds"])
	}
	// The table hears nothing.
	time.Sleep(150 * time.Millisecond)
	if bc.count(game.MsgTypeReactionAttached) != 0 {
		t.Fatal("a bystander was told that someone reacted")
	}
}
```

(O import de `enum` só é usado nos testes seguintes; se não usar, tire.)

Mais dois testes no mesmo arquivo:

```go
func TestE2E_ReactionAttachedNamesTheConsumedAction(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc, pc := collectFrom(master), collectFrom(player)

	actionID := f.openAttackOn(t, player, master, mc)
	// The victim (the player's too) queues something AFTER the attack opened, so the
	// attack is the one under the baton and this one waits in the queue.
	queued := mc.count(game.MsgTypeActionQueued)
	f.enqueueAttackFrom(t, player, f.victimID)
	if !awaitCount(mc, game.MsgTypeActionQueued, queued+1, 2*time.Second) {
		t.Fatal("the victim's action never reached the queue")
	}
	victimActionID := lastActionQueuedID(t, mc) // read actionId off the newest action_queued

	sword := "Sword"
	sendWS(t, player, string(game.MsgTypeAttachReaction), game.ActionPayload{
		ActorID: f.victimID, ReactToID: actionID, ReactionKind: "repel",
		Repel: &game.RepelPayload{Weapon: &sword},
	})
	if !pc.await(game.MsgTypeReactionAttached, 2*time.Second) || !mc.await(game.MsgTypeReactionAttached, 2*time.Second) {
		t.Fatal("reaction_attached never arrived")
	}
	for name, c := range map[string]*collector{"reactor": pc, "master": mc} {
		got := lastReactionAttached(t, c)
		if len(got.ConsumedActionIDs) != 1 || got.ConsumedActionIDs[0] != victimActionID {
			t.Fatalf("%s: consumedActionIds = %v, want [%v]", name, got.ConsumedActionIDs, victimActionID)
		}
	}
}

func TestE2E_TheMasterReactsThroughAnNPCAndIsAnsweredOnce(t *testing.T) {
	// The NPC is the target. Build it the way add_npc_e2e_test.go seats a live NPC (or, if the
	// fixture already has an option for an NPC target, use it): the attacker (the player's
	// character) targets the NPC, the master attaches a dodge with actorId = the NPC sheet.
	// Assert: the master gets exactly ONE reaction_attached (he is reactor and master at once),
	// with actorId = the NPC; the player gets none.
}
```

`lastActionQueuedID` não existe: escreva-o no arquivo (último `action_queued` do coletor,
`game.ActionQueuedPayload.ActionID`). Para o teste do NPC, leia `add_npc_e2e_test.go` e
`combat_e2e_test.go` (`withAddLiveNPC`, `newCombatFixture`) e monte o alvo NPC com o que existir —
se nada servir, acrescente ao fixture uma opção `withNPCTarget` que inscreve uma ficha de NPC do
mestre na sessão (`session.AddNPC(sheetUUID, sheet, f.masterUUID)`) e use `enqueueAttackFrom`
mirando nela. O corpo do teste é o do comentário acima; escreva-o completo.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/app/game/ -run 'TestE2E_ReactionAttached|TestE2E_TheMasterReactsThroughAnNPC' -v`
Expected: FAIL de compilação (`MsgTypeReactionAttached` indefinido).

- [ ] **Step 3: Implement**

`message.go` — o tipo, junto de `MsgTypeReactionOpened`:

```go
	// reaction_attached answers attach_reaction: to whoever reacted, and to the master — the
	// one message that names the queued actions a charged reaction consumed (Phase 7, item 3).
	MsgTypeReactionAttached MessageType = "reaction_attached"
```

e o payload, perto de `ReactionOpenedPayload`:

```go
// ReactionAttachedPayload answers an accepted attach_reaction. It goes to whoever reacted —
// the only one who did not otherwise learn the reaction's ID, which open_reaction and the
// reconnect's ownReactions key on — and to the master, because ConsumedActionIDs is the news
// that a pending action left the queue, and the queue is theirs. The same message, not two
// shapes for one fact (game-server.instructions.md, "fewer event types"). The table hears
// nothing: that someone reacted is not table news until the master opens it.
//
// ConsumedActionIDs is ALWAYS a list ([] on a free reaction): the client reconciles its
// declared actions against it, and "absent" would be one more case to read.
type ReactionAttachedPayload struct {
	TurnID            uuid.UUID   `json:"turnId"`
	ReactionID        uuid.UUID   `json:"reactionId"`
	ActorID           uuid.UUID   `json:"actorId"`
	ConsumedActionIDs []uuid.UUID `json:"consumedActionIds"`
}
```

`room.go`, `handleReaction` — monte o payload **na mesma seção crítica** do `Execute` e envie
depois do unlock, antes do `publishResolution`:

```go
	r.mu.Lock()
	result, err := r.deps.AttachReactionUC.Execute(context.Background(), session, client.userUUID, reaction)
	var turnID uuid.UUID
	var attached ReactionAttachedPayload
	if err == nil {
		turnID = session.CurrentTurnID()
		// Read under the lock: `reaction` is ours, but Execute just wrote ConsumedActionIDs into
		// it, and the session's copy is the one later readers use — both are written before the
		// unlock, never after.
		attached = ReactionAttachedPayload{
			TurnID: turnID, ReactionID: reaction.GetID(), ActorID: reaction.GetActorID(),
			ConsumedActionIDs: append([]uuid.UUID{}, reaction.ConsumedActionIDs...),
		}
	}
	r.mu.Unlock()
	if err != nil {
		client.SendMessage(NewErrorMessage("game_error", err.Error()))
		return
	}
	// Before the master's resolution_updated, on the same lane and goroutine: the reactor and
	// the master learn the reaction's ID before anything else names it.
	ack := NewServerMessage(MsgTypeReactionAttached, attached)
	client.SendMessage(ack)
	if !r.IsMaster(client.userUUID) {
		// The master reacting through an NPC is reactor AND master: one copy, not two.
		r.sendToMaster(ack)
	}
	r.publishResolution(turnID, result.Resolution)
```

(mantenha o comentário longo que já explica o `publishResolution`).

- [ ] **Step 4: Run tests**

Run: `go test ./internal/app/game/ -run 'ReactionAttached|MasterReactsThroughAnNPC' -v` → PASS; `go test -race ./internal/app/game/` → PASS.

- [ ] **Step 5: Contrato**

Em `match-combat-ws.md`:
- §2: depois da nota "NPC é do mestre", uma frase: "o mesmo vale para [`attach_reaction`](#attach_reaction): o mestre reage **pelo NPC** que é alvo, pela mesma checagem `charToPlayer`. A ficha de jogador continua negada a ele."
- §3, índice: `attach_reaction` → "jogador **alvo** da ação aberta, pelo próprio personagem; o mestre, pelo NPC alvo". Servidor → cliente: linha nova `reaction_attached` — "quem reagiu **+ mestre**".
- `attach_reaction`: "**Dispara:**" passa a ser `reaction_attached` (a quem reagiu e ao mestre; uma cópia só quando é o mestre), depois `resolution_updated` master-only. Tire "Não há ack próprio". Mantenha "não há broadcast: a mesa não é avisada".
- Seção nova `### reaction_attached` em §5, depois de `action_queued`: o exemplo do spec §4.2, a tabela de campos, o destino, "`consumedActionIds` é sempre lista", e o uso: "o front tira a ação consumida da lista de declaradas (dono) e da fila (mestre); ela **não** foi perdida — ver a regra de reconciliação em [`match_full_state`](#match_full_state)".

- [ ] **Step 6: Commit**

```bash
git add internal/app/game docs/dev/api/match-combat-ws.md
git commit -m "feat(game): reaction_attached responde a quem reagiu e ao mestre, com o que foi consumido

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 4: `reaction_opened` leva a reação projetada (spec §4.1) — **opus**

**Files:**
- Modify: `internal/app/game/message.go` (`ReactionOpenedPayload`)
- Modify: `internal/app/game/room.go` (arm `MsgTypeOpenReaction`; função nova `recordOpenedReactionMoveViews`)
- Modify: `internal/app/game/turn_writes.go` (campo `reactionMoveViews`)
- Test: `internal/app/game/reaction_opened_e2e_test.go` (novo)
- Docs: `docs/dev/api/match-combat-ws.md`

**Interfaces:**
- Consumes: `turnActionWireLocked(act action.Action, pid uuid.UUID, v domainservice.Viewer, origin *[3]int) actionwire.Action`, `openedMoveViewLocked`, `sessionPlayerViewsLocked(except uuid.UUID, gate func(pid uuid.UUID) (masteraction.View, bool)) map[uuid.UUID]masteraction.View`, `pieceSlotOf(characterID string) (*[3]int, bool)`, `viewerFor(playerID uuid.UUID, isMaster bool) domainservice.Viewer`, `ownerOfLocked`, `openTurnIDLocked`, `turnWritesLocked`, `OpenReactionResult.Opened` (ponteiro para a reação aberta).
- Produces:
  - `ReactionOpenedPayload{TurnID, ReactionID, Reaction actionwire.Action \`json:"reaction"\`}`.
  - `turnWrites.reactionMoveViews map[uuid.UUID]map[uuid.UUID]masteraction.View` (por `reactionID`), drenado por `takeTurnWritesLocked` como o resto — **a Task 6 o grava**.
  - `func (r *Room) reactionWireLocked(react action.Action, pid uuid.UUID, v domainservice.Viewer) actionwire.Action` — o corte de uma reação para um destinatário, com a origem = casa atual da peça do reator. **A Task 5 a reusa.**

- [ ] **Step 1: Write the failing e2e tests**

`reaction_opened_e2e_test.go` (`package game_test`). Use `connectTable` (mestre, dono, terceiro) e
`withVictimPiece`/`seedBoard` do `escape_e2e_test.go` para a peça do reator. Escreva:

1. `TestE2E_ReactionOpenedCarriesTheReactionCutPerRecipient` — `closedEscape` de `escapeFrom` para
   `escapeTo` (use `attachClosedEscape`), aberta pelo mestre. Leia o `reaction_opened` de cada um:
   - **mestre**: `reaction.reactionKind == "closedEscape"`; `reaction.skills` tem `Evasion`;
     `reaction.dodge.rollCheck.result != nil` (Full);
   - **dono** (o jogador): `reactionKind == "closedEscape"`; `dodge.rollCheck.result == nil`
     (Opened corta); `move.position == escapeTo`;
   - **terceiro**: `reactionKind == "escape"` (rebaixado); nenhuma entrada `Evasion` em `skills`;
     sem `consumedActionIds`; `dodge.rollCheck.result == nil`.
   Para ler os campos, decodifique `payload.reaction` em `actionwire.Action` (importe
   `internal/app/wire/actionwire`).
2. `TestE2E_ReactionOpenedGatesTheEscapeDestinationByTheBystandersSight` — table-driven no molde
   de `TestE2E_TheOpenedMoveIsGatedByTheRecipientsSight` (`hidden_move_e2e_test.go`), mas com o
   **reator** (vítima) no papel da peça que se move: casos "vê o destino" (`move.position`
   presente para o terceiro), "não vê nenhuma ponta" (sem `position`), "peça `visible: false`"
   (sem `position`). Mestre e dono sempre recebem `position`. Escreva um helper
   `seedReactorBoard(t, at [2]int, visible bool, walls ...mapentity.WallSegment)` copiando
   `seedMoverBoard` e pondo a peça da **vítima** em `at` (leia `seedMoverBoard` e `seedBoard`
   antes). O destino da fuga é o `to` de cada caso; a categoria `Dash` (`attachDashEscape`).
3. `TestE2E_ReactionOpenedReachesTheMasterBeforeTheRecomputedResolution` — no coletor do mestre,
   o índice do `reaction_opened` é menor que o do `resolution_updated` que veio depois do
   `open_reaction` (conte os `resolution_updated` antes de mandar o `open_reaction`, como
   `escapeStage` faz, e compare as posições em `snapshotMessages()`).
4. `TestE2E_OpeningAReactionRecordsWhatEachPlayerSawOfItsDestination` — depois de abrir uma fuga
   que o terceiro vê, feche o turno (`close_turn` com `confirm: true`) e confira, no
   `TurnCloseData` que o `mockRoundRepoHandler` recebeu, `ReactionMoveViews[reactionID][bystanderUUID] == masteraction.ViewFull`
   e nenhuma entrada para mestre nem dono. **Este teste só compila depois da Task 6** (o campo
   `TurnCloseData.ReactionMoveViews` nasce lá): escreva-o agora com `t.Skip("Task 6 wires TurnCloseData.ReactionMoveViews")`
   na primeira linha e deixe a Task 6 tirar o skip. Até lá, a prova de que o veredito foi guardado
   é um teste interno (`package game`) em `turn_writes_test.go` (crie se não existir) que chama
   `recordOpenedReactionMoveViews` numa `Room` montada como em `fog_dispatch_test.go`
   (`fogTestRoom`) e lê `r.pendingTurns[turnID].reactionMoveViews` — escreva esse.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/app/game/ -run 'ReactionOpened|OpeningAReactionRecords' -v`
Expected: FAIL — `reaction_opened` sem `reaction`.

- [ ] **Step 3: Implement**

`message.go`:

```go
// ReactionOpenedPayload announces who narrates next, and with what (Phase 7, item 1). Reaction
// is the opened reaction cut for THIS recipient by the very rule turn_opened.action uses
// (turnActionWireLocked): the master gets it whole (Full); everyone else — its own owner
// included — gets service.ProjectAction (closedDodge/closedEscape demoted, the Evasion entry
// and consumedActionIds stripped for a third party) at Opened, with move.position passing the
// reactor's piece's fog gate. Projected, so it travels on the DIRECT lane (dispatchPerPlayer),
// and the master's resolution_updated that follows is sent after it by the same goroutine.
type ReactionOpenedPayload struct {
	TurnID     uuid.UUID         `json:"turnId"`
	ReactionID uuid.UUID         `json:"reactionId"`
	Reaction   actionwire.Action `json:"reaction"`
}
```

`turn_writes.go`, em `turnWrites`:

```go
	// reactionMoveViews is, per opened reaction that moves (an escape), what each session
	// player saw of its destination when the master opened it (recordOpenedReactionMoveViews) —
	// written with the reaction's own row (actions.move_views): the history shows each reader
	// a reaction's move as they saw it then (Phase 7, item 1). Absent for a reaction never
	// opened: it was never shown.
	reactionMoveViews map[uuid.UUID]map[uuid.UUID]masteraction.View
```

`room.go` — a função de corte da reação, perto de `turnActionWireLocked`:

```go
// reactionWireLocked cuts an opened reaction for one recipient — the same rule as the turn's
// action (turnActionWireLocked), with the gate's origin at the reactor's piece NOW: a reaction
// never moves at the opening (an escape waits for the close, B13), so where the piece stands is
// both where it stood when the master opened it and what a reconnect can still read. A reaction
// carries no move.from (buildAction never derives one), so an "origin only" verdict leaves it
// with the category alone — position only reaches who sees the destination.
//
// Shared by the live reaction_opened and match_full_state's openTurn.reactions. The caller
// MUST hold r.mu (a read lock is enough).
func (r *Room) reactionWireLocked(react action.Action, pid uuid.UUID, v domainservice.Viewer) actionwire.Action {
	origin, _ := r.pieceSlotOf(react.GetActorID().String())
	return r.turnActionWireLocked(react, pid, v, origin)
}

// recordOpenedReactionMoveViews records what each session player saw of an opened reaction's
// destination — the verdict reactionWireLocked's gate reached for them — and holds it with the
// turn (turnWrites.reactionMoveViews), to be written with the reaction's row at the close.
// Only while that turn is still the open one, like recordOpenedMoveViews. The caller must NOT
// hold r.mu.
func (r *Room) recordOpenedReactionMoveViews(turnID uuid.UUID, react action.Action) {
	if react.Move == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.openTurnIDLocked() != turnID {
		return
	}
	actorID, to := react.GetActorID(), react.Move.Position
	origin, _ := r.pieceSlotOf(actorID.String())
	views := r.sessionPlayerViewsLocked(r.ownerOfLocked(actorID), func(pid uuid.UUID) (masteraction.View, bool) {
		return r.openedMoveViewLocked(pid, actorID, nil, &to, origin)
	})
	w := r.turnWritesLocked(turnID)
	if w.reactionMoveViews == nil {
		w.reactionMoveViews = map[uuid.UUID]map[uuid.UUID]masteraction.View{}
	}
	w.reactionMoveViews[react.GetID()] = views
}
```

> `pieceSlotOf` **não** pega lock: exige `r.mu` preso pelo chamador (doc dela, `room.go:3285`) —
> por isso as duas funções acima a chamam dentro da seção crítica. `sessionPlayerViewsLocked`
> recebe o dono a excluir; o mestre ela já pula por dentro.

O arm `MsgTypeOpenReaction` — copie a reação sob o lock e troque o broadcast por
`dispatchPerPlayer`:

```go
		r.mu.Lock()
		result, err := r.deps.OpenReactionUC.Execute(context.Background(), session, client.userUUID, payload.ReactionID)
		turnID := session.CurrentTurnID()
		var opened action.Action
		if err == nil {
			// A copy, under the lock: result.Opened aliases the turn's own reaction.
			opened = *result.Opened
		}
		r.mu.Unlock()
		if err != nil {
			client.SendMessage(NewErrorMessage("game_error", err.Error()))
			return
		}
		// (keep the existing comment about no escape displacing here)
		r.recordOpenedReactionMoveViews(turnID, opened)
		// Who narrates next, and with what, is public — cut per recipient, so on the direct lane;
		// the master's resolution_updated below goes after it on the same lane, from this same
		// goroutine, which is what makes "reaction_opened, then the recomputed resolution" a
		// promise rather than luck.
		r.dispatchPerPlayer(func(pid uuid.UUID, isMaster bool) *Message {
			r.mu.RLock()
			defer r.mu.RUnlock()
			msg := NewServerMessage(MsgTypeReactionOpened, ReactionOpenedPayload{
				TurnID: turnID, ReactionID: payload.ReactionID,
				Reaction: r.reactionWireLocked(opened, pid, r.viewerFor(pid, isMaster)),
			})
			return &msg
		})
		r.publishResolution(turnID, result.Resolution)
```

> `viewerFor` lê a sessão e não pega lock: ela roda **dentro** do `r.mu.RLock()` do `build`, como
> no `turn_opened` (`announceOpenedTurn`, `room.go:1919-1929`), que é o molde deste trecho.

Apague o `data, _ := json.Marshal(out); go func() { r.broadcast <- data }()` antigo.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/app/game/ -run 'ReactionOpened|turnWrites|OpeningAReaction' -v` → PASS (o teste 4 do e2e em skip); `go test -race ./internal/app/game/` → PASS. Rode também os e2e de escape e de visibilidade inteiros (`-run 'Escape|Visibility|Chain'`): nenhum pode ter dependido do `reaction_opened` sem `reaction`.

- [ ] **Step 5: Contrato**

Seção `reaction_opened` de `match-combat-ws.md`, reescrita:
- Destino: mesa inteira, **uma cópia por destinatário** (projetada) — pista direta.
- O exemplo do mestre (Full, `closedEscape` com `Evasion` e números) e o de um terceiro (`escape`, sem `Evasion`, sem números de `dodge`, `move` com `position` só porque vê o destino).
- A tabela de corte: "o mesmo de [`turn_opened`](#o-corte-de-action-design-spec-41)" — mestre `Full`; todo o resto, o dono incluído, `Opened` depois de `ProjectAction`; o rótulo fechado rebaixado e a entrada `Evasion` e `consumedActionIds` tirados para quem não é dono nem mestre.
- O `move`: mesmo portão de fog do `turn_opened`; origem julgada = a casa da peça do reator na abertura (ela não anda); uma reação nunca tem `move.from`, então "só a origem" vira só `category`. **O front tem que tolerar `move` sem `position`.**
- Ordem: "`reaction_opened` chega ao mestre **antes** do `resolution_updated` recomputado — mesma pista, mesmo goroutine".
- Em `open_reaction` ("**Dispara:**"): "`reaction_opened` à mesa, projetado por destinatário".
- §8, diagrama: a linha `reaction_opened {turnId, reactionId}` vira `reaction_opened {turnId, reactionId, reaction}` (projetado).

- [ ] **Step 6: Commit**

```bash
git add internal/app/game docs/dev/api/match-combat-ws.md
git commit -m "feat(game): reaction_opened leva a reação, cortada por destinatário e pelo fog

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 5: `match_full_state` — `openTurn.reactions` e `ownReactions` (spec §4.1, §4.2, §5) — **opus**

**Files:**
- Modify: `internal/app/game/message.go` (`OpenTurnPayload`, `MatchFullStatePayload`, payload novo)
- Modify: `internal/app/game/room.go` (`buildMatchFullState`)
- Test: `internal/app/game/reaction_reconnect_e2e_test.go` (novo)
- Docs: `docs/dev/api/match-combat-ws.md`

**Interfaces:**
- Consumes: `reactionWireLocked` (Task 4), `Turn.OpenedReactionIDs() []uuid.UUID`, `Turn.GetReactions() []action.Action`, `action.Action.ConsumedActionIDs` (Task 2), `session.GetCharToPlayer()`.
- Produces:
  - `OpenTurnPayload.Reactions []actionwire.Action \`json:"reactions,omitempty"\`` — abertas, na ordem de abertura.
  - `type OwnReactionPayload struct { ReactionID uuid.UUID \`json:"reactionId"\`; ActorID uuid.UUID \`json:"actorId"\`; ReactionKind string \`json:"reactionKind"\`; Opened bool \`json:"opened"\`; ConsumedActionIDs []uuid.UUID \`json:"consumedActionIds"\` }` (`ConsumedActionIDs` nunca nil).
  - `MatchFullStatePayload.OwnReactions []OwnReactionPayload \`json:"ownReactions,omitempty"\``.

- [ ] **Step 1: Write the failing e2e tests**

`reaction_reconnect_e2e_test.go`. Para reconectar, use o padrão de `own_queue_e2e_test.go`
(fechar a conexão do jogador e abrir outra com `connectWS`, depois ler o `match_full_state`) — leia
o arquivo e reuse os helpers dele.

1. `TestE2E_ReconnectingReactorGetsOwnReactions` — o jogador anexa uma reação cobrada (repel) com
   uma ação da vítima na fila (como na Task 3) e **não** é aberta; reconecta. `ownReactions` tem
   uma entrada: `actorId == victimID`, `reactionKind == "repel"`, `opened == false`,
   `consumedActionIds == [ação consumida]`. Depois o mestre abre a reação; o jogador reconecta de
   novo: `opened == true`.
2. `TestE2E_ReconnectingTableGetsOpenedReactionsInOpeningOrder` — com o `areaFixture`
   (`reaction_chain_e2e_test.go`: três alvos A, B, C, quatro jogadores): A anexa repel, B anexa
   nothing, o mestre abre **B e depois A**; o jogador C reconecta (veja como o `areaFixture` abre
   conexões — `wsConn` — e escreva um `reconnect` para ele se não houver): `openTurn.reactions`
   tem duas entradas, `[B, A]` pelo `uuid`; uma reação anexada e não aberta (faça C anexar
   `dodge` antes de reconectar, sem o mestre abrir) **não** aparece em `openTurn.reactions`, mas
   aparece no `ownReactions` de C com `opened: false`.
3. `TestE2E_MasterOwnReactionsCarryOnlyTheNPCs` — alvo jogador + alvo NPC no mesmo ataque (use a
   opção de NPC alvo da Task 3); o jogador e o mestre (pelo NPC) anexam; o mestre reconecta:
   `ownReactions` do mestre tem só a do NPC; o do jogador, só a dele.
4. `TestE2E_NoOpenTurnMeansNoReactionFields` — sem turno aberto, o JSON do `match_full_state` não
   tem `ownReactions` nem `openTurn` (leia como `map[string]json.RawMessage`).

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/app/game/ -run 'Reconnecting.*Reaction|OwnReactions|NoOpenTurnMeansNoReactionFields' -v`
Expected: FAIL de compilação (`OwnReactions` indefinido).

- [ ] **Step 3: Implement**

`message.go` — em `OpenTurnPayload`:

```go
	// Reactions are the open turn's OPENED reactions, in the order the master opened them —
	// each cut for this recipient exactly as the live reaction_opened cut it
	// (reactionWireLocked), so a reconnect shows the table the same balloons and escape ghosts,
	// in the same order (the order changes the outcome). A reaction attached but not opened is
	// never here: it was never announced. Absent when none is open.
	Reactions []actionwire.Action `json:"reactions,omitempty"`
```

Em `MatchFullStatePayload`, depois de `OwnQueue`:

```go
	// OwnReactions are the open turn's reactions whose actor belongs to this recipient
	// (charToPlayer — the master through their NPCs), opened or not, in arrival order: what
	// reaction_attached told them live, for a client that reconnected after it. Opened says
	// whether the master already gave it the floor. ReactionKind is the TRUE kind — the owner
	// sees their own. ConsumedActionIDs are the queued actions it consumed: a declared action
	// named there was consumed, not lost (the contract's reconciliation rule). Absent when the
	// recipient has none, or no turn is open — absent and empty mean the same here, unlike
	// OwnQueue: no reconciliation hinges on telling them apart.
	OwnReactions []OwnReactionPayload `json:"ownReactions,omitempty"`
```

e o tipo:

```go
// OwnReactionPayload is one entry of MatchFullStatePayload.OwnReactions — see its doc.
type OwnReactionPayload struct {
	ReactionID        uuid.UUID   `json:"reactionId"`
	ActorID           uuid.UUID   `json:"actorId"`
	ReactionKind      string      `json:"reactionKind"`
	Opened            bool        `json:"opened"`
	ConsumedActionIDs []uuid.UUID `json:"consumedActionIds"`
}
```

`buildMatchFullState`, dentro do `if round.HasOpenTurn()`, depois de montar `payload.OpenTurn`
(o lock já está preso pela função inteira — `reactionWireLocked` exige isso):

```go
			// The opened reactions, in opening order, each cut as the live reaction_opened cut it.
			reactions := t.GetReactions()
			byID := make(map[uuid.UUID]action.Action, len(reactions))
			for _, re := range reactions {
				byID[re.GetID()] = re
			}
			for _, id := range t.OpenedReactionIDs() {
				if re, ok := byID[id]; ok {
					payload.OpenTurn.Reactions = append(payload.OpenTurn.Reactions, r.reactionWireLocked(re, playerID, v))
				}
			}
			// This recipient's own reactions, opened or not (OwnReactions' doc).
			charToPlayer := session.GetCharToPlayer()
			opened := make(map[uuid.UUID]bool, len(t.OpenedReactionIDs()))
			for _, id := range t.OpenedReactionIDs() {
				opened[id] = true
			}
			for _, re := range reactions {
				if charToPlayer[re.GetActorID().String()] != playerID {
					continue
				}
				payload.OwnReactions = append(payload.OwnReactions, OwnReactionPayload{
					ReactionID: re.GetID(), ActorID: re.GetActorID(),
					ReactionKind: string(re.ReactionKind), Opened: opened[re.GetID()],
					ConsumedActionIDs: append([]uuid.UUID{}, re.ConsumedActionIDs...),
				})
			}
```

> Se `reactionWireLocked` usar um `pieceSlotOf` que pega lock por dentro, a Task 4 já trocou por
> uma variante `...Locked`; aqui o lock está preso pela função toda, então só a variante serve.

- [ ] **Step 4: Run tests**

Run: os testes do Step 2 → PASS; `go test -race ./internal/app/game/` → PASS.

- [ ] **Step 5: Contrato**

Em `match_full_state`:
- O exemplo do mestre ganha `openTurn.reactions` (uma reação aberta, Full) e `ownReactions` (vazio para ele no exemplo — então omita a chave e diga isso).
- Tabela de campos: linhas `openTurn.reactions` e `ownReactions`, com os textos dos docs acima.
- **Regra de reconciliação (B12)**: acrescente — "uma declarada cujo `actionId` aparece em `ownReactions[].consumedActionIds` — ou no `consumedActionIds` de uma reação do [histórico](match-history.md) — **foi consumida** por uma reação cobrada, não perdida: sai da lista sem o aviso de perda e sem devolver o rascunho. Ao vivo, o mesmo fato chega em [`reaction_attached`](#reaction_attached)."
- §9, linha "Turno aberto, reações anexadas, …": na coluna "Recarregar", "`openTurn` com `action` e `reactions` (abertas); `ownReactions` com as suas; mestre recebe `resolution`". Na coluna "Reiniciar": acrescente "as reações e o que elas consumiram vão junto".

- [ ] **Step 6: Commit**

```bash
git add internal/app/game docs/dev/api/match-combat-ws.md
git commit -m "feat(game): match_full_state devolve as reações abertas e as do próprio destinatário

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 6: persistência e histórico — o consumo e o destino da reação como visto ao vivo (spec §4.6, D3, D4)

**Files:**
- Create: `migrations/20261005000000_actions_consumed_action_ids.sql`
- Modify: `internal/application/match/i_repository.go` (`TurnCloseData.ReactionMoveViews`, `HistoryTurn.ReactionMoveViews`)
- Modify: `internal/app/game/room.go` (`persistClosedTurn`: passar `inTurn.reactionMoveViews`)
- Modify: `internal/gateway/pg/round/persist_turn_close.go` (`insertAction`: `consumed_action_ids`; reações com `move_views`)
- Modify: `internal/gateway/pg/round/find_match_history.go` (ler as duas colunas; reação com `move_views`)
- Modify: `internal/application/match/get_match_history.go` (`shownReactionMovesFor`; limpar `ReactionMoveViews`)
- Test: integração em `internal/gateway/pg/round/` (o arquivo que já testa `PersistTurnClose`/`FindMatchHistory` — `grep -ln "PersistTurnClose\|FindMatchHistory" internal/gateway/pg/round/*_test.go`); use case em `internal/application/match/get_match_history_test.go`; e o teste 4 da Task 4 (tirar o skip)
- Docs: `docs/dev/api/match-history.md`, `docs/documentation-map.yaml`

**Interfaces:**
- Consumes: `turnWrites.reactionMoveViews` (Task 4), `action.Action.ConsumedActionIDs` (Task 2).
- Produces:
  - `TurnCloseData.ReactionMoveViews map[uuid.UUID]map[uuid.UUID]masteraction.View` (por `reactionID`).
  - `HistoryTurn.ReactionMoveViews map[uuid.UUID]map[uuid.UUID]masteraction.View` (só leitura; o use case limpa antes do wire).
  - Coluna `actions.consumed_action_ids UUID[]`.

- [ ] **Step 1: Migração**

```sql
-- +goose Up
-- +goose StatementBegin
BEGIN;

-- What a charged REACTION took off the reactor's queue when it was attached — one pending
-- action per bar its kind charges, the best-keyed one (Phase 7, item 3). The owner reconciles
-- what they had declared against it: an action named here was CONSUMED, not lost, and a
-- reconnect after the turn closed — even after a restart — can only learn that here.
-- NULL means nothing consumed: a free reaction, a charged one that found nothing queued, every
-- plain action, and every row from before this column.
ALTER TABLE actions ADD COLUMN IF NOT EXISTS consumed_action_ids UUID[];

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

ALTER TABLE actions DROP COLUMN IF EXISTS consumed_action_ids;

COMMIT;
-- +goose StatementEnd
```

Acrescente no comentário da migração `20261001000000_actions_move_views.sql`? **Não** — migração
aplicada não se edita. O novo uso de `move_views` em linha de reação vai documentado no
`documentation-map.yaml` e no `match-history.md`.

Entrada em `documentation-map.yaml`, no molde da de `20261001000000_actions_move_views.sql`:
`code_path: migrations/20261005000000_actions_consumed_action_ids.sql`, `dev_docs`:
`match-history.md` (directly_affected), `match-combat-ws.md` (possibly_affected), `notes` em PT-BR
dizendo o que a coluna guarda e que `move_views` passa a valer também para a linha de uma reação
aberta.

- [ ] **Step 2: Write the failing integration test**

No arquivo de integração de `round/` (com `//go:build integration`), um teste
`TestPersistTurnClose_WritesTheReactionsConsumedActionsAndMoveViews`: monte um turno com uma ação e
uma reação cobrada com `ConsumedActionIDs = [x]` e um `Move`; chame `PersistTurnClose` com
`ReactionMoveViews = {reactionID: {bystander: "full"}}`; leia com `FindMatchHistory` e asserte
`turn.Reactions[0].ConsumedActionIDs == [x]` e `turn.ReactionMoveViews[reactionID][bystander] == masteraction.ViewFull`.
Uma reação **sem** consumo volta com `ConsumedActionIDs == nil`. Monte os dados copiando o teste
vizinho que já grava `MoveViews` (procure `MoveViews:` nos testes de `round/`).

No mesmo arquivo, `TestFindMatchHistory_KeepsTheReactionsTruthUnprojected` (spec §4.6.1 — o fim da
partida vai revelar tudo, então o banco tem que ter tudo): grave uma reação `closedEscape` com a
entrada `Evasion` em `Skills`, `Move.Position` e `ConsumedActionIDs`; o que `FindMatchHistory`
devolve (antes de qualquer projeção — o repositório não projeta) tem `ReactionKind ==
action.ReactClosedEscape`, a entrada `Evasion`, o `Move.Position` e os `ConsumedActionIDs`.

- [ ] **Step 3: Write the failing use-case test**

Em `get_match_history_test.go`, table-driven sobre `GetMatchHistory` com um `HistoryTurn` cuja
reação é uma fuga com `Move.Position` e `ConsumedActionIDs`:

| leitor | `ReactionMoveViews` | fuga | espera `ShownReactionMoves[id]` | espera `ConsumedActionIDs` |
|---|---|---|---|---|
| mestre | — | falhou | true | presente |
| dono do reator | — | falhou | true | presente |
| terceiro que viu na abertura | `{terceiro: full}` | **falhou** | **true** (D3) | nil |
| terceiro que não viu | `{}` | falhou | false | nil |
| terceiro, linha antiga | nil | escapou, sem `LandingViews` | false | nil |

E: o `HistoryTurn` projetado sai com `ReactionMoveViews == nil` (quem viu o quê não vai ao wire).
Copie a montagem de leitor/`HistoryTurn` dos testes vizinhos de `shownReactionMovesFor`/`moveSightFor`.

- [ ] **Step 4: Run tests to verify they fail**

Run: `go test ./internal/application/match/ -run 'GetMatchHistory' -v` e
`go test -tags=integration -p 1 ./internal/gateway/pg/round/ -run PersistTurnClose_WritesTheReactions -v`
Expected: FAIL de compilação (`ReactionMoveViews` indefinido).

- [ ] **Step 5: Implement**

`i_repository.go`:
- Em `TurnCloseData`, depois de `MoveViews`:

```go
	// ReactionMoveViews is, per opened reaction that moves (an escape), what each session player
	// saw of its destination when the master opened it (reaction_opened's own gate, Phase 7) —
	// held with the turn (turnWrites) and written to that reaction's actions.move_views. A
	// reaction never opened has no entry: it was never shown. Same keys and rules as MoveViews.
	ReactionMoveViews map[uuid.UUID]map[uuid.UUID]masteraction.View
```

- Em `HistoryTurn`, depois de `LandingViews`: o mesmo campo, com a nota "read side only: the use
  case turns it into ShownReactionMoves and clears it".

`room.go`, `persistClosedTurn`: no literal de `TurnCloseData`, `ReactionMoveViews: inTurn.reactionMoveViews,`.

`persist_turn_close.go`:
- `insertAction` ganha a coluna: acrescente `consumed_action_ids` à lista de colunas e `$21` aos
  valores, com o argumento `consumedOrNil(act.ConsumedActionIDs)`:

```go
// consumedOrNil is consumed_action_ids: NULL when the reaction consumed nothing (or the row is a
// plain action), never an empty array — one way to say "nothing", the way the history reads it.
func consumedOrNil(ids []uuid.UUID) []uuid.UUID {
	if len(ids) == 0 {
		return nil
	}
	return ids
}
```

  (confirme que o pgx do repo grava `[]uuid.UUID` nil como `NULL` em `UUID[]` — os `target_ids`
  já passam `[]uuid.UUID`; se nil virar `'{}'`, passe `any(nil)` no caso vazio).
- No laço das reações: `insertAction(ctx, tx, &reactions[i], t.GetID(), *finishedAt, d.ReactionMoveViews[reactions[i].GetID()])`, e troque o comentário "a reaction carries none" por "a reaction carries what each player saw of its destination when it was opened (nil if never opened); an escape's landing is recorded in the resolution above".

`find_match_history.go`:
- `SELECT` ganha `a.consumed_action_ids` no fim; o `Scan` ganha a variável (`var consumed []uuid.UUID`).
- `actionRow` ganha `consumed []uuid.UUID`; depois do `decode()`, `act.ConsumedActionIDs = row.consumed` (nos dois ramos — ação e reação). Faça isso dentro de `actionRow.decode` para não repetir.
- No ramo da **reação**: decodifique `moveViewsRaw` como o ramo da ação já faz (mesma política: valor ilegível é logado e vira nil) e guarde em `curTurn.ReactionMoveViews[react.GetID()]` (crie o mapa na primeira vez).

`get_match_history.go`:
- `shownReactionMovesFor`: a condição de "mostrar" passa a ser
  `viewer.SeesAllOf(react.GetActorID()) || sawAtOpening || (escaped && sawWhereItEnded(...))`, com
  `sawAtOpening := tu.ReactionMoveViews[react.GetID()][userUUID] == masteraction.ViewFull`.
  Atualize o comentário: a regra é "como visto ao vivo" — na abertura (`reaction_opened`) ou na
  chegada da peça (o `piece_moved` do fechamento).
- Onde o use case limpa `pt.MoveViews, pt.LandingViews = nil, nil`, acrescente `pt.ReactionMoveViews = nil`.
- `ConsumedActionIDs` já sai zerado para terceiros por `ProjectAction` (Task 2) — nada a fazer;
  o teste do Step 3 prova.

Tire o `t.Skip` do teste 4 da Task 4 (`TestE2E_OpeningAReactionRecordsWhatEachPlayerSawOfItsDestination`)
e faça-o ler `TurnCloseData.ReactionMoveViews` do `mockRoundRepoHandler` (veja como os testes de
`turn_scoped_e2e_test.go` leem o `TurnCloseData` gravado).

- [ ] **Step 6: Run tests**

Run: `go test ./internal/...`; `go test -race ./internal/app/game/`;
`go test -tags=integration -p 1 ./internal/gateway/pg/...` (com a migração aplicada no banco de
teste: `goose -dir migrations postgres "$TEST_DATABASE_URL" up`, ou o alvo do `Makefile` que o
repo usa para isso). Tudo PASS.

- [ ] **Step 7: Contrato do histórico**

`match-history.md`:
- Em "Notas sobre os campos de `action`/`reactions`": `consumedActionIds` — só numa reação cobrada que consumiu algo; **só para o mestre e o dono do reator** (o resto recebe sem a chave); o front reconcilia as declaradas por ele (uma ação nomeada ali foi consumida, não perdida).
- Na seção "O movimento como foi visto ao vivo", parágrafo **Reações**: reescreva — a reação passou a ir à mesa ao vivo na abertura (`reaction_opened`); o terceiro vê o `move.position` de uma reação se **viu o destino na abertura** (veredito gravado no `move_views` da linha da reação) **ou** viu a peça chegar (fuga que escapou). Por isso o destino de uma fuga que falhou aparece a quem o viu na abertura. Reação nunca aberta, ou linha antiga: só `category`.
- Atualize a frase "`move_views`… Only on the turn's action" onde ela aparecer no doc.

- [ ] **Step 8: Commit**

```bash
git add migrations internal docs/dev/api/match-history.md docs/documentation-map.yaml
git commit -m "feat(history): o consumo e o destino da reação ficam gravados como foram vistos ao vivo

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 7: `targets[]` na ordem da cadeia — provar e escrever (spec §4.5)

**Files:**
- Test: `internal/app/game/reaction_chain_e2e_test.go`
- Docs: `docs/dev/api/match-combat-ws.md`

**Interfaces:**
- Consumes: `areaFixture` (`reaction_chain_e2e_test.go`), `f.attachReaction`, `f.openReaction`.
- Produces: nenhum código de produção — **se o teste falhar**, a ordem está sendo perdida em algum lugar (`ProjectResolution` ou `newResolutionUpdatedPayload`) e o conserto é preservar a ordem de `CharacterResults` ali.

- [ ] **Step 1: Write the test**

`TestE2E_TargetsComeInChainOrder`: três alvos `[A, B, C]` em `targetId`. A anexa `dodge`, B anexa
`dodge`. O mestre abre **B, depois A**. Asserte:
1. no último `resolution_updated` aberto do mestre: `targets[].targetId == [B, A, C]`;
2. o mestre manda `close_turn` com `confirm: true`; no `resolution_updated` liquidado de **C**
   (terceiro, projetado): `targets[].targetId == [B, A, C]`;
3. antes do fechamento, o mestre reconecta (ou um segundo `match_full_state` lido de uma conexão
   nova do mestre): `resolution.targets[].targetId == [B, A, C]`.

Use os faces do `TestE2E_AreaAttackWithThreeTargetsReactingDifferently` como molde para o
`scriptedFaces` (duas esquivas rolam dados; conte os faces e deixe folga — o `overran` acusa falta).

- [ ] **Step 2: Run**

Run: `go test ./internal/app/game/ -run TestE2E_TargetsComeInChainOrder -v`
Expected: PASS (a ordem já é a da cadeia — spec §1). Se falhar, conserte a ordem onde ela se
perde e rode de novo.

- [ ] **Step 3: Contrato**

Em `resolution_updated`, na tabela de campos, linha nova **`targets`** (antes de `targets[].avoided`):
"**Na ordem da cadeia**: primeiro os alvos cuja reação foi aberta, na ordem em que o mestre as
abriu; depois os alvos sem reação aberta, na ordem de `action.targetId`. Vale para o payload aberto,
o liquidado projetado, o `match_full_state.resolution` e o histórico. É assim que a ordem de
abertura — que muda o desfecho — sobrevive à reconexão. Só personagens: parede não entra em
`targets`."

- [ ] **Step 4: Commit**

```bash
git add internal/app/game/reaction_chain_e2e_test.go docs/dev/api/match-combat-ws.md
git commit -m "test(game): targets[] vem na ordem da cadeia, aberto, liquidado e na reconexão

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 8: documento mestre, `reacoes.md`, `combat-engine.md`, `AGENTS.md` (spec §2, §4.8)

**Files:**
- Modify: `docs/superpowers/specs/2026-09-20-front-combat-phases.md` (§0, §7, §11.4, §12)
- Modify: `docs/game/combate/reacoes.md`
- Modify: `docs/dev/match/combat-engine.md`
- Modify: `AGENTS.md`

Sem código. Cada texto abaixo é o conteúdo a escrever (ajuste só a costura com o vizinho).

- [ ] **Step 1: Documento mestre**

- **§0, tabela**: linha da Fase 7 → "back feito (PR deste branch — preencha o número ao abrir o
  PR); front **próximo**". Linha do fechamento da 6 → "✅ feito (PRs #82 e #69)".
- **§7**: depois de "**Escopo:**", um bloco **"Decisões de 2026-10-05 (pacote de back da Fase 7)"**
  com a tabela do spec §2 (itens 1–5, 10, 7, 8) e as decisões de front 6 e 9 em uma linha cada:
  - 6: "Clicar em **Escapar** (ou Escape defensivo) arma a escolha da casa de destino no mapa; o
    toque na casa envia, sem diálogo. Clicar em **Repelir** envia com a arma do rascunho daquele
    personagem; sem rascunho, desarmado. Segurar abre a configuração em todos."
  - 9: "**Cinco botões**: Não fazer nada, Esquivar, Escapar, Escape defensivo, Repelir. As
    fechadas saem da configuração (segurar) com **um toque em Evasão**: Esquivar + Evasão =
    `closedDodge`; Escapar + Evasão = `closedEscape`."
  E o que este PR entregou, apontando para o contrato: `reaction_attached`, `reaction_opened.reaction`,
  `openTurn.reactions`, `ownReactions`, `consumedActionIds` (ao vivo, na reconexão e no
  histórico), segundo attach recusado, `targets[]` na ordem da cadeia, perícias derivadas.
- **§7, o bullet "O default do escape é Dash; o fechado é Shift (§11.4)"** → "**A categoria do
  movimento de cada escape é fixa por tipo** (matriz do §11.4) e validada no servidor: escape e
  escape defensivo usam Dash; o escape fechado usa Shift. Não é default e não há seletor."
- **§7, o bullet "Os botões aparecem para o alvo…"**: acrescente "— e, desde o pacote de back da
  Fase 7, o mestre também reage **pelo NPC** alvo: os botões aparecem ao lado dos NPCs alvo na tela
  dele".
- **§7, "Balões"**: acrescente "a mecânica da reação chega à mesa em `reaction_opened.reaction`;
  o resultado, no `resolution_updated` liquidado".
- **§7, o fantasma de espera**: acrescente "o destino chega à mesa em `reaction_opened.reaction.move.position`,
  pelo portão de fog — quem não vê a casa não recebe o destino e não desenha o fantasma".
- **§12**: título vira "## 12. Consertos de documentação — ✅ feitos no pacote de back da Fase 7"; mantenha o texto como registro.
- **§0, depois do parágrafo "Depois da Fase 8…"**: um parágrafo **"O histórico se revela ao fim
  da partida"** — direção do dono do produto (2026-10-05): quando a partida encerrar, os jogadores
  que participaram veem todos os dados do histórico. Ainda não desenhado nem implementado; o banco
  guarda a verdade e a projeção só acontece na leitura, então é um ramo no `GET /history`. As
  perguntas em aberto (quem conta como participante; se o que é do mestre também se revela; WS ou
  só REST) estão no spec do pacote de back da Fase 7, §4.6.1.

- [ ] **Step 2: `reacoes.md`** (linguagem de jogador, sem código — regra de `docs/game/`)

- Em "As duas reações que acontecem sozinhas", troque o trecho de "Se o reflexo não for
  suficiente, o sistema avisa você. Aí você escolhe:" até o fim da lista por:

  > Enquanto o golpe está no ar, **só o mestre vê os números** — você não sabe se o seu reflexo
  > basta, porque saber disso seria saber o quanto o ataque acertou. Então você aposta:
  >
  > - **ficar na passiva** — confiar no reflexo e, se ele falhar, na defesa;
  > - **arriscar** — rolar Reflexo + 2 D10 de verdade, torcendo por sorte acima da média;
  > - **gastar sua ação** em algo mais forte (escapar, repelir).
  >
  > Você decide sem saber se precisava. É aposta — e é por isso que arriscar existe.

  (Mantenha a nota "Arriscar não melhora sua média…".)
- "## As duas esquivas difíceis" → título "## As esquivas fechadas"; troque o primeiro parágrafo
  ("**Esquiva fechada** e **escape fechado** são propositalmente trabalhosas…") por: "**Esquiva
  fechada** e **escape fechado** são para quem quer esquivar no instante exato. Montar uma é
  simples: segure o botão (Esquivar ou Escapar) e toque em **Evasão**." Mantenha o resto da seção.
- "## Como configurar": reescreva:

  > Ao ser alvo, aparecem cinco botões ao lado do seu personagem: **Não fazer nada**,
  > **Esquivar**, **Escapar**, **Escape defensivo** e **Repelir**. Você tem dois gestos:
  >
  > - **Clicar** — o caminho rápido. Esquivar, Não fazer nada e Repelir saem na hora (Repelir com a
  >   arma que você estava usando; sem ela, de mãos nuas). Escapar e Escape defensivo pedem só uma
  >   coisa: **a casa para onde você vai** — toque nela no mapa e a reação sai.
  > - **Clicar e segurar** — abre a configuração, onde você monta a reação em detalhe. É ali que
  >   entra a **Evasão**: Esquivar + Evasão é a esquiva fechada; Escapar + Evasão é o escape
  >   fechado. A narração vem depois; primeiro você define a mecânica.
  >
  > **O tipo de escape decide o deslocamento** — você não escolhe:
  >
  > | Reação | Deslocamento | Como é |
  > |---|---|---|
  > | **Escapar** e **Escape defensivo** | **Dash** | Arranque rápido, medido pelo **Accelerate**. Durante o dash você está "no ar". |
  > | **Escape fechado** | **Shift** | Deslocamento controlado, medido pelo **Brake**. Não rola dado: usa o valor médio. |

  Tire os parágrafos "Segurando em **Escapar**, a tela já vem com **Accelerate** escolhido…", a
  tabela antiga de Accelerate/Brake, "**Num escape, o movimento precisa ser Shift**…" e
  "Segurando em **Esquivar** e adicionando **Evasão**…". Mantenha o parágrafo "Por que Brake governa
  o Shift…" e a nota do Gon logo depois da tabela nova.

- [ ] **Step 3: `combat-engine.md` e `AGENTS.md`**

- `combat-engine.md`: na seção do consumo (procure "A reação consome a action pendente"),
  acrescente que a lista do que foi consumido fica na reação (`Action.ConsumedActionIDs`), vai ao
  dono e ao mestre em `reaction_attached`, na reconexão (`ownReactions`) e no histórico
  (`actions.consumed_action_ids`); e que um segundo attach do mesmo personagem é recusado
  (`ErrReactorAlreadyReacted`). Na tabela das perícias da reação (procure `` `Dodge` (Reflexo) ``),
  uma nota: "os nomes são derivados no `buildAction`; o payload não decide nenhum".
- `AGENTS.md`, "Deferred to Phase 4 (reações)": o item "Reaction visibility: players see reactions
  only when master reveals (currently master-only)" vira "Reaction visibility — **feito** (Fase 7,
  pacote de back): a reação vai à mesa projetada em `reaction_opened` e na reconexão; o cálculo
  continua do mestre até o fechamento."

- [ ] **Step 4: Commit**

```bash
git add docs AGENTS.md
git commit -m "docs: decisões da Fase 7 no documento mestre, reacoes.md sem o aviso do reflexo e sem a contradição do Accelerate

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 9: verificação final e PR

- [ ] **Step 1: Suíte inteira**

```bash
go build ./... && go vet ./... && go vet -tags integration ./... && go vet -tags smoke ./...
go test ./internal/... && go test -race ./internal/app/game/
go test -tags=integration -p 1 ./internal/gateway/pg/...
```

Todos verdes. Cole a saída resumida no PR.

- [ ] **Step 2: Revisão do contrato contra o código**

Leia de ponta a ponta as seções `attach_reaction`, `reaction_attached`, `reaction_opened`,
`resolution_updated` (`targets`), `match_full_state`, §8 e §9 de `match-combat-ws.md` e os trechos
mudados de `match-history.md`. Cada campo citado existe com esse nome JSON no código; cada
destino bate com o envio real.

- [ ] **Step 3: Revisão da branch inteira** (subagent-driven-development: o revisor final)

- [ ] **Step 4: PR**

`gh pr create` para `main`, título "feat: Fase 7 — pacote de back das reações". Corpo em PT-BR:
- o que entrou (itens 1–5, 10 e a documentação), apontando para o spec;
- **o que foi verificado**: a suíte, `-race`, integração; os e2e contra a `Room` real por websocket;
- **o que não foi, e por quê**: nenhum smoke manual por REST+WS (spec §7) — montar o cenário à mão é
  desproporcional a um pacote sem tela; a verificação de ponta a ponta, com as três contas, é do PR
  de front da Fase 7, contra este back mergeado. Por isso **não** rodar o `dev-checkout.sh` aqui:
  não há o que o dono do produto validar na mão sem a tela;
- os pontos D1 e D3 do spec, para o revisor confirmar;
- termine com `🤖 Generated with [Claude Code](https://claude.com/claude-code)`.
