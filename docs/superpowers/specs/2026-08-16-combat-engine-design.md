# Motor de Batalha — spec de implementação

**Data:** 2026-08-16 · **Status:** aprovado para planejamento · **Repos:** `System_X_System` (Go) e `System_X_System_React`

> Este documento é **auto-contido**. Quem for implementar não precisa da conversa que o
> originou. As regras de jogo completas estão em:
> - `docs/dev/match/combat-engine.md` — modelo técnico
> - `docs/game/combate/reacoes.md` · `barra-de-acao.md` · `acoes.md` · `docs/game/dados.md` — regras
> - `docs/dev/match/flows/` — diagramas do bounded context como ele está hoje
> - `docs/superpowers/specs/2026-08-14-action-flow-design-notes.md` — registro bruto do design

---

## 1. Objetivo

Fazer a partida ganhar vida: um combate jogável em que os jogadores declaram intenção em
paralelo, o mestre rege a mesa abrindo ações, e o sistema resolve os números.

**A tese do produto:** em RPG de mesa, a maior parte do tempo é latência — esperar a vez,
esperar alguém decidir, esperar alguém somar dados. Aqui não existe fase de espera depois do
round 0. Os jogadores focam em imaginação, criatividade e interpretação; o sistema calcula em
paralelo sob gerência do mestre.

## 2. Onde estamos

O esqueleto **Cena → Round → Turno → Action** existe, é testado e está correto. O miolo que
calcula está vazio.

| Existe | Vazio |
|---|---|
| `entity/scene`, `entity/round`, `entity/turn`, `entity/action` | `service.RollCalculator` — retorna `0`, ninguém chama |
| `action.PriorityQueue` (max-heap por `Speed.Result`) | `service.TurnResolver` — ramo `TargetKindCharacter` vazio |
| `service.RoundOrchestrator` — ciclo Round/Turn | `match.CharacterStatus` — só comentário, nada instancia |
| `matchsession.MatchSession` — estado vivo + fachada | `battle.Blow` — campos privados, sem construtor |
| 7 use cases de sessão + persistência de turno | `action.Initiative` — órfão; `ChangeMode` ignora o parâmetro |
| Paredes: dano estrutural, interação, LOS/fog | `game.buildAction` — descarta Skills, Speed, Feint, Attack, Defense |

**Consequência prática:** toda action entra na fila com `Speed.Result == 0`, então a fila de
prioridade não tem prioridade. E nada no sistema altera uma ficha.

## 3. Decisões de design (fechadas)

Estas não se rediscutem durante a implementação. Se alguma se mostrar errada, para-se e
volta-se ao dono do produto.

### Princípios

1. **Servidor autoritativo em tudo.** Nenhuma fórmula de jogo em TypeScript. O client do
   mestre envia inputs (CD, vantagem, edições) e renderiza.
2. **"Abrir" é passar o microfone.** É regência da mesa — *"agora é a vez desta pessoa
   narrar"*. O cálculo é efeito colateral. Vale igual para action e reaction.
3. **Sorteio uma vez; resultado derivado quantas vezes precisar.** Os dados caem quando a
   action/reaction chega. O número final é refeito a cada edição do mestre e a cada reaction
   que colide. **O mestre nunca re-sorteia o dado do jogador.**
4. **O resultado de um teste é a margem, não um booleano.**
5. **A action não carrega status.** Onde ela está diz em que ponto do ciclo está.
6. **Todo momento em que o mestre edita exige recálculo** — sem exceção, sem novo sorteio.

### Regras de jogo essenciais

- **Teste = 2 D10 somados.** Crítico = ambos 10; erro crítico = ambos 1 — **é a combinação,
  não a soma**, logo os dados individuais precisam sobreviver ao cálculo.
- **Passivo usa o valor médio derivado do conjunto de dados** (11 para 2 D10, 10 para D20).
  Usado em: actionSpeed com `RoundMode == Free`, esquiva por reflexo (`Reflexo + 11`), defesa
  à mão livre, deslocamento Shift.
- **Duas barras por personagem** — `actionSpeed` e `moveSpeed` — com preços independentes,
  mas **um relógio só**. As duas estão na mesma escala (`perícia + 2 D10`).
- **Fórmula de fechamento do round, por barra:**
  `barra = média(velocidades daquela barra no round) − nº de ações × preço`, limitada a `+preço`.
- **Preço do round** = a menor velocidade daquela barra na rodada. Igual para todos.
- **Depois do round 0 não existe fase de coleta.** A fila é permanentemente viva.
- **Não há confirmação de action.** Abrir vale como aval; editar recalcula e reavisa.
- **O mestre sempre encerra o turno**, com ou sem reactions pendentes.
- **Bônus acumulado é sempre de actionSpeed, nunca de acerto.** Bônus é específico do alvo;
  penalidade é geral.

## 4. Modelo de domínio proposto

### 4.1 `CharacterStatus` — o que falta

Hoje é um arquivo de ~150 linhas de comentário que nada consome. Precisa virar código, mas
**sem os campos que outra camada já possui**.

```go
// internal/domain/match/character_status.go
type CharacterStatus struct {
    ActionBar ResourceBar   // saldo e histórico de velocidades da rodada
    MoveBar   ResourceBar
    Ledger    ModifierLedger // bônus/penalidades acumulados
    Stance    Stance         // reservado; a regra de posturas ainda não existe
    Velocity  action.Velocity
}
```

**`Position` NÃO entra.** As posições vivem no `Room` (vieram dos payloads de WS) e a sessão
já as alcança por interface — ver `matchsession.PiecePositionSource`. Duplicar posição criaria
duas fontes de verdade, e a que o mapa desenha continuaria sendo a do `Room`. Quando o motor
precisar da posição de um personagem (arremetida, alcance, `targetSpeed`), **estenda a
costura existente**:

```go
type PiecePositionSource interface {
    PlayerPiecePositions(playerID uuid.UUID) []service.Point2D
    CharacterPosition(charID uuid.UUID) (service.Point2D, bool) // ← novo
}
```

### 4.2 ⚠️ Chaveamento: NPCs não existem na sessão hoje

`charSheets`, `participants` e `charToPlayer` são **todos indexados por `playerUUID`**, e o
construtor faz `if p.Sheet.PlayerUUID != nil { ... }`. Um NPC tem `PlayerUUID` nulo e
`MasterUUID` preenchido — portanto é **silenciosamente descartado**.

Mas o mestre envia as ações dos NPCs, e controla vários personagens ao mesmo tempo.

**A entidade de combate é o personagem, não o jogador.**

| Mapa | Hoje | Proposto |
|---|---|---|
| `charSheets` | `map[playerUUID]*CharacterSheet` | `map[sheetUUID]*CharacterSheet` |
| `statuses` | não existe | `map[sheetUUID]*CharacterStatus` |
| `charToPlayer` | `map[sheetUUID]playerUUID` | mantém — é a ponte para autorização e fog |
| `participants` | `map[playerUUID]*Participant` | mantém — autorização é por jogador |

`sheetUUID` já é o `CharacterID` das peças no tabuleiro, então o chaveamento novo casa
naturalmente com o que o `Room` envia.

**Autorização continua por jogador**; o motor opera por personagem. Os dois eixos coexistem
via `charToPlayer`.

### 4.3 Onde mora a diferença acumulada

**Não é `RollCondition`.** `RollCondition` é a struct **do mestre**: `Bias` são os dados
(vantagem/desvantagem) e `Modifier` é o ajuste manual (*"+3 porque teve criatividade
estratégica"*).

A diferença acumulada mora no `ModifierLedger` dentro do `CharacterStatus`, com escopo e
validade:

```go
type Modifier struct {
    Amount     int
    AgainstID  *uuid.UUID // nil = vale contra qualquer um
    ExpiresAt  Scope      // fim do turno | fim do round
    Reason     string     // para o Action History e para SystemData
}
```

### 4.4 ⚠️ Conflito no `Bias`

O sistema gera Desvantagem automaticamente (trocar de action, converter action em reaction) e
o mestre também pode conceder/anular vantagem. Os dois querem o mesmo campo.

**Decisão:** `RollCondition.Bias` permanece **exclusivamente do mestre**. A desvantagem
gerada pelo sistema entra como um `Modifier` de origem `system` no ledger, e o
`RollCalculator` soma os dois vieses no momento de derivar o resultado. Assim o mestre pode
anular a desvantagem do sistema sem que um sobrescreva o outro, e a auditoria consegue
distinguir quem pôs o quê.

### 4.5 Visibilidade

**Nem todo campo de uma action é público, inclusive no histórico** — durante a partida.

Exemplo canônico: uma esquiva fechada **não pode revelar** que embutiu um teste de Evasão; o
adversário precisa deduzir dos números. O Action History é uma **superfície de jogo com
visibilidade por campo**, não um log.

**Decisão técnica:** um único evento de WS com **projeção por destinatário**, não canais
separados. O `Room` já faz exatamente isso em `dispatchPerPlayer` para o fog de guerra —
reusar o padrão em vez de inventar outro.

## 5. As fases

Cada fase é uma sessão e um PR. A ordem foi escolhida para haver **algo visível rodando na
Fase 2**, em vez de esperar até o fim.

---

### Fase 1 — Fundação

**Objetivo:** criar o estado e o motor de rolagem. Puro, sem I/O, testável sem partida.

**Escopo:**
- `CharacterStatus` conforme §4.1, com `ResourceBar` e `ModifierLedger`.
- Re-chavear `charSheets` para `sheetUUID` e **passar a carregar NPCs** (§4.2). Ajustar
  `InitMatchSessionUC` e os dois construtores de `MatchSession`.
- `RollCalculator` de verdade: 2 D10, crítico pela combinação, passivo vs. ativo,
  vantagem/desvantagem acumulável, e o retorno carregando **a margem** e **os dados
  individuais**.
- **Mecanismo de configuração de partida** — o mesmo que destrava o `fog_mode` hardcoded em
  `room.go` (ver Known Issues do `AGENTS.md`). Padrões: 2 D10, valor médio derivado do
  conjunto, timer de reação desligado, reação padrão ligada, `fog_mode: explored`.

**Fora de escopo:** ninguém chama o `RollCalculator` ainda.

**Pronto quando:** testes unitários cobrem crítico, erro crítico, passivo, vantagem,
desvantagem acumulada e margem; `go vet -tags=integration ./internal/...` passa; um NPC
aparece em `session.charSheets`.

---

### Fase 2 — Primeira colisão de verdade

**Objetivo:** um ataque contra um personagem resolvendo ponta a ponta. **Primeira coisa
visível no browser.**

**Escopo:**
- `buildAction` mapeando o payload **inteiro** (Skills, Speed, Feint, Attack, Defense) — hoje
  descarta quase tudo.
- `TurnResolver`, ramo `TargetKindCharacter`: chamar o `RollCalculator`, produzir
  `ActionResult`, popular `Blows` (dar construtor a `battle.Blow`).
- A escada de margem do repelir e da defesa (`docs/dev/match/combat-engine.md`).
- Aplicar o resultado: dano na ficha do alvo.
- `resolution_updated` com payload de verdade (hoje só `IsSettled`, e só para o mestre).

**Fora de escopo:** barras, economia de turno, reações ativas, vários alvos.

**Pronto quando:** um ataque enviado pelo browser produz dano visível na ficha do alvo, e o
`TurnResolution` carrega margem e dados individuais.

---

### Fase 3 — A economia do turno

**Objetivo:** o turno vira dinâmico.

**Escopo:**
- actionSpeed real alimentando `Action.Speed.Result` — a fila passa a ter prioridade.
- As duas barras, com preço por barra, média das velocidades, carry-over e teto.
- Recalculação forward-only da posição quando a média muda.
- Action enviada no meio do round entra com a velocidade rolada, **sem reordenação
  retroativa**.
- Fim de round quando as barras acabam.
- Ações compostas: ataque amarrado ao **fim do movimento** (`max(tempo do ataque, fim do
  movimento)`), para cait, arremetida e investida.

**Fora de escopo:** iniciativa e `RoundMode.Race` (o `Free` usa o valor passivo 11).

**Pronto quando:** o exemplo canônico da spec de jogo (p1=20, p2=23, p3=11) reproduz a ordem
`p2 → p1 → p3 → p2` e os saldos `+9 / 0 / −2` em teste automatizado.

---

### Fase 4 — Reações

**Objetivo:** o catálogo completo e a cadeia de resolução.

**Escopo:**
- Reações passivas (esquiva por reflexo → defesa) aplicadas automaticamente, na ordem certa.
- Reações ativas: escape, escape defensivo, esquiva fechada, escape fechado, repelir, não
  fazer nada. Custo por barra conforme a tabela em `combat-engine.md`.
- Conversão action→reaction com **Desvantagem** (pior das duas), não média.
- **Resolução em cadeia com vários alvos**: o estado do ataque sai alterado de cada resolução
  e entra na próxima. A ordem de abertura do mestre determina o resultado.
- Validar que só alvos podem reagir (hoje qualquer um pode).
- Timer de reação como regra de partida (padrão desligado).

**Fora de escopo:** posturas.

**Pronto quando:** teste de integração cobre um ataque em área com três alvos reagindo
diferente, e a ordem de abertura muda o resultado de forma verificável.

---

### Fase 5 — Regência e visibilidade

**Objetivo:** a mesa funcionando.

**Escopo:**
- Abrir reaction como operação de primeira classe (mesmo ciclo da action).
- Encerrar turno explicitamente, com diálogo de confirmação quando houver reactions enviadas
  e não abertas — elas entram no cálculo, mas perdem o momento de narrar.
- Visibilidade por campo, via projeção por destinatário (§4.5): mecânica pública, cálculo só
  do mestre até o encerramento.
- Notificação de action enfileirada **só para o mestre**.
- Action History como superfície de jogo, com os campos ocultos respeitados.
- Tabela `SystemData` — auditoria de toda interferência do mestre.
- `CloseRoundUC` plugado (hoje não é chamado por nada) e `round_closed` emitido (hoje
  declarado e nunca enviado).

**Pronto quando:** dois navegadores logados como jogadores diferentes veem projeções
distintas do mesmo turno, e a interferência do mestre aparece em `SystemData`.

---

### Fase 6 — Front

**Repo:** `System_X_System_React`. **A tela já existe** — é a partida em execução, com o mapa,
as peças e uma sidebar de personagens à direita. Faltam os componentes.

**Escopo:**
- Bottom sheet de action: alvo (clicando na peça), arma/habilidade, perícias, aura (futuro).
- **Rascunho persistente**: fechar a bottom sheet preserva a configuração; trocar de alvo ou
  de habilidade **migra** a configuração em vez de resetar.
- Botões de reação ao lado do alvo. **Clicar** envia direto; **clicar e segurar** abre a
  bottom sheet. Segurando em Scape, vem com Accelerate pré-setado.
- As duas barras: a própria e a geral (visível para todos).
- Balões acima do ícone: mecânica ao abrir, resultado ao encerrar.
- Action History.

**Em aberto:** se a sidebar direita de personagens continua onde está.

---

## 6. Riscos

| Risco | Mitigação |
|---|---|
| **`CharacterStatus` com a forma errada** — toca barras, ledger, postura, velocidade; errar cascateia | Desenhá-lo por inteiro na Fase 1, com os campos reservados já previstos, mesmo sem uso |
| Re-chavear `charSheets` quebra o fog de guerra | `charToPlayer` continua sendo a ponte; a suíte de fog (`fog_*_test.go`) é a rede de segurança |
| `MatchSession` não tem lock próprio | `room.go` é a única serialização — todo `Execute` novo herda a obrigação de segurar `r.mu` |
| Regras mudarem depois do MVP | Números viram configuração de partida; a forma da escada fica em código |

## 7. Pontas soltas conhecidas

Registradas, **fora de escopo** deste spec:

- **Percepção mental** — bloqueada: os subatributos mentais não existem. Só afeta o percept
  de início de batalha (quem vê os alvos), que é exceção, não regra.
- **Posturas** — em guarda / ofensiva / defensiva / evasiva. Quando existirem, o desconto do
  escape fechado passa a exigir postura evasiva; enquanto não existem, vale sem condição.
- **Movimento detalhado** — accelerate/brake/charge, teste de curva `f(v,x) = vel·|sen(x)|`,
  salto, salto rasante, momentum. Produz o `moveSpeed`; encaixa na costura existente.
- **Tipos de dano** — concusivo, cortante, perfurante, ultra perfurante. A jusante da colisão.
- **Iniciativa** — soma na actionSpeed e força `RoundMode.Race`. `ChangeMode` já existe e
  ignora o parâmetro.
- **Golpe simultâneo, combos, finta com direcionalidade, consumo de energy, desengaje.**
- **Mesa Livre** — `Match → Campaign → Scenario` com cenário compartilhado. É o que torna a
  auditoria (`SystemData`) importante de verdade.
