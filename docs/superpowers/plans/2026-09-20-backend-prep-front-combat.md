# Preparação do backend para a Fase 6 — plano de implementação

> **Para quem executa:** SUB-SKILL OBRIGATÓRIA — use `superpowers:subagent-driven-development`
> (recomendado) ou `superpowers:executing-plans` para implementar tarefa a tarefa. Os passos
> usam checkbox (`- [ ]`) para acompanhamento.

**Goal:** fechar as sete lacunas de backend que a Fase 6 do front precisa encontrar prontas —
o movimento que aplica, o snapshot de reconexão, o ID da própria ação, o catálogo de combate,
o `Push` no dano, a finta que volta a aparecer e a categoria de movimento dos escapes validada
no servidor.

**Architecture:** tudo acontece em duas camadas já existentes. As mensagens WS nascem em
`internal/app/game/` (`message.go` define o payload, `room.go` despacha, `dispatchPerPlayer`
projeta); as regras de jogo nascem em `internal/domain/match/`. Nenhuma camada nova, nenhum
mecanismo de projeção novo — o `dispatchPerPlayer` e o gate de fog do `handlePieceMoved` são
reusados, não reescritos.

**Tech Stack:** Go, `huma/v2` + `chi` no REST, `gorilla/websocket` no game server, `humatest`
para handler HTTP, testes e2e com clientes WS reais em `internal/app/game/*_e2e_test.go`.

**Spec:** [`../specs/2026-09-20-backend-prep-front-combat-design.md`](../specs/2026-09-20-backend-prep-front-combat-design.md)
— o plano argumenta a partir dele; leia os dois.

## Global Constraints

- **Branch:** `docs/front-combat-phases`. Ela já tem commits de documentação que **não devem
  ser tocados** (`102ea0d`, `9fe042a`, `77a3226`, `99bd2be`).
- **Wire em camelCase** dos dois lados, por tag de struct. Exceção conhecida: valores de enum
  de domínio (`rung`, `applies`, `expiresAt`, `againstKind`) viajam em snake_case porque são
  `String()` de enum, não tags.
- **`room.go` é o dono do lock.** `MatchSession` não tem lock próprio: quem lê ou escreve
  estado de sessão segura `r.mu` antes. Confira quem já segura o lock antes de chamar qualquer
  helper que também o pegue.
- **`actorId` é sempre o sheetUUID** do personagem, nunca o UUID do jogador.
- **Não encoste em `indexParticipants`** (`internal/domain/match/matchsession/match_session.go`).
  É território do pacote de rostering de NPC, que roda em paralelo.
- **Não toque no repo React.** O conserto do `combat_strength` é da Fase 6 — ver §11 do spec.
- `go vet ./...` ao fim de **cada** tarefa, não só no fim do PR.
- **Nas tarefas 1, 6 e 7, rode também `go vet -tags smoke ./...` e `go vet -tags integration ./...`.**
  `AGENTS.md` avisa: arquivos atrás de build tag (`smoke`, `integration` — 12 deles) **não são
  vistos** pelo `go vet ./...` nem pelo `go test ./...` comuns, e as três tarefas mexem em tag
  de struct e formato de wire. É exatamente o caso em que a deriva se esconde ali.
- Mensagens de commit em português, no estilo do repo (`feat(match):`, `fix(game):`,
  `docs(api):`), terminando com `Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>`.

---

## File Structure

| Arquivo | Responsabilidade | Tarefas |
|---|---|---|
| `internal/app/game/message.go` | define os payloads do wire e seus comentários de contrato | 1, 6, 7 |
| `internal/app/game/room.go` | despacho, lock, projeção por destinatário | 1, 6, 7 |
| `internal/app/game/action_mapper.go` | fronteira string→enum e recusa de payload malformado | 4 |
| `internal/domain/match/entity/action/reaction_kind.go` | onde vivem as regras próprias de cada `ReactionKind` | 4 |
| `internal/domain/match/service/damage.go` | a aritmética do dano | 2 |
| `internal/domain/match/service/turn_resolver.go` | a colisão, e quem chama `RawDamage` | 2 |
| `internal/domain/match/service/projection.go` | as duas políticas de visibilidade | 3 |
| `internal/application/match/get_match_history.go` | o único chamador de `ProjectAction` | 3 |
| `internal/app/api/sheet/get_combat_catalogue.go` | **novo** — o handler do catálogo | 5 |
| `internal/app/api/sheet/routes.go` | registro de rota | 5 |
| `cmd/api/main.go` | injeção do handler novo | 5 |

---

## Task 1: `action_enqueued` devolve o `actionId`

**Files:**
- Modify: `internal/app/game/message.go` (junto de `ActionQueuedPayload`, ~linha 184)
- Modify: `internal/app/game/room.go:746`
- Test: `internal/app/game/combat_e2e_test.go`

**Interfaces:**
- Consumes: nada de tarefas anteriores.
- Produces: `ActionEnqueuedPayload{ ActionID uuid.UUID `json:"actionId"` }`.

- [ ] **Step 1: Escreva o teste que falha**

Em `internal/app/game/combat_e2e_test.go`. Siga o setup dos testes vizinhos daquele arquivo
para subir a sala com um mestre e um jogador — **não invente um harness novo**.

```go
// O ack do jogador tem que nomear a MESMA ação que o mestre viu entrar na fila. Um actionId
// qualquer, não-zero, passaria num teste que só checasse "não é zero" — e um ID que não casa
// com o do mestre é pior que nenhum, porque pull_action falharia sem explicação.
func TestEnqueueActionAckNamesTheSameActionTheMasterSaw(t *testing.T) {
	h := newCombatHarness(t) // mesmo helper que os testes vizinhos deste arquivo usam
	defer h.Close()

	h.player.send("enqueue_action", map[string]any{
		"actorId": h.playerCharID.String(),
		"attack": map[string]any{
			"hit":    map[string]any{"skillName": "Accuracy"},
			"damage": map[string]any{"skillName": "Push"},
		},
	})

	ack := h.player.waitFor(t, "action_enqueued")
	queued := h.master.waitFor(t, "action_queued")

	var ackBody struct {
		ActionID uuid.UUID `json:"actionId"`
	}
	mustUnmarshal(t, ack.Payload, &ackBody)

	var queuedBody struct {
		ActionID uuid.UUID `json:"actionId"`
	}
	mustUnmarshal(t, queued.Payload, &queuedBody)

	if ackBody.ActionID == uuid.Nil {
		t.Fatal("action_enqueued came back with a zero actionId; the player cannot address their own action")
	}
	if ackBody.ActionID != queuedBody.ActionID {
		t.Fatalf("ack names %s but the master saw %s enter the queue", ackBody.ActionID, queuedBody.ActionID)
	}
}
```

> Se os helpers do arquivo tiverem outros nomes (`newCombatHarness`, `waitFor`,
> `mustUnmarshal`), **use os que existem**. O que não pode mudar é a asserção: igualdade entre
> os dois IDs.

- [ ] **Step 2: Rode o teste e confirme que falha**

```bash
go test ./internal/app/game/ -run TestEnqueueActionAckNamesTheSameActionTheMasterSaw -v
```

Esperado: FAIL — `action_enqueued` chega como `{}`, então `ackBody.ActionID` é `uuid.Nil`.

- [ ] **Step 3: Escreva o payload**

Em `message.go`, imediatamente antes de `ActionQueuedPayload`:

```go
// ActionEnqueuedPayload acks the sender's own enqueue AND names the action.
//
// The name is not decoration and it is not a leak: this goes only to the player who sent the
// action, about their own action. The queue stays secret — what the table cannot learn is what
// OTHER people queued, and action_queued (master-only) is still the only surface that names
// someone else's.
//
// Without it the player's browser cannot refer to what it just sent: it cannot cancel it,
// cannot highlight it on the general bar, cannot tell that the next one up is theirs. It is
// the same hole PendingReactions closed for the master, with the same consequence — an ID a
// client cannot learn is an operation a client cannot invoke.
type ActionEnqueuedPayload struct {
	ActionID uuid.UUID `json:"actionId"`
}
```

- [ ] **Step 4: Mande o payload no lugar do `struct{}{}`**

Em `room.go:746`, troque:

```go
client.SendMessage(NewServerMessage(MsgTypeActionEnqueued, struct{}{}))
```

por:

```go
client.SendMessage(NewServerMessage(MsgTypeActionEnqueued, ActionEnqueuedPayload{ActionID: a.GetID()}))
```

O `action_queued` logo abaixo **não muda**.

- [ ] **Step 5: Rode o teste e confirme que passa**

```bash
go test ./internal/app/game/ -run TestEnqueueActionAckNamesTheSameActionTheMasterSaw -v
go test ./internal/app/game/
go vet ./...
```

- [ ] **Step 6: Commit**

```bash
git add internal/app/game/message.go internal/app/game/room.go internal/app/game/combat_e2e_test.go
git commit -m "$(cat <<'EOF'
feat(game): action_enqueued passa a nomear a ação do próprio jogador

O ack era {} e o navegador do jogador não tinha como se referir ao que
acabou de mandar: não cancelava, não destacava na barra, não sabia que a
próxima da fila era dele. acoes.md diz que uma ação pode ser cancelada —
e ela era inendereçável.

Mesmo buraco que PendingReactions fechou para o mestre: um ID que o
cliente não recebe é uma operação que ele não consegue invocar. O verbo
de cancelar é da Fase 6; o ID é daqui.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: `Push` entra no `RawDamage`

**Files:**
- Modify: `internal/domain/match/service/damage.go:37`
- Modify: `internal/domain/match/service/turn_resolver.go:313` e `:385-391`
- Test: `internal/domain/match/service/damage_test.go` (crie se não existir)

**Interfaces:**
- Consumes: `skillValueOf(cs *csSheet.CharacterSheet, name string) int` (`turn_resolver.go:538`),
  `enum.Push` (`enum/skill_name.go:15`), `ResolveInput.Sheets map[uuid.UUID]*csSheet.CharacterSheet`.
- Produces: `RawDamage(dice []int, name *enum.WeaponName, cat *item.WeaponsManager, push int) (int, error)`
  e `(tr TurnResolver) actorPush(in ResolveInput, a action.Action) int`.

- [ ] **Step 1: Escreva o teste que falha**

```go
// O mesmo golpe, com a mesma arma e os mesmos dados, dói mais na mão de quem tem mais Push.
// Sem esta asserção um parâmetro ignorado passaria: os dois totais seriam iguais e o teste
// continuaria verde.
func TestRawDamageAddsThePush(t *testing.T) {
	cat := item.NewWeaponsManagerFactory().Build()
	sword := enum.Sword
	dice := []int{7, 3}

	weak, err := RawDamage(dice, &sword, cat, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	strong, err := RawDamage(dice, &sword, cat, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strong-weak != 5 {
		t.Fatalf("Push moved the damage by %d, want 5 (weak=%d strong=%d)", strong-weak, weak, strong)
	}
}
```

- [ ] **Step 2: Rode o teste e confirme que falha**

```bash
go test ./internal/domain/match/service/ -run TestRawDamageAddsThePush -v
```

Esperado: FAIL de compilação — `too many arguments in call to RawDamage`.

- [ ] **Step 3: Acrescente o parâmetro em `RawDamage`**

Em `damage.go`, troque a assinatura e some o `push`. Acrescente ao comentário existente
(**não o apague** — a nota sobre a margem do acerto não entrar no dano continua valendo):

```go
// RawDamage is the weapon's rolled dice plus its flat damage bonus plus the attacker's Push.
//
// Push is the skill that measures damage, and it is NOT selectable: the weapon is what deals
// the damage, and the player does not choose which skill measures it. Swapping Push for Grab
// is the master's prerogative and belongs to their editing surface, in Phase 8.
//
// The hit margin deliberately does NOT enter here. Product owner: "it will not add into the
// damage, at least not for now, because this system is already very punishing." It is the
// one place in the system where a margin does not circulate, and that is on purpose —
// revisiting it is a post-MVP playtest question, not a TODO.
func RawDamage(dice []int, name *enum.WeaponName, cat *item.WeaponsManager, push int) (int, error) {
	w, err := lookupWeapon(name, cat)
	if err != nil {
		return 0, err
	}
	total := w.GetDamage() + push
	for _, d := range dice {
		total += d
	}
	return total, nil
}
```

- [ ] **Step 4: Rode o teste e confirme que passa**

```bash
go test ./internal/domain/match/service/ -run TestRawDamageAddsThePush -v
```

- [ ] **Step 5: Ligue os dois call sites**

Em `turn_resolver.go`, acrescente o helper ao lado de `actorSheetMissing`:

```go
// actorPush reads the attacker's Push — the skill that measures damage.
//
// It nil-guards on purpose: the wall branch is NOT behind actorSheetMissing, so an action
// whose actor sheet never reached the resolver arrives here with nothing. Zero is the honest
// answer there, and the missing sheet is already reported as a ResolutionError by the
// character branch when it applies.
func (tr TurnResolver) actorPush(in ResolveInput, a action.Action) int {
	cs, ok := in.Sheets[a.GetActorID()]
	if !ok || cs == nil {
		return 0
	}
	return skillValueOf(cs, enum.Push.String())
}
```

Depois, nos dois call sites:

```go
// turn_resolver.go:~313 — o ataque a parede
raw, err := RawDamage(a.Attack.Damage.Attempts.Primary, a.Attack.Weapon, in.Weapons, tr.actorPush(in, a))
```

```go
// turn_resolver.go:~386 — seedChain, o ataque a personagem
func (tr TurnResolver) seedChain(in ResolveInput, a action.Action) ChainState {
	raw, err := RawDamage(a.Attack.Damage.Attempts.Primary, a.Attack.Weapon, in.Weapons, tr.actorPush(in, a))
	if err != nil {
		return ChainState{}
	}
	return ChainState{Residual: raw}
}
```

**O dano é o mesmo dano**: não há razão para o punho medir diferente contra uma porta e contra
uma pessoa. Os dois passam o `Push`.

- [ ] **Step 6: Rode a suíte inteira do domínio**

```bash
go test ./internal/domain/match/...
go vet ./...
```

Alguns testes existentes de colisão podem ter números esperados que mudam agora, porque o
`Push` das fixtures passou a somar. **Se um falhar, leia antes de consertar:** se a fixture tem
`Push` zero, o número não devia mudar e a falha é bug seu; se tem `Push` não-zero, o número
esperado é que estava incompleto.

- [ ] **Step 7: Commit**

```bash
git add internal/domain/match/service/
git commit -m "$(cat <<'EOF'
feat(match): o Push do atacante entra no dano

RawDamage somava só os dados da arma e o dano plano dela. A regra é que
o dano é medido por Push, que já existia em enum.SkillName sem ninguém
ler.

Sem seletor: a arma é que dá o dano, e o jogador não escolhe qual perícia
o mede. Trocar Push por Grab é prerrogativa do mestre e entra na Fase 8.

Os dois call sites passam o mesmo Push — não há razão para o punho medir
diferente contra uma porta e contra uma pessoa. O helper nil-guarda
porque o ramo da parede não está atrás de actorSheetMissing.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: a finta deixa de ser escondida para sempre

**Files:**
- Modify: `internal/domain/match/service/projection.go:96-116`
- Modify: `internal/application/match/get_match_history.go:107` e `:110`
- Test: `internal/domain/match/service/projection_test.go`

**Interfaces:**
- Consumes: `Viewer{IsMaster bool; Owns map[uuid.UUID]bool}` e `v.SeesAllOf(id)`.
- Produces: `ProjectAction(a action.Action, v Viewer, isSettled bool) action.Action` — **a
  assinatura muda**, e o único chamador é o histórico.

- [ ] **Step 1: Escreva o teste que falha**

```go
// A finta é segredo enquanto o turno está aberto e pública quando ele fecha. Quem caiu nela
// descobre dentro da resolução do mesmo turno: o sucesso dele foi contra um ataque falso, e o
// de verdade vem logo em seguida. Esconder depois do fechamento esconde para sempre — que é o
// que o histórico fazia meses depois do fato.
func TestProjectActionRevealsFeintOnceTheTurnIsSettled(t *testing.T) {
	feint := action.RollCheck{SkillName: enum.Feint.String()}
	a := action.Action{Feint: &feint}
	stranger := Viewer{} // nem mestre, nem dono

	open := ProjectAction(a, stranger, false)
	if open.Feint != nil {
		t.Fatal("an open turn leaked the feint; the target would be warned before falling for it")
	}

	settled := ProjectAction(a, stranger, true)
	if settled.Feint == nil {
		t.Fatal("a settled turn still hid the feint; it would stay hidden forever")
	}
}
```

> Se `action.Action` não puder ser montado como literal por causa de campos não exportados, use
> o construtor `action.NewAction(...)` que `action_mapper.go` já usa e atribua `a.Feint` depois.

- [ ] **Step 2: Rode o teste e confirme que falha**

```bash
go test ./internal/domain/match/service/ -run TestProjectActionRevealsFeintOnceTheTurnIsSettled -v
```

Esperado: FAIL de compilação — `not enough arguments in call to ProjectAction`.

- [ ] **Step 3: Acrescente o eixo temporal**

Em `projection.go`:

```go
// ProjectAction downgrades an action to what this viewer is entitled to see.
//
// isSettled is the TIME axis, the same one ProjectResolution reads. A feint is secret while
// the turn is open and public once it closes: the target who fell for it finds out inside that
// same turn's resolution, because their success was against a false attack and the real one
// follows. Hiding it after the turn closed hides it forever, which is not the rule.
func ProjectAction(a action.Action, v Viewer, isSettled bool) action.Action {
	if v.SeesAllOf(a.GetActorID()) {
		return a
	}
	out := a
	if !isSettled {
		// A revealed feint is not a feint — while the swing is still in the air.
		out.Feint = nil
	}
	// Trigger stays hidden on both axes. §4.7 speaks about the feint; action.Trigger is an
	// empty object today and its rule has not been written. Do not widen the decision by
	// symmetry.
	out.Trigger = nil
	out.ReactionKind = action.ReactionKind(publicKind(string(a.ReactionKind)))
	if len(a.Skills) > 0 {
		// The Evasion entry is NOT the same rule as the feint: it is the other half of the
		// closed dodge's secret, tied to the label demotion above, and that demotion is
		// permanent by design.
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

- [ ] **Step 4: Atualize o chamador do histórico**

Em `get_match_history.go`, linhas 107 e 110. **Passe o fato, não uma constante:**

```go
settled := tu.FinishedAt != nil
pt.Action = service.ProjectAction(tu.Action, viewer, settled)
for l, react := range tu.Reactions {
	pt.Reactions[l] = service.ProjectAction(react, viewer, settled)
}
```

> É verdade que o histórico só guarda turno fechado — `PersistTurnClose` é o único caminho de
> escrita. Mas ler o fato custa o mesmo que assumi-lo e não quebra no dia em que um turno aberto
> atravessar. Confira o nome real do campo na struct do turno antes de escrever `tu.FinishedAt`.

- [ ] **Step 5: Rode os testes e confirme que passam**

```bash
go test ./internal/domain/match/service/ -run TestProjectActionRevealsFeintOnceTheTurnIsSettled -v
go test ./internal/domain/match/... ./internal/application/match/...
go vet ./...
```

- [ ] **Step 6: Acrescente o teste do histórico**

Em `internal/application/match/`, junto dos testes de histórico existentes: um turno fechado
com finta, lido por um participante que não é dono nem mestre, entrega `Feint` não-nulo. Use o
mock/fixture que os testes vizinhos daquele pacote já montam.

- [ ] **Step 7: Commit**

```bash
git add internal/domain/match/service/ internal/application/match/
git commit -m "$(cat <<'EOF'
fix(match): a finta volta a aparecer quando o turno fecha

ProjectAction zerava Feint para todo não-dono, em qualquer tempo — e o
único chamador é o histórico, então a finta ficava escondida do alvo
meses depois do fato.

A regra é temporal, não por classe: quem cai numa finta descobre dentro
da resolução do mesmo turno, porque o sucesso dele foi contra um ataque
falso e o de verdade vem em seguida. Turno aberto esconde, turno fechado
revela — o mesmo eixo IsSettled que ProjectResolution já usa.

Trigger continua escondido e a entrada Evasion continua saindo: a segunda
é a outra metade do segredo da esquiva fechada, amarrada ao rebaixamento
do rótulo, que é permanente por desenho.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: a categoria de movimento dos escapes, validada no servidor

**Files:**
- Modify: `internal/domain/match/entity/action/reaction_kind.go` (ao lado de `Displaces()`)
- Modify: `internal/app/game/action_mapper.go` (~linha 148, o `case action.ComponentMove`)
- Test: `internal/domain/match/entity/action/reaction_kind_test.go` e
  `internal/app/game/action_mapper_test.go`

**Interfaces:**
- Consumes: `ReactionKind.Displaces() bool`, `enum.Dash`, `enum.Shift`,
  `ActionPayload.Move.Category string`.
- Produces: `(k ReactionKind) RequiredMoveCategory() (enum.MoveCategory, bool)` — o `bool` é
  "este kind exige categoria", falso para quem não desloca.

**A matriz, fechada:**

| Reação | Movimento |
|---|---|
| `escape` | **Dash** |
| `escapeGuard` | **Dash** |
| `closedEscape` | **Shift** |

O discriminador é **fechado × aberto**, não defensivo × padrão.

- [ ] **Step 1: Escreva o teste de domínio que falha**

```go
// A matriz vive ao lado de Bars() e Displaces() porque é a mesma classe de regra: o que este
// kind exige de quem o manda. O WS só a aplica.
func TestReactionKindRequiredMoveCategory(t *testing.T) {
	cases := []struct {
		kind     ReactionKind
		want     enum.MoveCategory
		required bool
	}{
		{ReactEscape, enum.Dash, true},
		{ReactEscapeGuard, enum.Dash, true},
		{ReactClosedEscape, enum.Shift, true},
		{ReactDodge, "", false},
		{ReactClosedDodge, "", false},
		{ReactRepel, "", false},
		{ReactNothing, "", false},
	}
	for _, c := range cases {
		got, required := c.kind.RequiredMoveCategory()
		if required != c.required {
			t.Fatalf("%s: required=%v, want %v", c.kind, required, c.required)
		}
		if required && got != c.want {
			t.Fatalf("%s: category=%s, want %s", c.kind, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Rode e confirme que falha**

```bash
go test ./internal/domain/match/entity/action/ -run TestReactionKindRequiredMoveCategory -v
```

Esperado: FAIL — `k.RequiredMoveCategory undefined`.

- [ ] **Step 3: Escreva o método, ao lado de `Displaces()`**

```go
// RequiredMoveCategory is which displacement each escape has to use. The second return is
// whether this kind demands one at all — false for everything that does not displace.
//
//	escape       → Dash
//	escapeGuard  → Dash
//	closedEscape → Shift
//
// The discriminator is CLOSED vs OPEN, not defensive vs standard. During a Dash the character
// is "in the air" and cannot dodge — exactly what the closed variants exist not to do — so the
// closed escape steps with a Shift, which Brake governs because Brake is what measures your
// ability to STOP. Someone who accelerates beyond what they can brake is fast but not agile.
//
// It lives here, beside Bars() and Displaces(), because it is the same class of rule: what
// this kind demands of whoever sends it. The WS boundary enforces; it does not decide. Leaving
// the rule in the client would make the client its owner — and a client sending closedEscape
// with a Dash would buy the closed variant's bar discount with the open one's mobility.
func (k ReactionKind) RequiredMoveCategory() (enum.MoveCategory, bool) {
	switch k {
	case ReactEscape, ReactEscapeGuard:
		return enum.Dash, true
	case ReactClosedEscape:
		return enum.Shift, true
	default:
		return "", false
	}
}
```

Acrescente o import de `enum` no arquivo se ele ainda não tiver (o pacote `action` já importa
`enum` em `move.go`, então não há ciclo).

- [ ] **Step 4: Rode e confirme que passa**

```bash
go test ./internal/domain/match/entity/action/ -run TestReactionKindRequiredMoveCategory -v
```

- [ ] **Step 5: Escreva o teste do mapper que falha**

Em `action_mapper_test.go`:

```go
// Seis casos: os três kinds que deslocam, cada um com a categoria certa e com a errada.
func TestBuildActionEnforcesEscapeMoveCategory(t *testing.T) {
	cases := []struct {
		kind     string
		category string
		wantErr  bool
	}{
		{"escape", "Dash", false},
		{"escape", "Shift", true},
		{"escapeGuard", "Dash", false},
		{"escapeGuard", "Shift", true},
		{"closedEscape", "Shift", false},
		{"closedEscape", "Dash", true},
	}
	for _, c := range cases {
		p := ActionPayload{
			ActorID:      uuid.New(),
			ReactToID:    uuid.New(),
			ReactionKind: c.kind,
			Dodge:        &DodgePayload{RollCheck: RollCheckPayload{SkillName: "Reflex"}},
			Move:         &MovePayload{Category: c.category, Position: [3]int{1, 0, 0}},
		}
		if c.kind == "closedEscape" {
			p.Skills = []ActionSkillPayload{{SkillName: "Evasion"}}
		}
		_, err := buildAction(p.ActorID, p)
		if c.wantErr && err == nil {
			t.Fatalf("%s with %s was accepted; the client would own the rule", c.kind, c.category)
		}
		if !c.wantErr && err != nil {
			t.Fatalf("%s with %s was refused: %v", c.kind, c.category, err)
		}
	}
}
```

> Confira os nomes reais das structs de payload (`DodgePayload`, `MovePayload`,
> `ActionSkillPayload`) em `message.go` antes de escrever — use os que existem.

- [ ] **Step 6: Rode e confirme que falha**

```bash
go test ./internal/app/game/ -run TestBuildActionEnforcesEscapeMoveCategory -v
```

Esperado: FAIL nos três casos `wantErr` — hoje nada olha a categoria.

- [ ] **Step 7: Aplique a regra no mapper**

Em `action_mapper.go`, no `case action.ComponentMove` que hoje só checa `p.Move == nil`:

```go
case action.ComponentMove:
	if p.Move == nil {
		return nil, fmt.Errorf("reaction %q must carry a move", p.ReactionKind)
	}
	// Displaces() says THAT it moves; RequiredMoveCategory says WITH WHAT. Both live on the
	// kind — this is enforcement, not a second copy of the rule.
	if want, required := kind.RequiredMoveCategory(); required && p.Move.Category != string(want) {
		return nil, fmt.Errorf(
			"reaction %q must move with %s, not %s", p.ReactionKind, want, p.Move.Category)
	}
```

- [ ] **Step 8: Rode e confirme que passa**

```bash
go test ./internal/app/game/ -run TestBuildActionEnforcesEscapeMoveCategory -v
go test ./internal/app/game/ ./internal/domain/match/...
go vet ./...
```

- [ ] **Step 9: Commit**

```bash
git add internal/domain/match/entity/action/ internal/app/game/
git commit -m "$(cat <<'EOF'
feat(match): a categoria de movimento dos escapes é do servidor

Displaces() só exigia que existisse um Move, sem olhar a categoria. Com a
regra só no front, o cliente virava dono dela — e quem mandasse
closedEscape com Dash comprava o desconto de barra da fechada com a
mobilidade da aberta.

A matriz (escape e escapeGuard com Dash, closedEscape com Shift) vive em
RequiredMoveCategory, ao lado de Bars() e Displaces(), porque é a mesma
classe de regra. O mapper aplica; não decide.

O discriminador é fechado × aberto, não defensivo × padrão: durante o
Dash o personagem está no ar e não consegue esquivar, que é exatamente o
que a fechada existe para não fazer.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: o catálogo de combate, por personagem

**Files:**
- Create: `internal/app/api/sheet/get_combat_catalogue.go`
- Create: `internal/app/api/sheet/get_combat_catalogue_test.go`
- Modify: `internal/app/api/sheet/routes.go` (campo em `Api` + `huma.Register`)
- Modify: `cmd/api/main.go:150-155` (injeção)

**Interfaces:**
- Consumes: `cs.IGetCharacterSheet.GetCharacterSheet(ctx, sheetID, userUUID)`,
  `(*CharacterSheet).GetCommonProficiencies() map[enum.WeaponName]prof.IProficiency`,
  `prof.IProficiency.GetLevel() int`, `item.NewWeaponsManagerFactory().Build()`,
  `(*item.WeaponsManager).GetDice/GetDamage/GetDefense(name)`, `enum.AllSkillNames()`,
  `enum.Fist`.
- Produces: rota `GET /charactersheets/{uuid}/combat-catalogue`.

> **Reuse `IGetCharacterSheet`.** É o que dá a autorização de graça — quem pode ler a ficha pode
> ler o catálogo dela. **Não invente uma terceira política de visibilidade.**

- [ ] **Step 1: Escreva o teste que falha**

Em `get_combat_catalogue_test.go`, seguindo o padrão de `get_character_sheet_test.go`
(`humatest` + os mocks de `mocks_test.go`):

```go
// As armas de uma action não são o catálogo do sistema: são as que ESTE personagem sabe usar,
// mais o golpe corporal, sempre. Quando existir inventário isto vira "o que ele carrega";
// hoje proficiência é a melhor aproximação — é o que ele sabe empunhar.
func TestCombatCatalogueListsProficientWeaponsPlusFist(t *testing.T) {
	sheet := newSheetWithProficiencies(t, map[enum.WeaponName]int{
		enum.Sword:  4,
		enum.Dagger: 2,
	})
	_, api := humatest.New(t, huma.DefaultConfig("test", "1.0.0"))
	// registre só esta rota, como os testes vizinhos fazem

	resp := api.Get("/charactersheets/"+sheet.UUID.String()+"/combat-catalogue", authHeader(t))

	var body GetCombatCatalogueResponseBody
	mustUnmarshalJSON(t, resp.Body.Bytes(), &body)

	names := map[string]int{}
	for _, w := range body.Weapons {
		names[w.Name] = w.ProficiencyLevel
	}
	if len(body.Weapons) != 3 {
		t.Fatalf("got %d weapons, want 3 (Sword, Dagger, Fist): %v", len(body.Weapons), names)
	}
	if _, ok := names["Fist"]; !ok {
		t.Fatal("Fist is missing; the bare-handed blow does not depend on training")
	}
	if names["Sword"] != 4 {
		t.Fatalf("Sword came back at level %d, want 4", names["Sword"])
	}
	if len(body.Skills) == 0 {
		t.Fatal("skills came back empty; the front would go on inventing strings like combat_strength")
	}
}

// Uma ficha que JÁ tem proficiência em Fist devolve Fist uma vez só, com o nível real — não
// duplicado pelo acréscimo incondicional.
func TestCombatCatalogueDoesNotDuplicateFist(t *testing.T) {
	sheet := newSheetWithProficiencies(t, map[enum.WeaponName]int{enum.Fist: 3})
	// ... mesmo setup ...

	fists := 0
	level := -1
	for _, w := range body.Weapons {
		if w.Name == "Fist" {
			fists++
			level = w.ProficiencyLevel
		}
	}
	if fists != 1 {
		t.Fatalf("Fist appeared %d times, want exactly 1", fists)
	}
	if level != 3 {
		t.Fatalf("Fist came back at level %d, want the real 3", level)
	}
}
```

> `newSheetWithProficiencies`, `authHeader` e `mustUnmarshalJSON` são helpers de teste: se não
> existirem equivalentes em `mocks_test.go`, escreva-os **neste arquivo**, pequenos.

- [ ] **Step 2: Rode e confirme que falha**

```bash
go test ./internal/app/api/sheet/ -run TestCombatCatalogue -v
```

Esperado: FAIL de compilação — nada disso existe.

- [ ] **Step 3: Escreva o handler**

`internal/app/api/sheet/get_combat_catalogue.go`:

```go
package sheet

import (
	"context"
	"errors"
	"log"
	"net/http"

	apiAuth "github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/application/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	cs "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/item"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type GetCombatCatalogueRequest struct {
	UUID string `path:"uuid" required:"true" doc:"UUID of the character sheet"`
}

// WeaponOptionResponse is one weapon this character can attack with, with the numbers that
// decide what it does. The numbers travel because the bottom sheet has to show what the weapon
// does BEFORE the player picks it — and because the front duplicating this table is how the
// two sides start disagreeing.
type WeaponOptionResponse struct {
	Name             string `json:"name"`
	Dice             []int  `json:"dice"`
	FlatDamage       int    `json:"flatDamage"`
	DefenseBonus     int    `json:"defenseBonus"`
	ProficiencyLevel int    `json:"proficiencyLevel"`
}

type GetCombatCatalogueResponseBody struct {
	Weapons []WeaponOptionResponse `json:"weapons"`
	// Skills is the vocabulary the wire accepts, not a menu. Phase 6 puts NO skill selector in
	// the bottom sheet: the chain of tests is not executed yet, and a control the player moves
	// that changes nothing is worse than no control. This list exists so the front never
	// invents a string again — combat_strength was born of its absence.
	Skills []string `json:"skills"`
}

type GetCombatCatalogueResponse struct {
	Body   GetCombatCatalogueResponseBody `json:"body"`
	Status int                            `json:"status"`
}

// GetCombatCatalogueHandler serves what THIS character can attack with.
//
// It is deliberately not the system's weapon catalogue: it is the character's own
// proficiencies — what they know how to wield — plus the bare-handed blow, always. When an
// inventory exists this becomes "what they are carrying"; proficiency is today's best
// approximation of it.
//
// Authorization is IGetCharacterSheet's, reused on purpose: whoever may read the sheet may read
// its catalogue. Do not grow a third visibility policy here.
func GetCombatCatalogueHandler(
	uc cs.IGetCharacterSheet,
) func(context.Context, *GetCombatCatalogueRequest) (*GetCombatCatalogueResponse, error) {

	return func(ctx context.Context, req *GetCombatCatalogueRequest) (*GetCombatCatalogueResponse, error) {
		userUUID, ok := ctx.Value(apiAuth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		charSheetID, err := uuid.Parse(req.UUID)
		if err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}

		sheet, err := uc.GetCharacterSheet(ctx, charSheetID, userUUID)
		if err != nil {
			switch {
			case errors.Is(err, cs.ErrCharacterSheetNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, campaign.ErrCampaignNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, auth.ErrInsufficientPermissions):
				return nil, huma.Error403Forbidden(err.Error())
			default:
				log.Printf("[ERROR] GetCombatCatalogue uuid=%s: %v", req.UUID, err)
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}

		catalogue := item.NewWeaponsManagerFactory().Build()

		// Fist starts at level 0 and is overwritten by the real proficiency when the character
		// has one. Seeding it first is what makes "always present, never duplicated" one rule
		// instead of two.
		levels := map[enum.WeaponName]int{enum.Fist: 0}
		for name, prof := range sheet.GetCommonProficiencies() {
			levels[name] = prof.GetLevel()
		}

		weapons := make([]WeaponOptionResponse, 0, len(levels))
		for _, name := range enum.GetAllWeaponNames() {
			level, ok := levels[name]
			if !ok {
				continue
			}
			dice, err := catalogue.GetDice(name)
			if err != nil {
				// A proficiency in a weapon the catalogue does not carry is a data fault, not a
				// request fault: skip it rather than fail the whole list.
				log.Printf("[WARN] combat catalogue: weapon %s missing from catalogue", name)
				continue
			}
			damage, _ := catalogue.GetDamage(name)
			defense, _ := catalogue.GetDefense(name)
			weapons = append(weapons, WeaponOptionResponse{
				Name:             name.String(),
				Dice:             dice,
				FlatDamage:       damage,
				DefenseBonus:     defense,
				ProficiencyLevel: level,
			})
		}

		skills := make([]string, 0)
		for _, s := range enum.AllSkillNames() {
			skills = append(skills, s.String())
		}

		return &GetCombatCatalogueResponse{
			Body:   GetCombatCatalogueResponseBody{Weapons: weapons, Skills: skills},
			Status: http.StatusOK,
		}, nil
	}
}
```

> **Por que iterar `GetAllWeaponNames()` em vez do mapa:** um mapa Go itera em ordem aleatória,
> e uma lista que muda de ordem a cada request faz a bottom sheet dançar na cara do jogador. A
> ordem do enum é estável e é de graça.

- [ ] **Step 4: Registre a rota**

Em `routes.go`, acrescente o campo em `Api`:

```go
GetCombatCatalogueHandler    Handler[GetCombatCatalogueRequest, GetCombatCatalogueResponse]
```

e o registro, ao lado do `GET /charactersheets/{uuid}`:

```go
huma.Register(api, huma.Operation{
	Method:      http.MethodGet,
	Path:        "/charactersheets/{uuid}/combat-catalogue",
	Description: "List the weapons this character can attack with, and the skill vocabulary the wire accepts",
	Tags:        []string{"character_sheets"},
	Errors: []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusInternalServerError,
	},
}, a.GetCombatCatalogueHandler)
```

Em `cmd/api/main.go`, junto das outras linhas de `sheetHandler.Api{...}`:

```go
GetCombatCatalogueHandler: sheetHandler.GetCombatCatalogueHandler(getCharacterSheetUC),
```

- [ ] **Step 5: Rode e confirme que passa**

```bash
go test ./internal/app/api/sheet/ -run TestCombatCatalogue -v
go test ./internal/app/api/...
go vet ./...
```

- [ ] **Step 6: Smoke curl com o servidor de pé**

Endpoint novo se prova respondendo, não só em teste de handler.

```bash
make run-dev   # ou o alvo que o repo usa
# autentique com uma conta de teste e guarde o JWT em $TOKEN
curl -s -H "Authorization: Bearer $TOKEN" \
  http://localhost:5000/charactersheets/<uuid-de-uma-ficha-sua>/combat-catalogue | head -40
```

Confira a olho: `Fist` presente, as armas da ficha presentes, `dice` não vazio, `skills` com a
lista inteira. **Registre no PR o que você viu.**

- [ ] **Step 7: Commit**

```bash
git add internal/app/api/sheet/ cmd/api/main.go
git commit -m "$(cat <<'EOF'
feat(api): catálogo de combate por personagem

O contrato usava "Strength", "Deception" e "sword" nos exemplos e os três
seriam recusados: Strength é atributo, a finta é Feint, e WeaponNameFrom
é case-sensitive. Não havia endpoint que entregasse o vocabulário — e é
dessa ausência que nasceu o combat_strength que o front manda.

As armas não são o catálogo do sistema: são as proficiências DESTE
personagem, mais o golpe corporal, sempre. Quando existir inventário isto
vira "o que ele carrega". Os números vão junto porque a bottom sheet tem
que mostrar o que a arma faz antes de o jogador escolher.

A lista de perícias não é menu — a Fase 6 não tem seletor de perícia,
porque a corrente de testes não é executada. É lista de validação.

Autorização reusada de IGetCharacterSheet: quem pode ler a ficha pode ler
o catálogo dela.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: `match_full_state`

**Files:**
- Modify: `internal/app/game/message.go` (tipo + payload)
- Modify: `internal/app/game/room.go` (`buildMatchFullState` + o braço `register` do `Run()`)
- Test: `internal/app/game/combat_e2e_test.go`

**Interfaces:**
- Consumes: `newBarsUpdatedPayload(session) BarsUpdatedPayload`, `r.barsSeq`,
  `session.GetActiveScene()`, `session.GetActiveRound().GetMode()`, `session.CurrentTurnID()`,
  `session.ResolveTurn(t) *service.TurnResolution`,
  `newResolutionUpdatedPayload(turnID, res)`, `r.buildMapFullState(playerID, isMaster)` como
  molde de forma.
- Produces: `MsgTypeMatchFullState MessageType = "match_full_state"`,
  `MatchFullStatePayload`, `(r *Room) buildMatchFullState(playerID uuid.UUID, isMaster bool) *Message`.

- [ ] **Step 1: Escreva o teste que falha**

```go
// Quem entra no meio — ou reconecta, e o hook do front reconecta até cinco vezes sozinho —
// fica sem barras, sem regime, sem cena, sem turno aberto e sem reações pendentes, até alguma
// coisa mudar por acaso.
func TestMatchFullStateOnConnectCarriesTheCombatState(t *testing.T) {
	h := newCombatHarness(t)
	defer h.Close()

	h.enqueueAndOpenATurn(t) // qualquer caminho que deixe um turno aberto

	late := h.connectPlayer(t)       // um jogador que chega agora
	lateMaster := h.reconnectMaster(t)

	playerState := late.waitFor(t, "match_full_state")
	masterState := lateMaster.waitFor(t, "match_full_state")

	var p, m MatchFullStatePayload
	mustUnmarshal(t, playerState.Payload, &p)
	mustUnmarshal(t, masterState.Payload, &m)

	if p.OpenTurn == nil || p.OpenTurn.TurnID == uuid.Nil {
		t.Fatal("the late player did not learn that a turn is open")
	}
	if p.RoundMode == "" {
		t.Fatal("the late player did not learn the round regime")
	}
	// Eixo do TEMPO: o turno está aberto, logo o cálculo é do mestre.
	if p.Resolution != nil {
		t.Fatal("an open turn's resolution reached a player; that calculation is the master's")
	}
	if m.Resolution == nil {
		t.Fatal("the master reconnected into an open turn and lost the calculation")
	}
	// O seq atravessa a reconexão: se viesse um seq novo, a guarda do cliente zeraria e o
	// próximo bars_updated atrasado seria aplicado por cima de um estado mais novo.
	if p.Bars.Seq != h.lastBarsSeq(t) {
		t.Fatalf("match_full_state stamped seq %d, want the current %d", p.Bars.Seq, h.lastBarsSeq(t))
	}
}
```

- [ ] **Step 2: Rode e confirme que falha**

```bash
go test ./internal/app/game/ -run TestMatchFullStateOnConnectCarriesTheCombatState -v
```

- [ ] **Step 3: Escreva o tipo e o payload**

Em `message.go`, junto dos outros `MessageType` de servidor:

```go
// Server → Client (combat snapshot)
// Sent to every client that registers while a match session is live, so a late joiner — or a
// reconnect, and the front's hook reconnects up to five times on its own — does not sit
// without bars, regime, scene, open turn or pending reactions until something changes by luck.
MsgTypeMatchFullState MessageType = "match_full_state"
```

E o payload:

```go
// MatchFullStatePayload is everything about the COMBAT that map_full_state does not carry.
//
// Projected per recipient, by the same two axes as everything else: Resolution is the open
// turn's, therefore master-only by the TIME axis, and PendingReactions travel inside it.
type MatchFullStatePayload struct {
	SceneID               uuid.UUID `json:"sceneId"`
	SceneCategory         string    `json:"sceneCategory"`
	SceneBriefDescription string    `json:"sceneBriefDescription"`
	RoundMode             string    `json:"roundMode"`
	// Bars is the WHOLE bars_updated payload, reused rather than re-shaped: a second bar
	// format would be a second thing to keep in sync with the first.
	//
	// ⚠️ Its Seq is the CURRENT counter, NOT a new one. The client keeps the highest seq it
	// applied and discards anything lower; stamping a fresh number here would reset that guard
	// across a reconnect, and the first late bars_updated to arrive afterwards would be applied
	// on top of newer state.
	Bars BarsUpdatedPayload `json:"bars"`
	// OpenTurn is nil when the master is sitting on "closed and nothing opened", which is a
	// state they are allowed to be in.
	OpenTurn *OpenTurnPayload `json:"openTurn,omitempty"`
	// Resolution is the open turn's, MASTER-ONLY. nil for everyone else, and nil for the
	// master too when no turn is open.
	Resolution *ResolutionUpdatedPayload `json:"resolution,omitempty"`
}

type OpenTurnPayload struct {
	TurnID  uuid.UUID `json:"turnId"`
	ActorID uuid.UUID `json:"actorId"`
}
```

> Confira o nome real de `ResolutionUpdatedPayload` em `message.go` — use o que existe, e não
> monte uma segunda forma para a mesma resolução.

- [ ] **Step 4: Escreva `buildMatchFullState`**

Em `room.go`, ao lado de `buildMapFullState`. **A montagem sai numa função própria de
propósito** — assim a mensagem é testável sem subir conexão, e é a forma que o vizinho já tem.

```go
// buildMatchFullState snapshots the combat for one recipient. Returns nil when there is no
// session: in the lobby there is no combat to sync.
//
// The caller must NOT hold r.mu — this takes it.
func (r *Room) buildMatchFullState(playerID uuid.UUID, isMaster bool) *Message {
	r.mu.RLock()
	session := r.session
	seq := r.barsSeq
	r.mu.RUnlock()
	if session == nil {
		return nil
	}

	r.mu.RLock()
	payload := MatchFullStatePayload{Bars: newBarsUpdatedPayload(session)}
	payload.Bars.Seq = seq

	if scene := session.GetActiveScene(); scene != nil {
		payload.SceneID = scene.GetID()
		payload.SceneCategory = string(scene.GetCategory())
		payload.SceneBriefDescription = scene.GetBriefInitialDescription()
	}
	if round := session.GetActiveRound(); round != nil {
		payload.RoundMode = string(round.GetMode())
		if t := round.CurrentTurn(); t != nil {
			payload.OpenTurn = &OpenTurnPayload{
				TurnID:  t.GetID(),
				ActorID: t.GetAction().GetActorID(),
			}
			if isMaster {
				// ResolveTurn is a pure recompute, never a re-roll: the dice fell when the
				// action arrived. attach_reaction and edit_action already call it the same way.
				if res := session.ResolveTurn(t); res != nil {
					p := newResolutionUpdatedPayload(t.GetID(), res)
					payload.Resolution = &p
				}
			}
		}
	}
	r.mu.RUnlock()

	msg := NewServerMessage(MsgTypeMatchFullState, payload)
	return &msg
}
```

> Confira os getters reais de `scene.Scene` e `round.Round` antes de escrever
> (`GetCategory`, `GetBriefInitialDescription`, `CurrentTurn`, `GetAction`). Use os que existem.

- [ ] **Step 5: Envie no `register`**

Em `Run()`, no braço `case client := <-r.register`, logo depois do bloco do `map_full_state`:

```go
if msg := r.buildMatchFullState(client.userUUID, r.IsMaster(client.userUUID)); msg != nil {
	client.SendMessage(*msg)
}
```

`dispatchPerPlayer` **não** entra aqui: o register é de um cliente só. Aquele helper existe
para quando a mesa inteira precisa de uma cópia cada.

- [ ] **Step 6: Rode e confirme que passa**

```bash
go test ./internal/app/game/ -run TestMatchFullStateOnConnectCarriesTheCombatState -v
go test ./internal/app/game/
go vet ./...
```

- [ ] **Step 7: Commit**

```bash
git add internal/app/game/
git commit -m "$(cat <<'EOF'
feat(game): match_full_state dá o combate a quem entra ou reconecta

map_full_state cobre o mapa e só. Quem chegava no meio — ou reconectava,
e o hook do front reconecta até cinco vezes sozinho — ficava sem barras,
sem regime, sem cena, sem turno aberto e sem reações pendentes, até
alguma coisa mudar por acaso.

Projetado pelos dois eixos de sempre: a resolução do turno aberto é
master-only pelo eixo do tempo, e as pendingReactions viajam dentro dela.

O seq das barras é o CORRENTE, não um novo: a guarda do cliente descarta
snapshot menor que o maior já aplicado, e estampar número novo aqui a
zeraria na reconexão.

A montagem sai numa função própria porque isso vale por si — é a forma
que buildMapFullState já tem, e torna a mensagem testável sem conexão.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: o movimento aplicado ao tabuleiro

**Files:**
- Modify: `internal/app/game/room.go` (extrair de `handlePieceMoved:1657`, chamar do
  `open_next_action`/`pull_action`)
- Test: `internal/app/game/combat_e2e_test.go`

**Interfaces:**
- Consumes: `PieceMovedPayload`, `PieceRemovedPayload`, `r.pieces`, `r.gridShape()`,
  `slotPayloadToWorld`, `r.visibilityFor(pid)`, `domainservice.IsVisible`,
  `r.dispatchPerPlayer`, `session.GetCharToPlayer()`, `session.RecomputeVisibility(playerID)`,
  `r.buildMapFullState`, `TurnTransition.Opened *turn.Turn`.
- Produces: `(r *Room) applyAndRelayPieceMove(payload PieceMovedPayload, origin uuid.UUID)`.

> **Descoberto na implementação (Task 8):** `TurnTransition.Opened` não existe — o campo
> real, em ambos os call sites (`open_next_action` e `pull_action`), é `result.OpenedTurn`
> (`*turnentity.Turn`). Os passos abaixo ainda dizem `transition.Opened`; leia `result.OpenedTurn`.

**A regra:**

| O movimento… | A peça |
|---|---|
| **não depende de teste** (shift/dash para slot livre) | desloca **na abertura** da action |
| **depende de uma CD** (salto, slot ocupado, passar colado) | **não desloca** |

⭐ **O front nunca calcula onde a peça para.** Ele desenha o pedido e depois desenha a posição
que chegou do servidor.

> **Só o ramo sem teste tem caso alcançável hoje.** `moveSpeedSkill` aceita **só `Dash` e
> `Shift`** e recusa `Back`, `Roll`, `Slide`, `Jump`, `FlatJump`. Nenhum dos dois aceitos rola
> contra CD. **Não invente salto para preencher a tabela.**

- [ ] **Step 1: Extraia o helper, sem mudar comportamento**

Refatoração pura, antes de qualquer teste novo. Tire o miolo de `handlePieceMoved` para:

```go
// applyAndRelayPieceMove puts a piece on the board and tells everyone entitled to know.
//
// The fog gate is the point, and it is a PAIR: whoever can see the destination gets
// piece_moved, whoever could only see the origin gets piece_removed (the piece walked out of
// sight), whoever sees neither gets nothing. That is why the server-applied move reuses this
// instead of growing a second path — and why it stays on piece_moved rather than a new type,
// which would have to duplicate the pair to keep the "walked out of sight" case.
//
// origin is the player whose own browser already applied this move locally and must therefore
// not be echoed back to. It is uuid.Nil when the SERVER is the mover — nobody predicted that
// one, so nobody is skipped.
//
// The owner is not a parameter: it comes from payload.CharacterID through GetCharToPlayer().
// Whoever owns the moved character gets a fresh map_full_state, because their line of sight
// just changed.
//
// The caller must NOT hold r.mu — this takes it.
func (r *Room) applyAndRelayPieceMove(payload PieceMovedPayload, origin uuid.UUID) {
	// ... o corpo atual de handlePieceMoved, com duas trocas:
	//   `client.userUUID` → `origin`   (no teste de quem pular)
	//   o bloco final de RecomputeVisibility passa a resolver o dono por
	//   GetCharToPlayer()[payload.CharacterID] e mandar o map_full_state PARA ELE,
	//   em vez de para `client`.
	// Quando origin == uuid.Nil, ninguém é pulado.
}

func (r *Room) handlePieceMoved(client *Client, payload PieceMovedPayload) {
	r.applyAndRelayPieceMove(payload, client.userUUID)
}
```

> A mensagem relaiada hoje é `NewClientMessage(..., client.userUUID, payload)`. Quando o
> servidor é o autor, use `NewServerMessage(MsgTypePieceMoved, payload)` — o envelope já marca
> mensagem de servidor com `senderId` zero, que é como o front distingue se algum dia precisar.

- [ ] **Step 2: Prove que a extração não mudou nada**

```bash
go test ./internal/app/game/
```

A suíte de fog (`fog_e2e_test.go`, `fog_dispatch_test.go`, `fog_regression_test.go`) é a rede de
segurança desta extração. **Ela tem que passar sem você alterar um único teste.** Se precisar
mexer num teste de fog, a extração mudou comportamento — desfaça e refaça.

- [ ] **Step 3: Commit da extração, sozinha**

```bash
git add internal/app/game/room.go
git commit -m "$(cat <<'EOF'
refactor(game): extrai o relay de peça com gate de fog para um helper

handlePieceMoved já fazia exatamente o que o movimento do motor precisa:
atualiza o tabuleiro e relaia por jogador com o par piece_moved /
piece_removed. O miolo sai para applyAndRelayPieceMove, com o remetente
virando parâmetro, para o caminho do servidor reusá-lo em vez de crescer
um segundo.

Sem mudança de comportamento: a suíte de fog passa intacta.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
EOF
)"
```

- [ ] **Step 4: Escreva o teste que falha**

```go
// Hoje uma ação de mover acontece e a peça não sai do lugar: nada aplica o movimento
// resolvido ao tabuleiro, e o fog não recalcula porque nada se moveu.
//
// Na ABERTURA, não no fechamento: o dano espera o fechamento porque pode ser editado; a
// posição não pode, porque as reactions seguintes dependem de onde a peça está.
func TestOpeningAMoveActionMovesThePieceBeforeTurnOpened(t *testing.T) {
	h := newCombatHarness(t)
	defer h.Close()

	h.player.send("enqueue_action", map[string]any{
		"actorId": h.playerCharID.String(),
		"move": map[string]any{
			"category": "Dash",
			"from":     []int{4, 4, 0},
			"position": []int{6, 4, 0},
		},
	})
	h.player.waitFor(t, "action_enqueued")
	h.master.send("open_next_action", map[string]any{})

	moved := h.master.waitFor(t, "piece_moved")
	var mp PieceMovedPayload
	mustUnmarshal(t, moved.Payload, &mp)
	if mp.Slot.Col != 6 || mp.Slot.Row != 4 {
		t.Fatalf("piece landed on (%d,%d), want (6,4)", mp.Slot.Col, mp.Slot.Row)
	}

	// A mesa não pode ver o turno abrir com a peça no lugar velho.
	opened := h.master.waitFor(t, "turn_opened")
	if opened.ReceivedAt.Before(moved.ReceivedAt) {
		t.Fatal("turn_opened arrived before piece_moved; the table saw the turn open with the piece still in the old slot")
	}
}

// O que de fato prova a projeção: quem não enxerga nem a origem nem o destino não é avisado.
func TestOpeningAMoveActionDoesNotLeakToWhoCannotSeeIt(t *testing.T) {
	h := newCombatHarness(t)
	defer h.Close()
	blind := h.connectPlayerWithNoLineOfSight(t, [3]int{4, 4, 0}, [3]int{6, 4, 0})

	h.enqueueDashAndOpen(t, [3]int{4, 4, 0}, [3]int{6, 4, 0})

	if msg, ok := blind.tryWaitFor("piece_moved", 200*time.Millisecond); ok {
		t.Fatalf("a player who sees neither end of the move was told about it: %v", msg)
	}
}
```

> Se o harness não tiver `ReceivedAt` nem `tryWaitFor`, acrescente-os — são pequenos e a
> asserção de ordem é o ponto do primeiro teste. `connectPlayerWithNoLineOfSight` provavelmente
> significa posicionar a peça dele atrás de uma parede; veja como `fog_e2e_test.go` monta isso.

- [ ] **Step 5: Rode e confirme que falha**

```bash
go test ./internal/app/game/ -run TestOpeningAMoveAction -v
```

Esperado: FAIL — nenhum `piece_moved` chega.

- [ ] **Step 6: Aplique o movimento na abertura**

Nos braços de `MsgTypeOpenNextAction` e `MsgTypePullAction` em `room.go`, **depois** de o
`TurnTransition` voltar e **antes** de emitir `turn_opened`:

```go
// The position cannot wait for the close the way damage does: the reactions that follow
// depend on where the piece IS. Only movement that does not test displaces here — and today
// the mapper accepts only Dash and Shift, neither of which rolls against a DC, so the other
// branch has no reachable case. Do not invent one.
//
// ⚠️ Descoberto na implementação (Task 8): o campo real é result.OpenedTurn, não
// transition.Opened — confira o nome da variável de retorno no braço que você está editando.
if result.OpenedTurn != nil {
	r.applyOpenedMove(result.OpenedTurn)
}
```

E o helper, ao lado de `applyAndRelayPieceMove`:

```go
// applyOpenedMove walks the opened action's Move onto the board.
//
// A character with no piece is NOT an error: there is simply nothing to move. Do not send an
// error for it.
//
// The caller must NOT hold r.mu — applyAndRelayPieceMove takes it.
func (r *Room) applyOpenedMove(opened *turnentity.Turn) {
	a := opened.GetAction()
	if a.Move == nil {
		return
	}
	actorID := a.GetActorID().String()

	r.mu.RLock()
	var piece PieceMovedPayload
	found := false
	for _, p := range r.pieces {
		if p.CharacterID == actorID {
			piece = p
			found = true
			break
		}
	}
	r.mu.RUnlock()
	if !found {
		return
	}

	col, row := a.Move.Position[0], a.Move.Position[1]
	piece.Slot = SlotPayload{Kind: "square", Col: &col, Row: &row}
	// Z is deliberately NOT touched — see the note below.

	// origin is uuid.Nil: the server moved this one and nobody's browser predicted it, so
	// nobody is skipped.
	r.applyAndRelayPieceMove(piece, uuid.Nil)
}
```

> Confira a forma real de `SlotPayload` (hex vs square) e de `PieceMovedPayload.Z` em
> `message.go`. Se o tabuleiro puder ser hexagonal, respeite `piece.Slot.Kind` em vez de
> assumir `"square"` — **não force o quadrado num mapa hex**.
>
> ⚠️ **Descoberto na implementação (Task 8), dois erros de tipo neste trecho:**
> `SlotPayload.Col`/`Row`/`Q`/`R` são `*int`, não `int` — o literal acima não compila como
> escrito, precisa dos ponteiros (`&col`, `&row`). E `PieceMovedPayload.Z` é `float64`, não
> `*float64` — o código real **não** escreve `z := a.Move.Position[2]; piece.Z = &z`: a
> implementação final preserva o `Z` que a peça já tinha, sem tocar nele, porque
> `Move.Position[2]` é o índice `z` da GRADE e `PieceMovedPayload.Z` é altura virtual em
> METROS — grandezas possivelmente diferentes, nunca reconciliadas. Ver a lacuna correspondente
> em `docs/dev/api/match-combat-ws.md` §9.

⚠️ **Confira quem segura `r.mu` no braço do `open_next_action` antes de chamar.** O `Execute`
roda com write lock; o helper também pega o lock. Chamar com o lock na mão é deadlock.

- [ ] **Step 7: Rode e confirme que passa**

```bash
go test ./internal/app/game/ -run TestOpeningAMoveAction -v
go test ./internal/app/game/
go vet ./...
```

- [ ] **Step 8: Commit**

```bash
git add internal/app/game/
git commit -m "$(cat <<'EOF'
feat(game): o movimento resolvido move a peça

Nenhuma mensagem servidor→cliente aplicava um movimento ao tabuleiro. O
piece_moved que existia é cliente→servidor, do lobby, e o motor só usava
move.from para checar parede — então uma ação de mover acontecia e a peça
não saía do lugar, com o fog nunca recalculando.

Na abertura, não no fechamento: o dano espera porque pode ser editado; a
posição não pode, porque as reactions seguintes dependem de onde a peça
está.

Continua saindo como piece_moved, com senderId zero: o par com
piece_removed é indivisível — o gate de fog precisa dos dois para não
perder o caso "saiu de vista" — e para o front isto é uma coisa só,
"chegou uma posição, desenhe". Ele nunca calcula onde a peça para.

Só o ramo sem teste: o mapper aceita só Dash e Shift, e nenhum rola
contra CD.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: a documentação, que é entregável

**Files:**
- Modify: `docs/dev/api/match-combat-ws.md`
- Modify: `docs/dev/api/character-sheet.md`
- Modify: `docs/dev/match/flows/05-lacunas.md`
- Modify: `docs/documentation-map.yaml`

**Interfaces:**
- Consumes: tudo que as tarefas 1–7 produziram, com os nomes finais.

> O contrato tem um aviso no topo dizendo que nenhum cliente real o leu ainda e que uma
> divergência encontrada é **bug do contrato**. Este PR é a primeira vez que alguém o corrige
> por esse motivo. **Mantenha o aviso** — ele continua valendo para a Fase 6.

- [ ] **Step 1: `match-combat-ws.md`**

- `match_full_state` — seção nova, com os dois eixos e a nota do `seq`;
- `piece_moved` como mensagem **de servidor** também, com o gate de fog e o par com
  `piece_removed`;
- `action_enqueued` deixa de ser `{}`;
- a finta: visibilidade temporal;
- `attack.damage.skillName` **descartado**, na mesma tabela em que `speed` já está;
- `nickname` documentado como **opcional** — `handler.go:134` já cai para os 8 primeiros
  caracteres do UUID quando vem vazio. Não é código, é o contrato que estava errado;
- os escapes: a matriz de categoria e o erro que o servidor devolve;
- **os exemplos consertados**: `"Strength"` → `"Push"`, `"Deception"` → `"Feint"`,
  `"sword"` → `"Sword"`.

- [ ] **Step 2: a seção de lacunas do contrato — conferir, não presumir**

As três que este PR fecha (nada move a peça, sem snapshot, `action_enqueued` vazio)
**nunca estiveram** naquela lista: saíram da auditoria do front. As nove que estão lá
(`targetId` ausente no `turn_opened`, `actionType` vazio, sem evento de HP, a corrente de
testes, `ReboundDamage`, armadura, `move`/`attack` do master action, nenhuma projeção de
declaração, NPC não age) **continuam todas válidas**. Não risque nenhuma.

- [ ] **Step 3: `character-sheet.md` e `05-lacunas.md`**

O `combat-catalogue` junto das outras rotas de ficha — é rota de ficha, não abra doc própria.
Em `05-lacunas.md`, o que deixou de estar oco.

- [ ] **Step 4: `documentation-map.yaml`**

As entradas novas, no formato que o arquivo já usa.

- [ ] **Step 5: Commit**

```bash
git add docs/
git commit -m "$(cat <<'EOF'
docs(api): o contrato WS alcança o que o backend passou a fazer

match_full_state, o piece_moved de servidor, o action_enqueued que nomeia
a ação, a finta temporal, o damage.skillName descartado, o nickname
opcional e a matriz de categoria dos escapes.

E os três exemplos que estavam errados desde o começo: Strength não é
perícia, a finta é Feint, e WeaponNameFrom é case-sensitive. O aviso do
topo dizia que uma divergência encontrada é bug do contrato — esta é a
primeira vez que alguém o corrige por esse motivo, e o aviso fica.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
EOF
)"
```

---

## Fechamento

- [ ] `go test ./...` inteiro, verde.
- [ ] `go vet ./...` limpo, **mais** `go vet -tags smoke ./...` e `go vet -tags integration ./...`.
- [ ] `check_documentation_impact` (ou diff manual contra `documentation-map.yaml`), como o
      `docs-workflow` exige de todo PR que mexe em `internal/` ou `cmd/`.
- [ ] Servidor de pé e smoke curl do `combat-catalogue` refeito, com a saída anotada.
- [ ] PR aberto dizendo **o que foi verificado e o que não foi**. Diga explicitamente: o
      `combat_strength` do front **não** foi consertado aqui, e por quê (§11 do spec).
- [ ] Se sobrou qualquer coisa não verificada, `./dev-checkout.sh docs/front-combat-phases` a
      partir de `System_X_System_Project/`, e diga no PR o que olhar.

## Self-review deste plano

**Cobertura do spec:** §3→Task 1, §4→Task 2, §5→Task 3, §6→Task 4, §7→Task 5, §8→Task 6,
§9→Task 7, §12→Task 8. §10 (o que não entra) e §11 (por que o React não é tocado) são
restrições, cobertas nas Global Constraints e no fechamento. §13 (verificação) está distribuído
nos passos e no fechamento. §14 (riscos) vira o aviso de lock na Task 7 Step 6, a nota do `seq`
na Task 6 Step 3, e o gate da suíte de fog na Task 7 Step 2.

**Consistência de tipos:** `ActionEnqueuedPayload.ActionID`, `RawDamage(..., push int)`,
`actorPush(in, a)`, `ProjectAction(a, v, isSettled)`, `RequiredMoveCategory() (enum.MoveCategory, bool)`,
`GetCombatCatalogueResponseBody{Weapons, Skills}`, `MatchFullStatePayload{Bars, OpenTurn, Resolution}`,
`applyAndRelayPieceMove(payload, origin)`, `applyOpenedMove(opened)` — cada nome é usado com a
mesma forma em todos os pontos onde aparece.

**Onde o plano manda conferir em vez de afirmar:** nomes de helpers de teste do harness,
getters de `scene.Scene`/`round.Round`, forma de `SlotPayload` (square vs hex), nome real do
campo de fechamento do turno no histórico, nome de `ResolutionUpdatedPayload`. São pontos em
que o plano não leu o código e **diz isso** em vez de inventar um nome plausível.
