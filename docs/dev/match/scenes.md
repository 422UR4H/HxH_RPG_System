# Cenas (Scenes)

## Ciclo de Vida

Uma `Scene` representa um segmento narrativo dentro de uma partida. O fluxo é linear:

```
NewScene(category, briefDescription) → AddTurn() ... AddTurn() → FinishScene(briefFinalDescription)
```

- `NewScene` recebe uma `enum.SceneCategory` (Battle ou Roleplay) e uma descrição inicial.
- `AddTurn` adiciona turnos sequencialmente. Falha com `ErrSceneIsFinished` se a cena já foi encerrada (`finishedAt != nil`).
- `FinishScene` define a descrição final e marca `finishedAt`. Falha se já estiver finalizada.

Campos relevantes:

| Campo                     | Tipo                 | Descrição                          |
|---------------------------|----------------------|------------------------------------|
| `category`                | `enum.SceneCategory` | Battle ou Roleplay                 |
| `BriefInitialDescription` | `string`             | Descrição do início da cena        |
| `BriefFinalDescription`   | `*string`            | Descrição do desfecho (nil = ativa)|
| `turns`                   | `[]*turn.Turn`       | Turnos executados nesta cena       |
| `createdAt`               | `time.Time`          | Timestamp de criação               |
| `finishedAt`              | `*time.Time`         | Timestamp de encerramento          |

## Category vs Mode

São eixos independentes:

- **Scene.category** (`enum.SceneCategory`): Battle ou Roleplay — define o *tipo* da cena.
- **Turn.mode** (`enum.TurnMode`): Free ou Race — define o *ritmo* do turno.

Uma cena de Battle pode conter turnos Free (exploração dentro de combate) e uma cena de Roleplay pode conter turnos Race (perseguição narrativa). Não há acoplamento entre os dois.

## Match → Scene

`Match` mantém uma lista de `scenes []*scene.Scene` e `events []GameEvent`.

`match.Engine` gerencia transições de cena via `ChangeScene(initiative)`, que alterna entre Battle e Roleplay. O engine rastreia a `sceneCategory` atual a partir da última cena.

```go
// TODO: create and finish Initiative and turn.engine.ChangeMode to continue here
func (e *Engine) ChangeScene(initiative *action.Initiative) { /* toggles Battle↔Roleplay */ }
```

> **Nota:** `ChangeScene` depende de `action.Initiative`, que ainda não está implementado. O TODO no código-fonte marca esse ponto de continuação.

## Referências de Código

| Arquivo              | Responsabilidade                          |
|----------------------|-------------------------------------------|
| `scene/scene.go`    | Struct `Scene`, `NewScene`, `AddTurn`, `FinishScene` |
| `scene/error.go`    | `ErrSceneIsFinished`                      |
| `match/match.go`    | Struct `Match` (agrega scenes e events)   |
| `match/engine.go`   | `Engine`, `NewEngine`, `ChangeScene`      |
