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

Fila de prioridade baseada em max-heap (`container/heap`). Ações com maior `Speed.Result` são processadas primeiro.

```go
func (aq PriorityQueue) Less(i, j int) bool {
    return aq[i].Speed.Result > aq[j].Speed.Result // invertido: max-heap
}
```

Operações:

| Método           | Complexidade | Descrição                              |
|------------------|--------------|----------------------------------------|
| `Insert`         | O(log n)     | Insere ação na fila                    |
| `ExtractMax`     | O(log n)     | Remove e retorna a ação mais rápida    |
| `Peek`           | O(1)         | Consulta a ação mais rápida sem remover|
| `ExtractByID`    | O(n)         | Busca linear por UUID e remove         |

## Componentes de Combate

**Attack** — ataque com arma opcional:
- `Weapon *enum.WeaponName`, `Hit RollCheck`, `Damage RollCheck`
- `Charge *RollCheck` (investida), `RelativeVelocity float64`

**Defense** — defesa com arma opcional:
- `Weapon *enum.WeaponName`, `RollCheck` (embeddado)

**Dodge** — esquiva por categoria:
- `Category enum.DodgeCategory`, `RollCheck` (embeddado)

**Move** — movimentação no espaço 3D:
- `Category enum.MoveCategory`, `Position [3]int` (coordenada 3D)
- `Speed *RollCheck`, `Charge *RollCheck`, `FinalSpeed int`

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
| `action/priority_queue.go`| `PriorityQueue` (max-heap por Speed.Result)     |
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
