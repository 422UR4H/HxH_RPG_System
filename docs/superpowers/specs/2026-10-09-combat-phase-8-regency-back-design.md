# Fase 8 — Regência — pacote de back — design

> **Escopo:** as lacunas de back e de contrato que a sessão de planejamento do front da Fase 8
> levantou em 2026-10-09, já decididas pelo dono do produto. **Um PR, repo `System_X_System`.** O
> front da Fase 8 vem **depois** deste merge, em outro PR, e lê o contrato que este PR escreve.
>
> Documento mestre: [`2026-09-20-front-combat-phases.md`](2026-09-20-front-combat-phases.md) §8,
> §6A.6 F7, §4.6, §11.1.
> Contrato: [`../../dev/api/match-combat-ws.md`](../../dev/api/match-combat-ws.md) — `edit_action`,
> `action_edited`, `resolution_updated`, `match_full_state`.
> Motor: [`../../dev/match/combat-engine.md`](../../dev/match/combat-engine.md), "A edição do mestre".
> Plano: [`../plans/2026-10-09-combat-phase-8-regency-back.md`](../plans/2026-10-09-combat-phase-8-regency-back.md).
>
> **Branch:** `feat/combat-phase-8-regency-back`, a partir de `main` em `74fb5c0` (depois do PR #84).

## 0. O workflow (cópia do §0.1 do documento mestre)

1. **Uma sessão por fase por repo.** Back e front rodam em paralelo quando não tocam arquivo em
   comum — são repos diferentes, então normalmente não tocam.
2. A sessão **lê o documento mestre e o contrato** (`docs/dev/api/match-combat-ws.md`), escreve o
   **design spec** e o **plano**, e **para** para o dono do produto revisar.
3. **Lacuna ou contradição: liste e pare.** Ela volta para o autor do documento, que corrige o
   texto. Não se contorna, e não se decide regra de jogo por conta.
4. Aprovado o spec, a sessão **compacta** e implementa **lendo o próprio plano do disco**. Se
   ela não conseguir implementar a partir do plano, o plano estava incompleto — é melhor
   descobrir nessa hora.
5. Implementação por **subagent-driven-development**, uma tarefa por subagente.
6. **Verificação no browser, com três contas** (`test@`, `test2@`, `test3@mail.com`, senha
   `12345678`): um mestre e dois jogadores, jogando o caminho que o usuário faz.
7. PR aberto dizendo o que foi verificado e **o que não foi**.

> **O passo 6, neste PR.** Este pacote não tem tela: os botões de editar só entram no PR de front.
> A verificação daqui é automatizada, contra a `Room` real por websocket (o padrão dos
> `*_e2e_test.go`), mais a integração do Postgres — ver §7. O caminho do usuário, com as três
> contas, é verificado no PR de front, contra este back já mergeado. Mesmo precedente do pacote de
> back da Fase 7.

**Effort.** A sessão que planejou rodou em effort baixo e compensou lendo o código com arquivo e
linha; as descobertas estão aqui. Implementadores: `model: sonnet`, effort `medium`, por um tipo
de agente `implementer` (Task 0 do plano o cria neste repo — o front já tem o seu). As tarefas
marcadas **opus** no plano tocam `room.go`/projeção e merecem o modelo maior.

## 1. O que você precisa saber antes

- **Turno** = uma action e suas reactions. `actorId` é o **sheetUUID**. O mestre edita a ação do
  turno aberto ou uma reação dela por `edit_action` (`actionId` ausente/zero = a ação do turno).
- **A action editada É a action** (`combat-engine.md`). A condição do mestre (`RollCondition`:
  viés, ajuste, motivo) mora dentro do `RollCheck` (`RollCheck.Context.Condition`), dentro da
  `Action`. Não há versão paralela. No fechamento, a action é gravada inteira na tabela `actions`
  (`persist_turn_close.go`, `insertAction`) — **a condição vai junto, em cada coluna JSON**.
- **O valor que a edição descartou** vai para `overridden_action_values`: uma linha por campo,
  guardando o ORIGINAL. Editar de volta ao original apaga a captura (`captureOverride`,
  `match_session.go`). **Não existe verbo de confirmação**: passar o bastão confirma.
- **A edição muda o desfecho, nunca a economia.** Barras cobradas, `Speeds` registradas e ordem já
  jogada não se refazem.
- **A edição não re-rola nada.** Os dados caíram quando a action chegou; o viés só escolhe qual dos
  dois conjuntos (`Attempts.Primary`/`Secondary`) é lido (`RollCalculator.Derive`). Uma leitura
  **passiva** não tem dado: o viés não tem o que escolher, só o ajuste conta.
- **O dano tem um conjunto só de dados** (`rollActionDice`: "Only Primary, because damage has no
  advantage"). É regra escrita no código.
- `room.go` é dono do lock (`r.mu`). `MatchSession` não tem lock. **Nada que envia a cliente roda
  com `r.mu` preso.**
- **O cálculo do turno aberto é só do mestre** (`publishResolution`: `!IsSettled` →
  `sendToMaster`). Fechado, ele é projetado a todos por `service.ProjectResolution`. O
  `match_full_state.resolution` existe só para o mestre e só com turno aberto
  (`buildMatchFullState`, `room.go` ~2797), calculado por `session.ResolveTurn(t)` — um recálculo
  puro.
- **O wire da action** (`internal/app/wire/actionwire`) é um mapeamento **explícito** campo a
  campo. Um campo novo no domínio **não** aparece no wire a menos que alguém o mapeie. É isso que
  mantém `RollCheck.Context` fora de toda superfície hoje.
- **Nada que uma projeção esconde é descartado na gravação** (regra da Fase 7): o dado vai inteiro
  para o banco e é escondido só na leitura.

## 2. As decisões recebidas

Fechadas pelo dono do produto em 2026-10-09, a partir das lacunas que a sessão do front listou.

| # | Lacuna | Decisão |
|---|---|---|
| 1 | Editar perícias não muda nada (§11.1 vale: ninguém lê o resultado de `Skills`, exceto a `Evasion` das fechadas) | O painel **não** ganha edição de perícias até existir a corrente de testes. A condição na `Evasion` das reações fechadas também fica **fora do painel** — uma tela especial para uma perícia só, que a corrente vai redesenhar. Escrever isso no documento mestre |
| 2 | `edit_action` não troca a perícia do dano (contrato, linha 200: "ainda não implementada") | Campo novo no payload. Aceita **qualquer perícia válida do enum** ("Grab, ou outra coisa, talvez"); a tela oferece Push e Grab. O original vai para `overridden_action_values`. A resolução diz qual perícia mediu o dano |
| 3 | A condição do mestre não está em superfície nenhuma — recarregar perde o que foi editado (§0.2) | A condição atual de cada rolagem vai **ao mestre**, com o turno aberto, inclusive no `match_full_state`. Só para o mestre |
| 4 | Editar de volta a condição não apaga a captura (original `nil`, wire só manda `{0,0,""}`) | Condição zerada `{bias: 0, modifier: 0, description: ""}` conta como "sem condição"; editar de volta apaga a captura. Com teste |
| 5 | O contrato diz que toda seção substitui a lista inteira | `conditions` se aplica **por rolagem**; só `skills` e `targetIds` substituem a lista inteira |
| 6 | Das 8 rolagens que `conditions` aceita, a resolução só lê `hit` e o `moveSpeed` das fugas | **Tudo é corrigido.** A tela esconde o viés onde ele não faz sentido; o back **não** remove nada que aceita hoje — a regra pode se expandir |
| 7 | Dano: o ajuste não é lido; o viés não tem o que escolher | O ajuste entra no dano bruto. **Viés no dano é recusado com erro** |
| 8 | `speed` e o `moveSpeed` da própria ação não mudam o desfecho do turno aberto | Ficam como estão no back. Editar a velocidade de uma ação **antes** dela agir (na fila) faz sentido e é feature futura — registrar como pendência |
| 9 | A condição do mestre fica guardada? | **Sim**, dentro da action, na tabela `actions` (§1). O que este pacote não faz é copiá-la também para `turns.resolution` |
| 10 | A defesa padrão não tem onde guardar a condição | Num alvo que reagiu com um tipo que mantém a defesa padrão (`dodge`, `closedDodge`, `escapeGuard`), editar `defense` na reação cria ali o portador da condição. **Alvo que não reagiu**: a defesa e a esquiva passivas dele ficam sem edição nesta fase — pendência no documento mestre |
| 11 | Depois de fechado, o jogador vê a perícia do dano? | Sim, onde já vê o dano bruto. A condição do mestre nunca vai a jogador nenhum |

## 3. O desenho

### 3.1 Toda condição que a resolução lê, ela lê (decisões 6, 10)

Hoje a resolução só passa `Condition` ao `Derive` em três lugares: o acerto
(`turn_resolver.go`, `resolveCharacterStep`), e `speed`/`moveSpeed` em `deriveSpeeds`
(`match_session.go`). Este pacote passa a condição em toda leitura que a resolução usa:

| Rolagem | Onde se lê | Portador da condição |
|---|---|---|
| Reflexo (a esquiva) | `deriveReflex` (`reaction_collision.go`) | `in.Reaction.Dodge.Context.Condition` |
| `Evasion` das fechadas | `deriveEvasion` | a entrada `Evasion` de `in.Reaction.Skills` (`Context.Condition`) |
| Aparo | `resolveRepel` | `in.Reaction.Repel.Context.Condition` |
| Defesa padrão | `ResolveReaction`, o `calc.Derive` da defesa passiva | `in.Reaction.DefaultDefense.Context.Condition` (novo, §3.2) |
| Dano | `RawDamage` via um helper novo (§3.3) | `a.Attack.Damage.Context.Condition` — só o `Modifier` |

Consequências que o revisor precisa enxergar:

- **A reserva da esquiva fechada passa a refletir a condição.** A reserva é `|Reflexo − Evasion|`
  (`dodgeAndReserve`), calculada sobre as leituras. Uma condição no Reflexo ou na Evasion muda as
  duas leituras, portanto muda a reserva. É a leitura que a regra descreve; não há caminho para
  separar sem duplicar a derivação. **(D4, confirmar.)**
- **O ajuste no acerto já mudava todos os alvos** (o golpe é um só). Nada muda aqui.
- **Finta continua ignorada.** Não existe resolução da finta (§11.2); o back aceita e guarda a
  condição em `feint`, como hoje, e nada a lê. O contrato diz isso.
- **`speed` e o `moveSpeed` da própria ação** continuam como estão (decisão 8): `deriveSpeeds` já os
  relê, e o que eles movem é a economia, que não se refaz. O contrato diz que eles não mudam o
  desfecho do turno aberto.
- **Viés numa leitura passiva** continua aceito e guardado (decisão 6); `Derive` o ignora no ramo
  passivo, como sempre. São passivas, com o turno aberto: a defesa padrão (sempre), o `moveSpeed`
  de um `closedEscape` (Shift), e `speed` fora do regime Race.

### 3.2 O portador da defesa padrão (decisão 10)

A defesa que a resolução usa é sempre a **passiva**, calculada da ficha do alvo; nenhuma action a
carrega. O campo `Action.Defense` existente é outra coisa: um componente que um jogador pode
declarar numa action (com arma), que a resolução **nunca lê**, e que **vai ao wire** — a declaração
viaja a todos (`actionwire/from.go`, bloco `a.Defense`).

**Campo novo: `Action.DefaultDefense *RollCheck`.** Existe só numa **reação** cujo tipo mantém a
defesa padrão (`ReactionKind.KeepsDefault()`, com `ReactionKind != ""`), e só depois que o mestre a
editou. Ele não carrega dado (a defesa padrão é passiva); carrega a condição.

- **`resolveRollCheck(FieldDefense)`**: numa reação com `KeepsDefault()`, devolve
  `a.DefaultDefense`, **criando-o** (`&RollCheck{SkillName: "Defense"}`) se for `nil`. Em qualquer
  outra action, o comportamento de hoje (`a.Defense`, ou `ErrConditionTargetMissing`). Na passada
  de validação, a criação acontece na cópia rasa (`shadow`) e não toca a action real.
- **Por que não reusar `a.Defense`:** ele vai ao wire. Uma reação que de repente aparecesse com um
  componente `defense` contaria ao dono que reconecta — e a todos, pela declaração — que o mestre
  mexeu na defesa antes do fechamento. `DefaultDefense` não é mapeado em `actionwire`, como
  `RollCheck.Context` não é. **(D1, confirmar.)**
- **Não muda a economia:** as barras de uma reação vêm de `ReactionKind.Bars()`, não dos
  componentes (`bar.go`, `Bars`).
- **Persistência:** coluna nova `actions.default_defense JSONB NULL` (migração goose), escrita por
  `insertAction`. `NULL` = o mestre não editou a defesa padrão desta reação (e toda linha anterior a
  esta coluna). O histórico **não** a lê de volta: nada a projeta, e a regra "nada que se esconde é
  descartado na gravação" é cumprida pela escrita.
- **Captura:** igual a qualquer condição — campo `defense.condition`, original `nil`.

### 3.3 O dano: ajuste lido, viés recusado (decisão 7)

- **Helper novo** `TurnResolver.rawDamage(in, a) (int, error)`: `RawDamage(dados, arma, catálogo,
  valor da perícia do dano)` mais o `Modifier` da condição em `a.Attack.Damage`, com **piso zero**.
  Substitui as duas chamadas diretas de `RawDamage` (`seedChain` e o ramo de parede em `Resolve`).
  O ajuste no dano bruto vale para a cadeia inteira (é a semente da cadeia) e para a parede.
- **Piso zero:** um ajuste negativo maior que o dano bruto daria dano bruto negativo, que a cadeia
  carregaria como resíduo negativo. `EffectiveDamage` e `floorZero` já fazem piso no resto do
  cálculo; este é o mesmo piso, um passo antes. **(D3, confirmar.)**
- **Viés no dano é recusado:** na passada de validação de `ApplyMasterAction`, uma `ConditionEdit`
  com `Field == FieldDamage` e `Bias != 0` devolve `ErrDamageHasNoAdvantage`
  (`"damage has no advantage: a damage condition takes no bias"`) e nada muda. Vai ao cliente como
  `game_error`, como as outras recusas da sessão.

### 3.4 A perícia do dano (decisão 2)

- **Domínio:** campo novo `Attack.DamageSkill enum.SkillName`. Zero (`""`) = `Push` — é o que toda
  action e toda linha antiga têm. `Attack` já é gravado inteiro em `actions.attack` (JSON), então o
  campo persiste sem migração.
- **Por que não `Attack.Damage.SkillName`:** é o campo que o **jogador** manda e o servidor
  descarta (contrato, linha 200); usá-lo deixaria o jogador escolher. E ele **vai ao wire** a todos
  (`rollCheck` em `actionwire/from.go` sempre mantém `SkillName`): um jogador que recarregasse com o
  turno aberto veria `Grab` antes do fechamento. `DamageSkill` não é mapeado em `actionwire`.
  **(D2, confirmar.)**
- **Resolução:** `actorPush` vira `actorDamageSkill(in, a)`: lê o valor da perícia
  `effectiveDamageSkill(a)` (`DamageSkill`, ou `Push` se vazio). `TurnResolution` ganha
  `DamageSkill string` — o nome efetivo, preenchido quando a action tem ataque, vazio sem ataque.
- **Edição:** `EditActionPayload.DamageSkill *string` (`json:"damageSkill,omitempty"`). O mapper
  valida com `enum.SkillNameFrom` (desconhecida → `invalid_action`, como as outras perícias).
  `MasterAction.DamageSkill *enum.SkillName`. Em `ApplyMasterAction`:
  - validação antes de qualquer mutação: a action alvo precisa ter `Attack` — senão
    `ErrNoDamageToMeasure` (`"damageSkill edit targets an action with no attack"`);
  - captura com campo `"damageSkill"`, origem **`OriginSystem`** (o `Push` é o motor que escolhe, não
    o jogador), atual = nome efetivo (`"Push"` quando vazio), novo = o nome mandado. Mandar `"Push"`
    de volta apaga a captura;
  - grava `target.Attack.DamageSkill = *ma.DamageSkill`.
  - `hasRollEdits` (`application/match/edit_action.go`) passa a contar `ma.DamageSkill != nil`.
- **Wire:** `ResolutionUpdatedPayload.DamageSkill string` (`json:"damageSkill,omitempty"`), topo do
  payload — é uma por ataque, não por alvo. Persistido em `resolutionRecord` (`damageSkill`) e
  devolvido pelo `GET /history` (`TurnResolutionResponse.damageSkill`). **Ausente** em turno sem
  ataque e em turno gravado antes deste PR (todos esses foram `Push`).
- **Projeção:** pública, como `rawDamage` (decisão 11). Com o turno aberto o payload inteiro já é só
  do mestre.

### 3.5 A condição atual vai ao mestre (decisão 3)

- **Domínio:** `TurnResolution.Conditions []CheckCondition`, onde
  `CheckCondition{ActionID uuid.UUID; Field string; SkillName string; Condition action.RollCondition}`.
  O resolvedor preenche **só quando `!IsSettled`**, percorrendo a action do turno e cada reação
  anexada (aberta ou não), em ordem: a action, depois as reações na ordem de `Turn.GetReactions()`.
  Em cada action, na ordem: `speed`, `feint`, `hit`, `damage`, `dodge`, `defense` (o `a.Defense`
  declarado — e, numa reação, `DefaultDefense`, que também se chama `defense`), `repel`,
  `moveSpeed`, depois cada entrada de `Skills` por `skillName`. Entra só rolagem com condição
  não-nula. Não há como uma action ter `a.Defense` editável **e** `DefaultDefense` ao mesmo tempo:
  numa reação com `KeepsDefault()` o campo `defense` sempre resolve para `DefaultDefense` (§3.2).
- **Por que no resolvedor:** o `match_full_state.resolution` do mestre já é `ResolveTurn(t)`, e o
  `resolution_updated` de toda edição também. Pôr as condições na resolução faz as duas superfícies
  carregarem a mesma coisa sem código novo em `room.go` — é o que sobrevive a recarregar (§0.2).
- **Wire:** `ResolutionUpdatedPayload.Conditions []ConditionPayload` (`json:"conditions,omitempty"`):

  ```json
  "conditions": [
    { "actionId": "3333…", "field": "hit", "bias": -1, "modifier": -2, "description": "escuridao" },
    { "actionId": "4444…", "field": "dodge", "bias": 1, "modifier": 0 }
  ]
  ```

  O mesmo formato de uma entrada de `edit_action.conditions`, mais `actionId` — **sempre o ID real**
  (a action do turno vem com o próprio ID, nunca zero). `field` e `skillName` são alternativos, como
  no `edit_action`. Ausente = nenhuma rolagem com condição.
- **Só o mestre, só com turno aberto:**
  - `ProjectResolution` zera `Conditions` para `!v.IsMaster` (defesa em profundidade: com o turno
    aberto o payload já só vai ao mestre, mas é o padrão de `PendingReactions`/`Errors`);
  - o resolvedor não preenche num turno fechado, então o `resolution_updated` liquidado não leva;
  - `encodeResolution` **não** grava (`resolutionRecord` não ganha o campo): as condições já estão na
    tabela `actions` (§1, decisão 9). O histórico não as devolve. **(D5, confirmar.)**

### 3.6 Condição zerada = sem condição (decisão 4)

Em `ApplyMasterAction`, no laço que aplica as condições: se `edit.Condition == (RollCondition{})`,
a rolagem volta a `Context.Condition = nil`, e a captura recebe **`nil` literal** (não um ponteiro
tipado nulo) como valor novo. `captureOverride` então:

- primeira edição zerada numa rolagem sem condição: `valuesEqual(nil, nil)` → não captura nada;
- edição zerada depois de uma edição de verdade: `valuesEqual(original nil, nil)` → **apaga** a
  captura.

A normalização mora no domínio (`ApplyMasterAction`), não no mapper: é a sessão que decide o que é
"o original", e os testes de domínio a cobrem sem passar pelo wire.

### 3.7 Contrato e documentos (decisão 5 e o resto)

**`match-combat-ws.md`:**
- `edit_action`: o parágrafo "Uma seção presente SUBSTITUI a lista inteira" passa a dizer que
  `skills` e `targetIds` substituem a lista inteira, e que **`conditions` se aplica por rolagem**:
  cada entrada substitui a condição daquela rolagem, as que não aparecem ficam como estão, e uma
  entrada zerada (`bias`, `modifier` e `description` ausentes ou zero) **limpa** — volta a "sem
  condição" e apaga a captura. Campo novo `damageSkill`, com exemplo. Tabela nova **"o que cada
  rolagem muda com o turno aberto"** (§3.1), incluindo quais são passivas e que `feint`,
  `speed` e o `moveSpeed` da própria ação não mudam o desfecho. `defense` numa reação que mantém a
  defesa padrão. Viés no dano recusado. Erros novos.
- Linha 200 (`attack.damage.skillName`): "Trocar Push por Grab é prerrogativa do mestre, ainda não
  implementada" → aponta para `edit_action.damageSkill`.
- `resolution_updated`: `damageSkill` e `conditions` no exemplo e na tabela de campos, com o eixo
  de visibilidade de cada um.
- `match_full_state`: a linha de `resolution` diz que ela traz `conditions` e `damageSkill` — é o
  que faz a edição sobreviver a recarregar.
- Tabela de erros: os dois `game_error` novos.

**`match-history.md`:** `resolution.damageSkill`.

**`combat-engine.md`, "A edição do mestre":** uma subseção curta "O que cada condição move" (a
tabela de §3.1), o portador `DefaultDefense` e por que não `Defense`, a perícia do dano, e a
condição zerada.

**Documento mestre (`2026-09-20-front-combat-phases.md`):**
- §0: estado da Fase 8 — back neste PR, front em seguida.
- §4.6: a troca Push → outra perícia existe (`edit_action.damageSkill`).
- §8 e §6A.6 F7: o painel fica completo, ao fim da Fase 8, com **edição de rolagem (viés, ajuste,
  motivo), troca da perícia do dano, escolha de onde cai o escape que falhou e dar a palavra às
  reações**. Editar **perícias** fica fora até existir a corrente de testes (§11.1), e o texto diz
  isso. A condição na `Evasion` das reações fechadas também fica fora do painel, pelo mesmo motivo.
- §11.1: a mesma nota, do lado da regra.
- Pendências novas (em §0, "depois da Fase 8", ou §11.5): **editar uma ação na fila antes de ela
  agir** (velocidade e o resto) — o `edit_action` só aceita o turno aberto; **a defesa e a esquiva
  passivas de um alvo que não reagiu** não têm onde guardar a edição.

**`documentation-map.yaml`:** a migração nova; `reaction_collision.go` e `turn_resolver.go` já
apontam para `combat-engine.md` — conferir as `notes`.

## 4. Resiliência (§0.2)

| Acontece | Resultado |
|---|---|
| **O mestre recarrega no meio de uma edição** | O `match_full_state.resolution` vem de `ResolveTurn(t)` sobre a action viva: números já editados **e** `conditions` + `damageSkill`. O editor abre com o que está valendo. Nada é reenviado |
| **Um jogador recarrega com o turno aberto** | Não recebe `resolution` (já é assim). O `openTurn.action` e `openTurn.reactions` passam por `actionwire`, que não mapeia `Context`, `DamageSkill` nem `DefaultDefense`: nada da edição chega a ele |
| **O servidor reinicia com o turno aberto** | O turno aberto se perde inteiro (já documentado, `match-combat-ws.md` §8): as edições vão junto, e as capturas também (são memória até o fechamento). Todo cliente fica sabendo pelo `match_full_state`. Ninguém reenvia, nada re-rola |
| **A conexão do mestre cai entre o `edit_action` e o `action_edited`** | A edição aplicou ou não, atômica (tudo é validado antes de mutar). Ao reconectar, o `match_full_state` mostra qual dos dois |
| **O turno fecha** | A action é gravada com as condições e `DamageSkill`; a reação com `default_defense`; as capturas em `overridden_action_values`; o cálculo liquidado em `turns.resolution` com `damageSkill` e sem `conditions` |

## 5. Quem vê o quê

| | Mestre, turno aberto | Jogador, turno aberto (dono incluído) | Todos, turno fechado | Banco |
|---|---|---|---|---|
| números recalculados | ✔ `resolution_updated` / `match_full_state` | ✗ | ✔ projetado | `turns.resolution` |
| `conditions` | ✔ | ✗ | ✗ | dentro de `actions` (cada `RollCheck`) |
| `damageSkill` | ✔ | ✗ | ✔ (onde `rawDamage` vai) | `actions.attack` e `turns.resolution` |
| `DefaultDefense` | via `conditions` (`field: "defense"`) | ✗ | ✗ | `actions.default_defense` |
| o valor descartado | ✗ (nenhuma superfície) | ✗ | ✗ | `overridden_action_values` |

## 6. Fora do escopo

- **Botão de perícias e a condição da `Evasion` fechada no painel** (decisão 1) — o back lê a
  condição da Evasion (§3.1), o painel não a oferece.
- **Editar ação na fila** (decisão 8) e **defesa/esquiva passivas de quem não reagiu** (decisão
  10) — pendências no documento mestre.
- **Resolução da finta** (§11.2).
- **Observado, não tratado:** as seções `targetIds` e `skills` do `edit_action` mudam a action viva,
  e `targetIds`/os nomes de `skills` **vão** ao wire — então um jogador que recarrega com o turno
  aberto provavelmente vê alvos editados pelo mestre antes do fechamento. Não verificado por teste.
  O front da Fase 8 não usa essas duas seções (perícias ficam fora; alvos não estão no escopo), então
  isso não trava nada; fica registrado para o autor do documento mestre decidir.

## 7. Verificação

- **Domínio** (`application/match/edit_action_test.go`, `service/*_test.go`,
  `matchsession/*_test.go`): cada leitura da §3.1 muda o total quando a condição muda; viés no dano
  recusado sem mutar nada; condição zerada apaga a captura; `damageSkill` mede o dano com a perícia
  nova, captura com `OriginSystem` e original `"Push"`, e voltar a `"Push"` apaga; `DefaultDefense`
  é criado na reação certa e recusado onde não cabe; a reserva fechada reflete a condição.
- **Wire, contra a `Room` real** (`internal/app/game/*_e2e_test.go`): o mestre recebe
  `conditions` e `damageSkill` no `resolution_updated`; o jogador não recebe nada do turno aberto;
  o mestre que **reconecta** recebe ambos no `match_full_state.resolution`; o jogador que reconecta
  não vê `damageSkill`/`defaultDefense`/condição em `openTurn`; fechado o turno, o
  `resolution_updated` liquidado leva `damageSkill` a todos e `conditions` a ninguém.
- **Postgres** (`-tags=integration`): `default_defense` gravado e `NULL` quando não houve edição;
  `attack` guarda `DamageSkill`; `turns.resolution` guarda `damageSkill` e não `conditions`; o
  `GET /history` devolve `damageSkill`.
- **Não verificado neste PR:** o caminho do usuário com as três contas, que é do PR de front. Por
  isso **não** rodar o `dev-checkout.sh` aqui — não há o que o dono do produto validar na mão sem a
  tela.

## 8. Decisões desta sessão, para o revisor confirmar

| # | Decisão | Por quê |
|---|---|---|
| D1 | A condição da defesa padrão mora num campo novo, `Action.DefaultDefense`, fora do wire, com coluna própria | `a.Defense` vai ao wire: reusá-lo contaria a edição antes do fechamento (§3.2) |
| D2 | A perícia do dano mora em `Attack.DamageSkill`, fora do wire | `Attack.Damage.SkillName` é o campo do jogador e vai ao wire (§3.4) |
| D3 | Dano bruto com piso zero depois do ajuste | Sem piso, a cadeia carregaria resíduo negativo; o resto do cálculo já faz piso (§3.3) |
| D4 | A condição no Reflexo/Evasion entra antes da reserva da esquiva fechada | A reserva é a diferença das leituras; a condição é parte da leitura (§3.1) |
| D5 | `conditions` não é gravado em `turns.resolution` | Já está na tabela `actions`; uma segunda cópia divergiria (decisão 9) |
| D6 | `damageSkill` é capturado com origem `system` | O `Push` é escolha do motor, não envio do jogador |
