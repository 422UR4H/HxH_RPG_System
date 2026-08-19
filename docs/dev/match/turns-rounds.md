# Turnos e Rounds (Turns & Rounds)

> ⚠️ **WIP — Refatoração Semântica em Andamento**
>
> O sistema de turnos está passando por uma refatoração semântica. **Ambos os pacotes coexistem** no código:
>
> - `turn/` — Semântica **antiga**: Turn agrupa Actions diretamente.
> - `round/` — Semântica **nova**: Round agrupa Turns (inversão da hierarquia).
>
> O código em `round/` é a direção futura. O código em `turn/` ainda é referenciado.

## Semântica Antiga — `turn/`

`turn.Engine` gerencia uma fila de ações (`action.PriorityQueue`) e o fluxo de turnos:

- **Modos**: Free ou Race (`enum.TurnMode`).
- **Fila de prioridade**: Ações ordenadas por `Speed.Result` (maior = primeiro).
- `Add(action)` → insere na fila.
- `NextAction()` → extrai a ação de maior velocidade (`ExtractMax`).
- `PullAction(id)` → extrai ação específica por UUID (`ExtractByID`, busca linear).
- `AttachReaction(reaction)` → valida que `ReactToID` corresponde à ação corrente. Falha com `ErrReactionNotCompatible`.
- `CloseTurn()` → cria novo turno com mesmo modo, reseta `closeTurnTriggered`.
- `ChangeMode(initiative)` → alterna Free↔Race.

```go
// TODO: refatorar trocando closeTurnTriggered por um método que chame CloseTurn e faça o trigger
```

```go
// TODO: create and finish Initiative to continue here
```

O `Turn` possui `coast *int` — se nil, o turno é Free; caso contrário, indica custo em Race.

Erros: `ErrReactionNotCompatible`, `ErrTurnIsEmpty`.

## Semântica Nova — `round/`

`round.Engine` tem estrutura similar, mas com hierarquia invertida:

- **Round agrupa Turns** (ao invés de Turn agrupar Actions).
- `preparedActions map[uuid.UUID]*action.Action` — campo novo, não presente em `turn.Engine`.
- `closeRoundTriggered` substitui `closeTurnTriggered`.
- `CloseRound()` substitui `CloseTurn()`.
- `NewEngine` retorna `error` (valida que `closeRoundTriggered != nil`), diferente de `turn.NewEngine`.

```go
// TODO: refatorar trocando closeRoundTriggered por um método que chame CloseRound e faça o trigger
```

```go
// TODO: create and finish Initiative to continue here
```

Erro adicional: `ErrCloseRoundTriggeredCantBeNil`.

## Diferenças — Tabela Comparativa

| Aspecto                | `turn/` (antigo)         | `round/` (novo)              |
|------------------------|--------------------------|------------------------------|
| Unidade principal      | `Turn` (agrupa Actions)  | `Round` (agrupa Turns)       |
| Ações preparadas       | —                        | `preparedActions map[uuid.UUID]*Action` |
| Flag de encerramento   | `closeTurnTriggered`     | `closeRoundTriggered`        |
| Método de encerramento | `CloseTurn()`            | `CloseRound()`               |
| Construtor retorna     | `*Engine`                | `(*Engine, error)`           |
| Validação nil no construtor | Não                 | Sim (`closeRoundTriggered`)  |
| Erro exclusivo         | —                        | `ErrCloseRoundTriggeredCantBeNil` |

## Referências de Código

| Arquivo              | Responsabilidade                                  |
|----------------------|---------------------------------------------------|
| `turn/turn.go`       | Struct `Turn` (mode, actions, events, coast)      |
| `turn/engine.go`     | `Engine` antigo (fila, modos, CloseTurn)          |
| `turn/error.go`      | `ErrReactionNotCompatible`, `ErrTurnIsEmpty`      |
| `round/round.go`     | Struct `Round` (mode, turns)                      |
| `round/engine.go`    | `Engine` novo (preparedActions, CloseRound)       |
| `round/error.go`     | `ErrCloseRoundTriggeredCantBeNil`                 |

## Fase 3 — economia por barra e fechamento automático do round

> O ciclo Round/Turn que roda hoje vive em `service.RoundOrchestrator` e
> `matchsession.MatchSession`, não nos `Engine`s descritos acima — a Fase 3 não mexeu nessa
> dívida de nomenclatura. O que ela acrescentou foi a economia que faltava para o round fechar
> sozinho. Detalhamento completo em
> [`combat-engine.md`](../combat-engine.md#o-que-a-fase-3-fixou-no-motor) § "O que a Fase 3
> fixou no motor"; aqui só o que muda no ciclo Round/Turn em si.

### Preço por barra

`Round.prices` substitui o antigo campo único `coast` mencionado nas tabelas acima: é um preço
por `action.Bar` (`action`/`move`), congelado por `RoundScheduler.FreezePrices` na primeira
seleção que vê trabalho pendente naquela barra, e nunca mais alterado no round corrente.

### Fechamento automático

`MatchSession.OpenNextAction` consulta `RoundScheduler.SelectNext` a cada chamada. Quando
nada pendente passa no porteiro que lhe cabe, a resposta vem com
`TurnTransition.RoundExhausted = true`, e `OpenNextActionUC`
(`internal/application/match/open_next_action.go`) chama `CloseRoundUC` na hora — o primeiro
chamador que esse use case ganha. O round não depende mais só do caminho indireto via
`change_scene`.

### Duas mensagens WS novas

- **`round_mode_changed`** — broadcast em resposta a `change_round_mode` (o mestre pede a
  troca de regime; `RoundModeChangedPayload` avisa a mesa inteira do novo `mode`).
- **`bars_updated`** — broadcast depois de qualquer operação que mexe nas barras
  (`open_next_action`, `pull_action`, `change_round_mode`): preços congelados, saldo e
  velocidades já registradas de cada personagem, e a ordem projetada. Nada que identifique a
  ação em si.

`round_closed` já existia declarado desde a Fase 2 (ver `flows/05-lacunas.md` §7) e **passa a
ser emitido de fato** no fechamento automático descrito acima.
