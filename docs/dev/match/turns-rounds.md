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
