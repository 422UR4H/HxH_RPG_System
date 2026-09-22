# Plano — as quatro dívidas do back com a Fase 6 (B1–B4)

**Branch:** `feat/live-npc-ws` (continuação; o PR #76 já está aberto).
**Origem:** `docs/superpowers/specs/2026-09-20-front-combat-phases.md` §4.10 (PR #77, branch
`docs/front-combat-phase-6-gaps`). Entram aqui, e não num PR próprio, porque vivem nos mesmos
`room.go`/`message.go` que esta branch já abriu — e porque a Fase 6 não deve esperar mais um PR.

Fora de escopo (§4.10 diz explicitamente): verbo de cancelar ação.
Já fechado por esta branch: o §2 do contrato ("NPC hoje não age").

## Global Constraints

- Go 1.23; `testing` padrão, table-driven com `t.Run`; TDD (teste antes).
- `room.go` é dono do lock (`r.mu`); nada que envia para cliente roda em seção crítica.
- Wire camelCase; tipos de mensagem snake_case.
- **Nunca remover comentários TODO.** Comentários explicam o porquê, no tom dos vizinhos.
- Commits terminam com `Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>`.
- Verificação por task: `go build ./...`, `go vet ./...`, `go test ./internal/app/game/`, e
  `go test -race ./internal/app/game/` em qualquer task que toque `room.go`.
- Todo item que muda o wire atualiza `docs/dev/api/match-combat-ws.md` **no mesmo commit**.

---

## Task 5 — B1: `turn_opened` carrega `actionId`

**Arquivos:** `internal/app/game/message.go`, `room.go` (`announceOpenedTurn`),
`docs/dev/api/match-combat-ws.md`, teste no pacote `game`.

- `TurnOpenedPayload` ganha `ActionID uuid.UUID \`json:"actionId"\`` ao lado de `TurnID`/`ActorID`.
- `announceOpenedTurn` preenche a partir da ação do turno aberto (`opened.GetAction()` devolve
  uma CÓPIA — atribua a uma variável antes de chamar getter com receiver de ponteiro, como o
  código vizinho já faz).
- É o **mesmo** `actionId` que o `action_enqueued` devolveu a quem enfileirou e que o
  `action_queued` deu ao mestre. Diga isso no contrato: é o que liga as três mensagens.
- Vale para os dois caminhos que abrem turno (`open_next_action` e `pull_action`), porque os
  dois terminam em `announceOpenedTurn` — um teste para cada.
- Contrato: §5 `turn_opened` (campo novo + para que serve: com duas ações do mesmo personagem
  na fila, é o que diz qual abriu).

**Testes:** e2e no padrão do pacote — jogador enfileira duas ações do mesmo ator, o mestre abre
uma, e o `turn_opened` traz o `actionId` daquela, não o da outra.

---

## Task 6 — B3: `turn_closed` nos dois caminhos

**Arquivos:** `internal/app/game/room.go`, contrato, testes.

Hoje só o braço `close_turn` emite `turn_closed` (room.go ~941). O fechamento implícito de
`open_next_action` e de `pull_action` (quando `result.ClosedTurn != nil`) fecha calado.

- Emita `turn_closed` nesses dois braços, com o mesmo payload
  (`TurnClosedPayload{TurnID: closedTurn.GetID()}`), **no mesmo lugar** onde o braço já trata o
  fechamento: junto de `applyClosedEscapes`/`persistClosedTurn`/`publishResolution`, e **antes**
  do `announceOpenedTurn` do turno seguinte — a mesa tem que ver o turno acabar antes de ver o
  próximo começar.
- Não duplique a montagem da mensagem em três lugares: extraia um helper pequeno
  (`broadcastTurnClosed(turnID)`) e use nos três.
- Ordem contra `resolution_updated`: mantenha a que o braço já pratica hoje para o
  `close_turn`; se as duas diferirem, alinhe pelo `close_turn` e diga no contrato qual é.
- Contrato: §5 `turn_closed` — "disparado por" passa a listar os três verbos, com a nota de que
  o fechamento implícito emite igual ao explícito.

**Testes:** e2e — abrir a próxima ação com um turno aberto emite `turn_closed` do turno velho
para a mesa inteira, e ele chega antes do `turn_opened` do novo. Um teste por caminho
(`open_next_action` e `pull_action`).

---

## Task 7 — B2: HP ao vivo por WS, projetado

**Arquivos:** `internal/app/game/message.go`, `room.go`, contrato, testes.

Mensagem nova, servidor → cliente:

```go
MsgTypeCharacterHpChanged MessageType = "character_hp_changed"

type CharacterHpChangedPayload struct {
	CharacterID uuid.UUID `json:"characterId"`
	HP          int       `json:"hp"`
	MaxHP       int       `json:"maxHp"`
	Damage      int       `json:"damage"`
}
```

- **Mensagem própria, não campo do `resolution_updated`**: HP vai mudar por cura e veneno também,
  e esses caminhos não têm resolução de turno. Diga isso no comentário do tipo.
- **Destino: mestre + dono da ficha**, via `dispatchPerPlayer` (é a projeção por destinatário
  que o fog e o `resolution_updated` já usam — não crie uma segunda). O dono sai de
  `charToPlayer[characterID]`; NPC → mestre, então o mestre recebe uma cópia só.
- **Emitido de onde o dano é aplicado**, não do `turn_closed`: os três `*Result` que carregam
  `Damaged []matchsession.DamagedCharacter` (`close_turn.go`, `open_next_action.go`,
  `pull_action.go`). Um helper único em room.go (`broadcastHpChanges(damaged)`) chamado nos três
  braços, logo depois de `broadcastBars`/`persistClosedTurn` do mesmo braço.
- `MaxHP` sai da mesma barra de que `NewHP` saiu (`DamagedCharacter.Sheet`,
  `GetAllStatusBar()[enum.Health]` — confira o getter de máximo real antes de escrever; se a
  barra não expõe máximo, **pare e relate**, não invente um número).
- Ler a ficha para o máximo é leitura de estado da sessão: sob `r.mu`, e o envio fora dele.
- Contrato: seção nova em §5, mais uma linha na tabela do índice §3, mais a remoção da lacuna
  "**Não existe evento de HP de personagem**" do §9 (e o mesmo item no `AGENTS.md` § Known
  Issues, que diz que o caminho ao vivo é trabalho de front da Fase 6 — agora não é mais).

**Testes:** e2e com três clientes (mestre, dono do alvo, terceiro jogador): fechar um turno com
dano faz o mestre e o dono receberem `character_hp_changed` com o HP novo, e o terceiro **não**
receber nada. Use a barreira de ordenação que os testes do pacote já usam para a asserção
negativa. Mais um teste de que o caminho implícito (`open_next_action`) também emite.

---

## Task 8 — B4: `attack.hit` derivado pelo servidor + dois consertos de texto

**Arquivos:** `internal/app/game/action_mapper.go`, contrato, testes.

- Em `buildAction`, o `SkillName` de `Attack.Hit` passa a ser **sempre `enum.Accuracy`**,
  ignorando o que o payload trouxe — exatamente como `actionSpeed` já faz com `Legerity`
  (comentário do mapper, logo acima). O `Context`/`Condition` que o payload manda **continua
  respeitado**: o que é derivado é a perícia, não o resto do `RollCheck`.
- Por que `Accuracy` e não a proficiência da arma: **não existe mapeamento arma → perícia** no
  código — o catálogo de combate devolve `proficiencyLevel` por arma, mas nada usa isso num
  rolamento. Escreva no comentário que é aqui que a regra muda no dia em que a proficiência
  entrar no acerto, e que hoje o valor é o padrão único.
- O payload `attack.hit` **continua aceito** no wire (não quebra cliente nenhum), só deixa de
  decidir. Se preferir deixá-lo opcional, **não** faça nesta task: mudar a obrigatoriedade do
  campo é mudança de contrato de entrada, e o front da Fase 6 ainda não foi escrito contra ela.
- Contrato: na seção `enqueue_action`, marcar `attack.hit.skillName` como **derivado pelo
  servidor** (com o valor: `Accuracy`), no mesmo tom com que `speed` e `move.speed` já estão
  marcados. Confira como aqueles dois estão escritos e siga.

**Segundo conserto de texto (sem código)**, em `match-combat-ws.md` (~linha 1281): o contrato
lista "um pouso em slot ocupado" entre os casos **sem caso alcançável**. Está errado: um `Dash`
para um slot ocupado é aceito hoje — a peça desloca e **empilha**. O que não tem caso alcançável
é o movimento de ação que exigiria **teste** (salto, aperto), porque `move.category` só aceita
`Dash` e `Shift`. Separe as duas coisas e trate o empilhamento como a mesma classe da colisão
com parede: regra de jogo que ainda não foi escrita, não validação esquecida.

**Testes:** unitário em `action_mapper_test.go` — um payload de ataque com
`hit.skillName: "Push"` (ou qualquer outra) produz `Attack.Hit.SkillName == "Accuracy"`, e o
`Context` do payload sobrevive.
