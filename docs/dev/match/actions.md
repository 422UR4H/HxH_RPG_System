# Sistema de Ações (Actions)

## Action

```go
// TODO: REFACTOR TO IMPLEMENT I_ACTION INTERFACE
```

`Action` representa uma ação de um personagem em combate. Componentes:

| Campo       | Tipo              | Descrição                                   |
|-------------|-------------------|---------------------------------------------|
| `id`        | `uuid.UUID`       | Identificador único da ação                 |
| `actorID`   | `uuid.UUID`       | Personagem que executa a ação               |
| `TargetID`  | `[]uuid.UUID`     | Alvos (pode ser múltiplos)                  |
| `ReactToID` | `uuid.UUID`       | Ação à qual esta é uma reação               |
| `Speed`     | `ActionSpeed`     | Velocidade (determina ordem na fila)        |
| `Skills`    | `[]Skill`         | Perícias usadas na ação                     |
| `Trigger`   | `*Trigger`        | Gatilho condicional (TODO: criar categorias)|
| `Feint`     | `*RollCheck`      | Finta                                       |
| `Move`      | `*Move`           | Componente de movimentação                  |
| `Attack`    | `*Attack`         | Componente de ataque                        |
| `Defense`   | `*Defense`        | Componente de defesa                        |
| `Dodge`     | `*Dodge`          | Componente de esquiva                       |

Timestamps: `openedAt` (quando a ação foi aberta) e `confirmedAt` (quando foi confirmada).

Tipos auxiliares:
- `MasterAction` — ação simplificada do mestre (TargetID, Skills, Move, Attack, ActionSpeed como RollCheck).
- `Initiative` — determina ordem de cena (targetID, skills, FinalResult).

## ActionSpeed & RollCheck

`ActionSpeed` compõe `RollCheck` + `Bar`:

```
ActionSpeed { Bar int; RollCheck }
```

`RollCheck` — resultado de uma rolagem de dado vinculada a uma perícia:

| Campo        | Tipo          | Descrição                       |
|--------------|---------------|---------------------------------|
| `Context`    | `RollContext` | Dados, condição e resultado     |
| `SkillName`  | `string`      | Nome da perícia testada         |
| `SkillValue` | `int`         | Valor da perícia                |
| `Result`     | `int`         | Resultado final da rolagem      |

`RollContext` contém `Dice []die.Die`, `Condition *RollCondition` e `Result *int`.

`RollCondition` modifica rolagens:

| Campo         | Tipo     | Descrição                                        |
|---------------|----------|--------------------------------------------------|
| `Bias`        | `int`    | 1 = vantagem, -1 = desvantagem (acumula)         |
| `Modifier`    | `int`    | Modificador numérico fixo                        |
| `Description` | `string` | Descrição da condição                            |

## PriorityQueue

Lista simples em ordem de chegada (não é mais heap): no regime Race a ordem é calculada na hora
por `service.RoundScheduler` (chave por personagem, que muda a cada ação dele); a ordem de chegada
desempata. No Free, `ExtractMax`/`Peek` escolhem pela `Action.OrderSpeed()` (a menor velocidade
entre as barras que a ação cobra). Nos dois, um movimento empilhado num pendente
(`Action.IsBlockedIn`, `action/stack.go`) não é candidato — ver `combat-engine.md`,
*Movimentos empilhados*.

| Método           | Descrição                                                    |
|------------------|--------------------------------------------------------------|
| `Insert`         | Põe a ação no fim da fila                                    |
| `ExtractMax`     | Remove e retorna a desbloqueada de maior `OrderSpeed` (Free) |
| `Peek`           | Consulta essa mesma ação sem remover                         |
| `ExtractByID`    | Busca linear por UUID e remove                               |
| `All`            | Cópia da fila, em ordem de chegada                           |

## Componentes de Combate

**Attack** — ataque com arma opcional:
- `Weapon *enum.WeaponName`, `Hit RollCheck`, `Damage RollCheck`
- `Charge *RollCheck` (investida), `RelativeVelocity float64`

**Defense** — defesa com arma opcional:
- `Weapon *enum.WeaponName`, `RollCheck` (embeddado)

**Dodge** — esquiva:
- `RollCheck` (embeddado). WHICH dodge (ativa, fechada, esquivas de fuga) é responsabilidade
  de `ReactionKind` na própria `Action`, não de um enum aqui — um `Category` antigo cobria o
  mesmo eixo de forma menos expressiva e foi removido.

**Move** — movimentação no espaço 3D:
- `Category enum.MoveCategory`, `From *[3]int` (origem: a casa da peça no enfileiramento, ou o
  destino do movimento anterior quando empilhado), `Position [3]int` (destino)
- `Speed *RollCheck`, `Charge *RollCheck`, `FinalSpeed int`
- `StackedOn uuid.UUID` — o movimento pendente do mesmo personagem sobre o qual este se empilhou
  (`uuid.Nil` = não empilhado; spec 2026-11-06 D3)

**Velocity** — vetor de velocidade:
- `Speed float64`, `DirectionPlan float64`, `DirectionAlt float64`

**Battle/Blow** — golpe resolvido entre dois personagens:
- `actorID`, `targetID`, `attack Attack`, `attackSkills/defenseSkills *Skill`, `defense Defense`

## Referências de Código

| Arquivo                   | Responsabilidade                               |
|---------------------------|-------------------------------------------------|
| `action/action.go`        | Struct `Action` (TODO: I_ACTION interface)      |
| `action/action_speed.go`  | `ActionSpeed` (Bar + RollCheck)                 |
| `action/roll_check.go`    | `RollCheck`                                     |
| `action/roll_context.go`  | `RollContext` (Dice, Condition, Result)          |
| `action/roll_condition.go`| `RollCondition` (Bias, Modifier)                |
| `action/priority_queue.go`| `PriorityQueue` (lista em ordem de chegada)     |
| `action/stack.go`         | `IsBlockedIn` (movimento empilhado bloqueado)   |
| `action/attack.go`        | `Attack`                                        |
| `action/defense.go`       | `Defense`                                       |
| `action/dodge.go`         | `Dodge`                                         |
| `action/move.go`          | `Move` (posição 3D, velocidade)                 |
| `action/skill.go`         | `Skill` (perícia usada em ação)                 |
| `action/velocity.go`      | `Velocity` (vetor de velocidade)                |
| `action/initiative.go`    | `Initiative` (ordem de cena)                    |
| `action/master_action.go` | `MasterAction` (ação simplificada do mestre)    |
| `action/trigger.go`       | `Trigger` (TODO: criar categorias de trigger)   |
| `battle/blow.go`          | `Blow` (golpe resolvido)                        |
