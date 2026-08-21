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
esperar alguém decidir, esperar alguém somar dados. Aqui não existe fase de espera nem no
round 0. Nele os jogadores estudam o que fazer e configuram sua action (alvo, arma, etc.).
Os jogadores focam em imaginação, criatividade e interpretação; o sistema calcula em
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
- **O bônus do repelir e a penalidade do aparar são de actionSpeed, nunca de acerto.** Bônus
  é específico do alvo; penalidade é geral. ⚠️ Isso é do **acúmulo do duelo** — não é lei
  global. Outras reservas modificam outras coisas; ver `combat-engine.md` § Modificadores.

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
    Amount    int        // ajuste numérico
    Bias      int        // −1/0/+1 — vantagem/desvantagem NOS DADOS, acumulável
    Source    Source     // system | master — quem originou
    AgainstID *uuid.UUID // nil = vale contra qualquer um
    ExpiresAt Scope      // EndOfTurn | EndOfRound
    Reason    string     // para o Action History e para SystemData
}
```

⚠️ **`Bias` e `Source` são obrigatórios, não opcionais.** Sem `Bias`, o ledger não consegue
carregar a Desvantagem que o sistema gera (§4.4) — viés não é um número somável, é uma
mudança na forma de rolar. Sem `Source`, a auditoria não distingue o que o sistema pôs do que
o mestre pôs, que é o propósito inteiro do `SystemData`.

```go
type ResourceBar struct {
    Balance int   // saldo corrente
    Speeds  []int // velocidades roladas nesta rodada, para a média
}
```

A aritmética de fechamento (média, preço, teto) é da **Fase 3**; a Fase 1 só define a forma.

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

### 4.6 `MatchRules` — value object primeiro, persistência depois

Uma versão anterior deste spec colocava "o mecanismo de configuração de partida" na Fase 1 e
dizia que ele destravaria o `fog_mode` hardcoded. **Isso era uma contradição**: a Fase 1 se
declara pura e sem I/O, e destravar o `fog_mode` exige persistência *e* delivery. Resolvido
partindo em dois.

**Fase 1 — só a forma.** `MatchRules` é um value object imutável, com padrões embutidos,
**recebido por parâmetro** por quem precisa dele (o `RollCalculator`, à frente o resolver).
Não é global, não é lido de lugar nenhum, não persiste.

```go
type MatchRules struct {
    DiceSet      DiceSet   // 2D10 (padrão) | D20
    PassiveValue int       // DERIVADO do DiceSet: 11 para 2D10, 10 para D20
    LadderStep   int       // 10
    ReactionTimer *time.Duration // nil = desligado
    DefaultReactions bool  // true
    FogMode      *fog.FogMode   // nil = herda do mapa — ver abaixo
}
```

`PassiveValue` **nunca é digitado à mão**: é derivado do `DiceSet`, senão trocar os dados
descalibra todos os testes passivos silenciosamente. Uma sobrescrita explícita pode existir
depois.

**Fatia própria, mais tarde — persistência + REST.** Onde mora (coluna JSONB em `matches`?
tabela nova? herança da campanha?), o endpoint para o mestre escolher, e o desbloqueio do
`fog_mode` em `room.go`. Não bloqueia nenhuma fase do motor: enquanto não existir, o
construtor usa os padrões.

#### `fog_mode`: o mapa é o padrão, a partida sobrepõe

Hoje `fog_mode` está persistido em **`maps.fog_mode`** (migration `20260616000000`) — nível
de mapa. O `AGENTS.md` diz que será configuração **de partida**. São escopos diferentes, e é
tentador achar que um tem que perder.

**Não precisa.** `MatchRules.FogMode` é um ponteiro: `nil` significa *"use o do mapa"*. A
resolução fica num lugar só:

```
fogMode = matchRules.FogMode ?? mapa.FogMode ?? explored
```

Um mapa é reutilizável entre partidas; o estilo de névoa é de **como esta mesa quer jogar**,
não do desenho. Sobreposição honra a intenção de produto sem orfanar os dados que já existem
nem forçar toda partida a declarar o campo.

### 4.7 Dano

**Duas famílias de rolagem, não uma.** O teste (acerto, perícia, actionSpeed) usa
`MatchRules.DiceSet` — 2 D10. O **dano usa os dados da própria arma**: `item.Weapon.dice`
(Espada = D10 + D4) mais o bônus fixo `Weapon.damage`. O `RollCalculator` da Fase 1 só conhece
o conjunto de teste; o caminho do dano é outro e nasce na Fase 2.

```
bruto = dados da arma + Weapon.damage

efetivo = bruto                              // se o alvo NÃO defendeu
efetivo = max(0, bruto − defesa aplicável)   // se defendeu com sucesso
```

**A subtração é condicional.** Só se subtrai defesa se o alvo **conseguir defender**. Não é
redução automática.

#### O que compõe a "defesa aplicável"

| Fonte | Regra |
|---|---|
| **Arma × arma** | Defender um ataque armado **com arma** → **não passa dano** |
| **Ataque desarmado** | Só ataques desarmados geram dano através da defesa |
| **Armado × defesa desarmada** | A defesa **não tem eficácia** contra perfurante ou cortante. Só funciona contra **concussivo** |
| **Armadura** | Subtrai também — mas **o mestre controla se o ataque "bate ou não" na armadura** |
| **Nen** | Vai reduzir o dano final. **Não existe ainda e não entra na conta.** |

> ⚠️ **Aparar (defesa armada) existe desde o início — não é provisório.** O que vem depois do
> MVP são as complexidades e os detalhes dela, não a mecânica. Implementar a forma inicial e
> deixar comentário no código dizendo isso: *é a versão inicial, será enriquecida*, e não
> *é temporária, será substituída*. Não inventar degraus que ainda não foram definidos.

#### A margem do acerto **não** entra no dano

Decisão do dono do produto: *"não somará no dano, pelo menos não por enquanto, porque esse
sistema já é muito punitivo e apelativo."*

> **Registrar em comentário no código:** somar a margem ao dano **será estudado nos testes
> pós-MVP**. É a única regra do sistema em que a margem não circula, e isso é deliberado.

#### Dry-run: calcular sempre, aplicar uma vez

**Toda colisão produz a redução de HP projetada desde o começo, para o mestre ver — sem tocar
na ficha.** É um *dry-run*. O `TurnResolution` carrega o dano projetado.

**A aplicação de verdade acontece no fechamento do turno.** Isso resolve o problema de
recalcular a cada reaction sem aplicar dano várias vezes: calcula-se quantas vezes for
preciso, aplica-se uma só.

> Convém: o fechamento do turno **já acontece hoje**, implicitamente, dentro de
> `OpenNextAction`/`PullAction`. A Fase 2 se pendura nele e **não precisa** do encerramento
> explícito, que é da Fase 5.

#### Visibilidade do resultado

No fechamento, **o dano pode ser mostrado a todos. O HP do alvo, não** — HP é dado privado de
cada personagem.

#### Em aberto

**Crítico e erro crítico não fazem nada no dano ainda.** O motor da Fase 1 já sinaliza os
dois e nada os consome. Pelas notas de design, crítico tem consequência **narrativa** (perder
um olho, ser decepado, morrer — ver Evasão em `reacoes.md`), não um multiplicador. A regra não
existe; **fora de escopo da Fase 2**, e o resolver deve deixar a flag passar intacta sem
interpretá-la.

## 5. As fases

Cada fase é uma sessão e um PR. A ordem foi escolhida para haver **algo visível rodando na
Fase 2**, em vez de esperar até o fim.

---

### Fase 1 — Fundação

**Objetivo:** criar o estado e o motor de rolagem. Puro, sem I/O, testável sem partida.

**Escopo:**
- `CharacterStatus` conforme §4.1, com `ResourceBar` e `ModifierLedger`.
- Re-chavear `charSheets` para `sheetUUID` e **tornar a sessão capaz de segurar NPCs** (§4.2).
  Ajustar `InitMatchSessionUC` e os dois construtores de `MatchSession`.
- Corrigir `MatchSession.CategorizeTarget`, que **já está errado hoje**: compara `TargetID`
  (que é o `sheetUUID` da peça) contra `participants`, chaveado por `playerUUID`. Passa a
  consultar o mapa novo.
- `RollCalculator`: 2 D10, crítico e erro crítico pela combinação, passivo vs. ativo,
  vantagem/desvantagem acumulável. Retorna um `RollOutcome` com **os dados individuais**, o
  total e as flags de crítico. **A margem é derivada** — `outcome.Margin(cd int)` —, porque a
  CD vem da rolagem oposta, que só existe na Fase 2.
- `MatchRules` como **value object em memória** (§4.6), com os padrões embutidos, recebido
  por parâmetro. Sem persistência, sem REST.

**Fora de escopo:**
- Ninguém chama o `RollCalculator` ainda.
- A aritmética de fechamento das barras (média, preço, teto) — é da Fase 3.
- **Persistência e REST de `MatchRules`, e o desbloqueio do `fog_mode`** — ver §4.6.
- **Como um NPC entra em `match_participants`** — ver §7.

**Pronto quando:**
- Testes unitários cobrem crítico, erro crítico, passivo, vantagem, desvantagem acumulada e
  `Margin(cd)`.
- Um teste constrói uma sessão com um participante de `PlayerUUID == nil` e assere que a
  ficha e o `CharacterStatus` dele estão presentes e alcançáveis por `sheetUUID`.
- `CategorizeTarget` devolve `TargetKindCharacter` para o `sheetUUID` de uma peça.
- `go vet -tags=integration ./internal/...` passa.

> ⚠️ O critério **não** é "um NPC aparece numa partida real": nada hoje cria um. Ver §7.

---

### Fase 2 — Primeira colisão de verdade

**Objetivo:** um ataque contra um personagem resolvendo ponta a ponta. **Primeira coisa
visível no browser.**

**Escopo:**
- `buildAction` mapeando o payload **inteiro** (Skills, Speed, Feint, Attack, Defense) — hoje
  descarta quase tudo. Inclui a fronteira `string → enum.SkillName`, rejeitando nome inválido
  com erro de WS.
- **`actorID` passa a ser `sheetUUID`**: entra no `ActionPayload`, `buildAction` o coloca em
  `Action.actorID`, e a autorização em `EnqueueAction` vira
  `charToPlayer[actorCharID] == playerUUID`. Ver `flows/05-lacunas.md` §3.
- **Esquiva por reflexo e defesa passivas** (`Reflexo + 11`; defesa em CD − 10). **Elas sobem
  para cá**, e a razão é estrutural: sem rolagem oposta não existe CD, sem CD não existe
  margem, e sem margem não há colisão nenhuma — o nome da fase deixaria de fazer sentido. A
  Fase 4 fica com as reações **ativas**.
- `TurnResolver`, ramo `TargetKindCharacter`: chamar o `RollCalculator`, produzir
  `ActionResult`, popular `Blows` (dar construtor e acessores a `battle.Blow`).
- **A escada de margem como função pura**, sem nenhuma reação ativa ligada nela. A Fase 4 liga
  o repelir; aqui ela só existe e é testada isoladamente.
- **Dano** conforme §4.7: dados da arma, subtração condicional, dry-run no `TurnResolution`,
  aplicação no fechamento do turno via `sheet.Repository.UpdateStatusBars`.
- `OpenNextAction`/`PullAction` passam a resolver **pelo resolver da sessão, com as fichas**,
  em vez de `service.TurnResolver{}.Resolve(opened, nil, session)`. Ver `05-lacunas.md` §9.
- Onde ficam os dados sorteados: `RollCheck` ganha os `RollAttempts`, com o tipo movido para o
  pacote `action` (`service` importa `action`, nunca o contrário).
- **Seam de rolagem determinística.** `RollCalculator.Derive` já é testável — recebe os dados
  prontos. Mas `Roll` chama `die.NewDie(s).Roll()` direto, sem ponto de injeção, então tudo
  que passa por ele é irreprodutível. As Fases 3 e 4 têm critérios de pronto com números
  exatos (a economia do round, a cadeia de vários alvos) que **não podem depender de sorte**.
  Introduzir aqui o ponto de injeção — quem chama `Roll` recebe a fonte, e os testes passam
  uma determinística.
- `resolution_updated` com payload de verdade, **mantido master-only**. Difusão para a mesa e
  projeção por destinatário são da Fase 5 — antecipar aqui vazaria o cálculo que só o mestre
  pode ver antes do encerramento. O payload é um **recorte**, não o `TurnResolution` inteiro.

**Fora de escopo:**
- Barras e economia de turno (Fase 3); reações **ativas** e vários alvos (Fase 4).
- Encerramento **explícito** de turno e projeção por destinatário (Fase 5).
- Evento WS de HP e qualquer trabalho de front (Fase 6).
- Efeito de crítico e erro crítico no dano — a regra não existe (§4.7).

**Pronto quando:**
- Um ataque enviado pelo browser, contra um alvo que não reage, produz dano; o mestre vê a
  projeção **antes** do fechamento e o HP só muda **depois** que o turno fecha.
- Recarregando a página, a sidebar mostra o HP novo. **Sem evento WS e sem tocar no front** —
  a sidebar já lê HP do REST.
- O `TurnResolution` carrega dados individuais, total, flags de crítico e a margem derivada.
- A escada de margem tem teste unitário próprio, desacoplada de qualquer reação.

---

### Fase 3 — A economia do turno

> ✅ **Implementada** (2026-08-18). O comportamento final — incluindo os pontos que
> divergiram deste plano — está registrado em `docs/dev/match/combat-engine.md` § "O que a
> Fase 3 fixou no motor", que passa a ser a fonte de verdade a partir de agora.

**Objetivo:** o turno vira dinâmico.

**Escopo:**
- actionSpeed real alimentando `Action.Speed.Result` — a fila passa a ter prioridade.
  **Perícia base: `Legerity`.**
- **moveSpeed** — e aqui **a perícia não é fixa**: ela vem da **categoria do movimento**.

  | Movimento | Perícia | Rola? |
  |---|---|---|
  | **Dash** (arrancada) | `Accelerate` | sim — `Accelerate + 2 D10` |
  | **Shift** (controlado) | `Brake` | **não** — usa o valor passivo |

  O mapper lê `Move.Category` e daí tira **qual perícia e se rola**. Isso puxa um pedaço
  mínimo do sistema de movimento para a Fase 3 — inevitável, porque sem ele a segunda barra
  não tem número — mas **é pouco**, não o sistema de movimento inteiro.

  **A escolha é do jogador, e é de movimento, não de perícia.** O front expõe os tipos de
  movimento tático explicitamente; trocar Dash por Shift na bottom sheet **troca a perícia
  sozinho**. O jogador nunca escolhe perícia de movimento à mão. É o mesmo gesto já descrito
  para o escape: a interface mostra que ele está fazendo um dash, ele troca ali para shift, e
  a perícia acompanha.

  ⚠️ **Isso vale só para o início do movimento.** Depois que o personagem está em movimento,
  quem alimenta a velocidade da move action é a **`Speed`** acumulada em
  `CharacterStatus.Velocity` — não mais o teste. Mecânica completa (transições Shift↔Dash,
  teste de `Brake` automático, `Charge` acumulando) em
  **`docs/dev/match/combat-engine.md` § Movimento**.

  > ❓ Decidir se o momentum (`Charge`) entra na Fase 3 ou fica para a fatia de movimento.
  > **A barra funciona sem ele** — só fica menos rica.

  > ✅ **`enum.Velocity` virou `Quickness`**, no primeiro commit desta fase — por decisão do
  > dono do produto, que preferiu não esperar um PR à parte. `action.Velocity`, o vetor, não foi
  > tocado. Detalhe em `combat-engine.md`.
- As duas barras, com preço por barra, média das velocidades, carry-over e teto.
- Recalculação forward-only da posição quando a média muda.
- Action enviada no meio do round entra com a velocidade rolada, **sem reordenação
  retroativa**.
- **`RoundMode.Race` alcançável**: o regime existe e pode ser ligado. Ver a nota abaixo.
- **Fim de round quando nenhuma action pendente passa no porteiro que lhe cabe** — não
  "quando as barras acabam" (a barra não zera; termina em qualquer valor, débito incluso).
  `docs/dev/match/combat-engine.md` é a fonte para o predicado exato; qualquer divergência
  resolve-se por lá, como já vale para as ações compostas. E portanto **`CloseRoundUC`
  plugado aqui**, não na Fase 5. Hoje ele existe e nada o chama; sem ele o round não fecha e a
  fase não entrega o próprio objetivo. `round_closed` passa a ser emitido (hoje é declarado e
  nunca enviado).
- Ações compostas: **uma action só**, com `Move` e `Attack` preenchidos, que **cobra as duas
  barras** e acontece **no tempo da mais lenta** (`min` das duas chaves). Não se divide em duas
  actions e não abre dois turnos. Forma exata em `combat-engine.md` § Ações compostas, que é a
  fonte; qualquer divergência resolve-se por lá.

> **`Race` sem iniciativa — separar o regime da regra que o liga.**
>
> **Tudo o que este spec descreve sobre economia de turno é comportamento de `Race`.** A
> barra, o preço, a média das velocidades, agir de novo, o carry-over — foi tudo desenhado
> olhando para o turno disputado. **O turno dinâmico em `Free` funciona de outro jeito e
> ainda não foi desenhado.**
>
> Por isso o `Race` precisa ser alcançável aqui: sem ele, a fase implementaria uma economia
> que não tem regra escrita. A separação é entre **o regime** (`Race` entra na Fase 3,
> ligável pelo mestre, rolando em vez de usar o passivo) e **a regra de jogo que normalmente
> o liga** (iniciativa, depois).
>
> ⚠️ Uma versão anterior justificava isso dizendo que em `Free` "todo mundo empata em
> `perícia + 11`". **Isso estava errado** — cada ficha tem o seu valor de perícia, então os
> totais diferem e a ordenação funciona. O motivo real é o de cima.
>
#### O que o `Free` faz

**Free existe para os jogadores terem liberdade sem o mestre aprovar cada gesto** — mover a
peça, abrir uma porta, investigar um local, pegar um item do chão. Não há disputa para ver
quem age primeiro porque não há batalha acontecendo.

**Mas não é liberdade sem trava.** Senão um jogador fica movendo a peça indiscriminadamente.

> **A partir da terceira ação do personagem no round, a action cai na fila** para o mestre
> liberar — ou não.

Duas notas que mudam a implementação:

- **Ações ofensivas não contam nessa contagem.** Qualquer ataque **dispara o fluxo de
  iniciativa**: já está na fronteira do `Race`, e sai do regime livre por definição.
- **A trava não é só anti-abuso.** Vários jogadores podem querer agir "ao mesmo tempo", e é
  aí que o mestre pode **ligar o `Race`** para haver disputa de ação — mesmo fora de batalha.
  A fila em `Free` é o mecanismo que dá ao mestre a chance de perceber isso e decidir.

**Em `Free` o movimento é normalmente Shift**, que não rola dado — coerente com não haver
disputa.

⚠️ **A economia de barra descrita neste spec é do `Race`.** Em `Free` o comportamento é o de
cima: ação livre até a terceira, fila a partir dali. Não há preço de round, média nem
carry-over enquanto o `Race` não estiver ligado.

**Fora de escopo:** iniciativa como regra; movimento detalhado; posturas.

- **Só `Dash` e `Shift` são aceitos.** `enum.MoveCategory` tem 7 valores; as outras cinco —
  `Back` (cait), `Roll`, `Slide`, `Jump`, `FlatJump` — respondem **erro de WS "categoria ainda
  não suportada"**. As perícias delas se definem na fatia de movimento, que é onde elas serão
  de fato exercitadas. **Não mapear por analogia**: tratar tudo que não é Dash como Shift
  funcionaria silenciosamente errado — um salto custaria como um passo controlado, e ninguém
  descobriria até alguém reclamar na mesa.
- **`Charge` fica fora.** O momentum acumulando na `Speed` é da fatia de movimento. A barra
  funciona sem ele.
- **O segundo regime de velocidade fica fora** — aquele em que, com o personagem já em
  movimento, a `Speed` acumulada alimenta a move action no lugar do teste. Ele **depende** de
  saber quando a `Velocity` é setada e quando decai, e essa regra não existe. Na Fase 3 vale
  só o primeiro regime: toda move action ou rola (`Dash`) ou usa o passivo (`Shift`).
- **A mecânica do modo `Free` sai desta fase.** A Fase 3 implementa a economia do `Race` —
  que é a única com regra escrita. A trava da terceira ação em `Free`, e o que acontece com
  as duas primeiras, viram **fatia própria**. Enquanto isso, a fase exige que o mestre ligue
  o `Race` para a economia valer.
- **A renomeação `enum.Velocity` → `Quickness` entrou nesta fase**, no primeiro commit, antes
  de qualquer código de motor — decisão do dono do produto, revertendo o "PR separado" que
  este spec pedia. Fica isolada no próprio commit para o diff continuar legível. A metade do
  front (três chaves `velocity`) é PR próprio no repo React, com cross-link.
- **O evento WS que carrega as barras nasce aqui.** A Fase 6 precisa desenhar "a sua barra e
  a barra geral", e hoje nenhuma mensagem as transporta. Ele é da Fase 3 porque é aqui que as
  barras passam a existir — não adianta a Fase 6 descobrir que não tem de onde ler.

**Pronto quando:**
- O exemplo canônico da doc de jogo (p1=20, p2=23, p3=11, segunda rolagem de p2 = 17)
  reproduz a ordem `p2 → p1 → p3 → p2` e os saldos `+9 / 0 / −2` em teste automatizado, com
  **as rolagens injetadas** (ver o seam da Fase 2) — um teste de economia não pode depender
  de sorte.
- Um round fecha sozinho quando nenhuma action pendente passa no porteiro que lhe cabe — não
  "quando as barras acabam" (`docs/dev/match/combat-engine.md` § "Quando o round fecha" é a
  fonte) —, e `round_closed` chega aos clients.

---

### Fase 4 — Reações

> ✅ **Implementada** (2026-08-20). O comportamento final — incluindo os pontos que
> divergiram deste plano — está registrado em `docs/dev/match/combat-engine.md` § "O que a
> Fase 4 fixou no motor", que passa a ser a fonte de verdade a partir de agora.

**Objetivo:** o catálogo completo e a cadeia de resolução.

**Escopo:**
- Reações **ativas**: escape, escape defensivo, esquiva fechada, escape fechado, repelir, não
  fazer nada. Custo por barra conforme a tabela em `combat-engine.md`.
  > As **passivas** (esquiva por reflexo → defesa) já vieram na Fase 2 — sem elas não haveria
  > CD nem colisão lá. Aqui elas só ganham a companhia das ativas e o desfecho "não fazer
  > nada", que recusa até os padrões.
- Ligar o **repelir** na escada de margem que a Fase 2 escreveu como função pura.
- Conversão action→reaction com **Desvantagem** (pior das duas), não média.
- **Abrir reaction como operação de primeira classe** — mesmo ciclo da action. **Vem para cá,
  não fica na Fase 5**: a cadeia com vários alvos *é* o mestre abrindo uma reaction por vez, e
  a ordem de abertura muda o resultado. Sem essa operação, a fase não consegue entregar o
  próprio objetivo nem provar o critério de pronto.
- **Resolução em cadeia com vários alvos**: o estado do ataque sai alterado de cada resolução
  e entra na próxima.
- Validar que só alvos podem reagir (hoje qualquer um pode).
- Timer de reação como **número na regra de partida**, sem relógio: com o padrão desligado,
  encerrar o turno já é o estouro. A contagem visível é da Fase 6.
- `ReactionKind` declarado no envio, e o custo por barra saindo dele — não do formato.

**Fora de escopo:** posturas; encerramento explícito de turno e projeção por destinatário
(Fase 5).

**A Fase 4 roda com personagens de jogador.** O critério de pronto — três alvos reagindo
diferente — é alcançável com três PCs. O que o rostering de NPCs trava é **o mestre enviar
ação de NPC**, que não é objetivo desta fase. Fatia própria, antes da Fase 5.

### O catálogo precisa de um tipo declarado

Detalhe completo em `combat-engine.md` § *O tipo da reação é declarado, não inferido*. O
resumo executável:

- `action.ReactionKind` com sete valores (`nothing`, `dodge`, `closedDodge`, `escape`,
  `escapeGuard`, `closedEscape`, `repel`), como **campo de `Action`**, não struct aninhada. O
  discriminador "isto é uma reação" continua sendo `ReactToID != uuid.Nil`; a validação exige
  os dois juntos ou nenhum.
- `action.Repel{Weapon *enum.WeaponName; RollCheck}`, no molde de `Defense`.
- `ReactionKind.Bars()` devolve o custo por tabela; `Action.Bars()` consulta o tipo primeiro.
  **`Bars()` passa a poder ser vazio** — só para reações, e nenhum caller quebra.
- `enum.DodgeCategory` é **removida**, absorvida pelo `ReactionKind`. Raio de alcance: 3
  arquivos, um deles passthrough (`action_mapper.go:107`).
- Wire: o tipo é campo do payload de reação, camelCase, e é **obrigatório** — o servidor nunca
  infere custo do formato do que chegou.

### O custo da reação — as quatro decisões

Detalhe e razão em `combat-engine.md` § *O custo da reação na economia de barra*.

| | Decisão |
|---|---|
| Velocidade registrada | reação que cobra barra passa por `deriveSpeeds` e registra a velocidade que ela rolou, em cada barra que cobra; a grátis não registra nada |
| Porteiro | **não se aplica** — reação nunca é negada por falta de saldo, só fica devendo |
| Momento da cobrança | no **attach**, não no open (é onde a colisão já é calculada) |
| Action pendente | sai da `activeQueue` — a de melhor chave naquela barra; reação grátis não consome nada |

Duas regras de jogo derivadas, não ditadas, e sinalizadas como tais no doc: **repelir também
abre mão das passivas**, e o **timer não precisa de relógio nesta fase** (encerrar o turno *é*
o estouro enquanto o padrão for desligado).

### Consertos em código já escrito que a Fase 4 carrega

- `match.Scope` ganha o degrau que falta: `end_of_turn` mata no fim do **próprio** turno, e o
  bônus do repelir vale **no próximo**.
- `CharacterStatus.ExpireModifiers` **não tem caller** — nada expira hoje. Ligar no fechamento
  de turno e de round.
- `Modifier` ganha `Applies Dimension`; `AgainstID *uuid.UUID` vira `Against` com três formas
  (§4.3 e `combat-engine.md` § *Modificadores*).
- O comentário de `RollInput.Ledger` ainda carrega a invariante generalizada demais
  (*"always an actionSpeed adjustment"*). Quem decide a dimensão passa a ser `Modifier.Applies`.

### Onde mora o viés de uma rolagem só

A Desvantagem da conversão action→reaction **não vai para o `ModifierLedger`**. O ledger é do
personagem e vale até expirar — jogá-la lá aplicaria desvantagem a **todas** as rolagens de
actionSpeed dele, não só à conversão. E `RollCondition.Bias` é exclusivo do mestre (§4.4).

Falta o terceiro lugar, e ele é o mais óbvio: **o viés de uma rolagem só mora na própria
rolagem.** `RollInput` ganha um campo de viés de origem sistêmica, somado junto com o do
mestre e o do ledger no momento de derivar. Três origens, três lugares, nenhuma
sobrescrevendo a outra:

| Origem | Onde mora | Alcance |
|---|---|---|
| Mestre | `RollCondition.Bias` | aquela rolagem |
| Sistema, situacional | **`RollInput`** ← novo | aquela rolagem |
| Sistema, acumulado | `ModifierLedger` | até expirar |

**Pronto quando:** teste de integração cobre um ataque em área com três alvos reagindo
diferente (um reaction ativa, um "não fazer nada", um sem resposta), e **abrir na ordem
inversa produz resultado diferente de forma verificável** — com as rolagens injetadas.

---

### Fase 5 — Regência e visibilidade

**Objetivo:** a mesa funcionando.

**Escopo:**
- Encerrar turno **explicitamente**, com diálogo de confirmação quando houver reactions
  enviadas e não abertas — elas entram no cálculo, mas perdem o momento de narrar. Até aqui o
  turno fecha implicitamente dentro de `OpenNextAction`/`PullAction`.
- Visibilidade por campo, via projeção por destinatário (§4.5): mecânica pública, cálculo só
  do mestre até o encerramento. Inclui a regra do §4.7 — **dano é público, HP não**.
- `resolution_updated` deixa de ser master-only e passa a ser projetado por destinatário.
- Notificação de action enfileirada **só para o mestre**.
- **Action History como superfície de jogo** — inclui o **caminho de leitura**, que não
  existe: os turnos são persistidos por `PersistTurnClose`, mas nenhum endpoint os devolve.

  ⚠️ **O que é gravado ainda não basta.** O caminho de escrita foi consertado no fim da Fase 4
  (FK para `character_sheets`, reações gravadas, `reaction_kind`/`repel`), mas ele grava só a
  **declaração** — nem dano, nem margem, nem desfecho. Recalcular depois é impossível: o
  `ModifierLedger` daquele instante não existe mais. **Dar forma ao resultado persistido é
  trabalho desta fase**, e vem antes da consulta.
  Precisa da consulta e da projeção por campo em cima dela.

  **A resposta é aninhada, não uma lista plana.** As cenas são os blocos lógicos que
  organizam a partida, e o histórico tem que sair dentro deles — o front renderiza cards de
  action **dentro do escopo de cada cena**. A hierarquia do domínio (Cena → Round → Turno →
  Action) é a mesma da resposta; não achatar.
- Tabela `SystemData` — auditoria de toda interferência do mestre.

> `CloseRoundUC` e `round_closed` **saíram daqui** — foram para a Fase 3, que precisa deles
> para fechar o round quando nenhuma action pendente passa no porteiro que lhe cabe — não
> "quando as barras acabam" (`docs/dev/match/combat-engine.md` é a fonte para o predicado
> exato). `Abrir reaction` foi para a Fase 4.

**Pronto quando:**
- **Dois clients WS** conectados como jogadores diferentes recebem, para o mesmo turno,
  payloads distintos — um com o campo oculto, outro sem. Verificável no backend, **sem
  depender do front**, que é da Fase 6.
- O Action History devolve um turno fechado com os campos ocultos respeitados.
- Uma edição do mestre aparece em `SystemData` com `Source: master`.

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

**Pronto quando:** um jogador compõe e envia uma action pela bottom sheet, vê o balão subir
quando o mestre a abre, reage clicando num botão ao lado do seu personagem, e acompanha as
duas barras — tudo sem recarregar a página. É a primeira vez que o loop inteiro é jogável de
ponta a ponta por uma pessoa, e é o critério que fecha a iniciativa.

---

## 6. Riscos

| Risco | Mitigação |
|---|---|
| **`CharacterStatus` com a forma errada** — toca barras, ledger, postura, velocidade; errar cascateia | Desenhá-lo por inteiro na Fase 1, com os campos reservados já previstos, mesmo sem uso |
| Re-chavear `charSheets` quebra o fog de guerra | `charToPlayer` continua sendo a ponte; a suíte de fog (`fog_*_test.go`) é a rede de segurança |
| `MatchSession` não tem lock próprio | `room.go` é a única serialização — todo `Execute` novo herda a obrigação de segurar `r.mu` |
| Regras mudarem depois do MVP | Números viram configuração de partida; a forma da escada fica em código |

## 7. Pontas soltas conhecidas

### ⛔ Bloqueador conhecido: não existe caminho para criar um NPC

`start_match.go` popula `match_participants` **apenas a partir de enrollments aceitas**. A
Fase 1 torna a sessão *capaz* de segurar um NPC — mas **nada no sistema cria um**.

- **Não bloqueia** as Fases 1 a 4: elas operam sobre personagens de jogador. A Fase 4
  entregou o catálogo de reações inteiro sem tocar rostering — ver `combat-engine.md` § "O
  que a Fase 4 fixou no motor".
- **Bloqueia a Fase 5 em diante**, onde o mestre precisa enviar ações de NPCs.

Precisa virar fatia própria antes da Fase 5, e tem componente de produto: como o mestre
adiciona um NPC à partida? O desenho fala em *"o mestre adiciona NPCs na primeira cena"* e
*"mestre pode gerenciar adicionando e removendo personagens a qualquer momento"* — o que
sugere um caminho de rostering sem enrollment, provavelmente com fichas de `MasterUUID`
preenchido e `PlayerUUID` nulo.

### Outras pontas soltas

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
